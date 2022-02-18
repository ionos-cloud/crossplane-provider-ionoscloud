package datacenter

import (
	"context"
	"strings"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
)

// APIClient is a wrapper around IONOS Service
type APIClient struct {
	*clients.IonosServices
}

// Client is a wrapper around IONOS Service Datacenter methods
type Client interface {
	GetDatacenter(ctx context.Context, datacenterID string) (sdkgo.Datacenter, *sdkgo.APIResponse, error)
	CreateDatacenter(ctx context.Context, datacenter sdkgo.Datacenter) (sdkgo.Datacenter, *sdkgo.APIResponse, error)
	UpdateDatacenter(ctx context.Context, datacenterID string, datacenter sdkgo.DatacenterProperties) (sdkgo.Datacenter, *sdkgo.APIResponse, error)
	DeleteDatacenter(ctx context.Context, datacenterID string) (*sdkgo.APIResponse, error)
}

// GetDatacenter based on datacenterID
func (cp *APIClient) GetDatacenter(ctx context.Context, datacenterID string) (sdkgo.Datacenter, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.DataCentersApi.DatacentersFindById(ctx, datacenterID).Execute()
}

// CreateDatacenter based on Datacenter properties
func (cp *APIClient) CreateDatacenter(ctx context.Context, datacenter sdkgo.Datacenter) (sdkgo.Datacenter, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.DataCentersApi.DatacentersPost(ctx).Datacenter(datacenter).Execute()
}

// UpdateDatacenter based on datacenterID and Datacenter properties
func (cp *APIClient) UpdateDatacenter(ctx context.Context, datacenterID string, datacenter sdkgo.DatacenterProperties) (sdkgo.Datacenter, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.DataCentersApi.DatacentersPatch(ctx, datacenterID).Datacenter(datacenter).Execute()
}

// DeleteDatacenter based on datacenterID
func (cp *APIClient) DeleteDatacenter(ctx context.Context, datacenterID string) (*sdkgo.APIResponse, error) {
	resp, err := cp.ComputeClient.DataCentersApi.DatacentersDelete(ctx, datacenterID).Execute()
	return resp, err
}

// GenerateCreateDatacenterInput returns CreateDatacenterRequest based on the CR spec
func GenerateCreateDatacenterInput(cr *v1alpha1.Datacenter) (*sdkgo.Datacenter, error) {
	instanceCreateInput := sdkgo.Datacenter{
		Properties: &sdkgo.DatacenterProperties{
			Name:              &cr.Spec.ForProvider.Name,
			Description:       &cr.Spec.ForProvider.Description,
			Location:          &cr.Spec.ForProvider.Location,
			SecAuthProtection: &cr.Spec.ForProvider.SecAuthProtection,
		},
	}
	return &instanceCreateInput, nil
}

// GenerateUpdateDatacenterInput returns PatchDatacenterRequest based on the CR spec modifications
func GenerateUpdateDatacenterInput(cr *v1alpha1.Datacenter) (*sdkgo.DatacenterProperties, error) {
	instanceUpdateInput := sdkgo.DatacenterProperties{
		Name:        &cr.Spec.ForProvider.Name,
		Description: &cr.Spec.ForProvider.Description,
	}
	return &instanceUpdateInput, nil
}

// IsDatacenterUpToDate returns true if the Datacenter is up-to-date or false if it does not
func IsDatacenterUpToDate(cr *v1alpha1.Datacenter, datacenter sdkgo.Datacenter) bool {
	switch {
	case cr == nil && datacenter.Properties == nil:
		return true
	case cr == nil && datacenter.Properties != nil:
		return false
	case cr != nil && datacenter.Properties == nil:
		return false
	}
	if *datacenter.Metadata.State == "BUSY" {
		return true
	}
	if strings.Compare(cr.Spec.ForProvider.Name, *datacenter.Properties.Name) != 0 {
		return false
	}
	return true
}
