package mocktest

import (
	"context"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	ipblockClient "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/ipblock"
)

// Compile-time assertion that StubIPBlockService implements ipblock.Client.
var _ ipblockClient.Client = &StubIPBlockService{}

// StubIPBlockService is a no-op implementation of ipblock.Client used by NIC and FirewallRule tests.
type StubIPBlockService struct {
	apiClient *sdkgo.APIClient
}

// NewStubIPBlockService creates a stub IPBlock service for the given server URL.
func NewStubIPBlockService(serverURL string) *StubIPBlockService {
	cfg := sdkgo.NewConfiguration("", "", "test-token", serverURL)
	return &StubIPBlockService{apiClient: sdkgo.NewAPIClient(cfg)}
}

func (f *StubIPBlockService) CheckDuplicateIPBlock(_ context.Context, _, _ string) (*sdkgo.IpBlock, error) {
	return nil, nil
}

func (f *StubIPBlockService) GetIPBlockID(_ *sdkgo.IpBlock) (string, error) {
	return "", nil
}

func (f *StubIPBlockService) GetIPBlock(_ context.Context, _ string) (sdkgo.IpBlock, *sdkgo.APIResponse, error) {
	return sdkgo.IpBlock{}, nil, nil
}

func (f *StubIPBlockService) CreateIPBlock(_ context.Context, _ sdkgo.IpBlock) (sdkgo.IpBlock, *sdkgo.APIResponse, error) {
	return sdkgo.IpBlock{}, nil, nil
}

func (f *StubIPBlockService) UpdateIPBlock(_ context.Context, _ string, _ sdkgo.IpBlockProperties) (sdkgo.IpBlock, *sdkgo.APIResponse, error) {
	return sdkgo.IpBlock{}, nil, nil
}

func (f *StubIPBlockService) DeleteIPBlock(_ context.Context, _ string) (*sdkgo.APIResponse, error) {
	return nil, nil
}

func (f *StubIPBlockService) GetIPs(_ context.Context, _ string, _ ...int) ([]string, error) {
	return nil, nil
}

func (f *StubIPBlockService) GetAPIClient() *sdkgo.APIClient {
	return f.apiClient
}
