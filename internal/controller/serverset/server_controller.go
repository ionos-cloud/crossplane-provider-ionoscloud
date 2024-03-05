package serverset

import (
	"context"
	"fmt"
	"strings"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/ionos-cloud/sdk-go/v6"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
)

type kubeServerControlManager interface {
	Create(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version, volumeVersion int) (v1alpha1.Server, error)
	Get(ctx context.Context, name, ns string) (*v1alpha1.Server, error)
}

// KubeServerController - kubernetes client wrapper for server resources
type KubeServerController struct {
	kube client.Client
	log  logging.Logger
}

// Create creates a server CR and waits until in reaches AVAILABLE state
func (k *KubeServerController) Create(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version, volumeVersion int) (v1alpha1.Server, error) {
	createServer := FromServerSetToServer(cr, replicaIndex, version, volumeVersion)
	k.log.Info("Creating Server", "name", createServer.Name)

	createServer.SetProviderConfigReference(cr.Spec.ProviderConfigReference)
	if err := k.kube.Create(ctx, &createServer); err != nil {
		return v1alpha1.Server{}, fmt.Errorf("while creating createServer %w ", err)
	}
	if err := WaitForKubeResource(ctx, ResourceReadyTimeout, k.isAvailable, createServer.Name, cr.Namespace); err != nil {
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
func (k *KubeServerController) Get(ctx context.Context, name, ns string) (*v1alpha1.Server, error) {
	obj := &v1alpha1.Server{}
	if err := k.kube.Get(ctx, types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

func (k *KubeServerController) isAvailable(ctx context.Context, name, namespace string) (bool, error) {
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
