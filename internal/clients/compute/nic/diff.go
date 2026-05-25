package nic

import (
	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/diff"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

// IsUpToDate returns whether the Nic is up-to-date and a diff string describing
// any differences between the CR spec and the observed SDK state.
func IsUpToDate(cr *v1alpha1.Nic, nic sdkgo.Nic, ips []string) (bool, string) {
	if cr == nil && nic.Properties == nil {
		return true, ""
	}
	if cr == nil || nic.Properties == nil {
		return false, "nic properties presence mismatch"
	}
	if nic.Metadata != nil && nic.Metadata.State != nil && *nic.Metadata.State == sdkgo.Busy {
		return true, ""
	}
	p := nic.Properties
	d := diff.New()
	if cr.Spec.ForProvider.Name != "" || p.Name != nil {
		d.Str("name", &cr.Spec.ForProvider.Name, p.Name)
	}
	if p.Dhcp != nil {
		d.Bool("dhcp", &cr.Spec.ForProvider.Dhcp, p.Dhcp)
	}
	if cr.Spec.ForProvider.DhcpV6 != nil && p.Dhcpv6 != nil {
		d.Bool("dhcpv6", cr.Spec.ForProvider.DhcpV6, p.Dhcpv6)
	}
	if p.FirewallActive != nil {
		d.Bool("firewallActive", &cr.Spec.ForProvider.FirewallActive, p.FirewallActive)
	}
	if cr.Spec.ForProvider.FirewallType != "" || p.FirewallType != nil {
		d.Str("firewallType", &cr.Spec.ForProvider.FirewallType, p.FirewallType)
	}
	if cr.Spec.ForProvider.Vnet != "" || p.Vnet != nil {
		d.Str("vnet", &cr.Spec.ForProvider.Vnet, p.Vnet)
	}
	if len(ips) > 0 {
		ipv4s, ipv6s := GetIPvSlices(ips)
		if len(ipv4s) != 0 && p.HasIps() && !utils.ContainsStringSlices(ipv4s, *p.Ips) {
			d.StrSliceUnordered("ips", &ipv4s, p.Ips)
		}
		if len(ipv6s) != 0 && p.HasIpv6Ips() && !utils.ContainsStringSlices(ipv6s, *p.Ipv6Ips) {
			d.StrSliceUnordered("ipv6Ips", &ipv6s, p.Ipv6Ips)
		}
	}
	return d.Result()
}
