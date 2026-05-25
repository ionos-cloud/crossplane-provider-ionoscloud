package mongo

import (
	"reflect"

	ionoscloud "github.com/ionos-cloud/sdk-go-dbaas-mongo"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/dbaas/mongo/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/compare"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/diff"
)

// IsClusterUpToDate returns whether the cluster is up-to-date and a diff string
// describing differences between the CR spec and the observed SDK state.
func IsClusterUpToDate(cr *v1alpha1.MongoCluster, clusterResponse ionoscloud.ClusterResponse) (bool, string) {
	if cr == nil && clusterResponse.Properties == nil {
		return true, ""
	}
	if cr == nil || clusterResponse.Properties == nil {
		return false, "mongo cluster properties presence mismatch"
	}
	if clusterResponse.Metadata != nil && clusterResponse.Metadata.State != nil && *clusterResponse.Metadata.State == ionoscloud.STATE_BUSY {
		return true, ""
	}
	p := clusterResponse.Properties
	d := diff.New()
	if cr.Spec.ForProvider.DisplayName != "" || p.DisplayName != nil {
		d.Str("displayName", &cr.Spec.ForProvider.DisplayName, p.DisplayName)
	}
	if p.MongoDBVersion != nil {
		d.Str("mongoDBVersion", &cr.Spec.ForProvider.MongoDBVersion, p.MongoDBVersion)
	}
	if p.Instances != nil {
		d.Int32("instances", &cr.Spec.ForProvider.Instances, p.Instances)
	}
	if p.Cores != nil {
		d.Int32("cores", &cr.Spec.ForProvider.Cores, p.Cores)
	}
	if p.Ram != nil {
		d.Int32("ram", &cr.Spec.ForProvider.RAM, p.Ram)
	}
	if p.StorageSize != nil {
		d.Int32("storageSize", &cr.Spec.ForProvider.StorageSize, p.StorageSize)
	}
	if p.BiConnector != nil && !reflect.DeepEqual(*p.BiConnector, cr.Spec.ForProvider.BiConnector) {
		d.Add("biConnector", "<changed>", "<changed>")
	}
	if p.Edition != nil {
		d.Str("edition", &cr.Spec.ForProvider.Edition, p.Edition)
	}
	if !compare.EqualMongoDatabaseMaintenanceWindow(cr.Spec.ForProvider.MaintenanceWindow, p.MaintenanceWindow) {
		d.Add("maintenanceWindow", "<changed>", "<changed>")
	}
	if !eqClusterConnections(cr, clusterResponse) {
		d.Add("connections", "<changed>", "<changed>")
	}
	return d.Result()
}

// IsUserUpToDate returns whether the user is up-to-date and a diff string.
func IsUserUpToDate(cr *v1alpha1.MongoUser, user ionoscloud.User) (bool, string) {
	if cr == nil && user.Properties == nil {
		return true, ""
	}
	if cr == nil || user.Properties == nil {
		return false, "mongo user properties presence mismatch"
	}
	d := diff.New()
	if user.Properties.Username != nil {
		d.Str("username", &cr.Spec.ForProvider.Credentials.Username, user.Properties.Username)
	}
	if !eqUserRoles(cr, user) {
		d.Add("roles", "<changed>", "<changed>")
	}
	return d.Result()
}
