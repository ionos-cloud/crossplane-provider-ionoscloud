package serverset

import (
	"context"
	"strings"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/ionos-cloud/sdk-go/v6"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
)

type KubeBootVolumeControlManager interface {
	Create(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int) (v1alpha1.Volume, error)
	Get(ctx context.Context, volumeName, ns string) (*v1alpha1.Volume, error)
}

// kubeBootVolumeController - kubernetes client wrapper  for server resources
type kubeBootVolumeController struct {
	kube client.Client
	log  logging.Logger
}

// Create creates a volume CR and waits until in reaches AVAILABLE state
func (k *kubeBootVolumeController) Create(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int) (v1alpha1.Volume, error) {
	name := getNameFromIndex(cr.Name, "bootvolume", replicaIndex, version)
	k.log.Info("Creating Volume", "name", name)

	createVolume := FromServerSetToVolume(cr, name, replicaIndex, version)
	createVolume.SetProviderConfigReference(cr.Spec.ProviderConfigReference)
	if err := k.kube.Create(ctx, &createVolume); err != nil {
		return v1alpha1.Volume{}, err
	}
	if err := WaitForKubeResource(ctx, ResourceReadyTimeout, k.isAvailable, name, cr.Namespace); err != nil {
		return v1alpha1.Volume{}, err
	}
	// get the volume again before returning to have the id populated
	kubeVolume, err := k.Get(ctx, name, cr.Namespace)
	if err != nil {
		return v1alpha1.Volume{}, err
	}
	k.log.Info("Finished creating Volume", "name", name)

	return *kubeVolume, nil
}

// Get - returns a volume kubernetes object
func (k *kubeBootVolumeController) Get(ctx context.Context, volumeName, ns string) (*v1alpha1.Volume, error) {
	obj := &v1alpha1.Volume{}
	err := k.kube.Get(ctx, types.NamespacedName{
		Namespace: ns,
		Name:      volumeName,
	}, obj)
	return obj, err
}

// IsVolumeAvailable - checks if a volume is available
func (k *kubeBootVolumeController) isAvailable(ctx context.Context, name, namespace string) (bool, error) {
	obj, err := k.Get(ctx, name, namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	if obj != nil && obj.Status.AtProvider.VolumeID != "" && strings.EqualFold(obj.Status.AtProvider.State, ionoscloud.Available) {
		return true, nil
	}
	return false, err
}
