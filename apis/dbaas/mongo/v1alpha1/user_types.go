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

// MongoUserParameters are the observable fields of a User.
// Required fields in order to create a DBaaS User:
// ClusterConfig,
// Credentials,
type MongoUserParameters struct {
	// +kubebuilder:validation:Required
	//
	ClusterCfg ClusterConfig `json:"clusterConfig"`
	// Database credentials - either set directly, or as secret/path/env
	//
	// +kubebuilder:validation:Required
	Credentials DBUser `json:"credentials"`

	// A list of mongodb user roles
	//
	// +kubebuilder:validation:Required
	Roles []UserRoles `json:"userRoles,omitempty"`
}

// UserRoles a list of mongodb user role.
type UserRoles struct {
	// Role to set for the user
	//
	// +kubebuilder:validation:Required
	Role string `json:"role,omitempty"`
	// Database on which to set the role
	//
	// +kubebuilder:validation:Required
	Database string `json:"database,omitempty"`
}

// ClusterConfig is used by resources that need to link mongo clusters via id or via reference.
type ClusterConfig struct {
	// ClusterID is the ID of the Cluster on which the resource will be created.
	// It needs to be provided via directly or via reference.
	//
	// +immutable
	// +kubebuilder:validation:Format=uuid
	// +crossplane:generate:reference:type=github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/dbaas/mongo/v1alpha1.MongoCluster
	// +crossplane:generate:reference:extractor=github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/dbaas/mongo/v1alpha1.ExtractMongoClusterID()
	ClusterID string `json:"ClusterId,omitempty"`
	// ClusterIDRef references to a Cluster to retrieve its ID.
	//
	// +optional
	// +immutable
	ClusterIDRef *xpv1.Reference `json:"ClusterIdRef,omitempty"`
	// ClusterIDSelector selects reference to a Cluster to retrieve its ClusterID.
	//
	// +optional
	ClusterIDSelector *xpv1.Selector `json:"ClusterIdSelector,omitempty"`
}

// +kubebuilder:object:root=true

// A MongoUser is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="USER ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="USERNAME",priority=1,type="string",JSONPath=".spec.forProvider.credentials.username"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud}
type MongoUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec               MongoUserSpec           `json:"spec"`
	Status             UserStatus              `json:"status,omitempty"`
	ManagementPolicies xpv1.ManagementPolicies `json:"managementPolicies"`
}

// A UserStatus represents the observed state of a user.
type UserStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          UserObservation `json:"atProvider,omitempty"`
}

// UserObservation are the observable fields of a Cluster.
type UserObservation struct {
	UserID string `json:"userId,omitempty"`
}

// A MongoUserSpec defines the desired state of a Cluster.
type MongoUserSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       MongoUserParameters `json:"forProvider"`
}

// SetManagementPolicies implement managed interface
func (mg *MongoUser) SetManagementPolicies(p xpv1.ManagementPolicies) {
	mg.ManagementPolicies = p
}

// GetManagementPolicies implement managed interface
func (mg *MongoUser) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.ManagementPolicies
}

// +kubebuilder:object:root=true

// MongoUserList contains a list of User
type MongoUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MongoUser `json:"items"`
}

// Cluster type metadata.
var (
	MongoUserKind             = reflect.TypeOf(MongoUser{}).Name()
	MongoUserGroupKind        = schema.GroupKind{Group: Group, Kind: MongoUserKind}.String()
	MongoUserKindAPIVersion   = MongoUserKind + "." + SchemeGroupVersion.String()
	MongoUserGroupVersionKind = SchemeGroupVersion.WithKind(MongoUserKind)
)

func init() {
	SchemeBuilder.Register(&MongoUser{}, &MongoUserList{})
}
