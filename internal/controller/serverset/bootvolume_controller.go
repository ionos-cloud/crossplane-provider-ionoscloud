package serverset

import (
	"context"
	"fmt"
	"strings"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/ionos-cloud/sdk-go/v6"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
)

type kubeBootVolumeControlManager interface {
	Create(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int) (v1alpha1.Volume, error)
	Get(ctx context.Context, volumeName, ns string) (*v1alpha1.Volume, error)
	Delete(ctx context.Context, name, namespace string) error
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

	createVolume := fromServerSetToVolume(cr, name, replicaIndex, version)
	createVolume.SetProviderConfigReference(cr.Spec.ProviderConfigReference)
	if err := k.kube.Create(ctx, &createVolume); err != nil {
		return v1alpha1.Volume{}, err
	}
	if err := WaitForKubeResource(ctx, resourceReadyTimeout, k.isAvailable, name, cr.Namespace); err != nil {
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

// Delete - deletes the bootvolume k8s client and waits until it is deleted
func (k *kubeBootVolumeController) Delete(ctx context.Context, name, namespace string) error {
	condemnedVolume, err := k.Get(ctx, name, namespace)
	if err != nil {
		return err
	}
	if err := k.kube.Delete(ctx, condemnedVolume); err != nil {
		fmt.Printf("error deleting volume %v", err)
		return err
	}
	return WaitForKubeResource(ctx, resourceReadyTimeout, k.isBootVolumeDeleted, condemnedVolume.Name, namespace)
}

// IsVolumeAvailable - checks if a volume is available
func (k *kubeBootVolumeController) isAvailable(ctx context.Context, name, namespace string) (bool, error) {
	obj, err := k.Get(ctx, name, namespace)
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

func (k *kubeBootVolumeController) isBootVolumeDeleted(ctx context.Context, name, namespace string) (bool, error) {
	k.log.Info("Checking if Volume is deleted", "name", name, "namespace", namespace)
	obj := &v1alpha1.Volume{}
	err := k.kube.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, obj)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			k.log.Info("Volume has been deleted", "name", name, "namespace", namespace)
			return true, nil
		}
		return false, nil
	}
	return false, nil
}

func fromServerSetToVolume(cr *v1alpha1.ServerSet, name string, replicaIndex, version int) v1alpha1.Volume {
	return v1alpha1.Volume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels: map[string]string{
				serverSetLabel:                          cr.Name,
				fmt.Sprintf(indexLabel, "bootvolume"):   fmt.Sprintf("%d", replicaIndex),
				fmt.Sprintf(versionLabel, "bootvolume"): fmt.Sprintf("%d", version),
			},
		},
		ManagementPolicies: xpv1.ManagementPolicies{"*"},
		Spec: v1alpha1.VolumeSpec{
			ForProvider: v1alpha1.VolumeParameters{
				DatacenterCfg:    cr.Spec.ForProvider.DatacenterCfg,
				Name:             name,
				AvailabilityZone: "AUTO",
				Size:             cr.Spec.ForProvider.BootVolumeTemplate.Spec.Size,
				Type:             cr.Spec.ForProvider.BootVolumeTemplate.Spec.Type,
				Image:            cr.Spec.ForProvider.BootVolumeTemplate.Spec.Image,
				// todo add to template(?)
				ImagePassword: "imagePassword776",
			},
		}}
}
