package datacenter

import (
	"context"
	"fmt"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

// APIClient is a wrapper around IONOS Service
type APIClient struct {
	*clients.IonosServices
}

// Client is a wrapper around IONOS Service Datacenter methods
type Client interface {
	CheckDuplicateDatacenter(ctx context.Context, datacenterName, location string) (*sdkgo.Datacenter, error)
	GetDatacenterID(datacenter *sdkgo.Datacenter) (string, error)
	GetDatacenter(ctx context.Context, datacenterID string) (sdkgo.Datacenter, *sdkgo.APIResponse, error)
	CreateDatacenter(ctx context.Context, datacenter sdkgo.DatacenterPost) (sdkgo.Datacenter, *sdkgo.APIResponse, error)
	UpdateDatacenter(ctx context.Context, datacenterID string, datacenter sdkgo.DatacenterPropertiesPut) (sdkgo.Datacenter, *sdkgo.APIResponse, error)
	DeleteDatacenter(ctx context.Context, datacenterID string) (*sdkgo.APIResponse, error)
	GetCPUFamiliesForDatacenter(ctx context.Context, datacenterID string) ([]string, error)
	GetAPIClient() *sdkgo.APIClient
}

// CheckDuplicateDatacenter based on datacenterName, and the immutable property location
func (cp *APIClient) CheckDuplicateDatacenter(ctx context.Context, datacenterName, location string) (*sdkgo.Datacenter, error) { // nolint: gocyclo
	datacenters, _, err := cp.IonosServices.ComputeClient.DataCentersApi.DatacentersGet(ctx).Depth(utils.DepthQueryParam).Execute()
	if err != nil {
		return nil, err
	}
	matchedItems := make([]sdkgo.Datacenter, 0)
	if itemsOk, ok := datacenters.GetItemsOk(); ok && itemsOk != nil {
		for _, item := range *itemsOk {
			if propertiesOk, ok := item.GetPropertiesOk(); ok && propertiesOk != nil {
				if nameOk, ok := propertiesOk.GetNameOk(); ok && nameOk != nil {
					if *nameOk == datacenterName {
						// After checking the name, check the immutable properties
						if locationOk, ok := propertiesOk.GetLocationOk(); ok && locationOk != nil {
							if *locationOk != location {
								return nil, fmt.Errorf("error: found datacenter with the name %v, but immutable property location different. expected: %v actual: %v", datacenterName, location, *locationOk)
							}
						}
						matchedItems = append(matchedItems, item)
					}
				}
			}
		}
	}
	if len(matchedItems) == 0 {
		return nil, nil
	}
	if len(matchedItems) > 1 {
		return nil, fmt.Errorf("error: found multiple datacenters with the name %v", datacenterName)
	}
	return &matchedItems[0], nil
}

// GetDatacenterID based on datacenter
func (cp *APIClient) GetDatacenterID(datacenter *sdkgo.Datacenter) (string, error) {
	if datacenter != nil {
		if idOk, ok := datacenter.GetIdOk(); ok && idOk != nil {
			return *idOk, nil
		}
		return "", fmt.Errorf("error: getting datacenter id")
	}
	return "", nil
}

// GetDatacenter based on datacenterID
func (cp *APIClient) GetDatacenter(ctx context.Context, datacenterID string) (sdkgo.Datacenter, *sdkgo.APIResponse, error) {
	return cp.IonosServices.ComputeClient.DataCentersApi.DatacentersFindById(ctx, datacenterID).Depth(utils.DepthQueryParam).Execute()
}

// CreateDatacenter based on Datacenter properties
func (cp *APIClient) CreateDatacenter(ctx context.Context, datacenter sdkgo.DatacenterPost) (sdkgo.Datacenter, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.DataCentersApi.DatacentersPost(ctx).Datacenter(datacenter).Execute()
}

// UpdateDatacenter based on datacenterID and Datacenter properties
func (cp *APIClient) UpdateDatacenter(ctx context.Context, datacenterID string, datacenter sdkgo.DatacenterPropertiesPut) (sdkgo.Datacenter, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.DataCentersApi.DatacentersPatch(ctx, datacenterID).Datacenter(datacenter).Execute()
}

// DeleteDatacenter based on datacenterID
func (cp *APIClient) DeleteDatacenter(ctx context.Context, datacenterID string) (*sdkgo.APIResponse, error) {
	resp, err := cp.IonosServices.ComputeClient.DataCentersApi.DatacentersDelete(ctx, datacenterID).Execute()
	return resp, err
}

// GetCPUFamiliesForDatacenter based on datacenterID
func (cp *APIClient) GetCPUFamiliesForDatacenter(ctx context.Context, datacenterID string) ([]string, error) {
	cpuFamiliesAvailable := make([]string, 0)
	datacenter, _, err := cp.IonosServices.ComputeClient.DataCentersApi.DatacentersFindById(ctx, datacenterID).Execute()
	if err != nil {
		return cpuFamiliesAvailable, err
	}
	if propertiesOk, ok := datacenter.GetPropertiesOk(); ok && propertiesOk != nil {
		if cpuArchitecturesOk, ok := propertiesOk.GetCpuArchitectureOk(); ok && cpuArchitecturesOk != nil && len(*cpuArchitecturesOk) > 0 {
			for _, cpuArchitecture := range *cpuArchitecturesOk {
				if cpuFamilyOk, ok := cpuArchitecture.GetCpuFamilyOk(); ok && cpuFamilyOk != nil {
					cpuFamiliesAvailable = append(cpuFamiliesAvailable, *cpuFamilyOk)
				}
			}
		}
	}
	return cpuFamiliesAvailable, nil
}

// GetAPIClient gets the APIClient
func (cp *APIClient) GetAPIClient() *sdkgo.APIClient {
	return cp.IonosServices.ComputeClient
}

// GenerateCreateDatacenterInput returns sdkgo.Datacenter based on the CR spec
func GenerateCreateDatacenterInput(cr *v1alpha1.Datacenter) (*sdkgo.DatacenterPost, error) {
	instanceCreateInput := sdkgo.DatacenterPost{
		Properties: &sdkgo.DatacenterPropertiesPost{
			Name:              &cr.Spec.ForProvider.Name,
			Description:       &cr.Spec.ForProvider.Description,
			Location:          &cr.Spec.ForProvider.Location,
			SecAuthProtection: &cr.Spec.ForProvider.SecAuthProtection,
		},
	}
	return &instanceCreateInput, nil
}

// GenerateUpdateDatacenterInput returns sdkgo.DatacenterProperties based on the CR spec modifications
func GenerateUpdateDatacenterInput(cr *v1alpha1.Datacenter) (*sdkgo.DatacenterPropertiesPut, error) {
	instanceUpdateInput := sdkgo.DatacenterPropertiesPut{
		Name:        &cr.Spec.ForProvider.Name,
		Description: &cr.Spec.ForProvider.Description,
	}
	return &instanceUpdateInput, nil
}

// IsDatacenterUpToDate returns true if the Datacenter is up-to-date or false if it does not
func IsDatacenterUpToDate(cr *v1alpha1.Datacenter, datacenter sdkgo.Datacenter) bool { // nolint:gocyclo
	switch {
	case cr == nil && datacenter.Properties == nil:
		return true
	case cr == nil && datacenter.Properties != nil:
		return false
	case cr != nil && datacenter.Properties == nil:
		return false
	case datacenter.Metadata.State != nil && *datacenter.Metadata.State == "BUSY":
		return true
	case datacenter.Properties.Name != nil && *datacenter.Properties.Name != cr.Spec.ForProvider.Name:
		return false
	case datacenter.Properties.Name == nil && cr.Spec.ForProvider.Name != "":
		return false
	case datacenter.Properties.Description != nil && *datacenter.Properties.Description != cr.Spec.ForProvider.Description:
		return false
	default:
		return true
	}
}
