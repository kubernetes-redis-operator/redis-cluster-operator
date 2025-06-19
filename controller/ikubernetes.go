package controller

import (
	"context"

	"github.com/kubernetes-redis-operator/redis-cluster-operator/api/v1alpha1"
	redisclusterv1alpha1 "github.com/kubernetes-redis-operator/redis-cluster-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IKubernetesManager is an interface depicting the controller's interaction with the Kubernetes API.
type IKubernetesManager interface {
	FetchRedisPods(ctx context.Context, cluster *v1alpha1.RedisCluster) (*v1.PodList, error)
	FetchConfigmap(ctx context.Context, cluster *v1alpha1.RedisCluster) (*v1.ConfigMap, error)
	CreateConfigMap(ctx context.Context, cluster *v1alpha1.RedisCluster) (*v1.ConfigMap, error)
	FetchService(ctx context.Context, cluster *v1alpha1.RedisCluster) (*v1.Service, error)
	CreateService(ctx context.Context, cluster *v1alpha1.RedisCluster) (*v1.Service, error)
	FetchRedisCluster(ctx context.Context, namespacedName types.NamespacedName) (*redisclusterv1alpha1.RedisCluster, error)
	UpdateResource(ctx context.Context, obj client.Object) error
	UpdateResourceStatus(ctx context.Context, obj client.Object) error
	FetchStatefulsets(ctx context.Context, cluster *v1alpha1.RedisCluster) (*appsv1.StatefulSet, []*appsv1.StatefulSet, error)
	CreateStatefulsets(ctx context.Context, cluster *v1alpha1.RedisCluster) (*appsv1.StatefulSet, []*appsv1.StatefulSet, error)
}
