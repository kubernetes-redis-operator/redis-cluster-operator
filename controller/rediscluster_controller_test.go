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
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/logr"
	cachev1alpha1 "github.com/kubernetes-redis-operator/redis-cluster-operator/api/v1alpha1"
	"github.com/kubernetes-redis-operator/redis-cluster-operator/internal/kubernetes"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

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
		if apierrors.IsNotFound(err) {
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
		if apierrors.IsNotFound(err) {
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
func TestRedisClusterReconciler_ensureConfigMap_CreatesConfigMapIfNotFound(t *testing.T) {
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

	calledCreate := false
	mockKM := &mockKubernetesManager{
		FetchConfigmapFunc: func(ctx context.Context, rc *cachev1alpha1.RedisCluster) (*corev1.ConfigMap, error) {
			return nil, apierrors.NewNotFound(corev1.Resource("configmap"), "redis-cluster-config")
		},
		CreateConfigMapFunc: func(ctx context.Context, rc *cachev1alpha1.RedisCluster) (*corev1.ConfigMap, error) {
			calledCreate = true
			return &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rc.Name + "-config",
					Namespace: rc.Namespace,
				},
			}, nil
		},
	}

	r := &RedisClusterReconciler{
		KubernetesManager: mockKM,
		Scheme:            s,
	}

	logger := logr.Discard()
	result, err := r.ensureConfigMap(context.TODO(), logger, redisCluster)
	assert.NoError(t, err)
	assert.True(t, calledCreate)
	assert.Equal(t, 5*time.Second, result.RequeueAfter)
}

func TestRedisClusterReconciler_ensureConfigMap_ReturnsErrorOnFetchError(t *testing.T) {
	s := scheme.Scheme
	_ = cachev1alpha1.AddToScheme(s)

	redisCluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}

	mockKM := &mockKubernetesManager{
		FetchConfigmapFunc: func(ctx context.Context, rc *cachev1alpha1.RedisCluster) (*corev1.ConfigMap, error) {
			return nil, fmt.Errorf("unexpected error")
		},
	}

	r := &RedisClusterReconciler{
		KubernetesManager: mockKM,
		Scheme:            s,
	}

	logger := logr.Discard()
	result, err := r.ensureConfigMap(context.TODO(), logger, redisCluster)
	assert.Error(t, err)
	assert.Equal(t, 30*time.Second, result.RequeueAfter)
}

func TestRedisClusterReconciler_ensureConfigMap_ReturnsErrorOnCreateError(t *testing.T) {
	s := scheme.Scheme
	_ = cachev1alpha1.AddToScheme(s)

	redisCluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}

	mockKM := &mockKubernetesManager{
		FetchConfigmapFunc: func(ctx context.Context, rc *cachev1alpha1.RedisCluster) (*corev1.ConfigMap, error) {
			return nil, apierrors.NewNotFound(corev1.Resource("configmap"), "redis-cluster-config")
		},
		CreateConfigMapFunc: func(ctx context.Context, rc *cachev1alpha1.RedisCluster) (*corev1.ConfigMap, error) {
			return nil, fmt.Errorf("create error")
		},
	}

	r := &RedisClusterReconciler{
		KubernetesManager: mockKM,
		Scheme:            s,
	}

	logger := logr.Discard()
	result, err := r.ensureConfigMap(context.TODO(), logger, redisCluster)
	assert.Error(t, err)
	assert.Equal(t, 30*time.Second, result.RequeueAfter)
}

func TestRedisClusterReconciler_ensureConfigMap_SetsOwnerReferenceIfConfigMapExists(t *testing.T) {
	s := scheme.Scheme
	_ = cachev1alpha1.AddToScheme(s)

	redisCluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
			UID:       "12345",
		},
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster-config",
			Namespace: "default",
		},
	}

	updateCalled := false
	mockKM := &mockKubernetesManager{
		FetchConfigmapFunc: func(ctx context.Context, rc *cachev1alpha1.RedisCluster) (*corev1.ConfigMap, error) {
			return configMap, nil
		},
		UpdateResourceFunc: func(ctx context.Context, obj client.Object) error {
			updateCalled = true
			return nil
		},
	}

	r := &RedisClusterReconciler{
		KubernetesManager: mockKM,
		Scheme:            s,
	}

	logger := logr.Discard()
	result, err := r.ensureConfigMap(context.TODO(), logger, redisCluster)
	assert.NoError(t, err)
	assert.True(t, updateCalled)
	assert.Equal(t, time.Duration(0), result.RequeueAfter)
}
func TestRedisClusterReconciler_ensureStatefulSet_CreatesStatefulSetIfNotFound(t *testing.T) {
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

	masterSS := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster-master",
			Namespace: "default",
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: func() *int32 { i := int32(3); return &i }(),
		},
	}
	replicaSS := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster-replica",
			Namespace: "default",
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: func() *int32 { i := int32(3); return &i }(),
		},
	}

	mkm := &mockKubernetesManager{}

	r := &RedisClusterReconciler{
		KubernetesManager: mkm,
		Scheme:            s,
	}

	logger := logr.Discard()
	mkm.FetchStatefulsetsFunc = func(ctx context.Context, rc *cachev1alpha1.RedisCluster) (*appsv1.StatefulSet, []*appsv1.StatefulSet, error) {
		return nil, nil, apierrors.NewNotFound(appsv1.Resource("statefulset"), "redis-cluster-master")
	}
	mkm.CreateStatefulsetsFunc = func(ctx context.Context, rc *cachev1alpha1.RedisCluster) (*appsv1.StatefulSet, []*appsv1.StatefulSet, error) {
		return masterSS, []*appsv1.StatefulSet{replicaSS}, nil
	}
	gotMaster, gotReplicas, result, err := r.ensureStatefulSet(context.TODO(), logger, redisCluster)
	assert.NoError(t, err)
	assert.Equal(t, masterSS, gotMaster)
	assert.Equal(t, []*appsv1.StatefulSet{replicaSS}, gotReplicas)
	assert.Equal(t, 5*time.Second, result.RequeueAfter)
}

func TestRedisClusterReconciler_ensureStatefulSet_ReturnsErrorOnFetchError(t *testing.T) {
	s := scheme.Scheme
	_ = cachev1alpha1.AddToScheme(s)

	redisCluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}

	mockKM := &mockKubernetesManager{
		FetchStatefulsetsFunc: func(ctx context.Context, rc *cachev1alpha1.RedisCluster) (*appsv1.StatefulSet, []*appsv1.StatefulSet, error) {
			return nil, nil, fmt.Errorf("unexpected error")
		},
	}

	r := &RedisClusterReconciler{
		KubernetesManager: mockKM,
		Scheme:            s,
	}

	logger := logr.Discard()
	gotMaster, gotReplicas, result, err := r.ensureStatefulSet(context.TODO(), logger, redisCluster)
	assert.Error(t, err)
	assert.Nil(t, gotMaster)
	assert.Nil(t, gotReplicas)
	assert.Equal(t, 30*time.Second, result.RequeueAfter)
}

func TestRedisClusterReconciler_ensureStatefulSet_ReturnsErrorOnCreateError(t *testing.T) {
	s := scheme.Scheme
	_ = cachev1alpha1.AddToScheme(s)

	redisCluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}

	mockKM := &mockKubernetesManager{
		FetchStatefulsetsFunc: func(ctx context.Context, rc *cachev1alpha1.RedisCluster) (*appsv1.StatefulSet, []*appsv1.StatefulSet, error) {
			return nil, nil, apierrors.NewNotFound(appsv1.Resource("statefulset"), "redis-cluster-master")
		},
		CreateStatefulsetsFunc: func(ctx context.Context, rc *cachev1alpha1.RedisCluster) (*appsv1.StatefulSet, []*appsv1.StatefulSet, error) {
			return nil, nil, fmt.Errorf("create error")
		},
	}

	r := &RedisClusterReconciler{
		KubernetesManager: mockKM,
		Scheme:            s,
	}

	logger := logr.Discard()
	gotMaster, gotReplicas, result, err := r.ensureStatefulSet(context.TODO(), logger, redisCluster)
	assert.Error(t, err)
	assert.Nil(t, gotMaster)
	assert.Nil(t, gotReplicas)
	assert.Equal(t, 30*time.Second, result.RequeueAfter)
}

func TestRedisClusterReconciler_ensureStatefulSet_SetsOwnerReference(t *testing.T) {
	s := scheme.Scheme
	_ = cachev1alpha1.AddToScheme(s)

	redisCluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
			UID:       "12345",
		},
	}

	masterSS := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster-master",
			Namespace: "default",
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: func() *int32 { i := int32(3); return &i }(),
		},
	}
	replicaSS := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster-replica",
			Namespace: "default",
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: func() *int32 { i := int32(3); return &i }(),
		},
	}

	fetchCount := 0
	updateCalled := 0

	mockKM := &mockKubernetesManager{
		FetchStatefulsetsFunc: func(ctx context.Context, rc *cachev1alpha1.RedisCluster) (*appsv1.StatefulSet, []*appsv1.StatefulSet, error) {
			fetchCount++
			return masterSS, []*appsv1.StatefulSet{replicaSS}, nil
		},
		UpdateResourceFunc: func(ctx context.Context, obj client.Object) error {
			updateCalled++
			return nil
		},
	}

	r := &RedisClusterReconciler{
		KubernetesManager: mockKM,
		Scheme:            s,
	}

	logger := logr.Discard()
	gotMaster, gotReplicas, result, err := r.ensureStatefulSet(context.TODO(), logger, redisCluster)
	assert.NoError(t, err)
	assert.Equal(t, masterSS, gotMaster)
	assert.Equal(t, []*appsv1.StatefulSet{replicaSS}, gotReplicas)
	assert.Equal(t, 0*time.Second, result.RequeueAfter)
	assert.True(t, updateCalled > 0)
}
func TestRedisClusterReconciler_ensureService_CreatesServiceIfNotFound(t *testing.T) {
	s := scheme.Scheme
	_ = cachev1alpha1.AddToScheme(s)

	redisCluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}

	calledCreate := false
	mockKM := &mockKubernetesManager{
		FetchServiceFunc: func(ctx context.Context, rc *cachev1alpha1.RedisCluster) (*corev1.Service, error) {
			return nil, apierrors.NewNotFound(corev1.Resource("service"), "redis-cluster")
		},
		CreateServiceFunc: func(ctx context.Context, rc *cachev1alpha1.RedisCluster) (*corev1.Service, error) {
			calledCreate = true
			return &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rc.Name,
					Namespace: rc.Namespace,
				},
			}, nil
		},
	}

	r := &RedisClusterReconciler{
		KubernetesManager: mockKM,
		Scheme:            s,
	}

	logger := logr.Discard()
	result, err := r.ensureService(context.TODO(), logger, redisCluster)
	assert.NoError(t, err)
	assert.True(t, calledCreate)
	assert.Equal(t, 5*time.Second, result.RequeueAfter)
}

func TestRedisClusterReconciler_ensureService_ReturnsErrorOnFetchError(t *testing.T) {
	s := scheme.Scheme
	_ = cachev1alpha1.AddToScheme(s)

	redisCluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}

	mockKM := &mockKubernetesManager{
		FetchServiceFunc: func(ctx context.Context, rc *cachev1alpha1.RedisCluster) (*corev1.Service, error) {
			return nil, fmt.Errorf("unexpected error")
		},
	}

	r := &RedisClusterReconciler{
		KubernetesManager: mockKM,
		Scheme:            s,
	}

	logger := logr.Discard()
	result, err := r.ensureService(context.TODO(), logger, redisCluster)
	assert.Error(t, err)
	assert.Equal(t, 30*time.Second, result.RequeueAfter)
}

func TestRedisClusterReconciler_ensureService_ReturnsErrorOnCreateError(t *testing.T) {
	s := scheme.Scheme
	_ = cachev1alpha1.AddToScheme(s)

	redisCluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}

	mockKM := &mockKubernetesManager{
		FetchServiceFunc: func(ctx context.Context, rc *cachev1alpha1.RedisCluster) (*corev1.Service, error) {
			return nil, apierrors.NewNotFound(corev1.Resource("service"), "redis-cluster")
		},
		CreateServiceFunc: func(ctx context.Context, rc *cachev1alpha1.RedisCluster) (*corev1.Service, error) {
			return nil, fmt.Errorf("create error")
		},
	}

	r := &RedisClusterReconciler{
		KubernetesManager: mockKM,
		Scheme:            s,
	}

	logger := logr.Discard()
	result, err := r.ensureService(context.TODO(), logger, redisCluster)
	assert.Error(t, err)
	assert.Equal(t, 30*time.Second, result.RequeueAfter)
}

func TestRedisClusterReconciler_ensureService_SetsOwnerReferenceIfServiceExists(t *testing.T) {
	s := scheme.Scheme
	_ = cachev1alpha1.AddToScheme(s)

	redisCluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
			UID:       "12345",
		},
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}

	updateCalled := false
	mockKM := &mockKubernetesManager{
		FetchServiceFunc: func(ctx context.Context, rc *cachev1alpha1.RedisCluster) (*corev1.Service, error) {
			return service, nil
		},
		UpdateResourceFunc: func(ctx context.Context, obj client.Object) error {
			updateCalled = true
			return nil
		},
	}

	r := &RedisClusterReconciler{
		KubernetesManager: mockKM,
		Scheme:            s,
	}

	logger := logr.Discard()
	result, err := r.ensureService(context.TODO(), logger, redisCluster)
	assert.NoError(t, err)
	assert.True(t, updateCalled)
	assert.Equal(t, time.Duration(0), result.RequeueAfter)
}
