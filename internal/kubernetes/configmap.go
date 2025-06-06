package kubernetes

import (
	"context"
	"fmt"
	"strings"

	"github.com/kubernetes-redis-operator/redis-cluster-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// FetchConfigmap is a function that fetches a ConfigMap object from the Kubernetes API server.
func (km *KubernetesManager) FetchConfigmap(ctx context.Context, cluster *v1alpha1.RedisCluster) (*v1.ConfigMap, error) {
	configMap := &v1.ConfigMap{}
	err := km.client.Get(ctx, types.NamespacedName{
		Namespace: cluster.Namespace,
		Name:      getConfigMapName(cluster),
	}, configMap)
	return configMap, err
}

func getConfigMapName(cluster *v1alpha1.RedisCluster) string {
	return fmt.Sprintf("%s-config", cluster.Name)
}

func getDefaultRedisConfig() map[string]string {
	return map[string]string{
		"port":                 "6379",
		"cluster-enabled":      "yes",
		"cluster-config-file":  "nodes.conf",
		"cluster-node-timeout": "5000",
	}
}

func getAppliedRedisConfig(cluster *v1alpha1.RedisCluster) map[string]string {
	config := getDefaultRedisConfig()
	redisConfig := getRedisConfigFromMultilineYaml(cluster.Spec.Config)
	for setting, value := range redisConfig {
		config[setting] = value
	}
	return config
}

func getRedisConfigAsMultilineYaml(config map[string]string) string {
	result := ""
	for setting, value := range config {
		result += fmt.Sprintf("%s %s\n", setting, value)
	}
	return result
}

func getRedisConfigFromMultilineYaml(config string) map[string]string {
	result := map[string]string{}
	for _, settingLine := range strings.Split(config, "\n") {
		if strings.TrimSpace(settingLine) == "" {
			continue
		}
		settingParts := strings.Split(settingLine, " ")
		setting := settingParts[0]
		value := strings.Join(settingParts[1:], " ")
		result[setting] = value
	}
	return result
}

func createConfigMapSpec(cluster *v1alpha1.RedisCluster) *v1.ConfigMap {
	redisConfig := getAppliedRedisConfig(cluster)
	configMap := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      getConfigMapName(cluster),
			Namespace: cluster.Namespace,
		},
		Data: map[string]string{
			"redis.conf": getRedisConfigAsMultilineYaml(redisConfig),
		},
	}
	return configMap
}

// CreateConfigMap is a function that creates a ConfigMap object in the Kubernetes API server.
func (km *KubernetesManager) CreateConfigMap(ctx context.Context, cluster *v1alpha1.RedisCluster) (*v1.ConfigMap, error) {
	configMap := createConfigMapSpec(cluster)
	err := km.client.Create(ctx, configMap)
	return configMap, err
}
