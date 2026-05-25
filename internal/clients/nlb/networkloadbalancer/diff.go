package networkloadbalancer

import (
	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/nlb/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/diff"
)

// IsUpToDate returns whether the NetworkLoadBalancer is up-to-date and a diff
// string describing any differences between the CR spec and the observed SDK state.
func IsUpToDate(cr *v1alpha1.NetworkLoadBalancer, observed sdkgo.NetworkLoadBalancer, listenerLan, targetLan int32, ips []string) (bool, string) {
	if cr == nil && observed.Properties == nil {
		return true, ""
	}
	if cr == nil || observed.Properties == nil {
		return false, "nlb properties presence mismatch"
	}
	if cr.Status.AtProvider.State == compute.BUSY || cr.Status.AtProvider.State == compute.UPDATING {
		return true, ""
	}
	p := observed.Properties
	d := diff.New()
	if cr.Spec.ForProvider.Name != "" || p.Name != nil {
		d.Str("name", &cr.Spec.ForProvider.Name, p.Name)
	}
	if p.ListenerLan != nil {
		d.Int32("listenerLan", &listenerLan, p.ListenerLan)
	}
	if p.TargetLan != nil {
		d.Int32("targetLan", &targetLan, p.TargetLan)
	}
	if p.Ips != nil || len(ips) != 0 {
		d.StrSliceUnordered("ips", &ips, p.Ips)
	}
	if p.LbPrivateIps != nil || len(cr.Spec.ForProvider.LbPrivateIps) != 0 {
		d.StrSliceUnordered("lbPrivateIps", &cr.Spec.ForProvider.LbPrivateIps, p.LbPrivateIps)
	}
	return d.Result()
}
