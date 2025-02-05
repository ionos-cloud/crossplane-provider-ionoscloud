package serverset

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
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/kube"
)

type kubeServerControlManager interface {
	Create(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int) (v1alpha1.Server, error)
	Get(ctx context.Context, name, ns string) (*v1alpha1.Server, error)
	Delete(ctx context.Context, name, namespace string) error
	Ensure(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version, volumeVersion int) error
	Update(ctx context.Context, server *v1alpha1.Server) error
}

// kubeServerController - kubernetes client wrapper for server resources
type kubeServerController struct {
	kube client.Client
	log  logging.Logger
}

// Create creates a server CR and waits until in reaches AVAILABLE state
func (k *kubeServerController) Create(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int) (v1alpha1.Server, error) {
	createServer := fromServerSetToServer(cr, replicaIndex, version)
	k.log.Info("Creating Server", "name", createServer.Name, "serverset", cr.Name)

	if err := k.kube.Create(ctx, &createServer); err != nil {
		return v1alpha1.Server{}, fmt.Errorf("while creating Server for serverset %s %w", cr.Name, err)
	}
	if err := kube.WaitForResource(ctx, kube.ResourceReadyTimeout, k.isAvailable, createServer.Name, cr.Namespace); err != nil {
		_ = k.Delete(ctx, createServer.Name, cr.Namespace)
		return v1alpha1.Server{}, fmt.Errorf("while waiting for Server to be populated in serverset %s %w ", cr.Name, err)
	}
	createdServer, err := k.Get(ctx, createServer.Name, cr.Namespace)
	if err != nil {
		return v1alpha1.Server{}, fmt.Errorf("while getting created Server for serverset %s %w", cr.Name, err)
	}
	k.log.Info("Finished creating Server", "name", createServer.Name, "serverset", cr.Name)

	return *createdServer, nil
}

// Get - returns a server object from kubernetes
func (k *kubeServerController) Get(ctx context.Context, name, ns string) (*v1alpha1.Server, error) {
	obj := &v1alpha1.Server{}
	if err := k.kube.Get(ctx, types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

func (k *kubeServerController) isAvailable(ctx context.Context, name, namespace string) (bool, error) {
	obj := &v1alpha1.Server{}
	err := k.kube.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, obj)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return false, nil
		}
	}
	if !kube.IsSuccessfullyCreated(obj) {
		conditions := obj.Status.ResourceStatus.Conditions
		return false, fmt.Errorf("resource name %s reason %s %w", obj.Name, conditions[len(conditions)-1].Message, kube.ErrExternalCreateFailed)
	}
	if obj.Status.AtProvider.ServerID != "" && strings.EqualFold(obj.Status.AtProvider.State, ionoscloud.Available) {
		return true, nil
	}

	return false, err
}

// Delete - deletes the server k8s client and waits until it is deleted
func (k *kubeServerController) Delete(ctx context.Context, name, namespace string) error {
	condemnedServer, err := k.Get(ctx, name, namespace)
	if err != nil {
		return err
	}
	if err := k.kube.Delete(ctx, condemnedServer); err != nil {
		return fmt.Errorf("error deleting server name %s %w", name, err)
	}
	return kube.WaitForResource(ctx, kube.ResourceReadyTimeout, k.isServerDeleted, condemnedServer.Name, namespace)
}

func (k *kubeServerController) isServerDeleted(ctx context.Context, name, namespace string) (bool, error) {
	k.log.Info("Checking if Server is deleted", "name", name, "namespace", namespace)
	obj := &v1alpha1.Server{}
	err := k.kube.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, obj)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			k.log.Info("Server has been deleted", "name", name, "namespace", namespace)
			return true, nil
		}
		return false, err
	}
	return false, nil
}

// fromServerSetToServer is a conversion function that converts a ServerSet resource to a Server resource
// attaches a bootvolume to the server based on replicaIndex
func fromServerSetToServer(cr *v1alpha1.ServerSet, replicaIndex, version int) v1alpha1.Server {
	return v1alpha1.Server{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getNameFrom(cr.Spec.ForProvider.Template.Metadata.Name, replicaIndex, version),
			Namespace: cr.Namespace,
			Labels: map[string]string{
				serverSetLabel: cr.Name,
				fmt.Sprintf(indexLabel, cr.GetName(), ResourceServer):   fmt.Sprintf("%d", replicaIndex),
				fmt.Sprintf(versionLabel, cr.GetName(), ResourceServer): fmt.Sprintf("%d", version),
			},
		},
		Spec: v1alpha1.ServerSpec{
			ResourceSpec: xpv1.ResourceSpec{
				ProviderConfigReference: cr.GetProviderConfigReference(),
				ManagementPolicies:      cr.GetManagementPolicies(),
				DeletionPolicy:          cr.GetDeletionPolicy(),
			},
			ForProvider: v1alpha1.ServerParameters{
				DatacenterCfg:    cr.Spec.ForProvider.DatacenterCfg,
				Name:             getNameFrom(cr.Spec.ForProvider.Template.Metadata.Name, replicaIndex, version),
				Cores:            cr.Spec.ForProvider.Template.Spec.Cores,
				RAM:              cr.Spec.ForProvider.Template.Spec.RAM,
				AvailabilityZone: GetZoneFromIndex(replicaIndex),
				CPUFamily:        cr.Spec.ForProvider.Template.Spec.CPUFamily,
				// todo revert if we go back to attaching volume on server creation
				// VolumeCfg: v1alpha1.VolumeConfig{
				// 	VolumeIDRef: &xpv1.Reference{
				// 		Name: getNameFrom(cr.Spec.ForProvider.BootVolumeTemplate.Metadata.Name, replicaIndex, volumeVersion),
				// 	},
				// },
			},
		}}
}

// GetZoneFromIndex returns ZONE_2 for odd and ZONE_1 for even index
func GetZoneFromIndex(index int) string {
	return fmt.Sprintf("ZONE_%d", index%2+1)
}

func (k *kubeServerController) Ensure(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version, volumeVersion int) error {
	k.log.Info("Ensuring Server", "index", replicaIndex, "version", version)
	res := &v1alpha1.ServerList{}
	if err := listResFromSSetWithIndexAndVersion(ctx, k.kube, cr.GetName(), ResourceServer, replicaIndex, version, res); err != nil {
		return err
	}
	servers := res.Items
	if len(servers) > 0 {
		k.log.Info("Server already exists", "name", servers[0].Name, "serverset", cr.Name)
		return nil
	}

	_, err := k.Create(ctx, cr, replicaIndex, version)
	if err != nil {
		return err
	}

	k.log.Info("Finished ensuring Server", "index", replicaIndex, "version", version, "serverset", cr.Name)

	return nil
}

func (k *kubeServerController) Update(ctx context.Context, server *v1alpha1.Server) error {
	return k.kube.Update(ctx, server)
}
