/*
Copyright 2020 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// GroupParameters are the observable fields of a Group.
// Required values when creating a Group:
// Name
type GroupParameters struct {
	// Name of the resource.
	//
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// AccessActivityLog privilege for a group to access activity logs.
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:default=false
	AccessActivityLog bool `json:"accessActivityLog"`
	// AccessAndManageCertificates privilege for a group to access and manage certificates.
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:default=false
	AccessAndManageCertificates bool `json:"accessAndManageCertificates"`
	// AccessAndManageDNS privilege for a group to access and manage dns records.
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:default=false
	AccessAndManageDNS bool `json:"accessAndManageDns"`
	// AccessAndManageMonitoring privilege for a group to access and manage monitoring related functionality
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:default=false
	AccessAndManageMonitoring bool `json:"accessAndManageMonitoring"`
	// CreateBackupUnit privilege to create backup unit resource
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:default=false
	CreateBackupUnit bool `json:"createBackupUnit"`
	// CreateDataCenter privilege to create datacenter resource
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:default=false
	CreateDataCenter bool `json:"createDataCenter"`
	// CreateFlowLog privilege to create flow log resource
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:default=false
	CreateFlowLog bool `json:"createFlowLog"`
	// CreateInternetAccess privilege to create internet access
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:default=false
	CreateInternetAccess bool `json:"createInternetAccess"`
	// CreateK8sCluster privilege to create kubernetes cluster
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:default=false
	CreateK8sCluster bool `json:"createK8sCluster"`
	// CreatePcc privilege to create private cross connect
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:default=false
	CreatePcc bool `json:"createPcc"`
	// CreateSnapshot privilege to create snapshot
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:default=false
	CreateSnapshot bool `json:"createSnapshot"`
	// ManageDBaaS privilege to manage DBaaS related functionality
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:default=false
	ManageDBaaS bool `json:"manageDBaaS"`
	// ManageDataPlatform privilege to access and manage the Data Platform
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:default=false
	ManageDataPlatform bool `json:"manageDataplatform"`
	// ManageRegistry privilege to access container registry related functionality
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:default=false
	ManageRegistry bool `json:"manageRegistry"`
	// ReserveIp privilege to reserve ip block
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:default=false
	ReserveIP bool `json:"reserveIp"`
	// S3Privilege privilege to access S3 functionality
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:default=false
	S3Privilege bool `json:"s3Privilege"`
	// In order to add a User as member to the Group, it is recommended to use UserCfg
	// to add an existing User as a member (via id or via reference).
	// To remove a User from the Group, update the CR spec by removing it.
	//
	// UserCfg contains information about an existing User resource
	// which will be added to the Group
	UserCfg []UserConfig `json:"userConfig,omitempty"`

	// SharedResources allows sharing privilege to resources between the members of the group
	// In order to share a resource within a group, it must be referenced either by providing its ID directly
	// or by specifying a set of values by which its K8s object can be identified
	ResourceShareCfg []ResourceShareConfig `json:"sharedResourcesConfig,omitempty"`
}

// ResourceShareConfig is used for referencing a resource to be added as a ResourceShare within a Group
type ResourceShareConfig struct {
	// ResourceShare
	ResourceShare `json:"resourceShare,omitempty"`
	// If ResourceID is not provided directly, the resource can be referenced through other attributes
	// These attributes mut all be provided for the Resource to be resolved successfully
	// Name of the kubernetes object instance of the Custom Resource
	//
	// +kubebuilder:validation:Optional
	Name string `json:"name,omitempty"`
	// Kind of the Custom Resource
	//
	// +kubebuilder:validation:Optional
	Kind string `json:"kind,omitempty"`
	// Version of the Custom Resource
	//
	// +kubebuilder:validation:Optional
	Version string `json:"version,omitempty"`
}

// ResourceShare can be added to a Group to grant privileges to its members on the resource referenced by ResourceID
type ResourceShare struct {
	// EditPrivilege for the Resource
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:default=false
	EditPrivilege bool `json:"editPrivilege,omitempty"`
	// SharePrivilege for the Resource
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:default=false
	SharePrivilege bool `json:"sharePrivilege,omitempty"`
	// ResourceID is the ID of the Resource to which Group members gain privileges
	// It can only be provided directly
	// +immutable
	// +kubebuilder:validation:Format=uuid
	ResourceID string `json:"resourceId,omitempty"`
}

// GroupConfig is used by resources that need to link Groups via id or via reference.
type GroupConfig struct {
	// GroupID is the ID of the Group on which the resource should have access.
	// It needs to be provided directly or via reference.
	//
	// +immutable
	// +kubebuilder:validation:Format=uuid
	// +crossplane:generate:reference:type=github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1.Group
	// +crossplane:generate:reference:extractor=github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1.ExtractGroupID()
	GroupID string `json:"groupId,omitempty"`
	// GroupIDRef references to a Group to retrieve its ID.
	//
	// +optional
	// +immutable
	GroupIDRef *xpv1.Reference `json:"groupIdRef,omitempty"`
	// GroupIDSelector selects reference to a Group to retrieve its GroupID.
	//
	// +optional
	GroupIDSelector *xpv1.Selector `json:"groupIdSelector,omitempty"`
}

// GroupObservation are the observable fields of a Group.
type GroupObservation struct {
	// GroupID is the group id
	GroupID string `json:"groupId,omitempty"`
	// UserIDs of the members of this Group
	UserIDs []string `json:"userIDs,omitempty"`
	// ResourceShares of this Group
	ResourceShares []ResourceShare `json:"resourceShare,omitempty"`
}

// A GroupSpec defines the desired state of a Group.
type GroupSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       GroupParameters `json:"forProvider"`
}

// A GroupStatus represents the observed state of a Group.
type GroupStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          GroupObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// Group is the Schema for the Group resource API
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="NAME",type="string",JSONPath=".spec.forProvider.name"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud}
type Group struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GroupSpec   `json:"spec"`
	Status GroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GroupList contains a list of Group
type GroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Group `json:"items"`
}

// Group type metadata
var (
	GroupKind             = reflect.TypeOf(Group{}).Name()
	GroupGroupKind        = schema.GroupKind{Group: APIGroup, Kind: GroupKind}.String()
	GroupAPIVersion       = GroupKind + "." + SchemeGroupVersion.String()
	GroupGroupVersionKind = SchemeGroupVersion.WithKind(GroupKind)
)

func init() {
	SchemeBuilder.Register(&Group{}, &GroupList{})
}
