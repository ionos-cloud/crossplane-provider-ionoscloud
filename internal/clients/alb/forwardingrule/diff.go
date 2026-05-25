package forwardingrule

import (
	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/alb/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/diff"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

// IsForwardingRuleUpToDate returns whether the ALB ForwardingRule is up-to-date
// and a diff string describing differences between the CR spec and the observed SDK state.
func IsForwardingRuleUpToDate(cr *v1alpha1.ForwardingRule, forwardingRule sdkgo.ApplicationLoadBalancerForwardingRule, listenerIP string) (bool, string) {
	if cr == nil && forwardingRule.Properties == nil {
		return true, ""
	}
	if cr == nil || forwardingRule.Properties == nil {
		return false, "alb forwarding rule properties presence mismatch"
	}
	if forwardingRule.Metadata != nil && forwardingRule.Metadata.State != nil &&
		(*forwardingRule.Metadata.State == "BUSY" || *forwardingRule.Metadata.State == "DEPLOYING") {
		return true, ""
	}
	p := forwardingRule.Properties
	d := diff.New()
	if cr.Spec.ForProvider.Name != "" || p.Name != nil {
		d.Str("name", &cr.Spec.ForProvider.Name, p.Name)
	}
	if cr.Spec.ForProvider.Protocol != "" || p.Protocol != nil {
		d.Str("protocol", &cr.Spec.ForProvider.Protocol, p.Protocol)
	}
	if listenerIP != "" || p.ListenerIp != nil {
		d.Str("listenerIp", &listenerIP, p.ListenerIp)
	}
	if p.ListenerPort != nil {
		d.Int32("listenerPort", &cr.Spec.ForProvider.ListenerPort, p.ListenerPort)
	}
	if p.ClientTimeout != nil {
		d.Int32("clientTimeout", &cr.Spec.ForProvider.ClientTimeout, p.ClientTimeout)
	}
	if p.ServerCertificates != nil && !utils.ContainsStringSlices(*p.ServerCertificates, cr.Spec.ForProvider.ServerCertificatesIDs) {
		d.StrSliceUnordered("serverCertificates", &cr.Spec.ForProvider.ServerCertificatesIDs, p.ServerCertificates)
	}
	if !equalHTTPRules(cr.Spec.ForProvider.HTTPRules, p.HttpRules) {
		d.Add("httpRules", "<changed>", "<changed>")
	}
	return d.Result()
}
