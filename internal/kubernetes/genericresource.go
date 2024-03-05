package kubernetes

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UpdateResource updates the given Kubernetes resource.
func (km *KubernetesManager) UpdateResource(ctx context.Context, obj client.Object) (error) {
	return km.client.Update(ctx, obj)
}