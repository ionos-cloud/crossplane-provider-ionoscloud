package serverset

import (
	"context"
	"maps"
	"strconv"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/kube"
)

type kubeConfigmapControlManager interface {
	Get(ctx context.Context, name, ns string) (*v1.ConfigMap, error)
	Delete(ctx context.Context) error
	CreateOrUpdate(ctx context.Context) error
	SetSubstitutionConfigMap(name, namespace string)
	SetIdentity(key, val string)
	FetchSubstitutionFromMap(ctx context.Context, key string, replicaIndex, version int) string
}

// kubeConfigmapController - kubernetes client wrapper  for server resources
type kubeConfigmapController struct {
	kube           client.Client
	log            logging.Logger
	substConfigMap substitutionConfig
}

func (k *kubeConfigmapController) SetIdentity(key, val string) {
	k.substConfigMap.identities[key] = val
}
func (k *kubeConfigmapController) SetSubstitutionConfigMap(name, namespace string) {
	if k.substConfigMap.name == "" {
		k.substConfigMap.name = name
		k.substConfigMap.namespace = namespace
		k.substConfigMap.identities = make(map[string]string)
	}
}

func (k *kubeConfigmapController) FetchSubstitutionFromMap(ctx context.Context, key string, replicaIndex, version int) string {
	substMap := &v1.ConfigMap{}
	err := k.kube.Get(ctx, client.ObjectKey{Namespace: k.substConfigMap.namespace, Name: k.substConfigMap.name}, substMap)
	if err != nil {
		k.log.Info("Error fetching configmap", "error", err)
		return ""
	}
	return substMap.Data[strconv.Itoa(replicaIndex)+"."+strconv.Itoa(version)+"."+key]
}

// CreateOrUpdate - creates a config map if it doesn't exist
func (k *kubeConfigmapController) CreateOrUpdate(ctx context.Context) error {
	cfgMap := &v1.ConfigMap{}
	err := k.kube.Get(ctx, client.ObjectKey{Namespace: k.substConfigMap.namespace, Name: k.substConfigMap.name}, cfgMap)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			cfgMap = &v1.ConfigMap{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      k.substConfigMap.name,
					Namespace: k.substConfigMap.namespace,
				},
				Data: k.substConfigMap.identities,
			}
			k.log.Info("Creating ConfigMap", "name", k.substConfigMap.name, "identities", k.substConfigMap.identities)
			return k.kube.Create(ctx, cfgMap)
		}
	} else {
		if len(k.substConfigMap.identities) > 0 && !maps.Equal(k.substConfigMap.identities, cfgMap.Data) {
			cfgMap = &v1.ConfigMap{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      k.substConfigMap.name,
					Namespace: k.substConfigMap.namespace,
				},
				Data: k.substConfigMap.identities,
			}
			k.log.Info("Updating ConfigMap", "name", k.substConfigMap.name, "identities", k.substConfigMap.identities)
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

func (k *kubeConfigmapController) Delete(ctx context.Context) error {
	cfgMap := &v1.ConfigMap{}
	err := k.kube.Get(ctx, client.ObjectKey{Namespace: k.substConfigMap.namespace, Name: k.substConfigMap.name}, cfgMap)
	if err != nil {
		return err
	}
	k.log.Info("Deleting ConfigMap", "name", k.substConfigMap.name)
	if err := k.kube.Delete(ctx, cfgMap); err != nil {
		return err
	}
	return kube.WaitForResource(ctx, kube.ResourceReadyTimeout, k.isDeleted, k.substConfigMap.name, k.substConfigMap.namespace)

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
