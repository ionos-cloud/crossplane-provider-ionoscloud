package serverset

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

const resourceReadyTimeout = 5 * time.Minute

// Implements lower level functions to interact with kubernetes

// IsResourceReady polls kube api to see if resource is available and observed(status populated)
type IsResourceReady func(ctx context.Context, name, namespace string) (bool, error)

// WaitForKubeResource - keeps retrying until resource meets condition, or until ctx is cancelled
func WaitForKubeResource(ctx context.Context, timeoutInMinutes time.Duration, fn IsResourceReady, name, namespace string) error {
	if name == "" {
		return fmt.Errorf("name is empty")
	}
	pollInterval := 2 * time.Second
	return wait.PollUntilContextTimeout(ctx, pollInterval, timeoutInMinutes, true, func(context.Context) (bool, error) {
		return fn(ctx, name, namespace)
	})
}

// getNameFromIndex - generates name consisting of name, kind and index
func getNameFromIndex(resourceName, resourceType string, idx, version int) string {
	return fmt.Sprintf("%s-%s-%d-%d", resourceName, resourceType, idx, version)
}
