package template

import (
	"context"
	"fmt"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

// APIClient is a wrapper around IONOS Service
type APIClient struct {
	*clients.IonosServices
}

// Client is a wrapper around IONOS Service Template methods
type Client interface {
	GetTemplates(ctx context.Context) (sdkgo.Templates, *sdkgo.APIResponse, error)
	GetTemplateIDByName(ctx context.Context, templateName string) (string, error)
	GetAPIClient() *sdkgo.APIClient
}

// GetTemplates returns all existing Templates
func (cp *APIClient) GetTemplates(ctx context.Context) (sdkgo.Templates, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.TemplatesApi.TemplatesGet(ctx).Depth(utils.DepthQueryParam).Execute()
}

// GetTemplateIDByName returns Template with the name specified
func (cp *APIClient) GetTemplateIDByName(ctx context.Context, templateName string) (string, error) {
	templates, _, err := cp.ComputeClient.TemplatesApi.TemplatesGet(ctx).Depth(utils.DepthQueryParam).Filter("name", templateName).Execute()
	if err != nil {
		return "", err
	}
	if items, ok := templates.GetItemsOk(); ok && items != nil {
		if len(*items) == 0 {
			return "", fmt.Errorf("error getting ID of the Template named: %s - no Templates found", templateName)
		}
		if len(*items) > 1 {
			return "", fmt.Errorf("error getting ID of the Template named: %s - multiple Templates with the same name found", templateName)
		}
		if len(*items) == 1 {
			templatesItems := *items
			if idOk, ok := templatesItems[0].GetIdOk(); ok && idOk != nil {
				return *idOk, nil
			}
		}
	}
	return "", fmt.Errorf("error getting ID of the Template named: %s", templateName)
}

// GetAPIClient gets the APIClient
func (cp *APIClient) GetAPIClient() *sdkgo.APIClient {
	return cp.ComputeClient
}
