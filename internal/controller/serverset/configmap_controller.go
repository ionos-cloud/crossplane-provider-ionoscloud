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

// nameAndNamespace returns the name/namespace substConfigMap holds for crName, under lock. A
// dedicated method (rather than inlining "k.mu.Lock(); ...; k.mu.Unlock()" at each call site) lets
// callers still use defer for panic-safety - e.g. if crName has no entry yet, the nil-pointer
// dereference below would otherwise leave mu permanently locked - while the lock itself is only
// held for this lookup, not across the k8s API calls callers make afterwards.
func (k *kubeConfigmapController) nameAndNamespace(crName string) (name, namespace string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	entry := k.substConfigMap[crName]
	return entry.name, entry.namespace
}

func (k *kubeConfigmapController) FetchSubstitutionFromMap(ctx context.Context, crName, key string, replicaIndex, version int) string {
	name, namespace := k.nameAndNamespace(crName)

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
	name, namespace, identities := k.snapshot(crName)

	cfgMap := &v1.ConfigMap{}
	err := k.kube.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, cfgMap)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			cfgMap = &v1.ConfigMap{
				TypeMeta:  metav1.TypeMeta{},
				Name:      name,
				Namespace: namespace,
				Data:      identities,
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

// snapshot returns a copy of substConfigMap's name/namespace/identities for crName, under lock.
// See nameAndNamespace for why this is a dedicated locked method rather than inlined per call
// site. identities is cloned (not just referenced) so callers can read/mutate it after the lock
// is released without racing SetIdentity, which writes into the live map.
func (k *kubeConfigmapController) snapshot(crName string) (name, namespace string, identities map[string]string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	entry := k.substConfigMap[crName]
	return entry.name, entry.namespace, maps.Clone(entry.identities)
}

func (k *kubeConfigmapController) Get(ctx context.Context, name, ns string) (*v1.ConfigMap, error) {
	cfgMap := &v1.ConfigMap{}
	err := k.kube.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, cfgMap)
	return cfgMap, err
}

func (k *kubeConfigmapController) Delete(ctx context.Context, crName string) error {
	name, namespace := k.nameAndNamespace(crName)

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
			k.clearEntry(name)
			k.log.Info("ConfigMap has been deleted", "name", name, "namespace", namespace)
			return true, nil
		}
		return false, err
	}
	return false, nil
}

// clearEntry removes substConfigMap's entry for name, under lock (see nameAndNamespace).
func (k *kubeConfigmapController) clearEntry(name string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.substConfigMap[name] != nil {
		maps2.Clear(k.substConfigMap[name].identities)
		k.substConfigMap[name] = nil
		delete(k.substConfigMap, name)
	}
}
