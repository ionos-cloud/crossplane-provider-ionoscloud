package applicationloadbalancer

import (
	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/alb/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/diff"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

// IsApplicationLoadBalancerUpToDate returns whether the ApplicationLoadBalancer is
// up-to-date and a diff string describing any differences between the CR spec and
// the observed SDK state.
func IsApplicationLoadBalancerUpToDate(cr *v1alpha1.ApplicationLoadBalancer, applicationloadbalancer sdkgo.ApplicationLoadBalancer, listenerLan, targetLan int32, ips []string) (bool, string) {
	if cr == nil && applicationloadbalancer.Properties == nil {
		return true, ""
	}
	if cr == nil || applicationloadbalancer.Properties == nil {
		return false, "alb properties presence mismatch"
	}
	if applicationloadbalancer.Metadata != nil && applicationloadbalancer.Metadata.State != nil &&
		(*applicationloadbalancer.Metadata.State == "BUSY" || *applicationloadbalancer.Metadata.State == "DEPLOYING") {
		return true, ""
	}
	p := applicationloadbalancer.Properties
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
	if p.Ips != nil && !utils.ContainsStringSlices(*p.Ips, cr.Status.AtProvider.PublicIPs) {
		d.StrSliceUnordered("ips", &cr.Status.AtProvider.PublicIPs, p.Ips)
	}
	if !utils.ContainsStringSlices(ips, cr.Status.AtProvider.PublicIPs) {
		d.StrSliceUnordered("publicIps", &cr.Status.AtProvider.PublicIPs, &ips)
	}
	if p.LbPrivateIps != nil && !utils.ContainsStringSlices(*p.LbPrivateIps, cr.Spec.ForProvider.LbPrivateIps) {
		d.StrSliceUnordered("lbPrivateIps", &cr.Spec.ForProvider.LbPrivateIps, p.LbPrivateIps)
	}
	return d.Result()
}
