package statefulserverset

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/volumeselector"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/kube"
)

const resourceLAN = "lan"

type kubeLANControlManager interface {
	Create(ctx context.Context, cr *v1alpha1.StatefulServerSet, lanIndex int) (v1alpha1.Lan, error)
	Get(ctx context.Context, lanName, ns string) (*v1alpha1.Lan, error)
	Delete(ctx context.Context, name, namespace string) error
	Ensure(ctx context.Context, cr *v1alpha1.StatefulServerSet, lanIndex int) error
	ListLans(ctx context.Context, cr *v1alpha1.StatefulServerSet) (*v1alpha1.LanList, error)
	Update(ctx context.Context, cr *v1alpha1.StatefulServerSet, lanIndex int) (v1alpha1.Lan, error)
}

// kubeLANController - kubernetes client wrapper  for server resources
type kubeLANController struct {
	kube client.Client
	log  logging.Logger
}

// Create creates a lan CR and waits until in reaches AVAILABLE state
func (k *kubeLANController) Create(ctx context.Context, cr *v1alpha1.StatefulServerSet, lanIndex int) (v1alpha1.Lan, error) {
	name := cr.Spec.ForProvider.Lans[lanIndex].Metadata.Name
	k.log.Info("Creating LAN", "name", name, "ssset", cr.Name)

	createLAN := fromStatefulServerSetToLAN(cr, name, lanIndex)
	createLAN.SetOwnerReferences(utils.NewControllerOwnerReference(cr.TypeMeta, cr.ObjectMeta, true, false))
	if err := k.kube.Create(ctx, &createLAN); err != nil {
		return v1alpha1.Lan{}, fmt.Errorf("while creating lan %s %w", createLAN.Name, err)
	}
	if err := kube.WaitForResource(ctx, kube.ResourceReadyTimeout, k.isAvailable, name, cr.Namespace); err != nil {
		_ = k.Delete(ctx, name, cr.Namespace)
		return v1alpha1.Lan{}, fmt.Errorf("while waiting for LAN to be populated %s %w", name, err)
	}
	kubeLAN, err := k.Get(ctx, name, cr.Namespace)
	if err != nil {
		return v1alpha1.Lan{}, err
	}
	k.log.Info("Finished creating LAN", "name", name, "ssset", cr.Name)

	return *kubeLAN, nil
}

// isLanUpToDate - checks if the lan is up-to-date and update the kube lan object if needed
func isLanUpToDate(spec *v1alpha1.StatefulServerSetLanSpec, lan *v1alpha1.Lan) bool {
	switch {
	case spec.IPv6cidr != v1alpha1.LANAuto && lan.Spec.ForProvider.Ipv6Cidr != spec.IPv6cidr:
		lan.Spec.ForProvider.Ipv6Cidr = spec.IPv6cidr
		return false
	case lan.Spec.ForProvider.Public != spec.Public:
		lan.Spec.ForProvider.Public = spec.Public
		return false
	}
	return true
}

// Update - updates the lan CR and waits until in reaches AVAILABLE state
func (k *kubeLANController) Update(ctx context.Context, cr *v1alpha1.StatefulServerSet, lanIndex int) (v1alpha1.Lan, error) {
	name := cr.Spec.ForProvider.Lans[lanIndex].Metadata.Name

	updateKubeLAN, err := k.Get(ctx, name, cr.Namespace)
	if err != nil {
		return v1alpha1.Lan{}, err
	}

	if isLanUpToDate(&cr.Spec.ForProvider.Lans[lanIndex].Spec, updateKubeLAN) {
		return v1alpha1.Lan{}, nil
	}

	k.log.Info("Updating LAN", "name", name, "ssset", cr.Name)

	if err := k.kube.Update(ctx, updateKubeLAN); err != nil {
		return v1alpha1.Lan{}, err
	}
	if err := kube.WaitForResource(ctx, kube.ResourceReadyTimeout, k.isAvailable, name, cr.Namespace); err != nil {
		return v1alpha1.Lan{}, fmt.Errorf("while waiting for resource %s to be populated %w ", name, err)
	}
	updateKubeLAN, err = k.Get(ctx, name, cr.Namespace)
	if err != nil {
		return v1alpha1.Lan{}, err
	}
	k.log.Info("Finished updating LAN", "name", name, "ssset", cr.Name)
	return *updateKubeLAN, nil
}

// ListLans - lists all lans for a given StatefulServerSet
func (k *kubeLANController) ListLans(ctx context.Context, cr *v1alpha1.StatefulServerSet) (*v1alpha1.LanList, error) {
	lans := &v1alpha1.LanList{}
	if err := k.kube.List(ctx, lans, client.InNamespace(cr.Namespace), client.MatchingLabels{
		statefulServerSetLabel: cr.Name,
	}); err != nil {
		return nil, err
	}
	return lans, nil
}

// Get - returns a lan kubernetes object
func (k *kubeLANController) Get(ctx context.Context, lanName, ns string) (*v1alpha1.Lan, error) {
	obj := &v1alpha1.Lan{}
	err := k.kube.Get(ctx, types.NamespacedName{
		Namespace: ns,
		Name:      lanName,
	}, obj)
	return obj, err
}

// Delete - deletes the lan kube object and waits until it is deleted
func (k *kubeLANController) Delete(ctx context.Context, name, namespace string) error {
	condemnedLAN, err := k.Get(ctx, name, namespace)
	if err != nil {
		return err
	}
	if err := k.kube.Delete(ctx, condemnedLAN); err != nil {
		return fmt.Errorf("while deleting lan %s %w", name, err)
	}
	return kube.WaitForResource(ctx, kube.ResourceReadyTimeout, k.isLANDeleted, condemnedLAN.Name, namespace)
}

// isAvailable - checks if a lan is available
func (k *kubeLANController) isAvailable(ctx context.Context, name, namespace string) (bool, error) {
	obj, err := k.Get(ctx, name, namespace)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	if !kube.IsSuccessfullyCreated(obj) {
		conditions := obj.Status.ResourceStatus.Conditions
		return false, fmt.Errorf("resource name %s reason %s %w", obj.Name, conditions[len(conditions)-1].Message, kube.ErrExternalCreateFailed)
	}
	if obj != nil && obj.Status.AtProvider.LanID != "" && strings.EqualFold(obj.Status.AtProvider.State, ionoscloud.Available) {
		return true, nil
	}
	return false, err
}

func (k *kubeLANController) isLANDeleted(ctx context.Context, name, namespace string) (bool, error) {
	k.log.Info("Checking if LAN is deleted", "name", name, "namespace", namespace)
	obj := &v1alpha1.Lan{}
	err := k.kube.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, obj)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			k.log.Info("LAN has been deleted", "name", name, "namespace", namespace)
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func fromStatefulServerSetToLAN(cr *v1alpha1.StatefulServerSet, name string, lanIndex int) v1alpha1.Lan {
	lan := v1alpha1.Lan{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels: map[string]string{
				statefulServerSetLabel: cr.Name,
				fmt.Sprintf(volumeselector.IndexLabel, getSSetName(cr), resourceLAN): strconv.Itoa(lanIndex),
			},
		},
		Spec: v1alpha1.LanSpec{
			ResourceSpec: xpv1.ResourceSpec{
				ProviderConfigReference: cr.GetProviderConfigReference(),
				ManagementPolicies:      cr.GetManagementPolicies(),
				DeletionPolicy:          cr.GetDeletionPolicy(),
			},
			ForProvider: v1alpha1.LanParameters{
				DatacenterCfg: cr.Spec.ForProvider.DatacenterCfg,
				Name:          cr.Spec.ForProvider.Lans[lanIndex].Metadata.Name,
				Public:        cr.Spec.ForProvider.Lans[lanIndex].Spec.Public,
			},
		}}

	if cr.Spec.ForProvider.Lans[lanIndex].Spec.IPv6cidr != "" {
		lan.Spec.ForProvider.Ipv6Cidr = cr.Spec.ForProvider.Lans[lanIndex].Spec.IPv6cidr
	}
	return lan
}

// Ensure - creates a lan if it does not exist
func (k *kubeLANController) Ensure(ctx context.Context, cr *v1alpha1.StatefulServerSet, lanIndex int) error {
	k.log.Info("Ensuring LAN", "lanIndex", lanIndex, "ssset", cr.Name)
	res := &v1alpha1.LanList{}
	if err := k.kube.List(ctx, res, client.MatchingLabels{
		fmt.Sprintf(volumeselector.IndexLabel, getSSetName(cr), resourceLAN): strconv.Itoa(lanIndex),
	}); err != nil {
		return err
	}
	lans := res.Items
	if len(lans) == 0 {
		_, err := k.Create(ctx, cr, lanIndex)
		return err
	}
	k.log.Info("Finished ensuring LAN", "lanIndex", lanIndex, "ssset", cr.Name)
	return nil
}
