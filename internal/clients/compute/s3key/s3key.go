package s3key

import (
	"context"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
)

// APIClient is a wrapper around IONOS Service
type APIClient struct {
	*clients.IonosServices
}

// GetS3Key based on userID, keyID
func (cp *APIClient) GetS3Key(ctx context.Context, userID, keyID string) (sdkgo.S3Key, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.UserS3KeysApi.UmUsersS3keysFindByKeyId(ctx, userID, keyID).Execute()
}

// CreateS3Key using userID
func (cp *APIClient) CreateS3Key(ctx context.Context, userID string) (sdkgo.S3Key, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.UserS3KeysApi.UmUsersS3keysPost(ctx, userID).Execute()
}

// UpdateS3Key based on datacenterID, userID, keyID abd s3Key
func (cp *APIClient) UpdateS3Key(ctx context.Context, userID, keyID string, s3Key sdkgo.S3Key) (sdkgo.S3Key, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.UserS3KeysApi.UmUsersS3keysPut(ctx, userID, keyID).S3Key(s3Key).Execute()
}

// DeleteS3Key based on userID, keyID
func (cp *APIClient) DeleteS3Key(ctx context.Context, userID, keyID string) (*sdkgo.APIResponse, error) {
	return cp.ComputeClient.UserS3KeysApi.UmUsersS3keysDelete(ctx, userID, keyID).Execute()
}

// GetAPIClient gets the APIClient
func (cp *APIClient) GetAPIClient() *sdkgo.APIClient {
	return cp.ComputeClient
}

// IsS3KeyUpToDate returns true if the S3Key is up-to-date or false if it does not
func IsS3KeyUpToDate(cr *v1alpha1.S3Key, s3Key sdkgo.S3Key) bool { // nolint:gocyclo
	if cr == nil {
		return false
	}
	switch {
	case s3Key.Properties == nil:
		return true
	case s3Key.Properties.Active != nil && cr.Spec.ForProvider.Active != *s3Key.Properties.Active:
		return false
	default:
		return true
	}
}

// GenerateUpdateSeKeyInput returns sdkgo.TargetGroupProperties based on the CR spec modifications
func GenerateUpdateSeKeyInput(cr *v1alpha1.S3Key) (*sdkgo.S3Key, error) {
	instanceUpdateInput := sdkgo.S3Key{
		Properties: &sdkgo.S3KeyProperties{
			Active: &cr.Spec.ForProvider.Active,
		},
	}

	return &instanceUpdateInput, nil
}
