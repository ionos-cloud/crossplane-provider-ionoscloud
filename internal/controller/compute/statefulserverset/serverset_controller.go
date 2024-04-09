package statefulserverset

import (
	"context"
	"errors"
	"fmt"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/kube"
)

type kubeSSetControlManager interface {
	Create(ctx context.Context, cr *v1alpha1.StatefulServerSet) (*v1alpha1.ServerSet, error)
	Ensure(ctx context.Context, cr *v1alpha1.StatefulServerSet) error
	Update(ctx context.Context, cr *v1alpha1.StatefulServerSet) (v1alpha1.ServerSet, error)
}

// kubeServerSetController - kubernetes client wrapper for server set resources
type kubeServerSetController struct {
	kube client.Client
	log  logging.Logger
}

// Create creates a server set CR and waits until in reaches AVAILABLE state
func (k *kubeServerSetController) Create(ctx context.Context, cr *v1alpha1.StatefulServerSet) (*v1alpha1.ServerSet, error) {
	SSet := extractSSetFromSSSet(cr)
	k.log.Info("Creating ServerSet", "name", SSet.Name)

	if err := k.kube.Create(ctx, SSet); err != nil {
		return nil, err
	}

	k.log.Info("Finished creating ServerSet", "name", SSet.Name)
	return SSet, nil
}

// Update updates a server set CR
func (k *kubeServerSetController) Update(ctx context.Context, cr *v1alpha1.StatefulServerSet) (v1alpha1.ServerSet, error) {
	name := getSSetName(cr)
	updateObj, err := k.Get(ctx, name, cr.Namespace)
	if err != nil {
		return v1alpha1.ServerSet{}, err
	}
	areResUpToDate, err := areSSetResourcesUpToDate(ctx, k.kube, cr)
	if err != nil {
		return v1alpha1.ServerSet{}, err
	}
	if areResUpToDate {
		k.log.Info("ServerSet resources are up to date", "name", name)
		return v1alpha1.ServerSet{}, nil
	}

	k.log.Info("Updating ServerSet", "name", name)
	updateObj.Spec.ForProvider.Replicas = cr.Spec.ForProvider.Replicas
	updateObj.Spec.ForProvider.Template = cr.Spec.ForProvider.Template
	updateObj.Spec.ForProvider.BootVolumeTemplate = cr.Spec.ForProvider.BootVolumeTemplate
	if err := k.kube.Update(ctx, updateObj); err != nil {
		return v1alpha1.ServerSet{}, err
	}

	updateObj, err = k.Get(ctx, name, cr.Namespace)
	if err != nil {
		return v1alpha1.ServerSet{}, err
	}
	k.log.Info("Finished updating ServerSet", "name", name)
	return *updateObj, nil
}

// isAvailable - checks if a lan is available
func (k *kubeServerSetController) isAvailable(ctx context.Context, name, namespace string) (bool, error) {
	obj, err := k.Get(ctx, name, namespace)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	if obj == nil {
		return false, nil
	}
	if obj.Annotations[meta.AnnotationKeyExternalCreateFailed] != "" {
		// todo add here the internal ReconcileError if possible
		return false, kube.ErrExternalCreateFailed
	}
	if (len(obj.Status.AtProvider.ReplicaStatuses) == obj.Spec.ForProvider.Replicas) &&
		(obj.GetCondition(xpv1.TypeReady).Equal(xpv1.Available())) {
		return true, nil
	}
	return false, err
}

// Ensure - creates a server set if it does not exist
func (k *kubeServerSetController) Ensure(ctx context.Context, cr *v1alpha1.StatefulServerSet) error {
	SSetName := getSSetName(cr)
	k.log.Info("Ensuring ServerSet", "name", SSetName)
	kubeSSet := &v1alpha1.ServerSet{}
	err := k.kube.Get(ctx, types.NamespacedName{Name: SSetName, Namespace: cr.Namespace}, kubeSSet)

	switch {
	case err != nil && apiErrors.IsNotFound(err):
		_, err := k.Create(ctx, cr)
		if err != nil {
			return err
		}

		if err = kube.WaitForResource(ctx, kube.ServersetReadyTimeout, k.isAvailable, SSetName, cr.Namespace); err != nil {
			if errors.Is(err, kube.ErrExternalCreateFailed) {
				_ = k.Delete(ctx, SSetName, cr.Namespace)
			}
			return err
		}
	case err != nil:
		return err
	default:
		k.log.Info("ServerSet already exists", "name", SSetName)
		return nil
	}
	return nil
}

// Get - returns a serverset kubernetes object
func (k *kubeServerSetController) Get(ctx context.Context, ssetName, ns string) (*v1alpha1.ServerSet, error) {
	obj := &v1alpha1.ServerSet{}
	err := k.kube.Get(ctx, types.NamespacedName{
		Namespace: ns,
		Name:      ssetName,
	}, obj)
	return obj, err
}

func extractSSetFromSSSet(sSSet *v1alpha1.StatefulServerSet) *v1alpha1.ServerSet {
	return &v1alpha1.ServerSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getSSetName(sSSet),
			Namespace: sSSet.Namespace,
			Labels: map[string]string{
				statefulServerSetLabel: sSSet.Name,
			},
		},
		Spec: v1alpha1.ServerSetSpec{
			ResourceSpec: xpv1.ResourceSpec{
				ProviderConfigReference: sSSet.GetProviderConfigReference(),
				ManagementPolicies:      sSSet.GetManagementPolicies(),
			},
			ForProvider: v1alpha1.ServerSetParameters{
				Replicas:           sSSet.Spec.ForProvider.Replicas,
				DatacenterCfg:      sSSet.Spec.ForProvider.DatacenterCfg,
				Template:           sSSet.Spec.ForProvider.Template,
				BootVolumeTemplate: sSSet.Spec.ForProvider.BootVolumeTemplate,
			},
		},
	}
}

func getSSetName(cr *v1alpha1.StatefulServerSet) string {
	return fmt.Sprintf("%s-%s", cr.Name, cr.Spec.ForProvider.Template.Metadata.Name)
}

// Delete - deletes the server k8s client and waits until it is deleted
func (k *kubeServerSetController) Delete(ctx context.Context, name, namespace string) error {
	condemnedServer, err := k.Get(ctx, name, namespace)
	if err != nil {
		return err
	}
	if err := k.kube.Delete(ctx, condemnedServer); err != nil {
		return fmt.Errorf("error deleting server %w", err)
	}
	return kube.WaitForResource(ctx, kube.ResourceReadyTimeout, k.isServerDeleted, condemnedServer.Name, namespace)
}

func (k *kubeServerSetController) isServerDeleted(ctx context.Context, name, namespace string) (bool, error) {
	k.log.Info("Checking if Serverset is deleted", "name", name, "namespace", namespace)
	obj := &v1alpha1.Server{}
	err := k.kube.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, obj)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			k.log.Info("Serverset has been deleted", "name", name, "namespace", namespace)
			return true, nil
		}
		return false, err
	}
	return false, nil
}
