package kube

import (
	"context"
	"fmt"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"
)

// ResourceReadyTimeout time to wait for resource to be ready
const ResourceReadyTimeout = 30 * time.Minute

// ServerSetReadyTimeout time to wait for serverset to be ready
const ServerSetReadyTimeout = 1 * time.Hour

// ErrExternalCreateFailed error when external create fails, so we know to delete kube object
var ErrExternalCreateFailed = errors.New("external create failed")

// Implements lower level functions to interact with kubernetes

// IsResourceReady polls kube api to see if resource is available and observed(status populated)
type IsResourceReady func(ctx context.Context, name, namespace string) (bool, error)

// WaitForResource - keeps retrying until resource meets condition, or until ctx is cancelled
func WaitForResource(ctx context.Context, timeoutInMinutes time.Duration, fn IsResourceReady, name, namespace string) error {
	if name == "" {
		return fmt.Errorf("name is empty")
	}
	pollInterval := 2 * time.Second
	return wait.PollUntilContextTimeout(ctx, pollInterval, timeoutInMinutes, true, func(context.Context) (bool, error) {
		return fn(ctx, name, namespace)
	})
}

// IsSuccessfullyCreated checks if the object was successfully created
func IsSuccessfullyCreated(obj resource.Managed) bool {
	return obj.GetAnnotations()[meta.AnnotationKeyExternalCreateFailed] == ""
}
