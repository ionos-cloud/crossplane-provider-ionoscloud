package serverset

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	"golang.org/x/sync/errgroup"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/kube"
)

type kubeNicControlManager interface {
	Create(ctx context.Context, cr *v1alpha1.ServerSet, serverID, lanName string, replicaIndex, nicIndex, version int) (v1alpha1.Nic, error)
	Get(ctx context.Context, name, ns string) (*v1alpha1.Nic, error)
	Delete(ctx context.Context, name, namespace string) error
	EnsureNICs(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int, serverID string) error
}

// kubeNicController - kubernetes client wrapper
type kubeNicController struct {
	kube client.Client
	log  logging.Logger
}

// getNicName - generates name for a NIC
func getNicName(resourceName string, replicaIndex, nicIndex, version int) string {
	return fmt.Sprintf("%s-%d-%d-%d", resourceName, replicaIndex, nicIndex, version)
}

// Create creates a NIC CR and waits until in reaches AVAILABLE state
func (k *kubeNicController) Create(ctx context.Context, cr *v1alpha1.ServerSet, serverID, lanName string, replicaIndex, nicIndex, version int) (v1alpha1.Nic, error) {
	name := getNicName(cr.Spec.ForProvider.Template.Spec.NICs[nicIndex].Name, replicaIndex, nicIndex, version)
	k.log.Info("Creating NIC", "name", name, "serverset", cr.Name)
	lan := v1alpha1.Lan{}
	if err := k.kube.Get(ctx, types.NamespacedName{
		Namespace: cr.GetNamespace(),
		Name:      lanName,
	}, &lan); err != nil {
		return v1alpha1.Nic{}, err
	}

	// no NIC found, create one
	createNic := k.fromServerSetToNic(cr, name, serverID, lan, replicaIndex, nicIndex, version)
	createNic.SetOwnerReferences([]metav1.OwnerReference{
		utils.NewOwnerReference(cr.TypeMeta, cr.ObjectMeta, true, false),
	})
	if err := k.kube.Create(ctx, &createNic); err != nil {
		return v1alpha1.Nic{}, fmt.Errorf("while creating NIC %s for serverset %s %w ", createNic.Name, cr.Name, err)
	}

	k.log.Info("Waiting for NIC to become available", "name", name, "serverset", cr.Name)
	err := kube.WaitForResource(ctx, kube.ResourceReadyTimeout, k.isAvailable, createNic.Name, cr.Namespace)
	if err != nil {
		if strings.Contains(err.Error(), utils.Error422) {
			k.log.Info("NIC failed to become available, deleting it", "name", name, "serverset", cr.Name)
			_ = k.Delete(ctx, createNic.Name, cr.Namespace)
		}
		return v1alpha1.Nic{}, fmt.Errorf("while waiting for NIC name %s to be populated for serverset %s %w ", createNic.Name, cr.Name, err)
	}
	createdNic, err := k.Get(ctx, createNic.Name, cr.Namespace)
	if err != nil {
		return v1alpha1.Nic{}, fmt.Errorf("while getting NIC name %s for serverset %s %w ", createNic.Name, cr.Name, err)
	}
	k.log.Info("Finished creating NIC", "name", name, "serverset", cr.Name)
	return *createdNic, nil
}

// isAvailable - checks if a volume is available
func (k *kubeNicController) isAvailable(ctx context.Context, name, namespace string) (bool, error) {
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

	if obj != nil && obj.Status.AtProvider.NicID != "" && strings.EqualFold(obj.Status.AtProvider.State, ionoscloud.Available) {
		return true, nil
	}
	return false, err
}

// Get - returns a nic kubernetes object
func (k *kubeNicController) Get(ctx context.Context, name, ns string) (*v1alpha1.Nic, error) {
	obj := &v1alpha1.Nic{}
	if err := k.kube.Get(ctx, types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

func (k *kubeNicController) isNicDeleted(ctx context.Context, name, namespace string) (bool, error) {
	k.log.Info("Checking if NIC is deleted", "name", name, "namespace", namespace)
	nic := &v1alpha1.Nic{}
	err := k.kube.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, nic)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			k.log.Info("NIC has been deleted", "name", name, "namespace", namespace)
			return true, nil
		}
		return false, err
	}
	return false, nil
}

// Delete - deletes the nic k8s client and waits until it is deleted
func (k *kubeNicController) Delete(ctx context.Context, name, namespace string) error {
	condemnedNIC, err := k.Get(ctx, name, namespace)
	if err != nil {
		return err
	}
	if err := k.kube.Delete(ctx, condemnedNIC); err != nil {
		return err
	}
	return kube.WaitForResource(ctx, kube.ResourceReadyTimeout, k.isNicDeleted, condemnedNIC.Name, namespace)
}

func (k *kubeNicController) fromServerSetToNic(cr *v1alpha1.ServerSet, name, serverID string, lan v1alpha1.Lan, replicaIndex, nicIndex, version int) v1alpha1.Nic {
	serverSetNic := cr.Spec.ForProvider.Template.Spec.NICs[nicIndex]
	nic := v1alpha1.Nic{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.GetNamespace(),
			Labels: map[string]string{
				serverSetLabel: cr.Name,
				// TODO: This label should be nicIndex instead of replicaIndex later
				fmt.Sprintf(indexLabel, cr.GetName(), resourceNIC):    fmt.Sprintf("%d", replicaIndex),
				fmt.Sprintf(nicIndexLabel, cr.GetName(), resourceNIC): fmt.Sprintf("%d", nicIndex),
				fmt.Sprintf(versionLabel, cr.GetName(), resourceNIC):  fmt.Sprintf("%d", version),
			},
		},
		Spec: v1alpha1.NicSpec{
			ResourceSpec: xpv1.ResourceSpec{
				ProviderConfigReference: cr.GetProviderConfigReference(),
				ManagementPolicies:      cr.GetManagementPolicies(),
				DeletionPolicy:          cr.GetDeletionPolicy(),
			},
			ForProvider: v1alpha1.NicParameters{
				Name:          name,
				DatacenterCfg: cr.Spec.ForProvider.DatacenterCfg,
				ServerCfg: v1alpha1.ServerConfig{
					ServerID: serverID,
				},
				LanCfg: v1alpha1.LanConfig{
					LanID: lan.Status.AtProvider.LanID,
				},
				Dhcp: serverSetNic.DHCP,
			},
		},
	}
	if lan.Spec.ForProvider.Ipv6Cidr != "" {
		nic.Spec.ForProvider.DhcpV6 = serverSetNic.DHCPv6
	} else {
		k.log.Debug("DHCPv6 will not be set on the NIC since Ipv6Cidr is not set on the LAN", "lan", lan.Name, "nic", nic.Name)
	}

	if serverSetNic.VNetID != "" {
		nic.Spec.ForProvider.Vnet = serverSetNic.VNetID
	}
	return nic
}

// EnsureNICs - creates NICS if they do not exist
func (k *kubeNicController) EnsureNICs(
	ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int, serverID string,
) error {
	defer func() {
		k.log.Info("Finished ensuring NICs", "index", replicaIndex, "version", version, "serverset", cr.Name)
	}()
	k.log.Info("Ensuring NICs", "index", replicaIndex, "version", version, "serverset", cr.Name)
	errGroup, ctx := errgroup.WithContext(ctx)
	for nicx := range cr.Spec.ForProvider.Template.Spec.NICs {
		errGroup.Go(func() error {
			return k.ensure(ctx, cr, serverID, cr.Spec.ForProvider.Template.Spec.NICs[nicx].LanReference, replicaIndex, nicx, version)
		})
	}
	return errGroup.Wait()
}

// EnsureNIC - creates a NIC if it does not exist
func (k *kubeNicController) ensure(ctx context.Context, cr *v1alpha1.ServerSet, serverID, lanName string, replicaIndex, nicIndex, version int) error {
	var nic *v1alpha1.Nic
	var err error

	nicName := getNicName(cr.Spec.ForProvider.Template.Spec.NICs[nicIndex].Name, replicaIndex, nicIndex, version)
	nic, err = k.Get(ctx, nicName, cr.Namespace)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			createdNic, err := k.Create(ctx, cr, serverID, lanName, replicaIndex, nicIndex, version)
			if err != nil {
				return err
			}
			nic = &createdNic
		} else {
			return err
		}
	}

	if !strings.EqualFold(nic.Status.AtProvider.State, ionoscloud.Available) {
		return fmt.Errorf("observed NIC %s got state %s but expected %s", nic.GetName(), nic.Status.AtProvider.State, ionoscloud.Available)
	}
	return nil
}

func getPCISlotFromNIC(ctx context.Context, kube client.Client, serversetName string, replicaIndex, nicIndex int) (pciSlot int32, err error) {
	obj := &v1alpha1.NicList{}
	err = kube.List(ctx, obj, client.MatchingLabels{
		fmt.Sprintf(indexLabel, serversetName, resourceNIC):    strconv.Itoa(replicaIndex),
		fmt.Sprintf(nicIndexLabel, serversetName, resourceNIC): fmt.Sprintf("%d", nicIndex),
	})
	if err != nil {
		return 0, err
	}
	if len(obj.Items) > 0 {
		return obj.Items[0].Status.AtProvider.PCISlot, nil
	}
	return 0, fmt.Errorf("no nics found for serversetName %s and replicaIndex %d", serversetName, replicaIndex)
}
