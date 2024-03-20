package statefulserverset

import (
	"context"
	"fmt"
	"strings"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/serverset"
)

type kubeDataVolumeControlManager interface {
	Create(ctx context.Context, cr *v1alpha1.StatefulServerSet, replicaIndex, volumeIndex int) (v1alpha1.Volume, error)
	Get(ctx context.Context, volumeName, ns string) (*v1alpha1.Volume, error)
	Delete(ctx context.Context, name, namespace string) error
	Ensure(ctx context.Context, cr *v1alpha1.StatefulServerSet, replicaIndex, version int) error
}

// kubeDataVolumeController - kubernetes client wrapper  for server resources
type kubeDataVolumeController struct {
	kube client.Client
	log  logging.Logger
}

// Create creates a volume CR and waits until in reaches AVAILABLE state
func (k *kubeDataVolumeController) Create(ctx context.Context, cr *v1alpha1.StatefulServerSet, replicaIndex, volumeIndex int) (v1alpha1.Volume, error) {
	name := getNameFromIndexes(cr.Name, resourceDataVolume, replicaIndex, volumeIndex)
	k.log.Info("Creating Data Volume", "name", name)

	createVolume := fromStatefulServerSetToVolume(cr, name, replicaIndex, volumeIndex)
	if err := k.kube.Create(ctx, &createVolume); err != nil {
		return v1alpha1.Volume{}, err
	}
	if err := serverset.WaitForKubeResource(ctx, serverset.ResourceReadyTimeout, k.isAvailable, name, cr.Namespace); err != nil {
		return v1alpha1.Volume{}, err
	}
	// get the volume again before returning to have the id populated
	kubeVolume, err := k.Get(ctx, name, cr.Namespace)
	if err != nil {
		return v1alpha1.Volume{}, err
	}
	k.log.Info("Finished creating Data Volume", "name", name)

	return *kubeVolume, nil
}

// Get - returns a volume kubernetes object
func (k *kubeDataVolumeController) Get(ctx context.Context, volumeName, ns string) (*v1alpha1.Volume, error) {
	obj := &v1alpha1.Volume{}
	err := k.kube.Get(ctx, types.NamespacedName{
		Namespace: ns,
		Name:      volumeName,
	}, obj)
	return obj, err
}

// Delete - deletes the datavolume k8s client and waits until it is deleted
func (k *kubeDataVolumeController) Delete(ctx context.Context, name, namespace string) error {
	condemnedVolume, err := k.Get(ctx, name, namespace)
	if err != nil {
		return err
	}
	if err := k.kube.Delete(ctx, condemnedVolume); err != nil {
		return fmt.Errorf("error deleting data volume %w", err)
	}
	return serverset.WaitForKubeResource(ctx, serverset.ResourceReadyTimeout, k.isDataVolumeDeleted, condemnedVolume.Name, namespace)
}

// isAvailable - checks if a volume is available
func (k *kubeDataVolumeController) isAvailable(ctx context.Context, name, namespace string) (bool, error) {
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

func (k *kubeDataVolumeController) isDataVolumeDeleted(ctx context.Context, name, namespace string) (bool, error) {
	k.log.Info("Checking if data volume is deleted", "name", name, "namespace", namespace)
	obj := &v1alpha1.Volume{}
	err := k.kube.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, obj)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			k.log.Info("Data volume has been deleted", "name", name, "namespace", namespace)
			return true, nil
		}
		return false, nil
	}
	return false, nil
}

func fromStatefulServerSetToVolume(cr *v1alpha1.StatefulServerSet, name string, replicaIndex, volumeIndex int) v1alpha1.Volume {
	vol := v1alpha1.Volume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels: map[string]string{
				statefulServerSetLabel: cr.Name,
				fmt.Sprintf(replicaIndexLabel, cr.Name, resourceDataVolume): fmt.Sprintf("%d", replicaIndex),
				fmt.Sprintf(volumeIndexLabel, cr.Name, resourceDataVolume):  fmt.Sprintf("%d", volumeIndex),
			},
		},
		Spec: v1alpha1.VolumeSpec{
			ResourceSpec: xpv1.ResourceSpec{
				ProviderConfigReference: cr.GetProviderConfigReference(),
				ManagementPolicies:      cr.GetManagementPolicies(),
				DeletionPolicy:          cr.GetDeletionPolicy(),
			},
			ForProvider: v1alpha1.VolumeParameters{
				DatacenterCfg:    cr.Spec.ForProvider.DatacenterCfg,
				Name:             name,
				AvailabilityZone: serverset.GetZoneFromIndex(replicaIndex),
				Size:             cr.Spec.ForProvider.Volumes[volumeIndex].Spec.Size,
				Type:             cr.Spec.ForProvider.Volumes[volumeIndex].Spec.Type,
			},
		}}
	if cr.Spec.ForProvider.Volumes[volumeIndex].Spec.Image != "" {
		vol.Spec.ForProvider.Image = cr.Spec.ForProvider.Volumes[volumeIndex].Spec.Image
		vol.Spec.ForProvider.ImagePassword = "imagePassword776"

	} else {
		vol.Spec.ForProvider.LicenceType = "UNKNOWN"

	}
	if cr.Spec.ForProvider.Volumes[volumeIndex].Spec.UserData != "" {
		vol.Spec.ForProvider.UserData = cr.Spec.ForProvider.Volumes[volumeIndex].Spec.UserData
	}
	return vol
}

// Ensure - creates a data volume if it does not exist
func (k *kubeDataVolumeController) Ensure(ctx context.Context, cr *v1alpha1.StatefulServerSet, replicaIndex, volumeIndex int) error {
	k.log.Info("Ensuring DataVolume", "replicaIndex", replicaIndex, "volumeIndex", volumeIndex)
	res := &v1alpha1.VolumeList{}
	if err := listResFromSSSetWithReplicaAndIndex(ctx, k.kube, cr.Name, resourceDataVolume, replicaIndex, volumeIndex, res); err != nil {
		return err
	}
	volumes := res.Items
	if len(volumes) == 0 {
		volume, err := k.Create(ctx, cr, replicaIndex, volumeIndex)
		k.log.Info("Data volume State", "state", volume.Status.AtProvider.State)
		return err
	}
	k.log.Info("Finished ensuring DataVolume", "replicaIndex", replicaIndex, "volumeIndex", volumeIndex)

	return nil
}
