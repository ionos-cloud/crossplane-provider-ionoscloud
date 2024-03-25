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
	ssCR := extractSSetFromSSet(cr)
	k.log.Info("Creating ServerSet CR", "name", ssCR.Name)

	if err := k.kube.Create(ctx, ssCR); err != nil {
		return nil, err
	}

	k.log.Info("Finished creating ServerSet CR", "name", ssCR.Name)
	return ssCR, nil
}

// Ensure - creates a server set if it does not exist
func (k *kubeServerSetController) Ensure(ctx context.Context, cr *v1alpha1.StatefulServerSet) error {
	ssName := getSSName(cr.Name, cr.Spec.ForProvider.Template.Metadata.Name)
	k.log.Info("Ensuring ServerSet CR", "name", ssName)
	kubeSSet := &v1alpha1.ServerSet{}
	err := k.kube.Get(ctx, types.NamespacedName{Name: ssName, Namespace: cr.Namespace}, kubeSSet)

	switch {
	case err != nil && apiErrors.IsNotFound(err):
		_, e := k.Create(ctx, cr)
		return e
	case err != nil:
		return err
	default:
		k.log.Info("ServerSet already exists", "name", ssName)
		return nil
	}
}

func extractSSetFromSSet(sssCR *v1alpha1.StatefulServerSet) *v1alpha1.ServerSet {
	return &v1alpha1.ServerSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getSSName(sssCR.Name, sssCR.Spec.ForProvider.Template.Metadata.Name),
			Namespace: sssCR.Namespace,
			Labels: map[string]string{
				statefulServerSetLabel: sssCR.Name,
			},
		},
		Spec: v1alpha1.ServerSetSpec{
			ResourceSpec: xpv1.ResourceSpec{
				ProviderConfigReference: sssCR.GetProviderConfigReference(),
				ManagementPolicies:      sssCR.GetManagementPolicies(),
			},
			ForProvider: v1alpha1.ServerSetParameters{
				Replicas:           sssCR.Spec.ForProvider.Replicas,
				DatacenterCfg:      sssCR.Spec.ForProvider.DatacenterCfg,
				Template:           sssCR.Spec.ForProvider.Template,
				BootVolumeTemplate: sssCR.Spec.ForProvider.BootVolumeTemplate,
			},
		},
	}
}

func getSSName(sssName, ssName string) string {
	return fmt.Sprintf("%s-%s", sssName, ssName)
}
