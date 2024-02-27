package kube

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
)

// Implements lower level functions to interact with kubernetes

// GetVolume - returns a volume object from kubernetes
func GetVolume(ctx context.Context, kube client.Client, volumeName, ns string) (*v1alpha1.Volume, error) {
	obj := &v1alpha1.Volume{}
	err := kube.Get(ctx, types.NamespacedName{
		Namespace: ns,
		Name:      volumeName,
	}, obj)
	return obj, err
}

// IsVolumeDeleted - checks if a volume is ready
func IsVolumeDeleted(ctx context.Context, c client.Client, name, namespace string) (bool, error) {
	_, err := GetVolume(ctx, c, name, namespace)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return true, nil
		}
		return false, nil
	}
	return false, nil
}

// IsVolumeAvailable - checks if a volume is available
func IsVolumeAvailable(ctx context.Context, c client.Client, name, namespace string) (bool, error) {
	obj, err := GetVolume(ctx, c, name, namespace)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	if obj != nil && obj.Status.AtProvider.VolumeID != "" && strings.EqualFold(obj.Status.AtProvider.State, ionoscloud.Available) {
		return true, nil
	}
	return false, err
}

// WaitForKubeResource - keeps retrying until resource meets condition, or until ctx is cancelled
func WaitForKubeResource(ctx context.Context, timeoutInMinutes time.Duration, fn IsResourceReady, kube client.Client, name, namespace string) error {
	if kube == nil {
		return fmt.Errorf("kube client is nil")
	}
	if name == "" {
		return fmt.Errorf("name is empty")
	}
	err := retry.RetryContext(ctx, timeoutInMinutes, func() *retry.RetryError {
		isReady, err := fn(ctx, kube, name, namespace)
		if isReady {
			return nil
		}
		if err != nil {
			retry.NonRetryableError(err)
		}
		return retry.RetryableError(fmt.Errorf("resource with name %v found, still trying ", name))
	})
	return err
}

// IsResourceReady polls kube api to see if resource is available and observed(status populated)
type IsResourceReady func(ctx context.Context, kube client.Client, name, namespace string) (bool, error)
