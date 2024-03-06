package serverset

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
)

const resourceReadyTimeout = 5 * time.Minute

// Implements lower level functions to interact with kubernetes

// wrapper - kubernetes client wrapper
type wrapper struct {
	kube client.Client
	log  logging.Logger
}

// getObjectTypeFromName - returns a kubernetes object type based on name. Naming convention must be <serverset-name>-<object-type>-<index>-<version>
func getObjectTypeFromName(name string) client.Object {
	s := strings.Split(name, "-")[1]
	switch s {
	case resourceServer:
		return &v1alpha1.Server{}
	case resourceVolume, resourceBootVolume:
		return &v1alpha1.Volume{}
	case resourceNIC:
		return &v1alpha1.Nic{}
	default:
		return nil
	}
}

// isResourceDeleted - checks if a kube object is deleted. Extracts resource type from name,
// support for server, bootvolume, volume, nic
func (w *wrapper) isResourceDeleted(ctx context.Context, name, namespace string) (bool, error) {
	w.log.Info("Checking if resource is deleted", "name", name, "namespace", namespace)
	obj := getObjectTypeFromName(name)
	err := w.kube.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, obj)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			w.log.Info("Resource has been deleted", "name", name, "namespace", namespace)
			return true, nil
		}
		return false, nil
	}
	return false, nil
}

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
