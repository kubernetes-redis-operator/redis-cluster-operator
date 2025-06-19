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

	"github.com/go-logr/logr"
	redisinternal "github.com/kubernetes-redis-operator/redis-cluster-operator/internal/redis"
	"github.com/kubernetes-redis-operator/redis-cluster-operator/internal/utils"
	"github.com/redis/go-redis/v9"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	redisclusterv1alpha1 "github.com/kubernetes-redis-operator/redis-cluster-operator/api/v1alpha1"
)

// RedisClusterReconciler reconciles a RedisCluster object
type RedisClusterReconciler struct {
	KubernetesManager IKubernetesManager
	Scheme            *runtime.Scheme
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
	redisCluster, err := r.KubernetesManager.FetchRedisCluster(ctx, req.NamespacedName)

	if err != nil {
		if apierrors.IsNotFound(err) {
			// The RedisCluster resource is not found on the cluster. Probably deleted by the user before this reconciliation. We'll quit early.
			logger.Info("RedisCluster not found during reconcile. Probably deleted by user. Exiting early.")
			return ctrl.Result{}, nil
		}
	}
	//endregion

	//region Try to get the ConfigMap for the RedisCluster
	result, err := r.ensureConfigMap(ctx, logger, redisCluster)
	if err != nil || result.RequeueAfter > 0 {
		return result, err
	}
	//endregion

	//region Ensure Statefulset
	masterSSet, replSSets, result, err := r.ensureStatefulSet(ctx, logger, redisCluster)
	if err != nil || result.RequeueAfter > 0 {
		return result, err
	}
	//endregion

	//region Ensure Service
	result, err = r.ensureService(ctx, logger, redisCluster)
	if err != nil || result.RequeueAfter > 0 {
		return result, err
	}
	//endregion
	clusterNodes := redisinternal.ClusterNodes{}
	if redisCluster.Status.GetInOperationCondition() == nil {
		// If the cluster is not in operation, we should set the in operation condition to false.
		logger.Info("Cluster is initializing.")
		redisCluster.Status.ClusterState = redisclusterv1alpha1.ClusterStateInitializing
		redisCluster.Status.SetInOperationCondition(true, "ClusterStateInitializing", "Cluster is initializing state")
		err = r.KubernetesManager.UpdateResourceStatus(ctx, redisCluster)
		if err != nil {
			return r.RequeueError(ctx, "Could not update RedisCluster condition", err)
		}
	}
	if redisCluster.Status.GetInOperationCondition().Status == metav1.ConditionTrue {
		// If the cluster is in operation, we should ensure that the operation is running correctly.
		logger.Info("Cluster is in operation. Reconciliation will check for the operation to complete.")
		switch redisCluster.Status.ClusterState {
		case redisclusterv1alpha1.ClusterStateScalingOut:
			// If the cluster is scaling out, we should ensure that the statefulset is scaled out correctly.
			result, err = r.reconcileScaleCluster(ctx, logger, redisCluster, masterSSet, replSSets)
			if err != nil || result.RequeueAfter > 0 {
				return result, err
			}
		case redisclusterv1alpha1.ClusterStateScalingIn:
			clusterNodes.DrainNodes(ctx, redisCluster)
			// If the cluster is scaling in, we should ensure that the statefulset is scaled in correctly.
			result, err = r.reconcileScaleCluster(ctx, logger, redisCluster, masterSSet, replSSets)
			if err != nil || result.RequeueAfter > 0 {
				return result, err
			}
		case redisclusterv1alpha1.ClusterStateIncreasingReplicas:
			// If the cluster is increasing replicas, we should ensure that the statefulset is updated correctly.
		case redisclusterv1alpha1.ClusterStateDecreasingReplicas:
			// If the cluster is decreasing replicas, we should ensure that the statefulset is updated correctly.
		case redisclusterv1alpha1.ClusterStateNormal:
			// If the cluster state is normal, there should be no operation in progress.
			logger.Info("Cluster is in normal state, but still marked as in operation. This should not happen. Resetting operation condition.")
			redisCluster.Status.SetInOperationCondition(false, "ClusterStateNormal", "Cluster is in normal state")
			err = r.KubernetesManager.UpdateResourceStatus(ctx, redisCluster)
			if err != nil {
				return r.RequeueError(ctx, "Could not update RedisCluster status", err)
			}
			// Reconcile the cluster again to check if a new operation is needed.
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
	} else {
		// If the cluster is not in operation, we can check the spec and compare it to the current state of the cluster to decide whether we need an operation.
		logger.Info("Cluster is not in operation. Continuing with reconciliation to check for potential operations.")
	}

	pods, err := r.KubernetesManager.FetchRedisPods(ctx, redisCluster)
	if err != nil {
		return r.RequeueError(ctx, "Could not fetch pods for redis cluster", err)
	}

	for _, pod := range pods.Items {
		if utils.IsPodReady(&pod) {
			node, err := redisinternal.NewNode(ctx, &redis.Options{
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

	redisCluster.Status.ClusterState = redisclusterv1alpha1.ClusterStateNormal
	redisCluster.Status.SetInOperationCondition(false, "ClusterStateNormal", "Cluster is in normal state")
	err = r.KubernetesManager.UpdateResourceStatus(ctx, redisCluster)
	if err != nil {
		return r.RequeueError(ctx, "Could not update RedisCluster status", err)
	}

	logger.Info("Reconciliation completed successfully for RedisCluster", "cluster", req.Name, "namespace", req.Namespace)
	// Return a result to requeue the reconciliation after 30 seconds
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
	masterSS     *appsv1.StatefulSet
	replicaSS    []*appsv1.StatefulSet
	masterPods   []*corev1.Pod
	replicaPods  []*corev1.Pod
	cm           *corev1.ConfigMap
	scr          *corev1.Secret
}

func (r *RedisClusterReconciler) ensureConfigMap(ctx context.Context, logger logr.Logger, redisCluster *redisclusterv1alpha1.RedisCluster) (ctrl.Result, error) {
	configMap, err := r.KubernetesManager.FetchConfigmap(ctx, redisCluster)
	if err != nil && !apierrors.IsNotFound(err) {
		logger.Error(err, "error checking the existence of ConfigMap")
		return ctrl.Result{
			RequeueAfter: 30 * time.Second,
		}, err
	}
	if apierrors.IsNotFound(err) {
		configMap, err = r.KubernetesManager.CreateConfigMap(ctx, redisCluster)
		if err != nil {
			logger.Error(err, "error creating ConfigMap for RedisCluster")
			return ctrl.Result{
				RequeueAfter: 30 * time.Second,
			}, err
		}
		logger.Info("Created ConfigMap for RedisCluster. Reconciling in 5 seconds.")
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
		configMap, err = r.KubernetesManager.FetchConfigmap(ctx, redisCluster)
		if err != nil {
			logger.Error(err, "error finding configMap")
			return err
		}
		err = ctrl.SetControllerReference(redisCluster, configMap, r.Scheme)
		if err != nil {
			logger.Error(err, "error setting owner reference of ConfigMap")
			return err
		}
		err = r.KubernetesManager.UpdateResource(ctx, configMap)
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
	return ctrl.Result{}, nil
}

func (r *RedisClusterReconciler) ensureStatefulSet(ctx context.Context, logger logr.Logger, redisCluster *redisclusterv1alpha1.RedisCluster) (*appsv1.StatefulSet, []*appsv1.StatefulSet, ctrl.Result, error) {
	masterSSet, replSSets, err := r.KubernetesManager.FetchStatefulsets(ctx, redisCluster)
	if err != nil && !apierrors.IsNotFound(err) {
		logger.Error(err, "Could not check whether statefulset exists due to error.")
		return nil, nil, ctrl.Result{
			RequeueAfter: 30 * time.Second,
		}, err
	}

	if apierrors.IsNotFound(err) {
		masterSSet, replSSets, err = r.KubernetesManager.CreateStatefulsets(ctx, redisCluster)
		if err != nil {
			logger.Error(err, "Failed to create Statefulset for RedisCluster")
			return nil, nil, ctrl.Result{
				RequeueAfter: 30 * time.Second,
			}, err
		}
		logger.Info("Created Statefulset for RedisCluster. Reconciling in 5 seconds.")
		return masterSSet, replSSets, ctrl.Result{
			RequeueAfter: 5 * time.Second,
		}, err
	}

	// Set owner reference for master statefulset
	masterSSet, replSSets, err = r.KubernetesManager.FetchStatefulsets(ctx, redisCluster)
	if err != nil {
		logger.Error(err, "Cannot find statefulsets")
		return nil, nil, ctrl.Result{
			RequeueAfter: 5 * time.Second,
		}, err
	}
	err = retry.RetryOnConflict(wait.Backoff{
		Steps:    5,
		Duration: 2 * time.Second,
		Factor:   1.0,
		Jitter:   0.1,
	}, func() error {
		err = ctrl.SetControllerReference(redisCluster, masterSSet, r.Scheme)
		if err != nil {
			logger.Error(err, fmt.Sprintf("Could not set owner name for master statefulset named %s", masterSSet.Name))
			return err
		}
		err = r.KubernetesManager.UpdateResource(ctx, masterSSet)
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
			err = ctrl.SetControllerReference(redisCluster, statefulset, r.Scheme)
			if err != nil {
				logger.Error(err, fmt.Sprintf("Could not set owner name for replica statefulset named %s", statefulset.Name))
				return err
			}
			err = r.KubernetesManager.UpdateResource(ctx, statefulset)
			if err != nil {
				logger.Error(err, fmt.Sprintf("Could not update replica statefulset named %s  with owner reference", statefulset.Name))
				return err
			}
			return err
		})
	}
	if err != nil {
		logger.Error(err, "Could not set owner reference for statefulset")
		return nil, nil, ctrl.Result{
			RequeueAfter: 10 * time.Second,
		}, err
	}
	return masterSSet, replSSets, ctrl.Result{}, nil
}

func (r *RedisClusterReconciler) ensureService(ctx context.Context, logger logr.Logger, redisCluster *redisclusterv1alpha1.RedisCluster) (ctrl.Result, error) {
	service, err := r.KubernetesManager.FetchService(ctx, redisCluster)
	if err != nil && !apierrors.IsNotFound(err) {
		logger.Error(err, "Could not check whether service exists due to error.")
		return ctrl.Result{
			RequeueAfter: 30 * time.Second,
		}, err
	}
	if apierrors.IsNotFound(err) {
		service, err = r.KubernetesManager.CreateService(ctx, redisCluster)
		if err != nil {
			logger.Error(err, "Failed to create Service for RedisCluster")
			return ctrl.Result{
				RequeueAfter: 30 * time.Second,
			}, err
		}
		logger.Info("Created Service for RedisCluster. Reconciling in 5 seconds.")
		return ctrl.Result{
			RequeueAfter: 5 * time.Second,
		}, err
	}

	// Set Service owner reference
	err = retry.RetryOnConflict(wait.Backoff{
		Steps:    5,
		Duration: 2 * time.Second,
		Factor:   1.0,
		Jitter:   0.1,
	}, func() error {
		service, err = r.KubernetesManager.FetchService(ctx, redisCluster)
		if err != nil {
			logger.Error(err, "Cannot find service")
			return err
		}
		err = ctrl.SetControllerReference(redisCluster, service, r.Scheme)
		if err != nil {
			logger.Error(err, "Could not set owner reference for service")
			return err
		}
		err = r.KubernetesManager.UpdateResource(ctx, service)
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
	return ctrl.Result{}, nil
}

// reconcileScaleOut checks if the cluster needs to be scaled out and performs the scaling if necessary.
func (r *RedisClusterReconciler) reconcileScaleCluster(ctx context.Context, logger logr.Logger, redisCluster *redisclusterv1alpha1.RedisCluster, masterSSet *appsv1.StatefulSet, replSSets []*appsv1.StatefulSet) (ctrl.Result, error) {
	replicas := redisCluster.Spec.Masters
	masterSSet.Spec.Replicas = &replicas
	err := r.KubernetesManager.UpdateResource(ctx, masterSSet)
	if err != nil {
		return r.RequeueError(ctx, "Could not update statefulset replicas", err)
	}
	for _, statefulset := range replSSets {
		if *statefulset.Spec.Replicas < redisCluster.Spec.Masters {
			statefulset.Spec.Replicas = &replicas
			err := r.KubernetesManager.UpdateResource(ctx, statefulset)
			if err != nil {
				return r.RequeueError(ctx, "Could not update statefulset replicas", err)
			}
		}
	}
	return ctrl.Result{
		RequeueAfter: 5 * time.Second,
	}, nil
}
