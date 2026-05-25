package k8snodepool

import (
	"reflect"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/k8s/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/k8s"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/compare"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/diff"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

// IsK8sNodePoolUpToDate returns whether the NodePool is up-to-date and a diff
// string describing differences between the CR spec and the observed SDK state.
func IsK8sNodePoolUpToDate(cr *v1alpha1.NodePool, nodepool sdkgo.KubernetesNodePool, publicIPs []string) (bool, string) {
	if cr == nil && nodepool.Properties == nil {
		return true, ""
	}
	if cr == nil || nodepool.Properties == nil {
		return false, "k8s nodepool properties presence mismatch"
	}
	if nodepool.Metadata != nil && nodepool.Metadata.State != nil && (*nodepool.Metadata.State == k8s.BUSY || *nodepool.Metadata.State == k8s.DEPLOYING) {
		return true, ""
	}
	p := nodepool.Properties
	d := diff.New()
	if cr.Spec.ForProvider.Name != "" || p.Name != nil {
		d.Str("name", &cr.Spec.ForProvider.Name, p.Name)
	}
	if cr.Spec.ForProvider.K8sVersion != "" && p.K8sVersion != nil {
		d.Str("k8sVersion", &cr.Spec.ForProvider.K8sVersion, p.K8sVersion)
	}
	if cr.Spec.ForProvider.ServerType != "" && p.ServerType != nil {
		serverType := string(*p.ServerType)
		d.Str("serverType", &cr.Spec.ForProvider.ServerType, &serverType)
	}
	if p.NodeCount != nil && utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.AutoScaling)) {
		d.Int32("nodeCount", &cr.Spec.ForProvider.NodeCount, p.NodeCount)
	}
	if p.PublicIps != nil && !utils.ContainsStringSlices(*p.PublicIps, publicIPs) {
		d.StrSliceUnordered("publicIps", &publicIPs, p.PublicIps)
	}
	if p.Labels != nil && !utils.IsEqStringMaps(*p.Labels, cr.Spec.ForProvider.Labels) {
		d.Add("labels", "<changed>", "<changed>")
	}
	if p.Annotations != nil && !utils.IsEqStringMaps(*p.Annotations, cr.Spec.ForProvider.Annotations) {
		d.Add("annotations", "<changed>", "<changed>")
	}
	if p.AutoScaling != nil {
		as := d.Sub("autoScaling")
		if p.AutoScaling.MinNodeCount != nil {
			as.Int32("minNodeCount", &cr.Spec.ForProvider.AutoScaling.MinNodeCount, p.AutoScaling.MinNodeCount)
		}
		if p.AutoScaling.MaxNodeCount != nil {
			as.Int32("maxNodeCount", &cr.Spec.ForProvider.AutoScaling.MaxNodeCount, p.AutoScaling.MaxNodeCount)
		}
	}
	if !compare.EqualKubernetesMaintenanceWindow(cr.Spec.ForProvider.MaintenanceWindow, p.MaintenanceWindow) {
		d.Add("maintenanceWindow", "<changed>", "<changed>")
	}
	if p.Lans != nil && !isEqKubernetesNodePoolLans(cr.Spec.ForProvider.Lans, *p.Lans) {
		d.Add("lans", "<changed>", "<changed>")
	}
	return d.Result()
}
