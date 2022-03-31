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

// ClusterParameters are the observable fields of a Cluster.
// Required fields in order to create a K8s Cluster:
// Name,
// Public.
type ClusterParameters struct {
	// A Kubernetes cluster name. Valid Kubernetes cluster name must be 63 characters or less and must be empty
	// or begin and end with an alphanumeric character ([a-z0-9A-Z]) with dashes (-), underscores (_), dots (.), and alphanumerics between.
	//
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// The Kubernetes version the cluster is running. This imposes restrictions on what Kubernetes versions can be run in a cluster's nodepools.
	// Additionally, not all Kubernetes versions are viable upgrade targets for all prior versions.
	// Example: 1.15.4
	//
	// +kubebuilder:validation:Optional
	K8sVersion string `json:"k8sVersion,omitempty"`
	// The maintenance window is used for updating the cluster's control plane and for upgrading the cluster's K8s version.
	// If no value is given, one is chosen dynamically, so there is no fixed default.
	//
	// +kubebuilder:validation:Optional
	MaintenanceWindow MaintenanceWindow `json:"maintenanceWindow,omitempty"`
	// The indicator if the cluster is public or private.
	// Be aware that setting it to false is currently in beta phase.
	//
	// +immutable
	// +kubebuilder:validation:Required
	Public bool `json:"public"`
	// Access to the K8s API server is restricted to these CIDRs. Traffic, internal to the cluster, is not affected by this restriction.
	// If no allow-list is specified, access is not restricted.
	// If an IP without subnet mask is provided, the default value is used: 32 for IPv4 and 128 for IPv6.
	// Example: "1.2.3.4/32", "2002::1234:abcd:ffff:c0a8:101/64", "1.2.3.4", "2002::1234:abcd:ffff:c0a8:101"
	//
	// +kubebuilder:validation:Optional
	APISubnetAllowList []string `json:"apiSubnetAllowList,omitempty"`
	// List of S3 bucket configured for K8s usage.
	// For now it contains only an S3 bucket used to store K8s API audit logs
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MaxItems=1
	S3Buckets []S3Bucket `json:"s3Buckets,omitempty"`
}

// MaintenanceWindow A weekly window, during which maintenance might occur
type MaintenanceWindow struct {
	Time string `json:"time,omitempty"`
	// DayOfTheWeek The name of the week day.
	DayOfTheWeek string `json:"dayOfTheWeek,omitempty"`
}

// S3Bucket configured for K8s usage.
type S3Bucket struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`
}

// ClusterConfig is used by resources that need to link clusters via id or via reference.
type ClusterConfig struct {
	// ClusterID is the ID of the Cluster on which the resource will be created.
	// It needs to be provided via directly or via reference.
	//
	// +immutable
	// +kubebuilder:validation:Format=uuid
	// +crossplane:generate:reference:type=Cluster
	// +crossplane:generate:reference:extractor=ExtractClusterID()
	ClusterID string `json:"clusterId,omitempty"`
	// ClusterIDRef references to a Cluster to retrieve its ID
	//
	// +optional
	// +immutable
	ClusterIDRef *xpv1.Reference `json:"clusterIdRef,omitempty"`
	// ClusterIDSelector selects reference to a Cluster to retrieve its clusterId
	//
	// +optional
	ClusterIDSelector *xpv1.Selector `json:"clusterIdSelector,omitempty"`
}

// ClusterObservation are the observable fields of a Cluster.
type ClusterObservation struct {
	ClusterID                string   `json:"clusterId,omitempty"`
	State                    string   `json:"state,omitempty"`
	AvailableUpgradeVersions []string `json:"availableUpgradeVersions,omitempty"`
	ViableNodePoolVersions   []string `json:"viableNodePoolVersions,omitempty"`
}

// A ClusterSpec defines the desired state of a Cluster.
type ClusterSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ClusterParameters `json:"forProvider"`
}

// A ClusterStatus represents the observed state of a Cluster.
type ClusterStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ClusterObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Cluster is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="CLUSTER ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="CLUSTER NAME",type="string",JSONPath=".spec.forProvider.name"
// +kubebuilder:printcolumn:name="K8S VERSION",priority=1,type="string",JSONPath=".spec.forProvider.k8sVersion"
// +kubebuilder:printcolumn:name="PUBLIC",priority=1,type="string",JSONPath=".spec.forProvider.public"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud}
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec"`
	Status ClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterList contains a list of Cluster
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

// Cluster type metadata.
var (
	ClusterKind             = reflect.TypeOf(Cluster{}).Name()
	ClusterGroupKind        = schema.GroupKind{Group: Group, Kind: ClusterKind}.String()
	ClusterKindAPIVersion   = ClusterKind + "." + SchemeGroupVersion.String()
	ClusterGroupVersionKind = SchemeGroupVersion.WithKind(ClusterKind)
)

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}
