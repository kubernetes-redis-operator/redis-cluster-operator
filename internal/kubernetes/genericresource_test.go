package kubernetes

import (
	"context"
	"testing"

	cachev1alpha1 "github.com/serdarkalayci/redis-cluster-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestKubernetesManager_UpdateResource(t *testing.T) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	_ = cachev1alpha1.AddToScheme(s)
	clientBuilder := fake.NewClientBuilder()

	clientBuilder.WithObjects(&v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster-config",
			Namespace: "default",
		},
	})

	client := clientBuilder.Build()


	km := NewKubernetesManager(client)

	err := km.UpdateResource(context.TODO(), &v1.ConfigMap{		
		ObjectMeta: metav1.ObjectMeta{
		Name:      "redis-cluster-config",
		Namespace: "default",
		Labels:   map[string]string{"app": "redis-cluster"},
		},
	})
	assert.NoError(t, err)
}
