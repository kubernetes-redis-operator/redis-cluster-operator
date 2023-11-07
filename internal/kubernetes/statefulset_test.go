package kubernetes

import (
	"context"
	"fmt"
	"sort"
	"testing"

	cachev1alpha1 "github.com/serdarkalayci/redis-cluster-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func generateStatefulSets(name, namespace string, replPerMaster int) []client.Object {
	sss := make([]client.Object, replPerMaster+1)
	sss[0] = &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-master",
			Namespace: namespace,
		},
	}
	for i := 0; i < replPerMaster; i++ {
		sss[i+1] = &v1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s-%d", name, "repl", i),
				Namespace: namespace,
			},
		}
	}
	return sss
}

func TestFetchExistingStatefulSetReturnsErrorIfNotFound(t *testing.T) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(cachev1alpha1.GroupVersion)
	clientBuilder := fake.NewClientBuilder()
	client := clientBuilder.Build()

	cluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}

	_, err := FetchExistingStatefulsets(context.TODO(), client, cluster)
	assert.Error(t, err)
	assert.True(t, apierrors.IsNotFound(err))
}

func TestFetchExistingStatefulSetReturnsErrorIfReplicaNotFound(t *testing.T) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(cachev1alpha1.GroupVersion)
	clientBuilder := fake.NewClientBuilder()
	sss := generateStatefulSets("redis-cluster", "default", 2) // This will create 3 statefulsets, 1 master and 2 replicas
	clientBuilder.WithObjects(sss[0:1]...)                     // Only create the master statefulset and one replica
	client := clientBuilder.Build()

	cluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
		Spec: cachev1alpha1.RedisClusterSpec{
			Masters:           1,
			ReplicasPerMaster: 2,
		},
	}

	_, err := FetchExistingStatefulsets(context.TODO(), client, cluster)
	assert.Error(t, err)
	assert.True(t, apierrors.IsNotFound(err))
}

func TestFetchExistingStatefulSetReturnsStatefulsetIfFound(t *testing.T) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(cachev1alpha1.GroupVersion)
	clientBuilder := fake.NewClientBuilder()

	clientBuilder.WithObjects(generateStatefulSets("redis-cluster", "default", 2)...)

	client := clientBuilder.Build()

	cluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
		Spec: cachev1alpha1.RedisClusterSpec{
			Masters:           1,
			ReplicasPerMaster: 2,
		},
	}

	statefulsets, err := FetchExistingStatefulsets(context.TODO(), client, cluster)
	assert.NoError(t, err)
	assert.Len(t, statefulsets, 3)
}

func TestFetchExistingStatefulSetReturnsCorrectStatefulsetIfMany(t *testing.T) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(cachev1alpha1.GroupVersion)
	clientBuilder := fake.NewClientBuilder()
	clientBuilder.WithObjects(generateStatefulSets("redis-cluster", "default", 3)...)
	clientBuilder.WithObjects(generateStatefulSets("redis-cluster-foo", "default", 2)...)
	clientBuilder.WithObjects(generateStatefulSets("redis-cluster", "foo", 2)...)
	client := clientBuilder.Build()

	cluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
		Spec: cachev1alpha1.RedisClusterSpec{
			Masters:           2,
			ReplicasPerMaster: 3,
		},
	}

	statefulsets, err := FetchExistingStatefulsets(context.TODO(), client, cluster)
	assert.NoError(t, err)
	assert.Len(t, statefulsets, 4)
	assert.Equal(t, "redis-cluster-master", statefulsets[0].Name)
}

func TestCreateStatefulset_CanCreateStatefulset(t *testing.T) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(cachev1alpha1.GroupVersion)
	clientBuilder := fake.NewClientBuilder()
	client := clientBuilder.Build()

	cluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
		Spec: cachev1alpha1.RedisClusterSpec{
			Masters:           3,
			ReplicasPerMaster: 2,
		},
	}

	_, err := CreateStatefulsets(context.TODO(), client, cluster)
	if err != nil {
		t.Fatalf("Expected Statefulset to be created sucessfully, but received an error %v", err)
	}

	statefulset := &v1.StatefulSet{}
	err = client.Get(context.TODO(), types.NamespacedName{
		Namespace: cluster.Namespace,
		Name:      cluster.Name + "-master",
	}, statefulset)
	assert.NoError(t, err)
	assert.Equal(t, cluster.Name+"-master", statefulset.Name)
}

func TestCreateStatefulset_ThrowsErrorIfStatefulsetAlreadyExists(t *testing.T) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(cachev1alpha1.GroupVersion)
	clientBuilder := fake.NewClientBuilder()

	clientBuilder.WithObjects(&v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster-master",
			Namespace: "default",
		},
	})

	client := clientBuilder.Build()
	cluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
		Spec: cachev1alpha1.RedisClusterSpec{
			Masters:           1,
			ReplicasPerMaster: 2,
		},
	}

	_, err := CreateStatefulsets(context.TODO(), client, cluster)
	assert.Error(t, err)
}

func TestCreateStatefulset_ThrowsErrorIfReplStatefulsetAlreadyExists(t *testing.T) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(cachev1alpha1.GroupVersion)
	clientBuilder := fake.NewClientBuilder()

	clientBuilder.WithObjects(&v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster-repl-0",
			Namespace: "default",
		},
	})

	client := clientBuilder.Build()
	cluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
		Spec: cachev1alpha1.RedisClusterSpec{
			Masters:           1,
			ReplicasPerMaster: 2,
		},
	}

	_, err := CreateStatefulsets(context.TODO(), client, cluster)
	assert.Error(t, err)
}

func TestCreateStatefulset_MountsConfigMapAsVolumeCorrectly(t *testing.T) {
	// Register operator types with the runtime scheme.
	cluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}
	statefulset := createStatefulsetSpec(cluster, "master")

	configVolume := corev1.Volume{}
	mounted := false
	for _, volume := range statefulset.Spec.Template.Spec.Volumes {
		if volume.Name == "redis-cluster-config" {
			configVolume = volume
			mounted = true
			break
		}
	}
	if !mounted {
		// The volume could not be found
		t.Fatalf("No configMap mounted into redis pods")
	}
	if configVolume.VolumeSource.ConfigMap.LocalObjectReference.Name != "redis-cluster-config" {
		t.Fatalf("Configmap mounted incorrectly to Redis pods")
	}

	redisContainer := corev1.Container{}
	for _, container := range statefulset.Spec.Template.Spec.Containers {
		if container.Name == "redis" {
			redisContainer = container
		}
	}
	configMount := corev1.VolumeMount{}
	mounted = false
	for _, mount := range redisContainer.VolumeMounts {
		if mount.Name == "redis-cluster-config" {
			configMount = mount
			mounted = true
			break
		}
	}
	if !mounted {
		// The volume could not be found
		t.Fatalf("Configmap not mounted into redis pods")
	}
	if configMount.MountPath != "/usr/local/etc/redis" {
		t.Fatalf("Configmap mounted on wrong directory in redis pod")
	}
}

func TestCreateStatefulset_SetsLivenessAndReadinessProbes(t *testing.T) {
	// Register operator types with the runtime scheme.
	cluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}
	statefulset := createStatefulsetSpec(cluster, "master")

	if statefulset.Spec.Template.Spec.Containers[0].ReadinessProbe == nil {
		t.Fatalf("Readiness probe not set on the Redis statefulset")
	}

	if statefulset.Spec.Template.Spec.Containers[0].ReadinessProbe == nil {
		t.Fatalf("Liveness probe not set on the Redis statefulset")
	}
}

func TestCreateStatefulsetSpec_CanAddAdditionalContainers(t *testing.T) {
	// Register operator types with the runtime scheme.
	cluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
		Spec: cachev1alpha1.RedisClusterSpec{
			Masters:           3,
			ReplicasPerMaster: 1,
			PodSpec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "metric-container",
						Image: "prometheus:1.0.0",
					},
				},
			},
		},
	}
	statefulset := createStatefulsetSpec(cluster, "master")

	sort.SliceStable(statefulset.Spec.Template.Spec.Containers, func(i, j int) bool {
		return statefulset.Spec.Template.Spec.Containers[i].Name > statefulset.Spec.Template.Spec.Containers[j].Name
	})

	// Assert and extra container has been added
	if len(statefulset.Spec.Template.Spec.Containers) != 2 {
		t.Fatalf("Additional container was not added")
	}

	if statefulset.Spec.Template.Spec.Containers[1].Name != "metric-container" || statefulset.Spec.Template.Spec.Containers[1].Image != "prometheus:1.0.0" {
		t.Fatalf("Additional container was incorrectly added")
	}
}

func TestCreateStatefulsetSpec_CanOverrideRedisConfigurations(t *testing.T) {
	// Register operator types with the runtime scheme.
	cluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
		Spec: cachev1alpha1.RedisClusterSpec{
			Masters:           3,
			ReplicasPerMaster: 1,
			PodSpec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "redis",
						Image: "custom-redis-image:1.0.0",
						Ports: []corev1.ContainerPort{
							{
								Name:          "custom-port",
								ContainerPort: 8080,
							},
						},
					},
				},
			},
		},
	}
	statefulset := createStatefulsetSpec(cluster, "master")

	// Assert and extra container has been added
	if len(statefulset.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("Too many containers for simple override")
	}

	if statefulset.Spec.Template.Spec.Containers[0].Image != "custom-redis-image:1.0.0" {
		t.Fatalf("Redis container image not correctly overridden")
	}

	if len(statefulset.Spec.Template.Spec.Containers[0].Ports) != 3 {
		t.Fatalf("Additional port was not added")
	}
}
