package kubernetes

import (
	"context"

	"github.com/kubernetes-redis-operator/redis-cluster-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// FetchService is a function that fetches a Service object from the Kubernetes API server.
func (km *KubernetesManager) FetchService(ctx context.Context, cluster *v1alpha1.RedisCluster) (*v1.Service, error) {
	service := &v1.Service{}
	err := km.client.Get(ctx, types.NamespacedName{
		Namespace: cluster.Namespace,
		Name:      cluster.Name,
	}, service)
	return service, err
}

func createServiceSpec(cluster *v1alpha1.RedisCluster) *v1.Service {
	service := &v1.Service{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:       "redis",
					Port:       6379,
					TargetPort: intstr.FromInt(6379),
				},
			},
			Selector: GetPodLabels(cluster),
			Type:     "ClusterIP",
		},
	}
	return service
}

// CreateService is a function that creates a Service object in the Kubernetes API server.
func (km *KubernetesManager) CreateService(ctx context.Context, cluster *v1alpha1.RedisCluster) (*v1.Service, error) {
	service := createServiceSpec(cluster)
	err := km.client.Create(ctx, service)
	return service, err
}
