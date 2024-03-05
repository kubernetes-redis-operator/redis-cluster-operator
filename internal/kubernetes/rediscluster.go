package kubernetes

import (
	"context"

	redisclusterv1alpha1 "github.com/serdarkalayci/redis-cluster-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
)

// FetchRedisCluster is a function that fetches a RedisCluster object from the Kubernetes API server.
func (km *KubernetesManager) FetchRedisCluster(ctx context.Context, namespacedName types.NamespacedName) (*redisclusterv1alpha1.RedisCluster, error) {
	redisCluster := &redisclusterv1alpha1.RedisCluster{}
	err := km.client.Get(ctx, namespacedName, redisCluster)
	return redisCluster, err
}