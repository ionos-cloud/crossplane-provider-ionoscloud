package targetgroup

import (
	"fmt"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/alb/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/diff"
)

// IsTargetGroupUpToDate returns whether the TargetGroup is up-to-date and a diff
// string describing differences between the CR spec and the observed SDK state.
func IsTargetGroupUpToDate(cr *v1alpha1.TargetGroup, targetGroup sdkgo.TargetGroup) (bool, string) {
	if cr == nil && targetGroup.Properties == nil {
		return true, ""
	}
	if cr == nil || targetGroup.Properties == nil {
		return false, "target group properties presence mismatch"
	}
	if targetGroup.Metadata != nil && targetGroup.Metadata.State != nil &&
		(*targetGroup.Metadata.State == "BUSY" || *targetGroup.Metadata.State == "DEPLOYING") {
		return true, ""
	}
	p := targetGroup.Properties
	d := diff.New()
	if cr.Spec.ForProvider.Name != "" || p.Name != nil {
		d.Str("name", &cr.Spec.ForProvider.Name, p.Name)
	}
	if cr.Spec.ForProvider.Protocol != "" || p.Protocol != nil {
		d.Str("protocol", &cr.Spec.ForProvider.Protocol, p.Protocol)
	}
	if cr.Spec.ForProvider.Algorithm != "" || p.Algorithm != nil {
		d.Str("algorithm", &cr.Spec.ForProvider.Algorithm, p.Algorithm)
	}
	if !equalTargetGroupTargets(cr.Spec.ForProvider.Targets, p.Targets) {
		d.Add("targets", fmt.Sprintf("%d", len(cr.Spec.ForProvider.Targets)), formatTargetCount(p.Targets))
	}
	if p.HealthCheck != nil && !equalTargetGroupHealthCheck(cr.Spec.ForProvider.HealthCheck, p.HealthCheck) {
		d.Add("healthCheck", "<changed>", "<changed>")
	}
	if !equalTargetGroupHTTPHealthCheck(cr.Spec.ForProvider.HTTPHealthCheck, p.HttpHealthCheck) {
		d.Add("httpHealthCheck", "<changed>", "<changed>")
	}
	return d.Result()
}

func formatTargetCount(targets *[]sdkgo.TargetGroupTarget) string {
	if targets == nil {
		return diff.NilSentinel
	}
	return fmt.Sprintf("%d", len(*targets))
}
