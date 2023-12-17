/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/serdarkalayci/redis-cluster-operator/internal/kubernetes"
	redis_internal "github.com/serdarkalayci/redis-cluster-operator/internal/redis"
	"github.com/serdarkalayci/redis-cluster-operator/internal/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	redisclusterv1alpha1 "github.com/serdarkalayci/redis-cluster-operator/api/v1alpha1"
)

// RedisClusterReconciler reconciles a RedisCluster object
type RedisClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=rediscluster.kuro.io,resources=redisclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rediscluster.kuro.io,resources=redisclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=rediscluster.kuro.io,resources=redisclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RedisCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *RedisClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Reconciling RedisCluster", "cluster", req.Name, "namespace", req.Namespace)

	//region Try to get the RedisCluster object from the cluster to make sure it still exists
	redisCluster := &redisclusterv1alpha1.RedisCluster{}
	err := r.Client.Get(ctx, req.NamespacedName, redisCluster)

	if err != nil {
		if apierrors.IsNotFound(err) {
			// The RedisCluster resource is not found on the cluster. Probably deleted by the user before this reconciliation. We'll quit early.
			logger.Info("RedisCluster not found during reconcile. Probably deleted by user. Exiting early.")
			return ctrl.Result{}, nil
		}
	}
	//endregion

	//region Try to get the ConfigMap for the RedisCluster
	configMap, err := kubernetes.FetchConfigmap(ctx, r.Client, redisCluster)
	if err != nil && !apierrors.IsNotFound(err) {
		// This is a legitimate error, we should log the error and exit early, requeuing the reconciliation
		logger.Error(err, "error checking the existence of ConfigMap")
		return ctrl.Result{
			RequeueAfter: 30 * time.Second,
		}, err
	}
	if apierrors.IsNotFound(err) {
		// We successfully checked but the ConfigMap does not exist. We need to create it.
		configMap, err = kubernetes.CreateConfigMap(ctx, r.Client, redisCluster)
		if err != nil {
			// Error creating the Configmap, we should log the error and exit early, requeuing the reconciliation
			logger.Error(err, "error creating ConfigMap for RedisCluster")
			return ctrl.Result{
				RequeueAfter: 30 * time.Second,
			}, err
		}

		// We've created the ConfigMap, we'll requeue the reconciliation in 5 seconds to give the ConfigMap time to be created
		logger.Info("Created ConfigMap for RedisCluster. Reconciling in 5 seconds.")
		return ctrl.Result{
			RequeueAfter: 5 * time.Second,
		}, err
	}
	//endregion

	//region Try to set ConfigMap owner reference
	err = retry.RetryOnConflict(wait.Backoff{
		Steps:    5,
		Duration: 2 * time.Second,
		Factor:   1.0,
		Jitter:   0.1,
	}, func() error {
		configMap, err = kubernetes.FetchConfigmap(ctx, r.Client, redisCluster)
		if err != nil {
			// At this point we definitely expect the statefulset to exist.
			logger.Error(err, "error finding configMap")
			return err
		}
		err = ctrl.SetControllerReference(redisCluster, configMap, r.Scheme)
		if err != nil {
			logger.Error(err, "error setting owner reference of ConfigMap")
			return err
		}
		err = r.Client.Update(ctx, configMap)
		if err != nil {
			logger.Error(err, "error updating owner reference of ConfigMap")
		}
		return err
	})
	if err != nil {
		logger.Error(err, "error setting/updating owner reference of ConfigMap")
		return ctrl.Result{
			RequeueAfter: 10 * time.Second,
		}, err
	}
	//endregion

	//region Ensure Statefulset
	masterSSet, replSSets, err := kubernetes.FetchExistingStatefulsets(ctx, r.Client, redisCluster)
	if (err != nil && !apierrors.IsNotFound(err)) {
		// We've got a legitimate error, we should log the error and exit early
		logger.Error(err, "Could not check whether statefulset exists due to error.")
		return ctrl.Result{
			RequeueAfter: 30 * time.Second,
		}, err
	}

	if apierrors.IsNotFound(err) {
		// We need to create the Statefulset
		masterSSet, replSSets, err = kubernetes.CreateStatefulsets(ctx, r.Client, redisCluster)
		if err != nil {
			logger.Error(err, "Failed to create Statefulset for RedisCluster")
			return ctrl.Result{
				RequeueAfter: 30 * time.Second,
			}, err
		}

		// We've created the Statefulset, and we can wait a bit before trying to do the rest.
		// We can trigger a new reconcile for this object in about 5 seconds
		logger.Info("Created Statefulset for RedisCluster. Reconciling in 5 seconds.")
		return ctrl.Result{
			RequeueAfter: 5 * time.Second,
		}, err
	}
	//endregion

	//region Set Statefsulset owner reference
	masterSSet, replSSets, err = kubernetes.FetchExistingStatefulsets(ctx, r.Client, redisCluster)
	if err != nil {
		// At this point we definitely expect the statefulset to exist.
		logger.Error(err, "Cannot find statefulsets")
		return ctrl.Result{
			RequeueAfter: 5 * time.Second,
		}, err
	}
	err = retry.RetryOnConflict(wait.Backoff{
		Steps:    5,
		Duration: 2 * time.Second,
		Factor:   1.0,
		Jitter:   0.1,
	}, func() error {
			err = ctrl.SetControllerReference(redisCluster, masterSSet, r.Scheme) // ToDo: This is a hack. We should update all statefulsets
			if err != nil {
				logger.Error(err, fmt.Sprintf("Could not set owner name for master statefulset named %s", masterSSet.Name))
				return err
			}
			err = r.Client.Update(ctx, masterSSet) // ToDo: This is a hack. We should update all statefulsets
			if err != nil {
				logger.Error(err, fmt.Sprintf("Could not update master statefulset named %s  with owner reference", masterSSet.Name))
				return err
			}
			return err
	})
	for _, statefulset := range replSSets {
		err = retry.RetryOnConflict(wait.Backoff{
			Steps:    5,
			Duration: 2 * time.Second,
			Factor:   1.0,
			Jitter:   0.1,
		}, func() error {
			err = ctrl.SetControllerReference(redisCluster, statefulset, r.Scheme) // ToDo: This is a hack. We should update all statefulsets
			if err != nil {
				logger.Error(err, fmt.Sprintf("Could not set owner name for replica statefulset named %s", statefulset.Name))
				return err
			}
			err = r.Client.Update(ctx, statefulset) // ToDo: This is a hack. We should update all statefulsets
			if err != nil {
				logger.Error(err, fmt.Sprintf("Could not update replica statefulset named %s  with owner reference", statefulset.Name))
				return err
			}
			return err
			
			})
		}
	if err != nil {
		logger.Error(err, "Could not set owner reference for statefulset")
		return ctrl.Result{
			RequeueAfter: 10 * time.Second,
		}, err
	}
	//endregion

	//region Ensure Service
	service, err := kubernetes.FetchService(ctx, r.Client, redisCluster)
	if err != nil && !apierrors.IsNotFound(err) {
		// We've got a legitimate error, we should log the error and exit early
		logger.Error(err, "Could not check whether service exists due to error.")
		return ctrl.Result{
			RequeueAfter: 30 * time.Second,
		}, err
	}
	if apierrors.IsNotFound(err) {
		// We need to create the Statefulset
		service, err = kubernetes.CreateService(ctx, r.Client, redisCluster)
		if err != nil {
			logger.Error(err, "Failed to create Service for RedisCluster")
			return ctrl.Result{
				RequeueAfter: 30 * time.Second,
			}, err
		}

		// We've created the Statefulset, and we can wait a bit before trying to do the rest.
		// We can trigger a new reconcile for this object in about 5 seconds
		logger.Info("Created Service for RedisCluster. Reconciling in 5 seconds.")
		return ctrl.Result{
			RequeueAfter: 5 * time.Second,
		}, err
	}
	// endregion

	//region Set Service owner reference
	err = retry.RetryOnConflict(wait.Backoff{
		Steps:    5,
		Duration: 2 * time.Second,
		Factor:   1.0,
		Jitter:   0.1,
	}, func() error {
		service, err = kubernetes.FetchService(ctx, r.Client, redisCluster)
		if err != nil {
			// At this point we definitely expect the statefulset to exist.
			logger.Error(err, "Cannot find service")
			return err
		}
		err = ctrl.SetControllerReference(redisCluster, service, r.Scheme)
		if err != nil {
			logger.Error(err, "Could not set owner reference for service")
			return err
		}
		err = r.Client.Update(ctx, service)
		if err != nil {
			logger.Error(err, "Could not update service with owner reference")
		}
		return err
	})
	if err != nil {
		logger.Error(err, "Could not set owner reference for service")
		return ctrl.Result{
			RequeueAfter: 10 * time.Second,
		}, err
	}
	//endregion

	//region Check if cluster needs to be scaled out
	scaleOut := false
	replicas := redisCluster.Spec.Masters
	// First check if master statefulset needs to be scaled out
	if *masterSSet.Spec.Replicas < redisCluster.Spec.Masters {
		scaleOut = true
		// The statefulset has less replicas than are needed for the cluster.
		// This means the user is trying to scale up the cluster, and we need to scale up the statefulset
		// and let the reconciliation take care of stabilising the cluster.
		logger.Info("Scaling up statefulset for Redis Cluster")
		masterSSet.Spec.Replicas = &replicas
		err = r.Client.Update(ctx, masterSSet)
		if err != nil {
			return r.RequeueError(ctx, "Could not update statefulset replicas", err)
		}
	}
	for _, statefulset := range replSSets {
		if *statefulset.Spec.Replicas < redisCluster.Spec.Masters {
			statefulset.Spec.Replicas = &replicas
			err = r.Client.Update(ctx, statefulset)
			if err != nil {
				return r.RequeueError(ctx, "Could not update statefulset replicas", err)
			}
		}
	}
	if scaleOut {
		// We've successfully updated the replicas for all statefulsets.
		// Now we can wait for the pods to come up and then continue on the
		// normal process for stabilising the Redis Cluster
		logger.Info("Scaling out statefulset for Redis Cluster successful. Reconciling again in 5 seconds.")
		return ctrl.Result{
			RequeueAfter: 5 * time.Second,
		}, nil
	}


	//endregion

	//region Check if cluster needs to be scaled in
	scaleIn := false
	// First we have to rebalance the masters if we have to scale in
	
	for _, statefulset := range replSSets {
		if *statefulset.Spec.Replicas > redisCluster.Spec.Masters {
			// The statefulset has less replicas than are needed for the cluster.
			// This means the user is trying to scale up the cluster, and we need to scale up the statefulset
			// and let the reconciliation take care of stabilising the cluster.
			logger.Info("Scaling up statefulset for Redis Cluster")
			statefulset.Spec.Replicas = &replicas
			err = r.Client.Update(ctx, statefulset)
			if err != nil {
				return r.RequeueError(ctx, "Could not update statefulset replicas", err)
			}
		}
	}
	if scaleIn {
		// We've successfully updated the replicas for all statefulsets.
		// Now we can wait for the pods to come up and then continue on the
		// normal process for stabilising the Redis Cluster
		logger.Info("Scaling in statefulset for Redis Cluster successful. Reconciling again in 5 seconds.")
		return ctrl.Result{
			RequeueAfter: 5 * time.Second,
		}, nil
	}
	//endregion

	pods, err := kubernetes.FetchRedisPods(ctx, r.Client, redisCluster)
	if err != nil {
		return r.RequeueError(ctx, "Could not fetch pods for redis cluster", err)
	}

	clusterNodes := redis_internal.ClusterNodes{}
	for _, pod := range pods.Items {
		if utils.IsPodReady(&pod) {
			node, err := redis_internal.NewNode(ctx, &redis.Options{
				Addr: pod.Status.PodIP + ":6379",
			}, &pod, redis.NewClient)
			if err != nil {
				return r.RequeueError(ctx, "Could not load Redis Client", err)
			}

			// make sure that the node knows about itself
			// This is necessary, as the nodes often startup without being able to retrieve their own IP address
			err = node.Client.ClusterMeet(ctx, pod.Status.PodIP, "6379").Err()
			if err != nil {
				return r.RequeueError(ctx, "Could not let node meet itself", err)
			}
			clusterNodes.Nodes = append(clusterNodes.Nodes, node)
		}
	}

	allPodsReady := len(clusterNodes.Nodes) == int(redisCluster.Spec.Masters)
	if !allPodsReady {
		logger.Info("Not all pods are ready. Reconciling again in 10 seconds")
		return ctrl.Result{
			RequeueAfter: 10 * time.Second,
		}, nil
	}

	if allPodsReady {
		// region Ensure Cluster Meet

		// todo we should check whether a cluster meet is necessary before just spraying cluster meets.
		// This can also be augmented by doing cluster meet for all ready nodes, and ignoring any none ready ones.
		// If the amount of ready pods is equal to the amount of nodes needed, we probably have some additional nodes we need to remove.
		// We can forget these additional nodes, as they are probably nodes which pods got killed.
		logger.Info("Meeting Redis nodes")
		err = clusterNodes.ClusterMeet(ctx)
		if err != nil {
			return r.RequeueError(ctx, "Could not meet all nodes together", err)
		}
		// We'll wait for 10 seconds to ensure the meet is propagated
		time.Sleep(time.Second * 5)
		// endregion

		logger.Info("Checking Cluster Master Replica Ratio")
		// region Ensure Cluster Replication Ratio
		err = clusterNodes.EnsureClusterReplicationRatio(ctx, redisCluster)
		if err != nil {
			return r.RequeueError(ctx, "Failed to ensure cluster ratio for cluster", err)
		}
		// endregion

		err = clusterNodes.ReloadNodes(ctx)
		if err != nil {
			return r.RequeueError(ctx, "Failed to reload node info for cluster", err)
		}

		// region Assign Slots
		logger.Info("Assigning Missing Slots")
		slotsAssignments := clusterNodes.CalculateSlotAssignment()
		for node, slots := range slotsAssignments {
			if len(slots) == 0 {
				continue
			}
			var slotsInt []int
			for _, slot := range slots {
				slotsInt = append(slotsInt, int(slot))
			}
			err = node.ClusterAddSlots(ctx, slotsInt...).Err()
			if err != nil {
				return r.RequeueError(ctx, "Could not assign node slots", err)
			}
		}
		// endregion

		logger.Info("Forgetting Failed Nodes No Longer Valid")
		failingNodes, err := clusterNodes.GetFailingNodes(ctx)
		if err != nil {
			return r.RequeueError(ctx, "could not fetch failing nodes", err)
		}
		for _, node := range failingNodes {
			err = clusterNodes.ForgetNode(ctx, node)
			if err != nil {
				return r.RequeueError(ctx, fmt.Sprintf("could not forget node %s", node.NodeAttributes.ID), err)
			}
		}

		logger.Info("Balancing Redis Cluster slots")
		err = clusterNodes.BalanceSlots(ctx, redisCluster)
		if err != nil {
			return r.RequeueError(ctx, "could not balance slots across nodes", err)
		}
		logger.Info("Finished balancing Redis Cluster slots")
	}

	return ctrl.Result{
		RequeueAfter: 30 * time.Second,
	}, nil
}

func (r *RedisClusterReconciler) RequeueError(ctx context.Context, message string, err error) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Error(err, message)
	return ctrl.Result{
		RequeueAfter: 10 * time.Second,
	}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *RedisClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&redisclusterv1alpha1.RedisCluster{}).
		Complete(r)
}

type clusterInfo struct {
	redisCluster *redisclusterv1alpha1.RedisCluster
	masterSS *appsv1.StatefulSet
	replicaSS []*appsv1.StatefulSet
	masterPods []*corev1.Pod
	replicaPods []*corev1.Pod
	cm *corev1.ConfigMap
	scr *corev1.Secret
}

func decorateRedisCluster(rdcl *redisclusterv1alpha1.RedisCluster)  {

}