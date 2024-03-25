package statefulserverset

import (
	"context"
	"fmt"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
)

type kubeSSetControlManager interface {
	Create(ctx context.Context, cr *v1alpha1.StatefulServerSet) (*v1alpha1.ServerSet, error)
	Ensure(ctx context.Context, cr *v1alpha1.StatefulServerSet) error
}

// kubeServerSetController - kubernetes client wrapper for server set resources
type kubeServerSetController struct {
	kube client.Client
	log  logging.Logger
}

// Create creates a server set CR and waits until in reaches AVAILABLE state
func (k *kubeServerSetController) Create(ctx context.Context, cr *v1alpha1.StatefulServerSet) (*v1alpha1.ServerSet, error) {
	SSet := extractSSetFromSSSet(cr)
	k.log.Info("Creating ServerSet CR", "name", SSet.Name)

	if err := k.kube.Create(ctx, SSet); err != nil {
		return nil, err
	}

	k.log.Info("Finished creating ServerSet CR", "name", SSet.Name)
	return SSet, nil
}

// Ensure - creates a server set if it does not exist
func (k *kubeServerSetController) Ensure(ctx context.Context, cr *v1alpha1.StatefulServerSet) error {
	SSetName := getSSetName(cr.Name, cr.Spec.ForProvider.Template.Metadata.Name)
	k.log.Info("Ensuring ServerSet CR", "name", SSetName)
	kubeSSet := &v1alpha1.ServerSet{}
	err := k.kube.Get(ctx, types.NamespacedName{Name: SSetName, Namespace: cr.Namespace}, kubeSSet)

	switch {
	case err != nil && apiErrors.IsNotFound(err):
		_, e := k.Create(ctx, cr)
		return e
	case err != nil:
		return err
	default:
		k.log.Info("ServerSet already exists", "name", SSetName)
		return nil
	}
}

func extractSSetFromSSSet(sSSet *v1alpha1.StatefulServerSet) *v1alpha1.ServerSet {
	return &v1alpha1.ServerSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getSSetName(sSSet.Name, sSSet.Spec.ForProvider.Template.Metadata.Name),
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

func getSSetName(sSSettName, sSetName string) string {
	return fmt.Sprintf("%s-%s", sSSettName, sSetName)
}
