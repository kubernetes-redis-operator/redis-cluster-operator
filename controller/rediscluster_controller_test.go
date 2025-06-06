/*
Copyright 2022.

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
	"testing"

	"github.com/kubernetes-redis-operator/redis-cluster-operator/api/v1alpha1"
	cachev1alpha1 "github.com/kubernetes-redis-operator/redis-cluster-operator/api/v1alpha1"
	"github.com/kubernetes-redis-operator/redis-cluster-operator/internal/kubernetes"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type mockKubernetes struct {
	fetchRedisPodsFunc func(ctx context.Context, kubeClient client.Client, cluster *v1alpha1.RedisCluster) (*corev1.PodList, error)
}

func (m *mockKubernetes) FetchRedisPods(ctx context.Context, kubeClient client.Client, cluster *v1alpha1.RedisCluster) (*corev1.PodList, error) {
	return m.fetchRedisPodsFunc(ctx, kubeClient, cluster)
}

func TestRedisClusterReconciler_Reconcile_ReturnsIfRedisClusterIsNotFound(t *testing.T) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	_ = cachev1alpha1.AddToScheme(s)
	clientBuilder := fake.ClientBuilder{}
	// Create a ReconcileMemcached object with the scheme and fake client.
	km := kubernetes.NewKubernetesManager(clientBuilder.Build())
	r := &RedisClusterReconciler{
		KubernetesManager: km,
		Scheme:            s,
	}

	// Mock request to simulate Reconcile() being called on an event for a
	// watched resource .
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}
	_, err := r.Reconcile(context.TODO(), req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
}

func TestRedisClusterReconciler_Reconcile_ReturnsErrorIfCannotGetStatefulset(t *testing.T) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	_ = cachev1alpha1.AddToScheme(s)
	clientBuilder := fake.NewClientBuilder()

	// Create a ReconcileMemcached object with the scheme and fake client.
	km := kubernetes.NewKubernetesManager(clientBuilder.Build())
	r := &RedisClusterReconciler{
		KubernetesManager: km,
		Scheme:            s,
	}

	// Mock request to simulate Reconcile() being called on an event for a
	// watched resource .
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}
	_, err := r.Reconcile(context.TODO(), req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
}

func TestRedisClusterReconciler_Reconcile_CreatesStatefulsetIfDoesntExist(t *testing.T) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	_ = cachev1alpha1.AddToScheme(s)

	clientBuilder := fake.NewClientBuilder()
	clientBuilder.WithObjects(&cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
		Spec: cachev1alpha1.RedisClusterSpec{
			Masters:           3,
			ReplicasPerMaster: 1,
		},
	})
	client := clientBuilder.Build()
	// Create a ReconcileMemcached object with the scheme and fake client.
	km := kubernetes.NewKubernetesManager(client)
	r := &RedisClusterReconciler{
		KubernetesManager: km,
		Scheme:            s,
	}

	// Mock request to simulate Reconcile() being called on an event for a
	// watched resource .
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}

	for i := 0; i < 8; i++ {
		_, err := r.Reconcile(context.TODO(), req)
		if err != nil {
			t.Fatalf("reconcile: (%v)", err)
		}
	}

	sts := &appsv1.StatefulSet{}
	err := client.Get(context.TODO(), types.NamespacedName{
		Name:      "redis-cluster-master",
		Namespace: "default",
	}, sts)
	if err != nil {
		t.Fatalf("Failed to fetch created Statefulset %v", err)
	}
}

func TestRedisClusterReconciler_Reconcile_DoesNotFailIfStatefulsetExists(t *testing.T) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	_ = cachev1alpha1.AddToScheme(s)

	replicas := int32(6)
	clientBuilder := fake.NewClientBuilder()
	clientBuilder.WithObjects(&cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
		Spec: cachev1alpha1.RedisClusterSpec{
			Masters:           3,
			ReplicasPerMaster: 1,
		},
	}, &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
		},
	})
	client := clientBuilder.Build()

	// Create a ReconcileMemcached object with the scheme and fake client.
	km := kubernetes.NewKubernetesManager(client)
	r := &RedisClusterReconciler{
		KubernetesManager: km,
		Scheme:            s,
	}

	// Mock request to simulate Reconcile() being called on an event for a
	// watched resource .
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}

	for i := 0; i < 8; i++ {
		_, err := r.Reconcile(context.TODO(), req)
		assert.NoError(t, err)
	}

	sts := &appsv1.StatefulSet{}
	err := client.Get(context.TODO(), types.NamespacedName{
		Name:      "redis-cluster",
		Namespace: "default",
	}, sts)
	assert.NoError(t, err)
}

func TestRedisClusterReconciler_Reconcile_StatefulsetHasOwnerReferenceSetToRedisCluster(t *testing.T) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	_ = cachev1alpha1.AddToScheme(s)

	redisCluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
			UID:       "5b85970f-d70e-4f32-a9f7-12b2cc81f125",
		},
		Spec: cachev1alpha1.RedisClusterSpec{
			Masters:           3,
			ReplicasPerMaster: 1,
		},
	}

	clientBuilder := fake.NewClientBuilder()
	clientBuilder.WithObjects(redisCluster)
	client := clientBuilder.Build()

	// Create a ReconcileMemcached object with the scheme and fake client.
	km := kubernetes.NewKubernetesManager(client)
	r := &RedisClusterReconciler{
		KubernetesManager: km,
		Scheme:            s,
	}

	// Mock request to simulate Reconcile() being called on an event for a
	// watched resource .
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}

	// We might need multiple reconciles to get to the result we need, as we return early most of the time.
	// Let's reconcile a couple times before assertions.
	for i := 0; i < 8; i++ {
		_, err := r.Reconcile(context.TODO(), req)
		if err != nil {
			t.Fatalf("reconcile: (%v)", err)
		}
	}

	sts := &appsv1.StatefulSet{}
	err := client.Get(context.TODO(), types.NamespacedName{
		Name:      "redis-cluster-master",
		Namespace: "default",
	}, sts)
	if err != nil {
		t.Fatalf("Failed to fetch created Statefulset %v", err)
	}

	if len(sts.GetOwnerReferences()) == 0 {
		t.Fatalf("Owner reference is not set on statefulset")
	}
	gvk, _ := apiutil.GVKForObject(redisCluster, s)
	if sts.GetOwnerReferences()[0].Name != redisCluster.Name || sts.GetOwnerReferences()[0].Kind != gvk.Kind {
		t.Fatalf("Owner not correctly set")
	}
}

func TestRedisClusterReconciler_Reconcile_CreatesConfigMapForRedisCluster(t *testing.T) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	_ = cachev1alpha1.AddToScheme(s)
	redisCluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
		Spec: cachev1alpha1.RedisClusterSpec{
			Masters:           3,
			ReplicasPerMaster: 1,
		},
	}

	clientBuilder := fake.NewClientBuilder()
	clientBuilder.WithObjects(redisCluster)
	client := clientBuilder.Build()

	// Create a ReconcileMemcached object with the scheme and fake client.
	km := kubernetes.NewKubernetesManager(client)
	r := &RedisClusterReconciler{
		KubernetesManager: km,
		Scheme:            s,
	}

	// Mock request to simulate Reconcile() being called on an event for a
	// watched resource .
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}

	// We might need multiple reconciles to get to the result we need, as we return early most of the time.
	// Let's reconcile a couple times before assertions.
	for i := 0; i < 8; i++ {
		_, err := r.Reconcile(context.TODO(), req)
		if err != nil {
			t.Fatalf("reconcile: (%v)", err)
		}
	}

	configMap := &corev1.ConfigMap{}
	err := client.Get(context.TODO(), types.NamespacedName{
		Name:      "redis-cluster-config",
		Namespace: "default",
	}, configMap)
	if err != nil {
		if errors.IsNotFound(err) {
			t.Fatalf("ConfigMap was not created during reconcile %v", err)
		}
		t.Fatalf("Failed to fetch created ConfigMap %v", err)
	}

	if configMap.Name != redisCluster.Name+"-config" || configMap.Namespace != redisCluster.Namespace {
		t.Fatalf("Failed to fetch correct ConfigMap in test %v", configMap)
	}
}

func TestRedisClusterReconciler_Reconcile_DoesNotFailIfConfigMapExists(t *testing.T) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	_ = cachev1alpha1.AddToScheme(s)
	redisCluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
		Spec: cachev1alpha1.RedisClusterSpec{
			Masters:           3,
			ReplicasPerMaster: 1,
		},
	}

	clientBuilder := fake.NewClientBuilder()
	clientBuilder.WithObjects(redisCluster, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster-config",
			Namespace: "default",
		},
		Data: map[string]string{
			"redis.conf": "",
		},
	})
	client := clientBuilder.Build()

	// Create a ReconcileMemcached object with the scheme and fake client.
	km := kubernetes.NewKubernetesManager(client)
	r := &RedisClusterReconciler{
		KubernetesManager: km,
		Scheme:            s,
	}

	// Mock request to simulate Reconcile() being called on an event for a
	// watched resource .
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}

	// We might need multiple reconciles to get to the result we need, as we return early most of the time.
	// Let's reconcile a couple times before assertions.
	for i := 0; i < 8; i++ {
		_, err := r.Reconcile(context.TODO(), req)
		if err != nil {
			t.Fatalf("reconcile: (%v)", err)
		}
	}

	configMap := &corev1.ConfigMap{}
	err := client.Get(context.TODO(), types.NamespacedName{
		Name:      "redis-cluster-config",
		Namespace: "default",
	}, configMap)
	if err != nil {
		if errors.IsNotFound(err) {
			t.Fatalf("ConfigMap was not created during reconcile %v", err)
		}
		t.Fatalf("Failed to fetch created ConfigMap %v", err)
	}

	if configMap.Name != redisCluster.Name+"-config" || configMap.Namespace != redisCluster.Namespace {
		t.Fatalf("Failed to fetch correct ConfigMap in test %v", configMap)
	}
}

func TestRedisClusterReconciler_Reconcile_ConfigMapHasOwnerReferenceSetToRedisCluster(t *testing.T) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	_ = cachev1alpha1.AddToScheme(s)

	redisCluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
			UID:       "5b85970f-d70e-4f32-a9f7-12b2cc81f125",
		},
		Spec: cachev1alpha1.RedisClusterSpec{
			Masters:           3,
			ReplicasPerMaster: 1,
		},
	}

	clientBuilder := fake.NewClientBuilder()
	clientBuilder.WithObjects(redisCluster)
	client := clientBuilder.Build()

	// Create a ReconcileMemcached object with the scheme and fake client.
	km := kubernetes.NewKubernetesManager(client)
	r := &RedisClusterReconciler{
		KubernetesManager: km,
		Scheme:            s,
	}

	// Mock request to simulate Reconcile() being called on an event for a
	// watched resource .
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}

	// We might need multiple reconciles to get to the result we need, as we return early most of the time.
	// Let's reconcile a couple times before assertions.
	for i := 0; i < 8; i++ {
		_, err := r.Reconcile(context.TODO(), req)
		if err != nil {
			t.Fatalf("reconcile: (%v)", err)
		}
	}

	configmap := &corev1.ConfigMap{}
	err := client.Get(context.TODO(), types.NamespacedName{
		Name:      "redis-cluster-config",
		Namespace: "default",
	}, configmap)
	if err != nil {
		t.Fatalf("Failed to fetch created ConfigMap %v", err)
	}

	if len(configmap.GetOwnerReferences()) == 0 {
		t.Fatalf("Owner reference is not set on ConfigMap")
	}
	gvk, _ := apiutil.GVKForObject(redisCluster, s)
	if configmap.GetOwnerReferences()[0].Name != redisCluster.Name || configmap.GetOwnerReferences()[0].Kind != gvk.Kind {
		t.Fatalf("Owner not correctly set")
	}
}
