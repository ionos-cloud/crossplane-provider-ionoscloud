package postgres

import (
	"context"

	ionoscloud "github.com/ionos-cloud/sdk-go-dbaas-postgres"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/ionosclients"
)

// ClusterAPIClient is a wrapper around IONOS Service
type ClusterAPIClient struct {
	*ionosclients.IonosServices
}

// ClusterClient is a wrapper around IONOS Service methods
type ClusterClient interface {
	GetCluster(ctx context.Context, clusterID string) (ionoscloud.ClusterResponse, *ionoscloud.APIResponse, error)
	DeleteCluster(ctx context.Context, clusterID string) error
	PostCluster(ctx context.Context, cluster ionoscloud.CreateClusterRequest) (ionoscloud.ClusterResponse, *ionoscloud.APIResponse, error)
	PatchCluster(ctx context.Context, clusterID string, cluster ionoscloud.PatchClusterRequest) (ionoscloud.ClusterResponse, *ionoscloud.APIResponse, error)
}

// GetCluster based on clusterID
func (cp *ClusterAPIClient) GetCluster(ctx context.Context, clusterID string) (ionoscloud.ClusterResponse, *ionoscloud.APIResponse, error) {
	return cp.DBaaSPostgresClient.ClustersApi.ClustersFindById(ctx, clusterID).Execute()
}

// DeleteCluster based on clusterID
func (cp *ClusterAPIClient) DeleteCluster(ctx context.Context, clusterID string) error {
	_, _, err := cp.DBaaSPostgresClient.ClustersApi.ClustersDelete(ctx, clusterID).Execute()
	return err
}

// PostCluster based on cluster properties
func (cp *ClusterAPIClient) PostCluster(ctx context.Context, cluster ionoscloud.CreateClusterRequest) (ionoscloud.ClusterResponse, *ionoscloud.APIResponse, error) {
	return cp.DBaaSPostgresClient.ClustersApi.ClustersPost(ctx).CreateClusterRequest(cluster).Execute()
}

// PatchCluster based on clusterID and cluster properties
func (cp *ClusterAPIClient) PatchCluster(ctx context.Context, clusterID string, cluster ionoscloud.PatchClusterRequest) (ionoscloud.ClusterResponse, *ionoscloud.APIResponse, error) {
	return cp.DBaaSPostgresClient.ClustersApi.ClustersPatch(ctx, clusterID).PatchClusterRequest(cluster).Execute()
}
