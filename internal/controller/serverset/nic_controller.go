package serverset

import (
	"context"
	"fmt"
	"strings"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	"k8s.io/apimachinery/pkg/api/errors"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/kube"
)

type kubeNicControlManager interface {
	Create(ctx context.Context, cr *v1alpha1.ServerSet, serverID, lanName string, replicaIndex, nicIndex, version int) (v1alpha1.Nic, error)
	Get(ctx context.Context, name, ns string) (*v1alpha1.Nic, error)
	Delete(ctx context.Context, name, namespace string) error
	EnsureNICs(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int) error
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
	k.log.Info("Creating NIC", "name", name)
	lan := v1alpha1.Lan{}
	if err := k.kube.Get(ctx, types.NamespacedName{
		Namespace: cr.GetNamespace(),
		Name:      lanName,
	}, &lan); err != nil {
		return v1alpha1.Nic{}, err
	}
	// no NIC found, create one
	createNic := k.fromServerSetToNic(cr, name, serverID, lan, replicaIndex, nicIndex, version)
	if err := k.kube.Create(ctx, &createNic); err != nil {
		return v1alpha1.Nic{}, fmt.Errorf("while creating NIC %w ", err)
	}

	err := kube.WaitForResource(ctx, kube.ResourceReadyTimeout, k.isAvailable, createNic.Name, cr.Namespace)
	if err != nil {
		_ = k.Delete(ctx, createNic.Name, cr.Namespace)
		return v1alpha1.Nic{}, fmt.Errorf("while waiting for NIC to be populated %w ", err)
	}
	createdNic, err := k.Get(ctx, createNic.Name, cr.Namespace)
	if err != nil {
		return v1alpha1.Nic{}, fmt.Errorf("while getting NIC %w ", err)
	}
	k.log.Info("Finished creating NIC", "name", name)
	return *createdNic, nil
}

// isAvailable - checks if a volume is available
func (k *kubeNicController) isAvailable(ctx context.Context, name, namespace string) (bool, error) {
	obj, err := k.Get(ctx, name, namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	if !kube.IsSuccessfullyCreated(obj) {
		conditions := obj.Status.ResourceStatus.Conditions
		return false, fmt.Errorf("reason %s %w", conditions[len(conditions)-1].Message, kube.ErrExternalCreateFailed)
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
		if errors.IsNotFound(err) {
			k.log.Info("NIC has been deleted", "name", name, "namespace", namespace)
			return true, nil
		}
		return false, err
	}
	return false, nil
}

// Delete - deletes the nic k8s client and waits until it is deleted
func (k *kubeNicController) Delete(ctx context.Context, name, namespace string) error {
	condemnedVolume, err := k.Get(ctx, name, namespace)
	if err != nil {
		return err
	}
	if err := k.kube.Delete(ctx, condemnedVolume); err != nil {
		return err
	}
	return kube.WaitForResource(ctx, kube.ResourceReadyTimeout, k.isNicDeleted, condemnedVolume.Name, namespace)
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
				fmt.Sprintf(indexLabel, cr.GetName(), resourceNIC):   fmt.Sprintf("%d", replicaIndex),
				fmt.Sprintf(versionLabel, cr.GetName(), resourceNIC): fmt.Sprintf("%d", version),
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
func (k *kubeNicController) EnsureNICs(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int) error {
	k.log.Info("Ensuring NICs", "index", replicaIndex, "version", version)
	res := &v1alpha1.ServerList{}
	if err := listResFromSSetWithIndexAndVersion(ctx, k.kube, cr.GetName(), ResourceServer, replicaIndex, version, res); err != nil {
		return err
	}
	servers := res.Items
	// check if the NIC is attached to the server
	if len(servers) > 0 {
		for nicx := range cr.Spec.ForProvider.Template.Spec.NICs {
			if err := k.ensure(ctx, cr, servers[0].Status.AtProvider.ServerID, cr.Spec.ForProvider.Template.Spec.NICs[nicx].Reference, replicaIndex, nicx, version); err != nil {
				return err
			}
		}
	}
	k.log.Info("Finished ensuring NICs", "index", replicaIndex, "version", version)

	return nil
}

// EnsureNIC - creates a NIC if it does not exist
func (k *kubeNicController) ensure(ctx context.Context, cr *v1alpha1.ServerSet, serverID, lanName string, replicaIndex, nicIndex, version int) error {
	var nic *v1alpha1.Nic
	var err error
	nic, err = k.Get(ctx, getNicName(cr.Spec.ForProvider.Template.Spec.NICs[nicIndex].Name, replicaIndex, nicIndex, version), cr.Namespace)
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
