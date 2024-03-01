package usergroup

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	"strings"
)

const separator = "-tosplit-"

func resourceToString(id string, editPrivilege, sharePrivilege bool) string {
	return fmt.Sprintf("%s%t%t", id, editPrivilege, sharePrivilege)
}
func hashToString(id, hash string) string {
	return fmt.Sprintf("%s%s%s", id, separator, hash)
}
func getResourceID(hash string) string {
	p := strings.Split(hash, separator)
	return p[0]
}
func (eu *externalUserGroup) hash(resource v1alpha1.Resource) string {
	s := resourceToString(resource.ID, resource.EditPrivilege, resource.SharePrivilege)

	h := sha256.New()
	h.Write([]byte(s))
	return hashToString(resource.ID, string(h.Sum(nil)))
}
func (eu *externalUserGroup) hashResources(resources []v1alpha1.Resource) []string {
	hashes := make([]string, 0)
	for _, resource := range resources {
		hashes = append(hashes, eu.hash(resource))
	}

	return hashes
}

func (eu *externalUserGroup) hashIonosResource(resource ionoscloud.GroupShare) string {
	p := utils.DereferenceOrZero(resource.Properties)
	id := utils.DereferenceOrZero(resource.Id)
	s := resourceToString(
		id,
		utils.DereferenceOrZero(p.EditPrivilege),
		utils.DereferenceOrZero(p.SharePrivilege))
	h := sha256.New()
	h.Write([]byte(s))
	return hashToString(id, string(h.Sum(nil)))
}

func (eu *externalUserGroup) addResources(ctx context.Context, groupID string, resources []v1alpha1.Resource) ([]string, error) {
	hashes := make([]string, 0)
	for _, resource := range resources {
		h := eu.hash(resource)
		_, err := eu.service.AddResource(ctx, groupID, resource)
		if err != nil {
			return hashes, err
		}
		hashes = append(hashes, h)
	}

	return hashes, nil
}

func (eu *externalUserGroup) hashIonosResources(resources ionoscloud.GroupShares) []string {
	hashes := make([]string, 0)
	for _, resource := range utils.DereferenceOrZero(resources.Items) {
		hashes = append(hashes, eu.hashIonosResource(resource))
	}

	return hashes
}

// updateResources adds missing resources and delete extra resources.
func (eu *externalUserGroup) updateResources(ctx context.Context, userGroupID string, resources []v1alpha1.Resource) error {
	ionosResources, resp, err := eu.service.GetResources(ctx, userGroupID)
	if err != nil {
		return compute.AddAPIResponseInfo(resp, err)
	}
	ionosHashedResources := eu.hashIonosResources(ionosResources)

	ionosMap := getResourcesMap(ionosHashedResources)
	// add any resource that exists in k8s but not in ionos
	for _, resource := range resources {
		hashedResource := eu.hash(resource)
		_, exist := ionosMap[hashedResource]
		if !exist {
			resp, err := eu.service.AddResource(ctx, userGroupID, resource)
			if err != nil {
				return compute.AddAPIResponseInfo(resp, err)
			}
		}
		ionosMap[hashedResource] = true
	}

	// find any resource exists in ionos but not in k8s
	for k, v := range ionosMap {
		if !v {
			resourceID := getResourceID(k)
			resp, err := eu.service.RemoveResourceFromGroup(ctx, userGroupID, resourceID)
			if err != nil {
				return compute.AddAPIResponseInfo(resp, err)
			}
		}
	}

	return nil
}

func getResourcesMap(resources []string) map[string]bool {
	m := make(map[string]bool, 0)
	for _, resource := range resources {
		m[resource] = false
	}

	return m
}
