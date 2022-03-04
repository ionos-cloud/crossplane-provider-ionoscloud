package lan

import (
	"context"
	"reflect"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
)

// APIClient is a wrapper around IONOS Service
type APIClient struct {
	*clients.IonosServices
}

// Client is a wrapper around IONOS Service Lan methods
type Client interface {
	GetLan(ctx context.Context, datacenterID, lanID string) (sdkgo.Lan, *sdkgo.APIResponse, error)
	CreateLan(ctx context.Context, datacenterID string, lan sdkgo.LanPost) (sdkgo.LanPost, *sdkgo.APIResponse, error)
	UpdateLan(ctx context.Context, datacenterID, lanID string, lan sdkgo.LanProperties) (sdkgo.Lan, *sdkgo.APIResponse, error)
	DeleteLan(ctx context.Context, datacenterID, lanID string) (*sdkgo.APIResponse, error)
	GetAPIClient() *sdkgo.APIClient
}

// GetLan based on datacenterID, lanID
func (cp *APIClient) GetLan(ctx context.Context, datacenterID, lanID string) (sdkgo.Lan, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.LANsApi.DatacentersLansFindById(ctx, datacenterID, lanID).Execute()
}

// CreateLan based on datacenterID and Lan properties
func (cp *APIClient) CreateLan(ctx context.Context, datacenterID string, lan sdkgo.LanPost) (sdkgo.LanPost, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.LANsApi.DatacentersLansPost(ctx, datacenterID).Lan(lan).Execute()
}

// UpdateLan based on datacenterID, lanID and Lan properties
func (cp *APIClient) UpdateLan(ctx context.Context, datacenterID, lanID string, lan sdkgo.LanProperties) (sdkgo.Lan, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.LANsApi.DatacentersLansPatch(ctx, datacenterID, lanID).Lan(lan).Execute()
}

// DeleteLan based on datacenterID, lanID
func (cp *APIClient) DeleteLan(ctx context.Context, datacenterID, lanID string) (*sdkgo.APIResponse, error) {
	resp, err := cp.ComputeClient.LANsApi.DatacentersLansDelete(ctx, datacenterID, lanID).Execute()
	return resp, err
}

// GetAPIClient gets the APIClient
func (cp *APIClient) GetAPIClient() *sdkgo.APIClient {
	return cp.ComputeClient
}

// GenerateCreateLanInput returns CreateLanRequest based on the CR spec
func GenerateCreateLanInput(cr *v1alpha1.Lan) (*sdkgo.LanPost, error) {
	instanceCreateInput := sdkgo.LanPost{
		Properties: &sdkgo.LanPropertiesPost{
			Public: &cr.Spec.ForProvider.Public,
		},
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Name)) {
		instanceCreateInput.Properties.SetName(cr.Spec.ForProvider.Name)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Pcc)) {
		instanceCreateInput.Properties.SetPcc(cr.Spec.ForProvider.Pcc)
	}
	return &instanceCreateInput, nil
}

// GenerateUpdateLanInput returns PatchLanRequest based on the CR spec modifications
func GenerateUpdateLanInput(cr *v1alpha1.Lan) (*sdkgo.LanProperties, error) {
	instanceUpdateInput := sdkgo.LanProperties{
		Public: &cr.Spec.ForProvider.Public,
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Name)) {
		instanceUpdateInput.SetName(cr.Spec.ForProvider.Name)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Pcc)) {
		instanceUpdateInput.SetPcc(cr.Spec.ForProvider.Pcc)
	}
	return &instanceUpdateInput, nil
}

// IsLanUpToDate returns true if the Lan is up-to-date or false if it does not
func IsLanUpToDate(cr *v1alpha1.Lan, lan sdkgo.Lan) bool { // nolint:gocyclo
	if lan.Properties == nil || lan.Metadata == nil || cr == nil {
		return false
	}
	switch {
	case *lan.Metadata.State == "BUSY":
		return true
	case *lan.Properties.Name != cr.Spec.ForProvider.Name:
		return false
	case *lan.Properties.Public != cr.Spec.ForProvider.Public:
		return false
	case *lan.Properties.Pcc != cr.Spec.ForProvider.Pcc:
		return false
	}
	return true
}
