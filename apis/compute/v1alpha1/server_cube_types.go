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

// CubeServerProperties are the observable fields of a Cube Server.
type CubeServerProperties struct {
	// DatacenterConfig contains information about the datacenter resource
	// on which the server will be created
	//
	// +immutable
	// +kubebuilder:validation:Required
	DatacenterCfg DatacenterConfig `json:"datacenterConfig"`
	// The ID or the name of the template for creating a CUBE server.
	//
	// +immutable
	// +kubebuilder:validation:Required
	Template Template `json:"template"`
	// The name of the  resource.
	Name string `json:"name,omitempty"`
	// The availability zone in which the server should be provisioned.
	//
	// +kubebuilder:validation:Enum=AUTO;ZONE_1;ZONE_2
	// +kubebuilder:default=AUTO
	AvailabilityZone string `json:"availabilityZone,omitempty"`
	// CPU architecture on which server gets provisioned; not all CPU architectures are available in all datacenter regions;
	// available CPU architectures can be retrieved from the datacenter resource.
	//
	// +kubebuilder:validation:Enum=AMD_OPTERON;INTEL_SKYLAKE;INTEL_XEON
	CPUFamily string `json:"cpuFamily,omitempty"`
	// DasVolumeProperties contains properties for the DAS volume attached to the Cube Server
	//
	// +kubebuilder:validation:Required
	DasVolumeProperties DasVolumeProperties `json:"volume"`
}

// DasVolumeProperties are the observable fields of a Cube Server's DAS Volume.
type DasVolumeProperties struct {
	// The name of the DAS Volume.
	//
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// The bus type of the volume.
	//
	// +kubebuilder:validation:Enum=VIRTIO;IDE;UNKNOWN
	// +kubebuilder:validation:Required
	Bus string `json:"bus"`
	// Image or snapshot ID to be used as template for this volume.
	// Make sure the image selected is compatible with the datacenter's location.
	// Note: when creating a volume, set image, image alias, or licence type
	//
	// +immutable
	Image string `json:"image,omitempty"`
	// Initial password to be set for installed OS. Works with public images only. Not modifiable, forbidden in update requests.
	// Password rules allows all characters from a-z, A-Z, 0-9.
	//
	// +immutable
	ImagePassword string `json:"imagePassword,omitempty"`
	// +immutable
	// Note: when creating a volume, set image, image alias, or licence type
	ImageAlias string `json:"imageAlias,omitempty"`
	// Public SSH keys are set on the image as authorized keys for appropriate SSH login to the instance using the corresponding private key.
	// This field may only be set in creation requests. When reading, it always returns null.
	// SSH keys are only supported if a public Linux image is used for the volume creation.
	//
	// +immutable
	SSHKeys []string `json:"sshKeys,omitempty"`
	// OS type for this volume.
	// Note: when creating a volume, set image, image alias, or licence type
	//
	// +immutable
	// +kubebuilder:validation:Enum=UNKNOWN;WINDOWS;WINDOWS2016;WINDOWS2022;LINUX;OTHER
	LicenceType string `json:"licenceType,omitempty"`
}

// Template refers to the template used for cube servers
type Template struct {
	// The name of the  resource.
	Name string `json:"name,omitempty"`
	// The ID of the  template.
	TemplateID string `json:"templateId,omitempty"`
}

// A CubeServerSpec defines the desired state of a Cube Server.
type CubeServerSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       CubeServerProperties `json:"forProvider"`
}

// +kubebuilder:object:root=true

// A CubeServer is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="DATACENTER ID",type="string",JSONPath=".spec.forProvider.datacenterConfig.datacenterId"
// +kubebuilder:printcolumn:name="SERVER ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="VOLUME ID",priority=1,type="string",JSONPath=".status.atProvider.volumeId"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud}
type CubeServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CubeServerSpec `json:"spec"`
	Status ServerStatus   `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CubeServerList contains a list of Server
type CubeServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CubeServer `json:"items"`
}

// CubeServer type metadata.
var (
	CubeServerKind             = reflect.TypeOf(CubeServer{}).Name()
	CubeServerGroupKind        = schema.GroupKind{Group: Group, Kind: CubeServerKind}.String()
	CubeServerKindAPIVersion   = CubeServerKind + "." + SchemeGroupVersion.String()
	CubeServerGroupVersionKind = SchemeGroupVersion.WithKind(CubeServerKind)
)

func init() {
	SchemeBuilder.Register(&CubeServer{}, &CubeServerList{})
}
