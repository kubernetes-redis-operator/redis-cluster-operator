package kubernetes

import (
	"context"
	"fmt"

	"errors"

	"github.com/kubernetes-redis-operator/redis-cluster-operator/api/v1alpha1"
	"github.com/kubernetes-redis-operator/redis-cluster-operator/internal/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

const (
	RedisNodeNameStatefulsetLabel = "rediscluster.kuro.io/cluster-name"
	RedisNodeComponentLabel       = "rediscluster.kuro.io/cluster-component"
)

// GetStatefulSetLabels is a function that returns the labels for a StatefulSet object.
func GetStatefulSetLabels(cluster *v1alpha1.RedisCluster) labels.Set {
	return labels.Set{
		RedisNodeNameStatefulsetLabel: cluster.Name,
	}
}

// GetPodLabels is a function that returns the labels for a Pod object.
func GetPodLabels(cluster *v1alpha1.RedisCluster) labels.Set {
	return labels.Set{
		RedisNodeNameStatefulsetLabel: cluster.Name,
		RedisNodeComponentLabel:       "redis",
	}
}

// FetchExistingStatefulsets is a function that fetches the existing StatefulSet objects from the Kubernetes API server.
func (km *KubernetesManager) FetchStatefulsets(ctx context.Context, cluster *v1alpha1.RedisCluster) (*appsv1.StatefulSet, []*appsv1.StatefulSet, error) {
	var errslice []error
	var replicass = make([]*appsv1.StatefulSet, cluster.Spec.ReplicasPerMaster)
	masterss := &appsv1.StatefulSet{}
	err := km.client.Get(ctx, types.NamespacedName{
		Namespace: cluster.Namespace,
		Name:      cluster.Name + "-master",
	}, masterss)
	if err != nil {
		errslice = append(errslice, err)
	}
	for i := 0; i < int(cluster.Spec.ReplicasPerMaster); i++ {
		replss := &appsv1.StatefulSet{}
		err := km.client.Get(ctx, types.NamespacedName{
			Namespace: cluster.Namespace,
			Name:      cluster.Name + fmt.Sprintf("-repl-%d", i),
		}, replss)
		if err != nil {
			errslice = append(errslice, err)
		}
		replicass[i] = replss
	}
	fetchError := errors.Join(errslice...)
	return masterss, replicass, fetchError
}

// CreateStatefulsets is a function that creates StatefulSet objects in the Kubernetes API server.
func (km *KubernetesManager) CreateStatefulsets(ctx context.Context, cluster *v1alpha1.RedisCluster) (*appsv1.StatefulSet, []*appsv1.StatefulSet, error) {
	var errslice []error
	var replicass = make([]*appsv1.StatefulSet, cluster.Spec.ReplicasPerMaster)
	masterss := createStatefulsetSpec(cluster, "master")
	masterss.Labels["rediscluster.kuro.io/cluster-role"] = "master"
	err := km.client.Create(ctx, masterss)
	if err != nil {
		errslice = append(errslice, err)
		return nil, nil, err
	}

	// Append to the list after creating and having no error just because we know what to clean up in case of an error later on
	for i := 0; i < int(cluster.Spec.ReplicasPerMaster) && len(errslice) == 0; i++ {
		replss := createStatefulsetSpec(cluster, fmt.Sprintf("repl-%d", i))
		replss.Labels["rediscluster.kuro.io/cluster-role"] = "replica"
		err := km.client.Create(ctx, replss)
		if err != nil {
			errslice = append(errslice, err)
			break
		}
		replicass = append(replicass, replss)
	}
	// At this point, we may have some statefulsets created and some not. We need to clean up the ones that are created if stop is true
	creationError := errors.Join(errslice...)
	return masterss, replicass, creationError
}

func createStatefulsetSpec(cluster *v1alpha1.RedisCluster, namesuffix string) *appsv1.StatefulSet {
	replicasNeeded := cluster.Spec.Masters
	statefulset := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + "-" + namesuffix,
			Namespace: cluster.Namespace,
			Labels:    GetStatefulSetLabels(cluster),
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicasNeeded,
			Selector: &metav1.LabelSelector{
				MatchLabels: GetPodLabels(cluster),
			},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: GetPodLabels(cluster),
					Annotations: map[string]string{
						"kubectl.kubernetes.io/default-container": "redis",
					},
				},
				Spec: corev1.PodSpec{
					Volumes: utils.MergeVolumes(
						[]corev1.Volume{
							{
								Name: "redis-cluster-config",
								VolumeSource: corev1.VolumeSource{
									ConfigMap: &corev1.ConfigMapVolumeSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: getConfigMapName(cluster),
										},
									},
								},
							},
						},
						cluster.Spec.PodSpec.Volumes,
					),
					InitContainers: utils.MergeContainers(
						[]corev1.Container{},
						cluster.Spec.PodSpec.InitContainers,
					),
					Containers: utils.MergeContainers(
						[]corev1.Container{
							{
								Name:  "redis",
								Image: "redis:7.0.0",
								Command: []string{
									"redis-server",
								},
								Args: []string{
									"/usr/local/etc/redis/redis.conf",
								},
								Ports: []corev1.ContainerPort{
									{
										Name:          "redis",
										ContainerPort: 6379,
									},
									{
										Name:          "redis-gossip",
										ContainerPort: 16379,
									},
								},
								LivenessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										Exec: &corev1.ExecAction{
											Command: []string{
												"redis-cli",
												"ping",
											},
										},
									},
									InitialDelaySeconds: 10,
									TimeoutSeconds:      5,
									PeriodSeconds:       3,
								},
								ReadinessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										Exec: &corev1.ExecAction{
											Command: []string{
												"redis-cli",
												"ping",
											},
										},
									},
									InitialDelaySeconds: 10,
									TimeoutSeconds:      5,
									PeriodSeconds:       3,
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "redis-cluster-config",
										MountPath: "/usr/local/etc/redis",
									},
								},
							},
						},
						cluster.Spec.PodSpec.Containers,
					),
					EphemeralContainers:           cluster.Spec.PodSpec.EphemeralContainers,
					RestartPolicy:                 cluster.Spec.PodSpec.RestartPolicy,
					TerminationGracePeriodSeconds: cluster.Spec.PodSpec.TerminationGracePeriodSeconds,
					ActiveDeadlineSeconds:         cluster.Spec.PodSpec.ActiveDeadlineSeconds,
					DNSPolicy:                     cluster.Spec.PodSpec.DNSPolicy,
					NodeSelector:                  cluster.Spec.PodSpec.NodeSelector,
					NodeName:                      cluster.Spec.PodSpec.NodeName,
					HostNetwork:                   cluster.Spec.PodSpec.HostNetwork,
					HostPID:                       cluster.Spec.PodSpec.HostPID,
					HostIPC:                       cluster.Spec.PodSpec.HostIPC,
					ShareProcessNamespace:         cluster.Spec.PodSpec.ShareProcessNamespace,
					SecurityContext:               cluster.Spec.PodSpec.SecurityContext,
					ImagePullSecrets:              cluster.Spec.PodSpec.ImagePullSecrets,
					Hostname:                      cluster.Spec.PodSpec.Hostname,
					Subdomain:                     cluster.Spec.PodSpec.Subdomain,
					Affinity:                      cluster.Spec.PodSpec.Affinity,
					SchedulerName:                 cluster.Spec.PodSpec.SchedulerName,
					Tolerations:                   cluster.Spec.PodSpec.Tolerations,
					HostAliases:                   cluster.Spec.PodSpec.HostAliases,
					PriorityClassName:             cluster.Spec.PodSpec.PriorityClassName,
					Priority:                      cluster.Spec.PodSpec.Priority,
					DNSConfig:                     cluster.Spec.PodSpec.DNSConfig,
					ReadinessGates:                cluster.Spec.PodSpec.ReadinessGates,
					RuntimeClassName:              cluster.Spec.PodSpec.RuntimeClassName,
					EnableServiceLinks:            cluster.Spec.PodSpec.EnableServiceLinks,
					PreemptionPolicy:              cluster.Spec.PodSpec.PreemptionPolicy,
					Overhead:                      cluster.Spec.PodSpec.Overhead,
					TopologySpreadConstraints:     cluster.Spec.PodSpec.TopologySpreadConstraints,
					SetHostnameAsFQDN:             cluster.Spec.PodSpec.SetHostnameAsFQDN,
					OS:                            cluster.Spec.PodSpec.OS,
				},
			},
			ServiceName:     cluster.Name,
			MinReadySeconds: 10,
		},
	}
	return statefulset
}
