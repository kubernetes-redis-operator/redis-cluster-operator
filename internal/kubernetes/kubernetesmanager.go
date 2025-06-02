package kubernetes

import "sigs.k8s.io/controller-runtime/pkg/client"

// KubernetesManager is a struct that holds a Kubernetes client, which also satisfies the IKubernetesManager interface.
type KubernetesManager struct {
	client client.Client
}

// NewKubernetesManager is a function that returns a new KubernetesManager.
func NewKubernetesManager(client client.Client) *KubernetesManager {
	return &KubernetesManager{client: client}
}
