package controller

import (
	"context"

	cachev1alpha1 "github.com/kubernetes-redis-operator/redis-cluster-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// --- Mock for IKubernetesManager ---

type mockKubernetesManager struct {
	FetchRedisPodsFunc       func(ctx context.Context, cluster *cachev1alpha1.RedisCluster) (*corev1.PodList, error)
	FetchConfigmapFunc       func(ctx context.Context, cluster *cachev1alpha1.RedisCluster) (*corev1.ConfigMap, error)
	CreateConfigMapFunc      func(ctx context.Context, cluster *cachev1alpha1.RedisCluster) (*corev1.ConfigMap, error)
	FetchServiceFunc         func(ctx context.Context, cluster *cachev1alpha1.RedisCluster) (*corev1.Service, error)
	CreateServiceFunc        func(ctx context.Context, cluster *cachev1alpha1.RedisCluster) (*corev1.Service, error)
	FetchRedisClusterFunc    func(ctx context.Context, namespacedName client.ObjectKey) (*cachev1alpha1.RedisCluster, error)
	UpdateResourceFunc       func(ctx context.Context, obj client.Object) error
	UpdateResourceStatusFunc func(ctx context.Context, obj client.Object) error
	FetchStatefulsetsFunc    func(ctx context.Context, cluster *cachev1alpha1.RedisCluster) (*appsv1.StatefulSet, []*appsv1.StatefulSet, error)
	CreateStatefulsetsFunc   func(ctx context.Context, cluster *cachev1alpha1.RedisCluster) (*appsv1.StatefulSet, []*appsv1.StatefulSet, error)
}

func (m *mockKubernetesManager) FetchRedisPods(ctx context.Context, cluster *cachev1alpha1.RedisCluster) (*corev1.PodList, error) {
	if m.FetchRedisPodsFunc != nil {
		return m.FetchRedisPodsFunc(ctx, cluster)
	}
	return nil, nil
}

func (m *mockKubernetesManager) FetchConfigmap(ctx context.Context, cluster *cachev1alpha1.RedisCluster) (*corev1.ConfigMap, error) {
	if m.FetchConfigmapFunc != nil {
		return m.FetchConfigmapFunc(ctx, cluster)
	}
	return nil, nil
}

func (m *mockKubernetesManager) CreateConfigMap(ctx context.Context, cluster *cachev1alpha1.RedisCluster) (*corev1.ConfigMap, error) {
	if m.CreateConfigMapFunc != nil {
		return m.CreateConfigMapFunc(ctx, cluster)
	}
	return nil, nil
}

func (m *mockKubernetesManager) FetchService(ctx context.Context, cluster *cachev1alpha1.RedisCluster) (*corev1.Service, error) {
	if m.FetchServiceFunc != nil {
		return m.FetchServiceFunc(ctx, cluster)
	}
	return nil, nil
}

func (m *mockKubernetesManager) CreateService(ctx context.Context, cluster *cachev1alpha1.RedisCluster) (*corev1.Service, error) {
	if m.CreateServiceFunc != nil {
		return m.CreateServiceFunc(ctx, cluster)
	}
	return nil, nil
}

func (m *mockKubernetesManager) FetchRedisCluster(ctx context.Context, namespacedName client.ObjectKey) (*cachev1alpha1.RedisCluster, error) {
	if m.FetchRedisClusterFunc != nil {
		return m.FetchRedisClusterFunc(ctx, namespacedName)
	}
	return nil, nil
}

func (m *mockKubernetesManager) UpdateResource(ctx context.Context, obj client.Object) error {
	if m.UpdateResourceFunc != nil {
		return m.UpdateResourceFunc(ctx, obj)
	}
	return nil
}

func (m *mockKubernetesManager) UpdateResourceStatus(ctx context.Context, obj client.Object) error {
	if m.UpdateResourceStatusFunc != nil {
		return m.UpdateResourceStatusFunc(ctx, obj)
	}
	return nil
}

func (m *mockKubernetesManager) FetchStatefulsets(ctx context.Context, cluster *cachev1alpha1.RedisCluster) (*appsv1.StatefulSet, []*appsv1.StatefulSet, error) {
	if m.FetchStatefulsetsFunc != nil {
		return m.FetchStatefulsetsFunc(ctx, cluster)
	}
	return nil, nil, nil
}

func (m *mockKubernetesManager) CreateStatefulsets(ctx context.Context, cluster *cachev1alpha1.RedisCluster) (*appsv1.StatefulSet, []*appsv1.StatefulSet, error) {
	if m.CreateStatefulsetsFunc != nil {
		return m.CreateStatefulsetsFunc(ctx, cluster)
	}
	return nil, nil, nil
}
