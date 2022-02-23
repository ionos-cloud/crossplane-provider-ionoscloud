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
	"context"
	"reflect"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reference"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// ServerProperties are the observable fields of a Server.
type ServerProperties struct {
	// +immutable
	DatacenterID string `json:"datacenterID,omitempty"`
	// DatacenterIDRef references to a Datacenter to retrieve its ID
	// +optional
	DatacenterIDRef *xpv1.Reference `json:"datacenterIDRef,omitempty"`
	// +optional
	DatacenterIDRefSelector *xpv1.Selector `json:"datacenterIDRefSelector,omitempty"`
	// The name of the  resource.
	Name string `json:"name,omitempty"`
	// The total number of cores for the server.
	Cores int32 `json:"cores"`
	// The memory size for the server in MB, such as 2048. Size must be specified in multiples of 256 MB with a minimum of 256 MB.
	// however, if you set ramHotPlug to TRUE then you must use a minimum of 1024 MB. If you set the RAM size more than 240GB,
	// then ramHotPlug will be set to FALSE and can not be set to TRUE unless RAM size not set to less than 240GB.
	// +kubebuilder:validation:MultipleOf=256
	RAM int32 `json:"ram"`
	// The availability zone in which the server should be provisioned.
	// +kubebuilder:validation:Enum=AUTO;ZONE_1;ZONE_2
	AvailabilityZone string `json:"availabilityZone,omitempty"`
	// CPU architecture on which server gets provisioned; not all CPU architectures are available in all datacenter regions;
	// available CPU architectures can be retrieved from the datacenter resource.
	// +kubebuilder:validation:Enum=AMD_OPTERON;INTEL_SKYLAKE;INTEL_XEON
	CPUFamily string `json:"cpuFamily,omitempty"`
}

// ServerObservation are the observable fields of a Server.
type ServerObservation struct {
	ServerID string `json:"serverID,omitempty"`
	State    string `json:"state,omitempty"`
}

// A ServerSpec defines the desired state of a Server.
type ServerSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ServerProperties `json:"forProvider"`
}

// A ServerStatus represents the observed state of a Server.
type ServerStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ServerObservation `json:"atProvider,omitempty"`
}

// ResolveReferences implements the ReferenceResolver interface for a resource.
// It's called to resolve references in the server to e.g. the datacenter.
func (in *Server) ResolveReferences(ctx context.Context, c client.Reader) error {
	r := reference.NewAPIResolver(c, in)
	const undefined = ""
	// Resolve spec.forProvider.datacenterID
	datacenterID, err := r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: in.Spec.ForProvider.DatacenterID,
		Reference:    in.Spec.ForProvider.DatacenterIDRef,
		Selector:     in.Spec.ForProvider.DatacenterIDRefSelector,
		To:           reference.To{Managed: &Datacenter{}, List: &DatacenterList{}},
		Extract: func(managed resource.Managed) string {
			c, ok := managed.(*Datacenter)
			if !ok {
				return undefined
			}
			if meta.GetExternalName(c) == c.Name {
				return undefined
			}
			return meta.GetExternalName(c)
		},
	})
	if err != nil {
		return errors.Wrap(err, "spec.forProvider.datacenterID")
	}
	if datacenterID.ResolvedValue == undefined {
		return errors.New("datacenter either not found or not reconciled yet")
	}
	in.Spec.ForProvider.DatacenterID = datacenterID.ResolvedValue
	in.Spec.ForProvider.DatacenterIDRef = datacenterID.ResolvedReference
	return nil
}

// +kubebuilder:object:root=true

// A Server is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="DATACENTER ID",type="string",JSONPath=".spec.forProvider.datacenterID"
// +kubebuilder:printcolumn:name="ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,template}
type Server struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServerSpec   `json:"spec"`
	Status ServerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ServerList contains a list of Server
type ServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Server `json:"items"`
}

// Server type metadata.
var (
	ServerKind             = reflect.TypeOf(Server{}).Name()
	ServerGroupKind        = schema.GroupKind{Group: Group, Kind: ServerKind}.String()
	ServerKindAPIVersion   = ServerKind + "." + SchemeGroupVersion.String()
	ServerGroupVersionKind = SchemeGroupVersion.WithKind(ServerKind)
)

func init() {
	SchemeBuilder.Register(&Server{}, &ServerList{})
}
