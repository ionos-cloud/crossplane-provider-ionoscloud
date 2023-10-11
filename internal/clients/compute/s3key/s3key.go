package s3key

import (
	"context"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	sdkgo "github.com/ionos-cloud/sdk-go/v6"
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

// LateInitializer fills the empty fields in *v1alpha1.S3KeyParameters with
// the values seen in sdkgo.S3Key.
func LateInitializer(in *v1alpha1.S3KeyParameters, s3Key *sdkgo.S3Key) {
	if s3Key == nil {
		return
	}
	// Add secretKey to the Spec, if it was set by the API
	if propertiesOk, ok := s3Key.GetPropertiesOk(); ok && propertiesOk != nil {
		if secretKeyOk, ok := propertiesOk.GetSecretKeyOk(); ok && secretKeyOk != nil {
			if in.SecretKey == "" {
				in.SecretKey = *secretKeyOk
			}
		}
	}
}

// IsS3KeyUpToDate returns true if the S3Key is up-to-date or false if it does not
func IsS3KeyUpToDate(cr *v1alpha1.S3Key, s3Key sdkgo.S3Key) bool { // nolint:gocyclo
	switch {
	case cr == nil && s3Key.Properties == nil:
		return true
	case cr == nil && s3Key.Properties != nil:
		return false
	case cr != nil && s3Key.Properties == nil:
		return false
	case s3Key.Properties.SecretKey != nil && *s3Key.Properties.SecretKey != cr.Spec.ForProvider.SecretKey:
		return false
	case s3Key.Properties.Active == nil && cr.Spec.ForProvider.Active != *s3Key.Properties.Active:
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
