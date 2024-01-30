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

// DataplatformClusterParameters are the observable fields of a DataplatformCluster.
// Required values when creating a DataplatformCluster cluster:
// Location.
type DataplatformClusterParameters struct {
	// A Datacenter, to which the user has access, to provision
	// the dataplatform cluster in.
	//
	// +immutable
	// +kubebuilder:validation:Required
	DatacenterCfg DatacenterConfig `json:"datacenterConfig"`
	// The name of the  resource.
	// +kubebuilder:validation:Pattern="^[A-Za-z0-9][-A-Za-z0-9_.]*[A-Za-z0-9]$"
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Required
	Name string `json:"name,omitempty"`
	// The version of the Data Platform.
	//
	// +kubebuilder:validation:Required
	Version string `json:"version"`
	// Starting time of a weekly 4 hour-long window, during which maintenance might occur in hh:mm:ss format
	//
	// +kubebuilder:validation:Optional
	MaintenanceWindow MaintenanceWindow `json:"maintenanceWindow,omitempty"`
}

// MaintenanceWindow A weekly window, during which maintenance might occur.
type MaintenanceWindow struct {
	// "Time at which the maintenance should start."
	Time string `json:"time,omitempty"`
	// DayOfTheWeek The name of the week day.
	DayOfTheWeek string `json:"dayOfTheWeek,omitempty"`
}

// GetTime returns the time of the maintenance window.
func (in *MaintenanceWindow) GetTime() string {
	return in.Time
}

// GetDayOfTheWeek returns the day of the week of the maintenance window.
func (in *MaintenanceWindow) GetDayOfTheWeek() string {
	return in.DayOfTheWeek
}

// DatacenterConfig is used by resources that need to link datacenters via id or via reference.
type DatacenterConfig struct {
	// DatacenterID is the ID of the Datacenter on which the resource should have access.
	// It needs to be provided via directly or via reference.
	//
	// +immutable
	// +kubebuilder:validation:Format=uuid
	// +crossplane:generate:reference:type=github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1.Datacenter
	// +crossplane:generate:reference:extractor=github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1.ExtractDatacenterID()
	DatacenterID string `json:"datacenterId,omitempty"`
	// DatacenterIDRef references to a Datacenter to retrieve its ID.
	//
	// +optional
	// +immutable
	DatacenterIDRef *xpv1.Reference `json:"datacenterIdRef,omitempty"`
	// DatacenterIDSelector selects reference to a Datacenter to retrieve its DatacenterID.
	//
	// +optional
	DatacenterIDSelector *xpv1.Selector `json:"datacenterIdSelector,omitempty"`
}

// DataplatformClusterObservation are the observable fields of a DataplatformCluster.
type DataplatformClusterObservation struct {
	DataplatformID string `json:"DataplatformId,omitempty"`
	Version        string `json:"version,omitempty"`
	State          string `json:"state,omitempty"`
}

// A DataplatformClusterSpec defines the desired state of a DataplatformCluster.
type DataplatformClusterSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       DataplatformClusterParameters `json:"forProvider"`
}

// A DataplatformClusterStatus represents the observed state of a DataplatformCluster.
type DataplatformClusterStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          DataplatformClusterObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A DataplatformCluster is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="DATAPLATFORM ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="DATAPLATFORM NAME",type="string",JSONPath=".spec.forProvider.name"
// +kubebuilder:printcolumn:name="LOCATION",type="string",JSONPath=".spec.forProvider.location"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud}
type DataplatformCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec               DataplatformClusterSpec   `json:"spec"`
	Status             DataplatformClusterStatus `json:"status,omitempty"`
	ManagementPolicies xpv1.ManagementPolicies   `json:"managementPolicies"`
}

// SetManagementPolicies implement managed interface
func (mg *DataplatformCluster) SetManagementPolicies(p xpv1.ManagementPolicies) {
	mg.ManagementPolicies = p
}

// GetManagementPolicies implement managed interface
func (mg *DataplatformCluster) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.ManagementPolicies
}

// +kubebuilder:object:root=true

// DataplatformClusterList contains a list of DataplatformCluster
type DataplatformClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DataplatformCluster `json:"items"`
}

// DataplatformCluster type metadata.
var (
	DataplatformClusterKind             = reflect.TypeOf(DataplatformCluster{}).Name()
	DataplatformClusterGroupKind        = schema.GroupKind{Group: Group, Kind: DataplatformClusterKind}.String()
	DataplatformClusterKindAPIVersion   = DataplatformClusterKind + "." + SchemeGroupVersion.String()
	DataplatformClusterGroupVersionKind = SchemeGroupVersion.WithKind(DataplatformClusterKind)
)

func init() {
	SchemeBuilder.Register(&DataplatformCluster{}, &DataplatformClusterList{})
}
