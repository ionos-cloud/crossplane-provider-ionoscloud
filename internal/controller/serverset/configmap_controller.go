package serverset

import (
	"context"
	"maps"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/kube"
)

// substConfigMap is used to store the substitutions
var substConfigMap substitutionConfigMap

func init() {
	substConfigMap = substitutionConfigMap{
		identities: make(map[string]string),
		// will be replaced with serverset name
		name:      "",
		namespace: "default",
	}
}

type kubeConfigmapControlManager interface {
	Get(ctx context.Context, name, ns string) (*v1.ConfigMap, error)
	Delete(ctx context.Context, name, namespace string) error
	CreateOrUpdate(ctx context.Context) error
}

// kubeConfigmapController - kubernetes client wrapper  for server resources
type kubeConfigmapController struct {
	kube       client.Client
	log        logging.Logger
	wasDeleted bool
}

// CreateOrUpdate - creates a config map if is doesn't exist
func (k *kubeConfigmapController) CreateOrUpdate(ctx context.Context) error {
	// we want to make sure the configmap was not deleted
	if k.wasDeleted {
		return nil
	}
	cfgMap := &v1.ConfigMap{}
	err := k.kube.Get(ctx, client.ObjectKey{Namespace: substConfigMap.namespace, Name: substConfigMap.name}, cfgMap)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			cfgMap = &v1.ConfigMap{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      substConfigMap.name,
					Namespace: substConfigMap.namespace,
				},
				Data: substConfigMap.identities,
			}
			k.log.Info("Creating ConfigMap", "name", substConfigMap.name, "identities", substConfigMap.identities)
			return k.kube.Create(ctx, cfgMap)
		}
	} else {
		// time for an update
		if len(substConfigMap.identities) > 0 && !maps.Equal(substConfigMap.identities, cfgMap.Data) && len(substConfigMap.identities) > len(cfgMap.Data) {
			cfgMap = &v1.ConfigMap{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      substConfigMap.name,
					Namespace: substConfigMap.namespace,
				},
				Data: substConfigMap.identities,
			}
			k.log.Info("Updating ConfigMap", "name", substConfigMap.name, "identities", substConfigMap.identities)
			return k.kube.Update(ctx, cfgMap)
		}
	}
	return nil
}

func (k *kubeConfigmapController) Get(ctx context.Context, name, ns string) (*v1.ConfigMap, error) {
	cfgMap := &v1.ConfigMap{}
	err := k.kube.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, cfgMap)
	return cfgMap, err
}

func (k *kubeConfigmapController) Delete(ctx context.Context, name, namespace string) error {
	cfgMap := &v1.ConfigMap{}
	err := k.kube.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, cfgMap)
	if err != nil {
		return err
	}
	k.log.Info("Deleting ConfigMap", "name", name)
	if err := k.kube.Delete(ctx, cfgMap); err != nil {
		return err
	}
	k.wasDeleted = true
	return kube.WaitForResource(ctx, kube.ResourceReadyTimeout, k.isDeleted, substConfigMap.name, substConfigMap.namespace)

}

func (k *kubeConfigmapController) isDeleted(ctx context.Context, name, namespace string) (bool, error) {
	k.log.Info("Checking if ConfigMap is deleted", "name", name, "namespace", namespace)
	_, err := k.Get(ctx, name, namespace)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			k.log.Info("ConfigMap has been deleted", "name", name, "namespace", namespace)
			return true, nil
		}
		return false, err
	}
	return false, nil
}
