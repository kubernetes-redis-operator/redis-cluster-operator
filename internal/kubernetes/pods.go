package kubernetes

import (
	"context"

	"github.com/serdarkalayci/redis-cluster-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FetchRedisPods is a function that fetches a list of Pods from the Kubernetes API server.
func (km *KubernetesManager) FetchRedisPods(ctx context.Context, cluster *v1alpha1.RedisCluster) (*v1.PodList, error) {
	pods := &v1.PodList{}
	err := km.client.List(
		ctx,
		pods,
		client.MatchingLabelsSelector{Selector: GetPodLabels(cluster).AsSelector()},
		client.InNamespace(cluster.Namespace),
	)
	return pods, err
}
