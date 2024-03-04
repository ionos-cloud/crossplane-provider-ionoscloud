package kube

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
)

const (
	ServersetIndexLabel   = "ionoscloud.com/serverset-%s-index"
	ServersetVersionLabel = "ionoscloud.com/serverset-%s-version"
	ServerSetLabel        = "ionoscloud.com/serverset"
)

// Implements lower level functions to interact with kubernetes

type Wrapper struct {
	Kube client.Client
	Log  logging.Logger
}

// GetVolume - returns a volume object from kubernetes
func (c *Wrapper) GetVolume(ctx context.Context, volumeName, ns string) (*v1alpha1.Volume, error) {
	obj := &v1alpha1.Volume{}
	err := c.Kube.Get(ctx, types.NamespacedName{
		Namespace: ns,
		Name:      volumeName,
	}, obj)
	return obj, err
}

func (c *Wrapper) GetNic(ctx context.Context, name, ns string) (*v1alpha1.Nic, error) {
	obj := &v1alpha1.Nic{}
	if err := c.Kube.Get(ctx, types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

func ListResourceFromIndex(ctx context.Context, kube client.Client, resType string, index int, obj client.ObjectList) error {
	if err := kube.List(ctx, obj, client.MatchingLabels{
		fmt.Sprintf("ionoscloud.com/serverset-%s-index", resType): strconv.Itoa(index),
	}); err != nil {
		return err
	}
	return nil
}

func ListResourceWithIndexAndVersion(ctx context.Context, kube client.Client, resType string, index, version int, obj client.ObjectList) error {
	if err := kube.List(ctx, obj, client.MatchingLabels{
		fmt.Sprintf("ionoscloud.com/serverset-%s-version", resType): strconv.Itoa(version),
		fmt.Sprintf("ionoscloud.com/serverset-%s-index", resType):   strconv.Itoa(index),
	}); err != nil {
		return err
	}
	return nil
}

// IsVolumeDeleted - checks if a volume is deleted
func (c *Wrapper) IsVolumeDeleted(ctx context.Context, name, namespace string) (bool, error) {
	c.Log.Info("Checking if volume is deleted")
	_, err := c.GetVolume(ctx, name, namespace)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return true, nil
		}
		return false, nil
	}
	return false, nil
}

// getObjectTypeFromName - returns a kubernetes object type based on name. Naming convention must be <serverset-name>-<object-type>-<index>-<version>
func getObjectTypeFromName(name string) client.Object {
	s := strings.Split(name, "-")[1]
	switch s {
	case "server":
		return &v1alpha1.Server{}
	case "volume", "bootvolume":
		return &v1alpha1.Volume{}
	case "nic":
		return &v1alpha1.Nic{}
	default:
		return nil
	}

}

// IsResourceDeleted - checks if a kube object is deleted. Extracts resource type from name,
// support for server, bootvolume, volume, nic
func (c *Wrapper) IsResourceDeleted(ctx context.Context, name, namespace string) (bool, error) {
	c.Log.Info("Checking if resource is deleted", "name", name, "namespace", namespace)
	obj := getObjectTypeFromName(name)
	err := c.Kube.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, obj)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			c.Log.Info("Resource has been deleted", "name", name, "namespace", namespace)
			return true, nil
		}
		return false, nil
	}
	return false, nil
}

// IsNicDeleted - checks if a nic is deleted
func (c *Wrapper) IsNicDeleted(ctx context.Context, name, namespace string) (bool, error) {
	_, err := c.GetNic(ctx, name, namespace)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return true, nil
		}
		return false, nil
	}
	return false, nil
}

// func IsServerDeleted(ctx context.Context, c client.Client, name, namespace string) (bool, error) {
// 	_, err := GetServer(ctx, c, name, namespace)
// 	if err != nil {
// 		if apiErrors.IsNotFound(err) {
// 			return true, nil
// 		}
// 		return false, nil
// 	}
// 	return false, nil
// }

// IsVolumeAvailable - checks if a volume is available
func (c *Wrapper) IsVolumeAvailable(ctx context.Context, name, namespace string) (bool, error) {
	obj, err := c.GetVolume(ctx, name, namespace)
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

// IsNIcAvailable - checks if a volume is available
func (c *Wrapper) IsNIcAvailable(ctx context.Context, name, namespace string) (bool, error) {
	obj, err := c.GetNic(ctx, name, namespace)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	if obj != nil && obj.Status.AtProvider.NicID != "" && strings.EqualFold(obj.Status.AtProvider.State, ionoscloud.Available) {
		return true, nil
	}
	return false, err
}

// WaitForKubeResource - keeps retrying until resource meets condition, or until ctx is cancelled
func WaitForKubeResource(ctx context.Context, timeoutInMinutes time.Duration, fn IsResourceReady, name, namespace string) error {
	if name == "" {
		return fmt.Errorf("name is empty")
	}
	return wait.PollUntilContextTimeout(ctx, 2*time.Second, timeoutInMinutes, true, func(context.Context) (bool, error) {
		return fn(ctx, name, namespace)
	})
}

// IsResourceReady polls Kube api to see if resource is available and observed(status populated)
type IsResourceReady func(ctx context.Context, name, namespace string) (bool, error)

// GetNameFromIndex - generates name consisting of name, kind and index
func GetNameFromIndex(resourceName, resourceType string, idx, version int) string {
	return fmt.Sprintf("%s-%s-%d-%d", resourceName, resourceType, idx, version)
}

func (c *Wrapper) IsServerAvailable(ctx context.Context, name, namespace string) (bool, error) {
	obj := &v1alpha1.Server{}
	err := c.Kube.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, obj)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return false, nil
		}
	}
	if obj != nil && obj.Status.AtProvider.ServerID != "" && strings.EqualFold(obj.Status.AtProvider.State, ionoscloud.Available) {
		return true, nil
	}
	return false, err
}

// CreateNic creates a NIC CR and waits until in reaches AVAILABLE state
func (c *Wrapper) CreateNic(ctx context.Context, cr *v1alpha1.ServerSet, serverID, lanName string, replicaIndex, version int) error {
	resourceType := "nic"
	name := GetNameFromIndex(cr.Name, resourceType, replicaIndex, version)
	c.Log.Info("Creating NIC", "name", name)
	network := v1alpha1.Lan{}
	if err := c.Kube.Get(ctx, types.NamespacedName{
		Namespace: cr.GetNamespace(),
		Name:      lanName,
	}, &network); err != nil {
		return err
	}
	lanID := network.Status.AtProvider.LanID
	// no NIC found, create one
	createNic := FromServerSetToNic(cr, name, serverID, lanID, replicaIndex, version)
	createNic.SetProviderConfigReference(cr.Spec.ProviderConfigReference)
	if err := c.Kube.Create(ctx, &createNic); err != nil {
		return err
	}

	err := WaitForKubeResource(ctx, ResourceReadyTimeout, c.IsNIcAvailable, createNic.Name, cr.Namespace)
	if err != nil {
		return err
	}
	c.Log.Info("Finished creating NIC", "name", name)
	return nil
}

const ResourceReadyTimeout = 5 * time.Minute
