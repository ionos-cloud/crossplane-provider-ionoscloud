package firewallrule

import (
	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/diff"
)

// IsFirewallRuleUpToDate returns whether the FirewallRule is up-to-date and a diff string.
func IsFirewallRuleUpToDate(cr *v1alpha1.FirewallRule, firewallRule sdkgo.FirewallRule, sourceIP, targetIP string) (bool, string) {
	if cr == nil && firewallRule.Properties == nil {
		return true, ""
	}
	if cr == nil || firewallRule.Properties == nil {
		return false, "firewall rule properties presence mismatch"
	}
	if firewallRule.Metadata != nil && firewallRule.Metadata.State != nil && *firewallRule.Metadata.State == "BUSY" {
		return true, ""
	}
	p := firewallRule.Properties
	d := diff.New()
	if cr.Spec.ForProvider.Name != "" || p.Name != nil {
		d.Str("name", &cr.Spec.ForProvider.Name, p.Name)
	}
	if cr.Spec.ForProvider.SourceMac != "" && p.SourceMac != nil {
		d.Str("sourceMac", &cr.Spec.ForProvider.SourceMac, p.SourceMac)
	}
	if p.SourceIp != nil {
		d.Str("sourceIp", &sourceIP, p.SourceIp)
	}
	if sourceIP != cr.Status.AtProvider.SourceIP {
		statusSrcIP := cr.Status.AtProvider.SourceIP
		d.Str("status.sourceIp", &sourceIP, &statusSrcIP)
	}
	if p.TargetIp != nil {
		d.Str("targetIp", &targetIP, p.TargetIp)
	}
	if targetIP != cr.Status.AtProvider.TargetIP {
		statusTgtIP := cr.Status.AtProvider.TargetIP
		d.Str("status.targetIp", &targetIP, &statusTgtIP)
	}
	if p.IcmpCode != nil {
		d.Int32("icmpCode", &cr.Spec.ForProvider.IcmpCode, p.IcmpCode)
	}
	if p.IcmpType != nil {
		d.Int32("icmpType", &cr.Spec.ForProvider.IcmpType, p.IcmpType)
	}
	if p.PortRangeStart != nil {
		d.Int32("portRangeStart", &cr.Spec.ForProvider.PortRangeStart, p.PortRangeStart)
	}
	if p.PortRangeEnd != nil {
		d.Int32("portRangeEnd", &cr.Spec.ForProvider.PortRangeEnd, p.PortRangeEnd)
	}
	if cr.Spec.ForProvider.Type != "" || p.Type != nil {
		d.Str("type", &cr.Spec.ForProvider.Type, p.Type)
	}
	return d.Result()
}
