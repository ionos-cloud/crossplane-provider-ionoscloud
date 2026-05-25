package flowlog

import (
	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/diff"
)

// IsUpToDate returns whether the FlowLog is up-to-date and a diff string
// describing any differences between the CR spec and the observed SDK state.
func IsUpToDate(cr customResource, observed sdkgo.FlowLog) (bool, string) {
	if cr == nil && observed.Properties == nil {
		return true, ""
	}
	if cr == nil || observed.Properties == nil {
		return false, "flow log properties presence mismatch"
	}
	if observed.Metadata != nil && observed.Metadata.State != nil && (*observed.Metadata.State == compute.BUSY || *observed.Metadata.State == compute.UPDATING) {
		return true, ""
	}
	p := observed.Properties
	d := diff.New()
	name, action, direction, bucket := cr.GetFlowLogName(), cr.GetAction(), cr.GetDirection(), cr.GetBucket()
	if name != "" || p.Name != nil {
		d.Str("name", &name, p.Name)
	}
	if action != "" || p.Action != nil {
		d.Str("action", &action, p.Action)
	}
	if direction != "" || p.Direction != nil {
		d.Str("direction", &direction, p.Direction)
	}
	if bucket != "" || p.Bucket != nil {
		d.Str("bucket", &bucket, p.Bucket)
	}
	return d.Result()
}
