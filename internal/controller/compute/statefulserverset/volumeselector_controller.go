package statefulserverset

import (
	"context"
	"fmt"
	"strings"
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/kube"
)

// volumeSelectorName <serverset_name>-volume-selector
const volumeSelectorName = "%s-volume-selector"

type kubeVolumeSelectorManager interface {
	Get(ctx context.Context, name, ns string) (*v1alpha1.Volumeselector, error)
	Delete(ctx context.Context, name, ns string) error
	CreateOrUpdate(ctx context.Context, cr *v1alpha1.StatefulServerSet) error
}

// kubeBootVolumeController - kubernetes client wrapper  for server resources
type kubeVolumeSelectorController struct {
	kube client.Client
	log  logging.Logger
}

func (k *kubeVolumeSelectorController) Delete(ctx context.Context, name, ns string) error {
	obj, err := k.Get(ctx, name, ns)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if err := k.kube.Delete(ctx, obj); err != nil {
		return err
	}
	err = kube.WaitForResource(ctx, kube.ResourceReadyTimeout, k.isDeleted, name, ns)
	if err != nil {
		return fmt.Errorf("an error occurred while deleting %w", err)
	}
	return nil
}

func (k *kubeVolumeSelectorController) isDeleted(ctx context.Context, name, namespace string) (bool, error) {
	_, err := k.Get(ctx, name, namespace)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

// Get - returns a volume selector kubernetes object
func (k *kubeVolumeSelectorController) Get(ctx context.Context, name, ns string) (*v1alpha1.Volumeselector, error) {
	obj := &v1alpha1.Volumeselector{}
	err := k.kube.Get(ctx, types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}, obj)
	return obj, err
}

// CreateOrUpdate - creates a boot volume if it does not exist, or updates it if replicas changed
func (k *kubeVolumeSelectorController) CreateOrUpdate(ctx context.Context, cr *v1alpha1.StatefulServerSet) error {
	vsName := fmt.Sprintf(volumeSelectorName, cr.Name)
	k.log.Info("CreateOrUpdate VolumeSelector", "name", vsName)
	volumeSelector, err := k.Get(ctx, vsName, cr.Namespace)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			_, err = k.Create(ctx, cr)
			return err
		}
		return err
	}
	if volumeSelector != nil && volumeSelector.Spec.ForProvider.Replicas != cr.Spec.ForProvider.Replicas {
		volumeSelector.Spec.ForProvider.Replicas = cr.Spec.ForProvider.Replicas
		if err = k.kube.Update(ctx, volumeSelector); err != nil {
			return err
		}
	}
	k.log.Info("Finished CreateOrUpdate VolumeSelector", "name", vsName)

	return nil
}

// Create creates a volume selector CR and waits until in reaches AVAILABLE state
func (k *kubeVolumeSelectorController) Create(ctx context.Context, cr *v1alpha1.StatefulServerSet) (v1alpha1.Volumeselector, error) {
	name := fmt.Sprintf(volumeSelectorName, cr.Name)
	k.log.Info("Creating VolumeSelector", "name", name)

	volSelector := fromStatefulServerSetToVolumeSelector(cr)
	volSelector.SetProviderConfigReference(cr.Spec.ProviderConfigReference)
	if err := k.kube.Create(ctx, &volSelector); err != nil {
		return v1alpha1.Volumeselector{}, err
	}
	if err := kube.WaitForResource(ctx, kube.ResourceReadyTimeout, k.isAvailable, name, cr.Namespace); err != nil {
		return v1alpha1.Volumeselector{}, err
	}
	// get the volume again before returning to have the id populated
	kubeVolume, err := k.Get(ctx, name, cr.Namespace)
	if err != nil {
		return v1alpha1.Volumeselector{}, err
	}
	k.log.Info("Finished creating Volume", "name", name)

	return *kubeVolume, nil
}

// IsVolumeAvailable - checks if a volume selector is available
func (k *kubeVolumeSelectorController) isAvailable(ctx context.Context, name, namespace string) (bool, error) {
	obj, err := k.Get(ctx, name, namespace)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	if obj != nil && strings.EqualFold(obj.Status.AtProvider.State, ionoscloud.Available) {
		return true, nil
	}
	return false, err
}

func fromStatefulServerSetToVolumeSelector(cr *v1alpha1.StatefulServerSet) v1alpha1.Volumeselector {
	return v1alpha1.Volumeselector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf(volumeSelectorName, cr.GetName()),
			Namespace: cr.GetNamespace(),
			CreationTimestamp: metav1.Time{
				Time: time.Now(),
			},
			Labels: map[string]string{
				statefulServerSetLabel: cr.GetName(),
			},
		},
		Spec: v1alpha1.VolumeselectorSpec{
			ResourceSpec: xpv1.ResourceSpec{
				ManagementPolicies: cr.GetManagementPolicies(),
			},
			ForProvider: v1alpha1.VolumeSelectorParameters{
				Replicas:      cr.Spec.ForProvider.Replicas,
				ServersetName: getSSetName(cr),
			},
		},
	}
}
