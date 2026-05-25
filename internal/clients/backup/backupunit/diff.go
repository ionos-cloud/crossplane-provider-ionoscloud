package backupunit

import (
	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/backup/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/diff"
)

// IsBackupUnitUpToDate returns whether the BackupUnit is up-to-date and a diff
// string describing differences between the CR spec and the observed SDK state.
func IsBackupUnitUpToDate(cr *v1alpha1.BackupUnit, backupUnit sdkgo.BackupUnit) (bool, string) {
	if cr == nil && backupUnit.Properties == nil {
		return true, ""
	}
	if cr == nil || backupUnit.Properties == nil {
		return false, "backup unit properties presence mismatch"
	}
	if backupUnit.Metadata != nil && backupUnit.Metadata.State != nil && *backupUnit.Metadata.State == "BUSY" {
		return true, ""
	}
	p := backupUnit.Properties
	d := diff.New()
	if cr.Spec.ForProvider.Name != "" || p.Name != nil {
		d.Str("name", &cr.Spec.ForProvider.Name, p.Name)
	}
	if cr.Spec.ForProvider.Email != "" || p.Email != nil {
		d.Str("email", &cr.Spec.ForProvider.Email, p.Email)
	}
	if cr.Spec.ForProvider.Password != oldPassword {
		d.Add("password", "<changed>", "<changed>")
	}
	return d.Result()
}
