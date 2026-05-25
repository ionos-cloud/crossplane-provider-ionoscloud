package postgrescluster

import (
	ionoscloud "github.com/ionos-cloud/sdk-go-bundle/products/dbaas/psql/v2"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/dbaas/postgres/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/compare"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/diff"
)

// IsClusterUpToDate returns whether the cluster is up-to-date and a diff string
// describing differences between the CR spec and the observed SDK state.
func IsClusterUpToDate(cr *v1alpha1.PostgresCluster, clusterResponse ionoscloud.ClusterResponse) (bool, string) {
	if cr == nil && clusterResponse.Properties == nil {
		return true, ""
	}
	if cr == nil || clusterResponse.Properties == nil {
		return false, "postgres cluster properties presence mismatch"
	}
	if clusterResponse.Metadata != nil && clusterResponse.Metadata.State != nil && *clusterResponse.Metadata.State == ionoscloud.STATE_BUSY {
		return true, ""
	}
	p := clusterResponse.Properties
	d := diff.New()
	if cr.Spec.ForProvider.DisplayName != "" || p.DisplayName != nil {
		d.Str("displayName", &cr.Spec.ForProvider.DisplayName, p.DisplayName)
	}
	if p.PostgresVersion != nil {
		d.Str("postgresVersion", &cr.Spec.ForProvider.PostgresVersion, p.PostgresVersion)
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
	if p.Connections != nil && !compare.EqualConnections(cr.Spec.ForProvider.Connections, p.Connections) {
		d.Add("connections", "<changed>", "<changed>")
	}
	if !compare.EqualDatabaseMaintenanceWindow(cr.Spec.ForProvider.MaintenanceWindow, p.MaintenanceWindow) {
		d.Add("maintenanceWindow", "<changed>", "<changed>")
	}
	if !compare.EqualConnectionPooler(cr.Spec.ForProvider.ConnectionPooler, p.ConnectionPooler) {
		d.Add("connectionPooler", "<changed>", "<changed>")
	}
	return d.Result()
}

// IsUserUpToDate returns whether the user is up-to-date and a diff string.
func IsUserUpToDate(cr *v1alpha1.PostgresUser, user ionoscloud.UserResource) (bool, string) {
	d := diff.New()
	username := user.Properties.Username
	d.Str("username", &cr.Spec.ForProvider.Credentials.Username, &username)
	return d.Result()
}
