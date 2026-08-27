package serverset

import (
	"context"
	"maps"
	"strconv"
	"sync"

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
	// mu guards substConfigMap. Different ServerSets are reconciled concurrently by
	// controller-runtime, and this single kubeConfigmapController instance (and its map) is
	// shared across all of them - every access must be synchronized.
	mu sync.Mutex
	// substConfigMap is shared between all serversets
	substConfigMap map[string]*substitutionConfig
}

func (k *kubeConfigmapController) SetIdentity(crName, key, val string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.substConfigMap[crName].identities[key] = val
}

func (k *kubeConfigmapController) SetSubstitutionConfigMap(name, namespace string) {
	k.mu.Lock()
	defer k.mu.Unlock()
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
	k.mu.Lock()
	name, namespace := k.substConfigMap[crName].name, k.substConfigMap[crName].namespace
	k.mu.Unlock()

	substMap := &v1.ConfigMap{}
	err := k.kube.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, substMap)
	if err != nil {
		k.log.Info("Error fetching configmap", "name", name, "namespace", namespace, "error", err)
		return ""
	}
	return substMap.Data[strconv.Itoa(replicaIndex)+"."+strconv.Itoa(version)+"."+key]
}

// CreateOrUpdate - creates a config map if it doesn't exist
func (k *kubeConfigmapController) CreateOrUpdate(ctx context.Context, cr *v1alpha1.ServerSet) error {
	crName := cr.Name

	k.mu.Lock()
	name, namespace := k.substConfigMap[crName].name, k.substConfigMap[crName].namespace
	identities := maps.Clone(k.substConfigMap[crName].identities)
	k.mu.Unlock()

	cfgMap := &v1.ConfigMap{}
	err := k.kube.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, cfgMap)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			cfgMap = &v1.ConfigMap{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Data: identities,
			}

			cfgMap.SetOwnerReferences([]metav1.OwnerReference{
				utils.NewOwnerReference(cr.TypeMeta, cr.ObjectMeta, true, false),
			})
			k.log.Info("Creating ConfigMap", "name", name, "namespace", namespace, "identities", identities)
			return k.kube.Create(ctx, cfgMap)
		}
	} else {
		if len(identities) > 0 && !maps.Equal(identities, cfgMap.Data) {
			maps.Copy(cfgMap.Data, identities)

			k.log.Info("Updating ConfigMap", "name", name, "namespace", namespace, "identities", identities)
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
	k.mu.Lock()
	name, namespace := k.substConfigMap[crName].name, k.substConfigMap[crName].namespace
	k.mu.Unlock()

	cfgMap := &v1.ConfigMap{}
	err := k.kube.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, cfgMap)
	if err != nil {
		return err
	}
	k.log.Info("Deleting ConfigMap", "name", name, "namespace", namespace)
	if err := k.kube.Delete(ctx, cfgMap); err != nil {
		return err
	}
	return kube.WaitForResource(ctx, kube.ResourceReadyTimeout, k.isDeleted, name, namespace)
}

func (k *kubeConfigmapController) isDeleted(ctx context.Context, name, namespace string) (bool, error) {
	_, err := k.Get(ctx, name, namespace)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			k.mu.Lock()
			if k.substConfigMap[name] != nil {
				maps2.Clear(k.substConfigMap[name].identities)
				k.substConfigMap[name] = nil
				delete(k.substConfigMap, name)
			}
			k.mu.Unlock()
			k.log.Info("ConfigMap has been deleted", "name", name, "namespace", namespace)
			return true, nil
		}
		return false, err
	}
	return false, nil
}
