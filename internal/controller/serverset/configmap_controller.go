package serverset

import (
	"context"
	"maps"
	"strconv"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	maps2 "golang.org/x/exp/maps"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/kube"
)

type kubeConfigmapControlManager interface {
	Get(ctx context.Context, name, ns string) (*v1.ConfigMap, error)
	Delete(ctx context.Context, crName string) error
	CreateOrUpdate(ctx context.Context, cr *v1alpha1.ServerSet) error
	SetSubstitutionConfigMap(name, namespace string)
	SetIdentity(crName, key, val string)
	FetchSubstitutionFromMap(ctx context.Context, crName, key string, replicaIndex, version int) string
}

// kubeConfigmapController - kubernetes client wrapper  for server resources
type kubeConfigmapController struct {
	kube client.Client
	log  logging.Logger
	// substConfigMap is shared between all serversets
	substConfigMap map[string]*substitutionConfig
}

func (k *kubeConfigmapController) SetIdentity(crName, key, val string) {
	k.substConfigMap[crName].identities[key] = val
}
func (k *kubeConfigmapController) SetSubstitutionConfigMap(name, namespace string) {
	if k.substConfigMap == nil {
		k.substConfigMap = make(map[string]*substitutionConfig)
	}
	if k.substConfigMap[name] == nil {
		k.substConfigMap[name] = &substitutionConfig{}
		k.substConfigMap[name].name = name
		k.substConfigMap[name].namespace = namespace
		k.substConfigMap[name].identities = make(map[string]string)
	}
}

func (k *kubeConfigmapController) FetchSubstitutionFromMap(ctx context.Context, crName, key string, replicaIndex, version int) string {
	substMap := &v1.ConfigMap{}
	err := k.kube.Get(ctx, client.ObjectKey{Namespace: k.substConfigMap[crName].namespace, Name: k.substConfigMap[crName].name}, substMap)
	if err != nil {
		k.log.Info("Error fetching configmap", "name", k.substConfigMap[crName].name, "namespace", k.substConfigMap[crName].namespace, "error", err)
		return ""
	}
	return substMap.Data[strconv.Itoa(replicaIndex)+"."+strconv.Itoa(version)+"."+key]
}

// CreateOrUpdate - creates a config map if it doesn't exist
func (k *kubeConfigmapController) CreateOrUpdate(ctx context.Context, cr *v1alpha1.ServerSet) error {
	cfgMap := &v1.ConfigMap{}
	crName := cr.Name
	err := k.kube.Get(ctx, client.ObjectKey{Namespace: k.substConfigMap[crName].namespace, Name: k.substConfigMap[crName].name}, cfgMap)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			cfgMap = &v1.ConfigMap{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      k.substConfigMap[crName].name,
					Namespace: k.substConfigMap[crName].namespace,
				},
				Data: k.substConfigMap[crName].identities,
			}

			cfgMap.SetOwnerReferences([]metav1.OwnerReference{
				utils.NewOwnerReference(cr.TypeMeta, cr.ObjectMeta, true, false),
			})
			k.log.Info("Creating ConfigMap", "name", k.substConfigMap[crName].name, "namespace", k.substConfigMap[crName].namespace, "identities", k.substConfigMap[crName].identities)
			return k.kube.Create(ctx, cfgMap)
		}
	} else {
		if len(k.substConfigMap[crName].identities) > 0 && !maps.Equal(k.substConfigMap[crName].identities, cfgMap.Data) {
			cfgMap = &v1.ConfigMap{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      k.substConfigMap[crName].name,
					Namespace: k.substConfigMap[crName].namespace,
				},
				Data: k.substConfigMap[crName].identities,
			}
			k.log.Info("Updating ConfigMap", "name", k.substConfigMap[crName].name, "namespace", k.substConfigMap[crName].namespace, "identities", k.substConfigMap[crName].identities)
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

func (k *kubeConfigmapController) Delete(ctx context.Context, crName string) error {
	cfgMap := &v1.ConfigMap{}
	err := k.kube.Get(ctx, client.ObjectKey{Namespace: k.substConfigMap[crName].namespace, Name: k.substConfigMap[crName].name}, cfgMap)
	if err != nil {
		return err
	}
	k.log.Info("Deleting ConfigMap", "name", k.substConfigMap[crName].name, "namespace", k.substConfigMap[crName].namespace)
	if err := k.kube.Delete(ctx, cfgMap); err != nil {
		return err
	}
	return kube.WaitForResource(ctx, kube.ResourceReadyTimeout, k.isDeleted, k.substConfigMap[crName].name, k.substConfigMap[crName].namespace)
}

func (k *kubeConfigmapController) isDeleted(ctx context.Context, name, namespace string) (bool, error) {
	_, err := k.Get(ctx, name, namespace)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			if k.substConfigMap[name] != nil {
				maps2.Clear(k.substConfigMap[name].identities)
				k.substConfigMap[name] = nil
				delete(k.substConfigMap, name)
			}
			k.log.Info("ConfigMap has been deleted", "name", name, "namespace", namespace)
			return true, nil
		}
		return false, err
	}
	return false, nil
}
