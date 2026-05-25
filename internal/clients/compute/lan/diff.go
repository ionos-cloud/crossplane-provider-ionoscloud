package lan

import (
	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/diff"
)

// IsUpToDate returns whether the Lan is up-to-date and a diff string describing
// any differences between the CR spec and the observed SDK state.
func IsUpToDate(cr *v1alpha1.Lan, lan sdkgo.Lan) (bool, string) {
	if cr == nil && lan.Properties == nil {
		return true, ""
	}
	if cr == nil || lan.Properties == nil {
		return false, "lan properties presence mismatch"
	}
	if lan.Metadata != nil && lan.Metadata.State != nil && *lan.Metadata.State == "BUSY" {
		return true, ""
	}
	p := lan.Properties
	d := diff.New()
	if cr.Spec.ForProvider.Name != "" || p.Name != nil {
		d.Str("name", &cr.Spec.ForProvider.Name, p.Name)
	}
	if p.Public != nil {
		d.Bool("public", &cr.Spec.ForProvider.Public, p.Public)
	}
	if cr.Spec.ForProvider.Ipv6Cidr != "" || p.Ipv6CidrBlock != nil {
		d.Str("ipv6CidrBlock", &cr.Spec.ForProvider.Ipv6Cidr, p.Ipv6CidrBlock)
	}
	pccID := cr.Spec.ForProvider.Pcc.PrivateCrossConnectID
	if pccID != "" || p.Pcc != nil {
		d.Str("pcc", &pccID, p.Pcc)
	}
	return d.Result()
}

// IsIPFailoverUpToDate returns whether the IPFailover is up-to-date and a diff string.
func IsIPFailoverUpToDate(cr *v1alpha1.IPFailover, lanIPFailovers []sdkgo.IPFailover, ipSetByUser string) (bool, string) {
	if cr == nil {
		return false, "cr is nil"
	}
	d := diff.New()
	if cr.Status.AtProvider.IP != ipSetByUser {
		statusIP := cr.Status.AtProvider.IP
		d.Str("status.ip", &ipSetByUser, &statusIP)
	}
	if cr.Status.AtProvider.State != "AVAILABLE" {
		state := cr.Status.AtProvider.State
		expected := "AVAILABLE"
		d.Str("status.state", &expected, &state)
	}
	if !IsIPFailoverPresent(lanIPFailovers, ipSetByUser, cr.Spec.ForProvider.NicCfg.NicID) {
		d.Add("ipFailover", ipSetByUser+"@"+cr.Spec.ForProvider.NicCfg.NicID, diff.NilSentinel)
	}
	return d.Result()
}
