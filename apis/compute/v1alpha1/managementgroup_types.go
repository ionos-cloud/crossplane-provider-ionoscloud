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

// ManagementGroupParameters are the observable fields of a ManagementGroup.
// Required values when creating a ManagementGroup:
// Name
type ManagementGroupParameters struct {
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
}

// ManagementGroupObservation are the observable fields of a ManagementGroup.
type ManagementGroupObservation struct {
	ManagementGroupID           string `json:"groupId,omitempty"`
	AccessActivityLog           bool   `json:"accessActivityLog"`
	AccessAndManageCertificates bool   `json:"accessAndManageCertificates"`
	AccessAndManageDNS          bool   `json:"accessAndManageDns"`
	AccessAndManageMonitoring   bool   `json:"accessAndManageMonitoring"`
	CreateBackupUnit            bool   `json:"createBackupUnit"`
	CreateDataCenter            bool   `json:"createDataCenter"`
	CreateFlowLog               bool   `json:"createFlowLog"`
	CreateInternetAccess        bool   `json:"createInternetAccess"`
	CreateK8sCluster            bool   `json:"createK8sCluster"`
	CreatePcc                   bool   `json:"createPcc"`
	CreateSnapshot              bool   `json:"createSnapshot"`
	ManageDBaaS                 bool   `json:"manageDBaaS"`
	ManageDataPlatform          bool   `json:"manageDataplatform"`
	ManageRegistry              bool   `json:"manageRegistry"`
	ReserveIP                   bool   `json:"reserveIp"`
	S3Privilege                 bool   `json:"s3Privilege"`
}

// A ManagementGroupSpec defines the desired state of a ManagementGroup.
type ManagementGroupSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ManagementGroupParameters `json:"forProvider"`
}

// A ManagementGroupStatus represents the observed state of a ManagementGroup.
type ManagementGroupStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ManagementGroupObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// ManagementGroup is the Schema for the ManagementGroup resource API
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="NAME",type="string",JSONPath=".spec.forProvider.name"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud}
type ManagementGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec               ManagementGroupSpec     `json:"spec"`
	Status             ManagementGroupStatus   `json:"status,omitempty"`
	ManagementPolicies xpv1.ManagementPolicies `json:"managementPolicies"`
}

// SetManagementPolicies implement managed interface
func (mg *ManagementGroup) SetManagementPolicies(p xpv1.ManagementPolicies) {
	mg.ManagementPolicies = p
}

// GetManagementPolicies implement managed interface
func (mg *ManagementGroup) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.ManagementPolicies
}

// +kubebuilder:object:root=true

// ManagementGroupList contains a list of ManagementGroup
type ManagementGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagementGroup `json:"items"`
}

// ManagementGroup type metadata
var (
	ManagementGroupResourceKind             = reflect.TypeOf(ManagementGroup{}).Name()
	ManagementGroupResourceGroupKind        = schema.GroupKind{Group: Group, Kind: ManagementGroupResourceKind}.String()
	ManagementGroupResourceAPIVersion       = ManagementGroupResourceKind + "." + SchemeGroupVersion.String()
	ManagementGroupResourceGroupVersionKind = SchemeGroupVersion.WithKind(ManagementGroupResourceKind)
)

func init() {
	SchemeBuilder.Register(&ManagementGroup{}, &ManagementGroupList{})
}
