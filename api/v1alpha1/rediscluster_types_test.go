package v1alpha1

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestSetInOperationCondition(t *testing.T) {
    status := &RedisClusterStatus{}

    // Set InOperation to true
    status.SetInOperationCondition(true, "Scaling", "Cluster is scaling out")
    cond := status.GetInOperationCondition()
    assert.NotNil(t, cond, "Condition should not be nil after setting")
    assert.Equal(t, "InOperation", cond.Type)
    assert.Equal(t, metav1.ConditionTrue, cond.Status)
    assert.Equal(t, "Scaling", cond.Reason)
    assert.Equal(t, "Cluster is scaling out", cond.Message)

    // Set InOperation to false
    status.SetInOperationCondition(false, "Idle", "Cluster is idle")
    cond = status.GetInOperationCondition()
    assert.NotNil(t, cond, "Condition should not be nil after setting to false")
    assert.Equal(t, metav1.ConditionFalse, cond.Status)
    assert.Equal(t, "Idle", cond.Reason)
    assert.Equal(t, "Cluster is idle", cond.Message)
}

func TestGetInOperationCondition_Empty(t *testing.T) {
    status := &RedisClusterStatus{}
    cond := status.GetInOperationCondition()
    assert.Nil(t, cond, "Condition should be nil if not set")
}