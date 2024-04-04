package statefulserverset

import (
	"context"
	"fmt"
	"strconv"
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
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/volumeselector"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/kube"
)

type kubeDataVolumeControlManager interface {
	Create(ctx context.Context, cr *v1alpha1.StatefulServerSet, replicaIndex, volumeIndex int) (v1alpha1.Volume, error)
	ListVolumes(ctx context.Context, cr *v1alpha1.StatefulServerSet) (*v1alpha1.VolumeList, error)
	Get(ctx context.Context, volumeName, ns string) (*v1alpha1.Volume, error)
	Update(ctx context.Context, cr *v1alpha1.StatefulServerSet, replicaIndex, volumeIndex int) (v1alpha1.Volume, error)
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
	name := generateNameFrom(cr.Name, volumeselector.ResourceDataVolume, replicaIndex, volumeIndex)
	k.log.Info("Creating DataVolume", "name", name)

	createVolume := fromSSSetToVolume(cr, name, replicaIndex, volumeIndex)
	if err := k.kube.Create(ctx, &createVolume); err != nil {
		return v1alpha1.Volume{}, err
	}
	if err := kube.WaitForResource(ctx, kube.ResourceReadyTimeout, k.isAvailable, name, cr.Namespace); err != nil {
		return v1alpha1.Volume{}, err
	}
	// get the volume again before returning to have the id populated
	kubeVolume, err := k.Get(ctx, name, cr.Namespace)
	if err != nil {
		return v1alpha1.Volume{}, err
	}
	k.log.Info("Finished creating DataVolume", "name", name)

	return *kubeVolume, nil
}

// ListVolumes - lists all volumes for a given StatefulServerSet
func (k *kubeDataVolumeController) ListVolumes(ctx context.Context, cr *v1alpha1.StatefulServerSet) (*v1alpha1.VolumeList, error) {
	objs := &v1alpha1.VolumeList{}
	if err := k.kube.List(ctx, objs, client.InNamespace(cr.Namespace), client.MatchingLabels{
		statefulServerSetLabel: cr.Name,
	}); err != nil {
		return nil, err
	}
	return objs, nil
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
		return fmt.Errorf("error deleting DataVolume %w", err)
	}
	return kube.WaitForResource(ctx, kube.ResourceReadyTimeout, k.isDataVolumeDeleted, condemnedVolume.Name, namespace)
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
	k.log.Info("Checking if DataVolume is deleted", "name", name, "namespace", namespace)
	obj := &v1alpha1.Volume{}
	err := k.kube.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, obj)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			k.log.Info("DataVolume has been deleted", "name", name, "namespace", namespace)
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func fromSSSetToVolume(cr *v1alpha1.StatefulServerSet, name string, replicaIndex, volumeIndex int) v1alpha1.Volume {
	vol := v1alpha1.Volume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels: map[string]string{
				statefulServerSetLabel: cr.Name,
				// todo replace with function
				fmt.Sprintf(volumeselector.IndexLabel, getSSetName(cr), volumeselector.ResourceDataVolume):       fmt.Sprintf("%d", replicaIndex),
				fmt.Sprintf(volumeselector.VolumeIndexLabel, getSSetName(cr), volumeselector.ResourceDataVolume): fmt.Sprintf("%d", volumeIndex),
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
				Name:             generateProviderNameFromIndex(cr.Spec.ForProvider.Volumes[volumeIndex].Metadata.Name, volumeIndex),
				AvailabilityZone: serverset.GetZoneFromIndex(replicaIndex),
				Size:             cr.Spec.ForProvider.Volumes[volumeIndex].Spec.Size,
				Type:             cr.Spec.ForProvider.Volumes[volumeIndex].Spec.Type,
			},
		}}
	if cr.Spec.ForProvider.Volumes[volumeIndex].Spec.Image != "" {
		vol.Spec.ForProvider.Image = cr.Spec.ForProvider.Volumes[volumeIndex].Spec.Image
		// todo - this will not work without a password
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

	if err := k.kube.List(ctx, res, client.MatchingLabels{
		createVolumeLabelKey(volumeselector.VolumeIndexLabel, getSSetName(cr)): strconv.Itoa(volumeIndex),
		createVolumeLabelKey(volumeselector.IndexLabel, getSSetName(cr)):       strconv.Itoa(replicaIndex),
	}); err != nil {
		return err
	}
	volumes := res.Items
	if len(volumes) == 0 {
		_, err := k.Create(ctx, cr, replicaIndex, volumeIndex)
		return err
	}
	k.log.Info("Finished ensuring DataVolume", "replicaIndex", replicaIndex, "volumeIndex", volumeIndex)

	return nil
}

// Update - updates the lan CR and waits until in reaches AVAILABLE state
func (k *kubeDataVolumeController) Update(ctx context.Context, cr *v1alpha1.StatefulServerSet, replicaIndex, volumeIndex int) (v1alpha1.Volume, error) {
	name := generateNameFrom(cr.GetName(), volumeselector.ResourceDataVolume, replicaIndex, volumeIndex)

	updateKubeDataVolume, err := k.Get(ctx, name, cr.Namespace)
	if err != nil {
		return v1alpha1.Volume{}, err
	}

	if isVolumeUpToDate(&cr.Spec.ForProvider.Volumes[volumeIndex].Spec, updateKubeDataVolume) {
		return v1alpha1.Volume{}, nil
	}

	k.log.Info("Updating DataVolume", "name", name)

	if err := k.kube.Update(ctx, updateKubeDataVolume); err != nil {
		return v1alpha1.Volume{}, err
	}
	if err := kube.WaitForResource(ctx, kube.ResourceReadyTimeout, k.isAvailable, name, cr.Namespace); err != nil {
		return v1alpha1.Volume{}, err
	}
	updateKubeDataVolume, err = k.Get(ctx, name, cr.Namespace)
	if err != nil {
		return v1alpha1.Volume{}, err
	}
	k.log.Info("Finished updating DataVolume", "name", name)
	return *updateKubeDataVolume, nil
}

// isVolumeUpToDate - checks if the lan is up-to-date and update the kube lan object if needed
func isVolumeUpToDate(spec *v1alpha1.StatefulServerSetVolumeSpec, lan *v1alpha1.Volume) bool {
	if lan.Spec.ForProvider.Size != spec.Size {
		lan.Spec.ForProvider.Size = spec.Size
		return false
	}
	return true
}

func createVolumeLabelKey(label string, name string) string {
	return fmt.Sprintf(label, name, volumeselector.ResourceDataVolume)
}

// generateNameFrom - generates name consisting of name, kind, index and version/second index
func generateNameFrom(resourceName, resourceType string, idx, version int) string {
	return fmt.Sprintf("%s-%s-%d-%d", resourceName, resourceType, idx, version)
}

func generateProviderNameFromIndex(resourceName string, idx int) string {
	return fmt.Sprintf("%s-%d", resourceName, idx)
}
