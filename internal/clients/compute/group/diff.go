package group

import (
	sdkgo "github.com/ionos-cloud/sdk-go/v6"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/diff"
)

// IsGroupUpToDate returns whether the Group is up-to-date and a diff string.
func IsGroupUpToDate(cr *v1alpha1.Group, observed sdkgo.Group) (bool, string) {
	if cr == nil && observed.Properties == nil {
		return true, ""
	}
	if cr == nil || observed.Properties == nil {
		return false, "group properties presence mismatch"
	}
	p := observed.Properties
	d := diff.New()
	if p.Name != nil {
		d.Str("name", &cr.Spec.ForProvider.Name, p.Name)
	}
	if p.AccessActivityLog != nil {
		d.Bool("accessActivityLog", &cr.Spec.ForProvider.AccessActivityLog, p.AccessActivityLog)
	}
	if p.AccessAndManageCertificates != nil {
		d.Bool("accessAndManageCertificates", &cr.Spec.ForProvider.AccessAndManageCertificates, p.AccessAndManageCertificates)
	}
	if p.AccessAndManageDns != nil {
		d.Bool("accessAndManageDns", &cr.Spec.ForProvider.AccessAndManageDNS, p.AccessAndManageDns)
	}
	if p.AccessAndManageMonitoring != nil {
		d.Bool("accessAndManageMonitoring", &cr.Spec.ForProvider.AccessAndManageMonitoring, p.AccessAndManageMonitoring)
	}
	if p.CreateBackupUnit != nil {
		d.Bool("createBackupUnit", &cr.Spec.ForProvider.CreateBackupUnit, p.CreateBackupUnit)
	}
	if p.CreateDataCenter != nil {
		d.Bool("createDataCenter", &cr.Spec.ForProvider.CreateDataCenter, p.CreateDataCenter)
	}
	if p.CreateFlowLog != nil {
		d.Bool("createFlowLog", &cr.Spec.ForProvider.CreateFlowLog, p.CreateFlowLog)
	}
	if p.CreateInternetAccess != nil {
		d.Bool("createInternetAccess", &cr.Spec.ForProvider.CreateInternetAccess, p.CreateInternetAccess)
	}
	if p.CreateK8sCluster != nil {
		d.Bool("createK8sCluster", &cr.Spec.ForProvider.CreateK8sCluster, p.CreateK8sCluster)
	}
	if p.CreatePcc != nil {
		d.Bool("createPcc", &cr.Spec.ForProvider.CreatePcc, p.CreatePcc)
	}
	if p.CreateSnapshot != nil {
		d.Bool("createSnapshot", &cr.Spec.ForProvider.CreateSnapshot, p.CreateSnapshot)
	}
	if p.ManageDBaaS != nil {
		d.Bool("manageDBaaS", &cr.Spec.ForProvider.ManageDBaaS, p.ManageDBaaS)
	}
	if p.ManageDataplatform != nil {
		d.Bool("manageDataplatform", &cr.Spec.ForProvider.ManageDataPlatform, p.ManageDataplatform)
	}
	if p.ManageRegistry != nil {
		d.Bool("manageRegistry", &cr.Spec.ForProvider.ManageRegistry, p.ManageRegistry)
	}
	if p.ReserveIp != nil {
		d.Bool("reserveIp", &cr.Spec.ForProvider.ReserveIP, p.ReserveIp)
	}
	if p.S3Privilege != nil {
		d.Bool("s3Privilege", &cr.Spec.ForProvider.S3Privilege, p.S3Privilege)
	}
	configuredMemberIDs := memberIDsSet(cr)
	observedMemberIDs := sets.New[string](cr.Status.AtProvider.UserIDs...)
	if !observedMemberIDs.Equal(configuredMemberIDs) {
		d.Add("members", "<changed>", "<changed>")
	}
	configuredResourceShares := resourceSharesSet(cr)
	observedResourceShares := sets.New[v1alpha1.ResourceShare](cr.Status.AtProvider.ResourceShares...)
	if !observedResourceShares.Equal(configuredResourceShares) {
		d.Add("resourceShares", "<changed>", "<changed>")
	}
	return d.Result()
}
