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

// PostgresUserParameters are the observable fields of a User.
// Required fields in order to create a DBaaS User:
// ClusterConfig,
// Credentials,
type PostgresUserParameters struct {
	// +kubebuilder:validation:Required
	//
	ClusterCfg ClusterConfig `json:"clusterConfig"`
	// The total number of instances in the cluster (one master and n-1 standbys).
	//
	// Database credentials - either set directly, or as secret/path/env
	//
	// +kubebuilder:validation:Required
	Credentials DBUser `json:"credentials"`
}

// ClusterConfig is used by resources that need to link psql clusters via id or via reference.
type ClusterConfig struct {
	// ClusterID is the ID of the Cluster on which the resource will be created.
	// It needs to be provided via directly or via reference.
	//
	// +immutable
	// +kubebuilder:validation:Format=uuid
	// +crossplane:generate:reference:type=github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/dbaas/postgres/v1alpha1.PostgresCluster
	// +crossplane:generate:reference:extractor=github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/dbaas/postgres/v1alpha1.ExtractPostgresClusterID()
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

// A PostgresUser is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="USER ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="USERNAME",priority=1,type="string",JSONPath=".spec.forProvider.credentials.username"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud},shortName=pgu;pguser;psqlu
type PostgresUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgresUserSpec `json:"spec"`
	Status UserStatus       `json:"status,omitempty"`
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

// A PostgresUserSpec defines the desired state of a Cluster.
type PostgresUserSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       PostgresUserParameters `json:"forProvider"`
}

// +kubebuilder:object:root=true

// PostgresUserList contains a list of User
type PostgresUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgresUser `json:"items"`
}

// Cluster type metadata.
var (
	PostgresUserKind             = reflect.TypeOf(PostgresUser{}).Name()
	PostgresUserGroupKind        = schema.GroupKind{Group: Group, Kind: PostgresUserKind}.String()
	PostgresUserKindAPIVersion   = PostgresUserKind + "." + SchemeGroupVersion.String()
	PostgresUserGroupVersionKind = SchemeGroupVersion.WithKind(PostgresUserKind)
)

func init() {
	SchemeBuilder.Register(&PostgresUser{}, &PostgresUserList{})
}
