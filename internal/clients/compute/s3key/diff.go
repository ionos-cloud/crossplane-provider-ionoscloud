package s3key

import (
	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/diff"
)

// IsS3KeyUpToDate returns whether the S3Key is up-to-date and a diff string.
func IsS3KeyUpToDate(cr *v1alpha1.S3Key, s3Key sdkgo.S3Key) (bool, string) {
	if cr == nil {
		return false, "cr is nil"
	}
	if s3Key.Properties == nil {
		return true, ""
	}
	d := diff.New()
	if s3Key.Properties.Active != nil {
		d.Bool("active", &cr.Spec.ForProvider.Active, s3Key.Properties.Active)
	}
	return d.Result()
}
