package serverset

import (
	"context"
	"fmt"
	"strings"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/ionos-cloud/sdk-go/v6"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
)

type kubeNicControlManager interface {
	Create(ctx context.Context, cr *v1alpha1.ServerSet, serverID, lanName string, replicaIndex, version int) (v1alpha1.Nic, error)
	Get(ctx context.Context, name, ns string) (*v1alpha1.Nic, error)
	Delete(ctx context.Context, name, namespace string) error
}

// kubeNicController - kubernetes client wrapper
type kubeNicController struct {
	kube client.Client
	log  logging.Logger
}

// Create creates a NIC CR and waits until in reaches AVAILABLE state
func (k *kubeNicController) Create(ctx context.Context, cr *v1alpha1.ServerSet, serverID, lanName string, replicaIndex, version int) (v1alpha1.Nic, error) {
	name := getNameFromIndex(cr.Name, resourceNIC, replicaIndex, version)
	k.log.Info("Creating NIC", "name", name)
	network := v1alpha1.Lan{}
	if err := k.kube.Get(ctx, types.NamespacedName{
		Namespace: cr.GetNamespace(),
		Name:      lanName,
	}, &network); err != nil {
		return v1alpha1.Nic{}, err
	}
	lanID := network.Status.AtProvider.LanID
	// no NIC found, create one
	createNic := fromServerSetToNic(cr, name, serverID, lanID, replicaIndex, version)
	createNic.SetProviderConfigReference(cr.Spec.ProviderConfigReference)
	if err := k.kube.Create(ctx, &createNic); err != nil {
		return v1alpha1.Nic{}, err
	}

	err := WaitForKubeResource(ctx, resourceReadyTimeout, k.isAvailable, createNic.Name, cr.Namespace)
	if err != nil {
		return v1alpha1.Nic{}, err
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
	k.log.Info("Checking if Nic is deleted", "name", name, "namespace", namespace)
	nic := &v1alpha1.Nic{}
	err := k.kube.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, nic)
	if err != nil {
		if errors.IsNotFound(err) {
			k.log.Info("Nic has been deleted", "name", name, "namespace", namespace)
			return true, nil
		}
		return false, nil
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
	return WaitForKubeResource(ctx, resourceReadyTimeout, k.isNicDeleted, condemnedVolume.Name, namespace)
}

func fromServerSetToNic(cr *v1alpha1.ServerSet, name, serverID, lanID string, replicaIndex, version int) v1alpha1.Nic {
	return v1alpha1.Nic{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.GetNamespace(),
			Labels: map[string]string{
				serverSetLabel:                   cr.Name,
				fmt.Sprintf(indexLabel, "nic"):   fmt.Sprintf("%d", replicaIndex),
				fmt.Sprintf(versionLabel, "nic"): fmt.Sprintf("%d", version),
			},
		},
		Spec: v1alpha1.NicSpec{
			ForProvider: v1alpha1.NicParameters{
				Name:          name,
				DatacenterCfg: cr.Spec.ForProvider.DatacenterCfg,
				ServerCfg: v1alpha1.ServerConfig{
					ServerID: serverID,
				},
				LanCfg: v1alpha1.LanConfig{
					LanID: lanID,
				},
			},
		},
	}
}
