package forwardingrule

import (
	"fmt"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/nlb/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/diff"
)

// IsUpToDate returns whether the ForwardingRule is up-to-date and a diff string
// describing any differences between the CR spec and the observed SDK state.
func IsUpToDate(cr *v1alpha1.ForwardingRule, observed sdkgo.NetworkLoadBalancerForwardingRule, listenerIP string, targetsIPs map[string]v1alpha1.ForwardingRuleTarget) (bool, string) {
	if cr == nil && observed.Properties == nil {
		return true, ""
	}
	if cr == nil || observed.Properties == nil {
		return false, "forwarding rule properties presence mismatch"
	}
	if observed.Metadata != nil && observed.Metadata.State != nil && (*observed.Metadata.State == compute.BUSY || *observed.Metadata.State == compute.UPDATING) {
		return true, ""
	}
	p := observed.Properties
	d := diff.New()
	if cr.Spec.ForProvider.Name != "" || p.Name != nil {
		d.Str("name", &cr.Spec.ForProvider.Name, p.Name)
	}
	if cr.Spec.ForProvider.Algorithm != "" || p.Algorithm != nil {
		d.Str("algorithm", &cr.Spec.ForProvider.Algorithm, p.Algorithm)
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
	diffRuleHealthCheck(d.Sub("healthCheck"), cr.Spec.ForProvider.HealthCheck, p.HealthCheck)
	diffTargets(d.Sub("targets"), targetsIPs, p.Targets)
	return d.Result()
}

func diffRuleHealthCheck(d *diff.Builder, cr v1alpha1.ForwardingRuleHealthCheck, observed *sdkgo.NetworkLoadBalancerForwardingRuleHealthCheck) {
	if observed == nil {
		return
	}
	if observed.Retries != nil {
		d.Int32("retries", &cr.Retries, observed.Retries)
	}
	if observed.ClientTimeout != nil {
		d.Int32("clientTimeout", &cr.ClientTimeout, observed.ClientTimeout)
	}
	if observed.ConnectTimeout != nil {
		d.Int32("connectTimeout", &cr.ConnectTimeout, observed.ConnectTimeout)
	}
	if observed.TargetTimeout != nil {
		d.Int32("targetTimeout", &cr.TargetTimeout, observed.TargetTimeout)
	}
}

func diffTargets(d *diff.Builder, configured map[string]v1alpha1.ForwardingRuleTarget, observed *[]sdkgo.NetworkLoadBalancerForwardingRuleTarget) {
	if observed == nil {
		if len(configured) != 0 {
			d.Add("", fmt.Sprintf("%d", len(configured)), diff.NilSentinel)
		}
		return
	}
	if len(*observed) != len(configured) {
		got := len(*observed)
		want := len(configured)
		d.Int("len", &want, &got)
		return
	}
	for i, obsTarget := range *observed {
		if obsTarget.Ip == nil {
			continue
		}
		cfgTarget, ok := configured[*obsTarget.Ip]
		if !ok {
			d.Index(i).Add("ip", diff.NilSentinel, *obsTarget.Ip)
			continue
		}
		diffTarget(d.Index(i), cfgTarget, &obsTarget, *obsTarget.Ip)
	}
}

func diffTarget(d *diff.Builder, cr v1alpha1.ForwardingRuleTarget, observed *sdkgo.NetworkLoadBalancerForwardingRuleTarget, ip string) {
	if observed.Ip != nil {
		d.Str("ip", &ip, observed.Ip)
	}
	if observed.Port != nil {
		d.Int32("port", &cr.Port, observed.Port)
	}
	if observed.Weight != nil {
		d.Int32("weight", &cr.Weight, observed.Weight)
	}
	if observed.ProxyProtocol != nil {
		d.Str("proxyProtocol", &cr.ProxyProtocol, observed.ProxyProtocol)
	}
	diffTargetHealthCheck(d.Sub("healthCheck"), cr.HealthCheck, observed.HealthCheck)
}

func diffTargetHealthCheck(d *diff.Builder, cr v1alpha1.ForwardingRuleTargetHealthCheck, observed *sdkgo.NetworkLoadBalancerForwardingRuleTargetHealthCheck) {
	if observed == nil {
		return
	}
	if observed.Check != nil {
		d.Bool("check", &cr.Check, observed.Check)
	}
	if observed.CheckInterval != nil {
		d.Int32("checkInterval", &cr.CheckInterval, observed.CheckInterval)
	}
	if observed.Maintenance != nil {
		d.Bool("maintenance", &cr.Maintenance, observed.Maintenance)
	}
}
