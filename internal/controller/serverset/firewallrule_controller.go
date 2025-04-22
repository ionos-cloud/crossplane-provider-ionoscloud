package serverset

import (
	"context"
	"fmt"
	"strings"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/kube"
)

type kubeFirewallRuleControlManager interface {
	Create(
		ctx context.Context, cr *v1alpha1.ServerSet, nic *v1alpha1.Nic,
		firewallRuleSpec v1alpha1.ServerSetTemplateFirewallRuleSpec,
		serverID, firewallRuleName string,
	) (v1alpha1.FirewallRule, error)
	Get(ctx context.Context, name, namespace string) (*v1alpha1.FirewallRule, error)
	Delete(ctx context.Context, name, namespace string) error
	EnsureFirewallRules(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int, serverID string) error
}

type kubeFirewallRuleController struct {
	kube client.Client
	log  logging.Logger
}

// Get - retrieves a Firewall Rule object
func (k *kubeFirewallRuleController) Get(ctx context.Context, name, namespace string) (*v1alpha1.FirewallRule, error) {
	obj := &v1alpha1.FirewallRule{}
	if err := k.kube.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// Create - creates a Firewall Rule object
func (k *kubeFirewallRuleController) Create(
	ctx context.Context, cr *v1alpha1.ServerSet, nic *v1alpha1.Nic,
	firewallRuleSpec v1alpha1.ServerSetTemplateFirewallRuleSpec,
	serverID, firewallRuleName string,
) (v1alpha1.FirewallRule, error) {
	k.log.Info("Creating Firewall Rule", "name", firewallRuleName, "serverset", cr.Name)

	toBeCreatedFirewallRule := fwRule(nic, cr, firewallRuleSpec, serverID, firewallRuleName)
	toBeCreatedFirewallRule.SetOwnerReferences([]metav1.OwnerReference{
		utils.NewOwnerReference(cr.TypeMeta, cr.ObjectMeta, true, false),
	})
	if err := k.kube.Create(ctx, &toBeCreatedFirewallRule); err != nil {
		return v1alpha1.FirewallRule{}, fmt.Errorf(
			"while creating Firewall Rule %s for serverset %s %w", firewallRuleName, cr.Name, err,
		)
	}

	k.log.Info("Waiting for Firewall Rule to become available", "name", firewallRuleName, "serverset", cr.Name)
	if err := kube.WaitForResource(
		ctx, kube.ResourceReadyTimeout, k.isAvailable, toBeCreatedFirewallRule.Name, cr.Namespace,
	); err != nil {
		if strings.Contains(err.Error(), utils.Error422) {
			k.log.Info(
				"Firewall Rule failed to become available, deleting it", "name", toBeCreatedFirewallRule.Name,
				"serverset", cr.Name,
			)
			_ = k.Delete(ctx, toBeCreatedFirewallRule.Name, cr.Namespace)
		}
		return v1alpha1.FirewallRule{}, fmt.Errorf(
			"while waiting for Firewall Rule name %s to be populated for serverset %s %w", toBeCreatedFirewallRule.Name,
			cr.Name, err,
		)
	}

	createdFirewallRule, err := k.Get(ctx, toBeCreatedFirewallRule.Name, cr.Namespace)
	if err != nil {
		return v1alpha1.FirewallRule{}, fmt.Errorf(
			"while getting Firewall Rule %s for serverset %s %w", toBeCreatedFirewallRule.Name, cr.Name, err,
		)
	}

	k.log.Info("Finished creating Firewall Rule", "name", firewallRuleName, "serverset", cr.Name)
	return *createdFirewallRule, nil
}

// Delete - deletes a Firewall Rule object
func (k *kubeFirewallRuleController) Delete(ctx context.Context, name, namespace string) error {
	condemnedFirewallRule, err := k.Get(ctx, name, namespace)
	if err != nil {
		return err
	}

	if err := k.kube.Delete(ctx, condemnedFirewallRule); err != nil {
		return err
	}

	return kube.WaitForResource(ctx, kube.ResourceReadyTimeout, k.isDeleted, name, namespace)
}

// EnsureFirewallRules - ensures that the firewall rules are created for the server set
func (k *kubeFirewallRuleController) EnsureFirewallRules(
	ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int, serverID string,
) error {
	// loop through all NICs and attempt to ensure firewall rules
	for nicIdx, nicSpec := range cr.Spec.ForProvider.Template.Spec.NICs {
		// if firewall is not active, ignore any firewall rules declared in the NIC spec
		if !nicSpec.FirewallActive {
			continue
		}

		nicName := getNicName(nicSpec.Name, replicaIndex, nicIdx, version)
		nic := &v1alpha1.Nic{}
		if err := k.kube.Get(ctx, client.ObjectKey{Name: nicName, Namespace: cr.Namespace}, nic); err != nil {
			return err
		}

		k.log.Info("Ensuring Firewall Rules", "NIC", nicSpec.Name, "index", replicaIndex, "version", version)
		for firewallIdx, firewallRuleSpec := range nicSpec.FirewallRules {
			firewallRuleName := getFirewallRuleName(
				firewallRuleSpec.Name, replicaIndex, nicIdx, firewallIdx, version,
			)
			if err := k.ensure(ctx, cr, firewallRuleSpec, nic, firewallRuleName, serverID); err != nil {
				return err
			}
		}
		k.log.Info("Finished ensuring Firewall Rules", "NIC", nicSpec.Name, "index", replicaIndex, "version", version)
	}

	return nil
}

func (k *kubeFirewallRuleController) ensure(
	ctx context.Context, cr *v1alpha1.ServerSet, firewallRuleSpec v1alpha1.ServerSetTemplateFirewallRuleSpec,
	nic *v1alpha1.Nic, firewallRuleName, serverID string,
) error {
	var firewallRule *v1alpha1.FirewallRule
	var err error

	firewallRule, err = k.Get(ctx, firewallRuleName, cr.Namespace)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			createdFirewallRule, err := k.Create(ctx, cr, nic, firewallRuleSpec, serverID, firewallRuleName)
			if err != nil {
				return err
			}
			firewallRule = &createdFirewallRule
		} else {
			return err
		}
	}

	if !strings.EqualFold(firewallRule.Status.AtProvider.State, ionoscloud.Available) {
		return fmt.Errorf(
			"observed Firewall Rule %s got state %s but expected %s",
			firewallRule.GetName(), firewallRule.Status.AtProvider.State, ionoscloud.Available,
		)
	}

	return nil
}

func (k *kubeFirewallRuleController) isAvailable(ctx context.Context, name string, namespace string) (bool, error) {
	k.log.Info("Checking if Firewall Rule is available", "name", name, "namespace", namespace)
	firewallRule := &v1alpha1.FirewallRule{}

	if err := k.kube.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, firewallRule); err != nil {
		if apiErrors.IsNotFound(err) {
			return false, nil
		}

		return false, err
	}

	if !kube.IsSuccessfullyCreated(firewallRule) {
		conditions := firewallRule.Status.Conditions

		return false, fmt.Errorf(
			"resource name %s reason %s %w", name, conditions[len(conditions)-1].Message, kube.ErrExternalCreateFailed,
		)
	}

	if firewallRule != nil && firewallRule.Status.AtProvider.FirewallRuleID != "" && strings.EqualFold(
		firewallRule.Status.AtProvider.State, ionoscloud.Available,
	) {
		return true, nil
	}
	return false, nil
}

func (k *kubeFirewallRuleController) isDeleted(ctx context.Context, name string, namespace string) (bool, error) {
	k.log.Info("Checking if Firewall Rule is deleted", "name", name, "namespace", namespace)
	firewallRule := &v1alpha1.FirewallRule{}

	if err := k.kube.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, firewallRule); err != nil {
		if apiErrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}

	return false, nil
}

// getFirewallRuleName generates a name for the firewall rule.
func getFirewallRuleName(resourceName string, replicaIndex, nicIndex, firewallRuleIndex, version int) string {
	return fmt.Sprintf(
		"%s-%d-%d-%d-%d", resourceName, replicaIndex, nicIndex, firewallRuleIndex, version,
	)
}

func fwRule(
	nic *v1alpha1.Nic, cr *v1alpha1.ServerSet, fwr v1alpha1.ServerSetTemplateFirewallRuleSpec, serverID, fwrName string,
) v1alpha1.FirewallRule {
	return v1alpha1.FirewallRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fwrName,
			Namespace: cr.Namespace,
			Labels: map[string]string{
				serverSetLabel: cr.Name,
			},
		},
		Spec: v1alpha1.FirewallRuleSpec{
			ResourceSpec: xpv1.ResourceSpec{
				ProviderConfigReference: cr.GetProviderConfigReference(),
				ManagementPolicies:      cr.GetManagementPolicies(),
				DeletionPolicy:          cr.GetDeletionPolicy(),
			},
			ForProvider: v1alpha1.FirewallRuleParameters{
				Name:          fwrName,
				DatacenterCfg: cr.Spec.ForProvider.DatacenterCfg,
				ServerCfg: v1alpha1.ServerConfig{
					ServerID: serverID,
				},
				NicCfg: v1alpha1.NicConfig{
					NicID: nic.Status.AtProvider.NicID,
				},
				Protocol:       fwr.Protocol,
				SourceMac:      fwr.SourceMac,
				SourceIPCfg:    fwr.SourceIPCfg,
				TargetIPCfg:    fwr.TargetIPCfg,
				IcmpCode:       fwr.IcmpCode,
				IcmpType:       fwr.IcmpType,
				PortRangeStart: fwr.PortRangeStart,
				PortRangeEnd:   fwr.PortRangeEnd,
				Type:           fwr.Type,
			},
		},
	}
}
