package serverset

import (
	"context"
	"fmt"
	"strings"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
)

type kubeServerControlManager interface {
	Create(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version, volumeVersion int) (v1alpha1.Server, error)
	Get(ctx context.Context, name, ns string) (*v1alpha1.Server, error)
	Delete(ctx context.Context, name, namespace string) error
	EnsureServer(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int) error
}

// kubeServerController - kubernetes client wrapper for server resources
type kubeServerController struct {
	kube client.Client
	log  logging.Logger
}

// Create creates a server CR and waits until in reaches AVAILABLE state
func (k *kubeServerController) Create(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version, volumeVersion int) (v1alpha1.Server, error) {
	createServer := fromServerSetToServer(cr, replicaIndex, version, volumeVersion)
	k.log.Info("Creating Server", "name", createServer.Name)

	createServer.SetProviderConfigReference(cr.Spec.ProviderConfigReference)
	if err := k.kube.Create(ctx, &createServer); err != nil {
		return v1alpha1.Server{}, fmt.Errorf("while creating createServer %w ", err)
	}
	if err := WaitForKubeResource(ctx, resourceReadyTimeout, k.isAvailable, createServer.Name, cr.Namespace); err != nil {
		return v1alpha1.Server{}, fmt.Errorf("while waiting for createServer to be populated %w ", err)
	}
	createdServer, err := k.Get(ctx, createServer.Name, cr.Namespace)
	if err != nil {
		return v1alpha1.Server{}, fmt.Errorf("while getting createServer %w ", err)
	}
	k.log.Info("Finished creating Server", "name", createServer.Name)

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
		if errors.IsNotFound(err) {
			return false, nil
		}
	}
	if obj != nil && obj.Status.AtProvider.ServerID != "" && strings.EqualFold(obj.Status.AtProvider.State, ionoscloud.Available) {
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
		fmt.Printf("error deleting server %v", err)
		return err
	}
	return WaitForKubeResource(ctx, resourceReadyTimeout, k.isServerDeleted, condemnedServer.Name, namespace)
}

func (k *kubeServerController) isServerDeleted(ctx context.Context, name, namespace string) (bool, error) {
	k.log.Info("Checking if Server is deleted", "name", name, "namespace", namespace)
	obj := &v1alpha1.Server{}
	err := k.kube.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, obj)
	if err != nil {
		if errors.IsNotFound(err) {
			k.log.Info("Server has been deleted", "name", name, "namespace", namespace)
			return true, nil
		}
		return false, nil
	}
	return false, nil
}

// fromServerSetToServer is a conversion function that converts a ServerSet resource to a Server resource
// attaches a bootvolume to the server based on replicaIndex
func fromServerSetToServer(cr *v1alpha1.ServerSet, replicaIndex, version, volumeVersion int) v1alpha1.Server {
	serverType := "server"
	return v1alpha1.Server{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getNameFromIndex(cr.Name, serverType, replicaIndex, version),
			Namespace: cr.Namespace,
			Labels: map[string]string{
				serverSetLabel:                        cr.Name,
				fmt.Sprintf(indexLabel, serverType):   fmt.Sprintf("%d", replicaIndex),
				fmt.Sprintf(versionLabel, serverType): fmt.Sprintf("%d", version),
			},
		},
		Spec: v1alpha1.ServerSpec{
			ForProvider: v1alpha1.ServerParameters{
				DatacenterCfg:    cr.Spec.ForProvider.DatacenterCfg,
				Name:             getNameFromIndex(cr.Name, serverType, replicaIndex, version),
				Cores:            cr.Spec.ForProvider.Template.Spec.Cores,
				RAM:              cr.Spec.ForProvider.Template.Spec.RAM,
				AvailabilityZone: "AUTO",
				CPUFamily:        cr.Spec.ForProvider.Template.Spec.CPUFamily,
				VolumeCfg: v1alpha1.VolumeConfig{
					VolumeIDRef: &xpv1.Reference{
						Name: getNameFromIndex(cr.Name, "bootvolume", replicaIndex, volumeVersion),
					},
				},
			},
		}}
}

// EnsureServer - creates a server CR if it does not exist
func (k *kubeServerController) EnsureServer(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int) error {
	k.log.Info("Ensuring Server", "index", replicaIndex, "version", version)
	res := &v1alpha1.ServerList{}
	err := ListResFromSSetWithIndexAndVersion(ctx, k.kube, resourceServer, replicaIndex, version, res)
	if err != nil {
		return err
	}
	servers := res.Items
	if len(servers) > 0 {
		k.log.Info("Server already exists", "name", servers[0].Name)
	} else {
		_, err := k.Create(ctx, cr, replicaIndex, version, version)
		if err != nil {
			return err
		}
	}
	k.log.Info("Finished ensuring Server", "index", replicaIndex, "version", version)

	return nil
}
