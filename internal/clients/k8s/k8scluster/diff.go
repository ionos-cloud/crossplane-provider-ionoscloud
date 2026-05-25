package k8scluster

import (
	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/k8s/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/k8s"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/compare"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/diff"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

// IsK8sClusterUpToDate returns whether the K8sCluster is up-to-date and a diff
// string describing differences between the CR spec and the observed SDK state.
func IsK8sClusterUpToDate(cr *v1alpha1.Cluster, cluster sdkgo.KubernetesCluster) (bool, string) {
	if cr == nil && cluster.Properties == nil {
		return true, ""
	}
	if cr == nil || cluster.Properties == nil {
		return false, "k8s cluster properties presence mismatch"
	}
	if cluster.Metadata != nil && cluster.Metadata.State != nil && (*cluster.Metadata.State == k8s.BUSY || *cluster.Metadata.State == k8s.DEPLOYING) {
		return true, ""
	}
	p := cluster.Properties
	d := diff.New()
	if cr.Spec.ForProvider.Name != "" || p.Name != nil {
		d.Str("name", &cr.Spec.ForProvider.Name, p.Name)
	}
	if cr.Spec.ForProvider.K8sVersion != "" && p.K8sVersion != nil {
		d.Str("k8sVersion", &cr.Spec.ForProvider.K8sVersion, p.K8sVersion)
	}
	if p.S3Buckets != nil && !isEqS3Buckets(cr.Spec.ForProvider.S3Buckets, *p.S3Buckets) {
		d.Add("s3Buckets", "<changed>", "<changed>")
	}
	if p.ApiSubnetAllowList != nil && !utils.ContainsStringSlices(*p.ApiSubnetAllowList, cr.Spec.ForProvider.APISubnetAllowList) {
		d.StrSliceUnordered("apiSubnetAllowList", &cr.Spec.ForProvider.APISubnetAllowList, p.ApiSubnetAllowList)
	}
	if !compare.EqualKubernetesMaintenanceWindow(cr.Spec.ForProvider.MaintenanceWindow, p.MaintenanceWindow) {
		d.Add("maintenanceWindow", "<changed>", "<changed>")
	}
	if p.Public != nil {
		d.Bool("public", &cr.Spec.ForProvider.Public, p.Public)
	}
	return d.Result()
}
