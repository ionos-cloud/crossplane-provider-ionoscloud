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

// DataplatformNodepoolParameters are the observable fields of a DataplatformNodepool.
// Required values when creating a DataplatformNodepool cluster:
// Location.
type DataplatformNodepoolParameters struct {
	// A Datacenter, to which the user has access, to provision
	// the dataplatform NodePool in
	//
	// +immutable
	// +kubebuilder:validation:Optional
	DatacenterCfg DatacenterConfig `json:"datacenterConfig"`
	// The Dataplatform Cluster on which the NodePool will be created.
	//
	// +immutable
	// +kubebuilder:validation:Required
	ClusterCfg ClusterConfig `json:"clusterConfig"`
	// The name of the resource.
	// +kubebuilder:validation:Pattern="^[A-Za-z0-9][-A-Za-z0-9_.]*[A-Za-z0-9]$"
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Required
	Name string `json:"name,omitempty"`
	// The version of the Data Platform NodePool
	//
	// +kubebuilder:validation:Optional
	Version string `json:"version"`
	// The number of nodes that make up the NodePool
	//
	// +kubebuilder:validation:Required
	NodeCount int32 `json:"nodeCount"`
	// A valid CPU family name or `AUTO` if the platform shall choose the best fitting option. Available CPU architectures can be retrieved from the datacenter resource.
	//
	// +kubebuilder:validation:Optional
	CPUFamily string `json:"cpuFamily"`
	// The number of CPU cores per node
	//
	// +kubebuilder:validation:Optional
	CoresCount int32 `json:"coresCount"`
	// The RAM size for one node in MB. Must be set in multiples of 1024 MB, with a minimum size is of 2048 MB.
	//
	// +kubebuilder:validation:Optional
	RAMSize int32 `json:"ramSize"`
	// The availability zone in which the NodePool resources should be provisioned.
	// Possible values: AUTO;ZONE_1;ZONE_2
	//
	// +kubebuilder:validation:Optional
	AvailabilityZone string `json:"availabilityZone,omitempty"`
	// The type of hardware for the NodePool
	// Possible values HDD;SSD
	//
	// +immutable
	// +kubebuilder:validation:Optional
	StorageType string `json:"storageType"`
	// The amount of storage per instance in megabytes
	//
	// +kubebuilder:validation:Optional
	// +immutable
	StorageSize int32 `json:"storageSize"`
	// Starting time of a weekly 4 hour-long window, during which maintenance might occur in hh:mm:ss format
	//
	// +kubebuilder:validation:Optional
	MaintenanceWindow MaintenanceWindow `json:"maintenanceWindow,omitempty"`
	// Map of labels attached to NodePool
	//
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`
	// Map of annotations attached to NodePool
	//
	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ClusterConfig is used by resources that need to link dataplatform clusters via id or via reference.
type ClusterConfig struct {
	// ClusterID is the ID of the Cluster on which the resource will be created.
	// It needs to be provided via directly or via reference.
	//
	// +immutable
	// +kubebuilder:validation:Format=uuid
	// +crossplane:generate:reference:type=github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/dataplatform/v1alpha1.DataplatformCluster
	// +crossplane:generate:reference:extractor=github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/dataplatform/v1alpha1.ExtractDataplatformClusterID()
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

// DataplatformNodepoolObservation are the observable fields of a DataplatformNodepool.
type DataplatformNodepoolObservation struct {
	DataplatformID string `json:"datacenterId,omitempty"`
	ClusterID      string `json:"ClusterId,omitempty"`
	Version        string `json:"version,omitempty"`
	State          string `json:"state,omitempty"`
}

// A DataplatformNodepoolSpec defines the desired state of a DataplatformNodepool.
type DataplatformNodepoolSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       DataplatformNodepoolParameters `json:"forProvider"`
}

// A DataplatformNodepoolStatus represents the observed state of a DataplatformNodepool.
type DataplatformNodepoolStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          DataplatformNodepoolObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A DataplatformNodepool is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="DATAPLATFORM ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="DATAPLATFORM NAME",type="string",JSONPath=".spec.forProvider.name"
// +kubebuilder:printcolumn:name="LOCATION",type="string",JSONPath=".spec.forProvider.location"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud},shortName=dpn;datanp
type DataplatformNodepool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DataplatformNodepoolSpec   `json:"spec"`
	Status DataplatformNodepoolStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DataplatformNodepoolList contains a list of DataplatformNodepool
type DataplatformNodepoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DataplatformNodepool `json:"items"`
}

// DataplatformNodepool type metadata.
var (
	DataplatformNodepoolKind             = reflect.TypeOf(DataplatformNodepool{}).Name()
	DataplatformNodepoolGroupKind        = schema.GroupKind{Group: Group, Kind: DataplatformNodepoolKind}.String()
	DataplatformNodepoolKindAPIVersion   = DataplatformNodepoolKind + "." + SchemeGroupVersion.String()
	DataplatformNodepoolGroupVersionKind = SchemeGroupVersion.WithKind(DataplatformNodepoolKind)
)

func init() {
	SchemeBuilder.Register(&DataplatformNodepool{}, &DataplatformNodepoolList{})
}
