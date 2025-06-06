package serverset

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
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/ccpatch"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/ccpatch/substitution"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/kube"
)

type kubeBootVolumeControlManager interface {
	Create(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int) (v1alpha1.Volume, error)
	Get(ctx context.Context, volumeName, ns string) (*v1alpha1.Volume, error)
	Delete(ctx context.Context, name, namespace string) error
	Ensure(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int) error
}

// kubeBootVolumeController - kubernetes client wrapper  for server resources
type kubeBootVolumeController struct {
	kube          client.Client
	log           logging.Logger
	mapController kubeConfigmapControlManager
}

// Create creates a volume CR and waits until in reaches AVAILABLE state
func (k *kubeBootVolumeController) Create(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int) (v1alpha1.Volume, error) {
	name := getNameFrom(cr.Spec.ForProvider.BootVolumeTemplate.Metadata.Name, replicaIndex, version)
	hostname := getNameFrom(cr.Spec.ForProvider.Template.Metadata.Name, replicaIndex, version)
	k.log.Info("Creating BootVolume", "name", name, "serverset", cr.Name)
	var userDataPatcher *ccpatch.CloudInitPatcher
	var err error
	userDataPatcher, err = k.setPatcher(ctx, cr, replicaIndex, version, name, k.kube)
	if err != nil {
		return v1alpha1.Volume{}, err
	}
	createVolume := fromServerSetToVolume(cr, name, replicaIndex, version)
	userDataPatcher.SetEnv("hostname", hostname)
	createVolume.Spec.ForProvider.UserData = userDataPatcher.Patch("hostname", hostname).Encode()

	createVolume.SetOwnerReferences([]metav1.OwnerReference{
		utils.NewOwnerReference(cr.TypeMeta, cr.ObjectMeta, true, false),
	})
	if err := k.kube.Create(ctx, &createVolume); err != nil {
		return v1alpha1.Volume{}, err
	}

	k.log.Info("Waiting for BootVolume to become available", "name", name, "serverset", cr.Name)
	if err := kube.WaitForResource(ctx, kube.ResourceReadyTimeout, k.isAvailable, name, cr.Namespace); err != nil {
		if strings.Contains(err.Error(), utils.Error422) {
			k.log.Info("BootVolume failed to become available, deleting it", "name", name, "serverset", cr.Name)
			_ = k.Delete(ctx, createVolume.Name, cr.Namespace)
		}
		return v1alpha1.Volume{}, fmt.Errorf("while waiting for BootVolume %s to be populated %w ", createVolume.Name, err)
	}
	// get the volume again before returning to have the id populated
	kubeVolume, err := k.Get(ctx, name, cr.Namespace)
	if err != nil {
		return v1alpha1.Volume{}, err
	}
	k.log.Info("Finished creating BootVolume", "name", name, "serverset", cr.Name)

	return *kubeVolume, nil
}

// one global state where to hold used ip addressed for substitutions for each statefulserverset
var globalStateMap = make(map[string]substitution.GlobalState)

func (k *kubeBootVolumeController) setPatcher(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int, name string, kube client.Client) (*ccpatch.CloudInitPatcher, error) { // nolint:gocyclo
	var userDataPatcher *ccpatch.CloudInitPatcher
	var err error
	userData := cr.Spec.ForProvider.BootVolumeTemplate.Spec.UserData
	if _, ok := globalStateMap[cr.Name]; !ok {
		globalStateMap[cr.Name] = substitution.GlobalState{}
	}
	if len(cr.Spec.ForProvider.BootVolumeTemplate.Spec.Substitutions) > 0 {
		identifier := substitution.Identifier(name)
		substitutions := extractSubstitutions(cr.Spec.ForProvider.BootVolumeTemplate.Spec.Substitutions)
		userDataPatcher, err = ccpatch.NewCloudInitPatcherWithSubstitutions(userData, identifier, substitutions, ionoscloud.ToPtr(globalStateMap[cr.Name]))
		if err != nil {
			return userDataPatcher, fmt.Errorf("while creating cloud init patcher with substitutions for BootVolume %s on serverset %s %w", name, cr.Name, err)
		}
		namespace := "default"
		if cr.Spec.ForProvider.IdentityConfigMap.Namespace != "" {
			namespace = cr.Spec.ForProvider.IdentityConfigMap.Namespace
		}
		k.mapController.SetSubstitutionConfigMap(cr.Name, namespace)
		for substIndex, subst := range substitutions {
			if stateMapVal, exists := globalStateMap[cr.Name]; exists {
				stateSlice := stateMapVal.GetByIdentifier(identifier)
				if len(stateSlice) > 0 && substIndex <= len(stateSlice)-1 {
					val := stateSlice[substIndex].Value
					k.mapController.SetIdentity(cr.Name, strconv.Itoa(replicaIndex)+"."+strconv.Itoa(version)+"."+subst.Key, val)
				}
			}
		}
		err := k.mapController.CreateOrUpdate(ctx, cr)
		if err != nil {
			k.log.Info("while writing to substConfig map", "error", err, "serverset", cr.Name)
		}
	} else {
		userDataPatcher, err = ccpatch.NewCloudInitPatcher(userData)
		if err != nil {
			return userDataPatcher, fmt.Errorf("while creating cloud init patcher for BootVolume %s serverset %s  %w", name, cr.Name, err)
		}
	}
	err = setPCINICSlotEnv(ctx, cr.Spec.ForProvider.Template.Spec.NICs, cr.Name, replicaIndex, kube, *userDataPatcher)
	if err != nil {
		return userDataPatcher, err
	}

	return userDataPatcher, nil
}

func setPCINICSlotEnv(ctx context.Context, nics []v1alpha1.ServerSetTemplateNIC, serversetName string, replicaIndex int, kube client.Client, userDataPatcher ccpatch.CloudInitPatcher) error {
	for nicIndex := range nics {
		pciSlot, err := getPCISlotFromNIC(ctx, kube, serversetName, replicaIndex, nicIndex)
		if err != nil {
			return err
		}
		const nicPCISlotPrefix = "nic_pcislot_"
		snakeCaseName := strings.ReplaceAll(nics[nicIndex].Name, "-", "_")
		userDataPatcher.SetEnv(nicPCISlotPrefix+snakeCaseName, strconv.Itoa(int(pciSlot)))
	}
	return nil
}

func extractSubstitutions(v1Substitutions []v1alpha1.Substitution) []substitution.Substitution {
	substitutions := make([]substitution.Substitution, len(v1Substitutions))
	for idx, subst := range v1Substitutions {
		substitutions[idx] = substitution.Substitution{
			Type:                 subst.Type,
			Key:                  subst.Key,
			Unique:               subst.Unique,
			AdditionalProperties: subst.Options,
		}
	}
	return substitutions
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
		return fmt.Errorf("error deleting BootVolume %s %w", name, err)
	}
	return kube.WaitForResource(ctx, kube.ResourceReadyTimeout, k.isBootVolumeDeleted, condemnedVolume.Name, namespace)
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
	if !kube.IsSuccessfullyCreated(obj) {
		conditions := obj.Status.ResourceStatus.Conditions
		return false, fmt.Errorf("resource name %s reason %s %w", obj.Name, conditions[len(conditions)-1].Message, kube.ErrExternalCreateFailed)
	}
	if obj != nil && obj.Status.AtProvider.VolumeID != "" && strings.EqualFold(obj.Status.AtProvider.State, ionoscloud.Available) {
		return true, nil
	}
	return false, err
}

func (k *kubeBootVolumeController) isBootVolumeDeleted(ctx context.Context, name, namespace string) (bool, error) {
	k.log.Info("Checking if BootVolume is deleted", "name", name, "namespace", namespace)
	obj := &v1alpha1.Volume{}
	err := k.kube.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, obj)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			k.log.Info("BootVolume has been deleted", "name", name, "namespace", namespace)
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func fromServerSetToVolume(cr *v1alpha1.ServerSet, name string, replicaIndex, version int) v1alpha1.Volume {
	vol := v1alpha1.Volume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels: map[string]string{
				serverSetLabel: cr.Name,
				fmt.Sprintf(indexLabel, cr.GetName(), resourceBootVolume):   strconv.Itoa(replicaIndex),
				fmt.Sprintf(versionLabel, cr.GetName(), resourceBootVolume): strconv.Itoa(version),
			},
		},
		Spec: v1alpha1.VolumeSpec{
			ResourceSpec: xpv1.ResourceSpec{
				ProviderConfigReference: cr.GetProviderConfigReference(),
				ManagementPolicies:      cr.GetManagementPolicies(),
				DeletionPolicy:          cr.GetDeletionPolicy(),
			},
			ForProvider: v1alpha1.VolumeParameters{
				DatacenterCfg:       cr.Spec.ForProvider.DatacenterCfg,
				Name:                name,
				AvailabilityZone:    GetZoneFromIndex(replicaIndex),
				Size:                cr.Spec.ForProvider.BootVolumeTemplate.Spec.Size,
				Type:                cr.Spec.ForProvider.BootVolumeTemplate.Spec.Type,
				Image:               cr.Spec.ForProvider.BootVolumeTemplate.Spec.Image,
				UserData:            cr.Spec.ForProvider.BootVolumeTemplate.Spec.UserData,
				CPUHotPlug:          true,
				RAMHotPlug:          true,
				NicHotPlug:          true,
				NicHotUnplug:        true,
				DiscVirtioHotPlug:   true,
				DiscVirtioHotUnplug: true,
			},
		}}
	if cr.Spec.ForProvider.BootVolumeTemplate.Spec.ImagePassword != "" {
		vol.Spec.ForProvider.ImagePassword = cr.Spec.ForProvider.BootVolumeTemplate.Spec.ImagePassword
	}
	if len(cr.Spec.ForProvider.BootVolumeTemplate.Spec.SSHKeys) > 0 {
		vol.Spec.ForProvider.SSHKeys = cr.Spec.ForProvider.BootVolumeTemplate.Spec.SSHKeys
	}
	return vol
}

// Ensure - creates a boot volume if it does not exist
func (k *kubeBootVolumeController) Ensure(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int) error {
	k.log.Info("Ensuring BootVolume from serverset", "name", cr.Name, "replicaIndex", replicaIndex, "version", version)
	res := &v1alpha1.VolumeList{}
	if err := listResFromSSetWithIndexAndVersion(ctx, k.kube, cr.GetName(), resourceBootVolume, replicaIndex, version, res); err != nil {
		return err
	}
	volumes := res.Items
	if len(volumes) == 0 {
		_, err := k.Create(ctx, cr, replicaIndex, version)
		if err != nil {
			return err
		}
	}
	k.log.Info("Finished ensuring BootVolume", "replicaIndex", replicaIndex, "version", version, "serverset", cr.Name)
	return nil
}

func init() {
	globalStateMap = make(map[string]substitution.GlobalState)
}
