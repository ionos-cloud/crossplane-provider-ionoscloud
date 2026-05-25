package ipblock

import (
	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/diff"
)

// IsIPBlockUpToDate returns whether the IPBlock is up-to-date and a diff string.
func IsIPBlockUpToDate(cr *v1alpha1.IPBlock, ipBlock sdkgo.IpBlock) (bool, string) {
	if cr == nil && ipBlock.Properties == nil {
		return true, ""
	}
	if cr == nil || ipBlock.Properties == nil {
		return false, "ip block properties presence mismatch"
	}
	if ipBlock.Metadata != nil && ipBlock.Metadata.State != nil && *ipBlock.Metadata.State == "BUSY" {
		return true, ""
	}
	p := ipBlock.Properties
	d := diff.New()
	if cr.Spec.ForProvider.Name != "" || p.Name != nil {
		d.Str("name", &cr.Spec.ForProvider.Name, p.Name)
	}
	return d.Result()
}
