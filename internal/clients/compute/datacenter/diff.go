package datacenter

import (
	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/diff"
)

// IsDatacenterUpToDate returns whether the Datacenter is up-to-date and a diff string.
func IsDatacenterUpToDate(cr *v1alpha1.Datacenter, datacenter sdkgo.Datacenter) (bool, string) {
	if cr == nil && datacenter.Properties == nil {
		return true, ""
	}
	if cr == nil || datacenter.Properties == nil {
		return false, "datacenter properties presence mismatch"
	}
	if datacenter.Metadata != nil && datacenter.Metadata.State != nil && *datacenter.Metadata.State == "BUSY" {
		return true, ""
	}
	p := datacenter.Properties
	d := diff.New()
	if cr.Spec.ForProvider.Name != "" || p.Name != nil {
		d.Str("name", &cr.Spec.ForProvider.Name, p.Name)
	}
	if cr.Spec.ForProvider.Description != "" && p.Description != nil {
		d.Str("description", &cr.Spec.ForProvider.Description, p.Description)
	}
	return d.Result()
}
