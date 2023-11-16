package v1alpha1

import (
	"bytes"
	"testing"

	"k8s.io/apimachinery/pkg/util/yaml"
)

func TestRedisConfigIsProcessedCorrectly(t *testing.T) {
	redisClusterYaml := `---
apiVersion: rediscluster.kuro.io/v1alpha1
kind: RedisCluster
metadata:
  name: redis-cluster
spec:
  masters: 3
  replicasPerMaster: 0
  config: |
    maxmemory 128mb
    port 6379
`
	redisCluster := &RedisCluster{}
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(redisClusterYaml)), 1000)
	err := decoder.Decode(&redisCluster)
	if err != nil {
		t.Fatalf("Failed to load yaml into RedisCluster type")
	}
	expectedConfig := `maxmemory 128mb
port 6379
`
	if redisCluster.Spec.Config != expectedConfig {
		t.Fatalf(`RedisCluster is not processing config correctly.
# Expected:
%v
# Got: 
%v`, expectedConfig, redisCluster.Spec.Config)
	}
}
