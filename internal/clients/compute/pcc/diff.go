package pcc

import (
	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/diff"
)

// IsPrivateCrossConnectUpToDate returns whether the Pcc is up-to-date and a diff string.
func IsPrivateCrossConnectUpToDate(cr *v1alpha1.Pcc, privateCrossConnect sdkgo.PrivateCrossConnect) (bool, string) {
	if cr == nil && privateCrossConnect.Properties == nil {
		return true, ""
	}
	if cr == nil || privateCrossConnect.Properties == nil {
		return false, "pcc properties presence mismatch"
	}
	if privateCrossConnect.Metadata != nil && privateCrossConnect.Metadata.State != nil && *privateCrossConnect.Metadata.State == "BUSY" {
		return true, ""
	}
	p := privateCrossConnect.Properties
	d := diff.New()
	if cr.Spec.ForProvider.Name != "" {
		d.Str("name", &cr.Spec.ForProvider.Name, p.Name)
	}
	if cr.Spec.ForProvider.Description != "" {
		d.Str("description", &cr.Spec.ForProvider.Description, p.Description)
	}
	return d.Result()
}
