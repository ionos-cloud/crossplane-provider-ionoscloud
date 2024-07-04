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

// PostgresDatabaseParameters are the observable fields of a Database.
// Required fields in order to create a DBaaS Database:
// ClusterConfig,
// Name,
// Owner,
type PostgresDatabaseParameters struct {
	// +kubebuilder:validation:Required
	//
	ClusterCfg ClusterConfig `json:"clusterConfig"`
	// The databasename of a given database.
	//
	// Database credentials - either set directly, or as secret/path/env
	//
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// The name of the role owning a given database.
	//
	// +kubebuilder:validation:Required
	Owner UserConfig `json:"owner"`
}

// +kubebuilder:object:root=true

// A PostgresDatabase is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="Database ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud},shortName=pgdb;psqldb;pgdb
type PostgresDatabase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgresDatabaseSpec `json:"spec"`
	Status DatabaseStatus       `json:"status,omitempty"`
}

// A DatabaseStatus represents the observed state of a Database.
type DatabaseStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          DatabaseObservation `json:"atProvider,omitempty"`
}

// DatabaseObservation are the observable fields of a Cluster.
type DatabaseObservation struct {
	DatabaseID string `json:"DatabaseId,omitempty"`
}

// A PostgresDatabaseSpec defines the desired state of a Cluster.
type PostgresDatabaseSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       PostgresDatabaseParameters `json:"forProvider"`
}

// +kubebuilder:object:root=true

// PostgresDatabaseList contains a list of Database
type PostgresDatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgresDatabase `json:"items"`
}

// UserConfig is used by resources that need to link postgres users via id or via reference.
type UserConfig struct {
	// UserName is the Name of the User on which the resource will be created.
	// It needs to be provided directly or via reference.
	//
	// +immutable
	// +crossplane:generate:reference:type=github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/dbaas/postgres/v1alpha1.PostgresUser
	// +crossplane:generate:reference:extractor=github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/dbaas/postgres/v1alpha1.ExtractPostgresUserID()
	UserName string `json:"userName,omitempty"`
	// UserIDRef references to a User to retrieve its Name.
	//
	// +optional
	// +immutable
	UserNameRef *xpv1.Reference `json:"UserNameRef,omitempty"`
	// UserNameSelector selects reference to a User to retrieve its UserName.
	//
	// +optional
	UserNameSelector *xpv1.Selector `json:"UserNameSelector,omitempty"`
}

// Cluster type metadata.
var (
	PostgresDatabaseKind             = reflect.TypeOf(PostgresDatabase{}).Name()
	PostgresDatabaseGroupKind        = schema.GroupKind{Group: Group, Kind: PostgresDatabaseKind}.String()
	PostgresDatabaseKindAPIVersion   = PostgresDatabaseKind + "." + SchemeGroupVersion.String()
	PostgresDatabaseGroupVersionKind = SchemeGroupVersion.WithKind(PostgresDatabaseKind)
)

func init() {
	SchemeBuilder.Register(&PostgresDatabase{}, &PostgresDatabaseList{})
}
