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
// Required fields in order to create a DBaaS Postgres Cluster:
// PostgresVersion,
// Instances,
// Cores,
// RAM,
// Storage Size,
// Storage Type,
// Connection (Datacenter ID or Reference, Lan ID and CIDR),
// Location (in sync with Datacenter),
// DisplayName,
// Credentials,
// Synchronization Mode.
type ClusterParameters struct {
	// The PostgreSQL version of your cluster.
	//
	// +kubebuilder:validation:Required
	PostgresVersion string `json:"postgresVersion"`
	// The total number of instances in the cluster (one master and n-1 standbys).
	//
	// +kubebuilder:validation:Required
	Instances int32 `json:"instances"`
	// The number of CPU cores per instance.
	//
	// +kubebuilder:validation:Required
	Cores int32 `json:"cores"`
	// The amount of memory per instance in megabytes. Has to be a multiple of 1024.
	//
	// +kubebuilder:validation:MultipleOf=1024
	// +kubebuilder:validation:Required
	RAM int32 `json:"ram"`
	// The amount of storage per instance in megabytes.
	//
	// +kubebuilder:validation:Required
	StorageSize int32 `json:"storageSize"`
	// +kubebuilder:validation:Enum=HDD;SSD;SSD Standard;SSD Premium;DAS;ISO
	// +kubebuilder:validation:Required
	StorageType string `json:"storageType"`
	// +kubebuilder:validation:MaxItems=1
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:Required
	Connections []Connection `json:"connections"`
	// Location The physical location where the cluster will be created.
	// This will be where all of your instances live.
	// Property cannot be modified after datacenter creation.
	//
	// +immutable
	// +kubebuilder:validation:Enum=de/fra;us/las;us/ewr;de/txl;gb/lhr;es/vit
	// +kubebuilder:validation:Required
	Location string `json:"location"`
	// The friendly name of your cluster.
	//
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`
	// +kubebuilder:validation:Optional
	MaintenanceWindow MaintenanceWindow `json:"maintenanceWindow,omitempty"`
	// +kubebuilder:validation:Required
	Credentials DBUser `json:"credentials"`
	// SynchronizationMode Represents different modes of replication.
	//
	// +kubebuilder:validation:Enum=ASYNCHRONOUS;STRICTLY_SYNCHRONOUS;SYNCHRONOUS
	// +kubebuilder:validation:Required
	SynchronizationMode string `json:"synchronizationMode"`
	// +kubebuilder:validation:Optional
	FromBackup CreateRestoreRequest `json:"fromBackup,omitempty"`
}

// Connection Details about the network connection for your cluster.
type Connection struct {
	// DatacenterConfig contains information about the datacenter resource
	//
	// +kubebuilder:validation:Required
	DatacenterCfg DatacenterConfig `json:"datacenterConfig"`
	// LanConfig contains information about the lan resource
	//
	// +kubebuilder:validation:Required
	LanCfg LanConfig `json:"lanConfig"`
	// The IP and subnet for your cluster. Note the following unavailable IP ranges: 10.233.64.0/18 10.233.0.0/18 10.233.114.0/24
	//
	// +kubebuilder:validation:Required
	Cidr string `json:"cidr"`
}

// MaintenanceWindow A weekly 4 hour-long window, during which maintenance might occur
type MaintenanceWindow struct {
	Time string `json:"time,omitempty"`
	// DayOfTheWeek The name of the week day.
	DayOfTheWeek string `json:"dayOfTheWeek,omitempty"`
}

// DBUser Credentials for the database user to be created.
type DBUser struct {
	// The username for the initial postgres user.
	// Some system usernames are restricted (e.g. \"postgres\", \"admin\", \"standby\").
	//
	// +kubebuilder:validation:Required
	Username string `json:"username"`
	// +kubebuilder:validation:Required
	Password string `json:"password"`
}

// CreateRestoreRequest The restore request.
type CreateRestoreRequest struct {
	// The unique ID of the backup you want to restore.
	//
	// +kubebuilder:validation:Required
	BackupID string `json:"backupId"`
	// If this value is supplied as ISO 8601 timestamp, the backup will be replayed up until the given timestamp.
	// If empty, the backup will be applied completely.
	RecoveryTargetTime string `json:"recoveryTargetTime,omitempty"`
}

// DatacenterConfig is used by resources that need to link datacenters via id or via reference.
type DatacenterConfig struct {
	// DatacenterID is the ID of the Datacenter on which the resource will be created.
	// It needs to be provided via directly or via reference.
	//
	// +immutable
	// +kubebuilder:validation:Format=uuid
	// +crossplane:generate:reference:type=github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1.Datacenter
	// +crossplane:generate:reference:extractor=github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1.ExtractDatacenterID()
	DatacenterID string `json:"datacenterId,omitempty"`
	// DatacenterIDRef references to a Datacenter to retrieve its ID
	//
	// +optional
	// +immutable
	DatacenterIDRef *xpv1.Reference `json:"datacenterIdRef,omitempty"`
	// DatacenterIDSelector selects reference to a Datacenter to retrieve its datacenterId
	//
	// +optional
	DatacenterIDSelector *xpv1.Selector `json:"datacenterIdSelector,omitempty"`
}

// LanConfig is used by resources that need to link lans via id or via reference.
type LanConfig struct {
	// LanID is the ID of the Lan on which the cluster will connect to.
	// It needs to be provided via directly or via reference.
	//
	// +immutable
	// +crossplane:generate:reference:type=github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1.Lan
	// +crossplane:generate:reference:extractor=github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1.ExtractLanID()
	LanID string `json:"lanId,omitempty"`
	// LanIDRef references to a Lan to retrieve its ID
	//
	// +optional
	// +immutable
	LanIDRef *xpv1.Reference `json:"lanIdRef,omitempty"`
	// LanIDSelector selects reference to a Lan to retrieve its lanId
	//
	// +optional
	LanIDSelector *xpv1.Selector `json:"lanIdSelector,omitempty"`
}

// ClusterObservation are the observable fields of a Cluster.
type ClusterObservation struct {
	ClusterID string `json:"clusterId,omitempty"`
	State     string `json:"state,omitempty"`
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

// A PostgresCluster is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="CLUSTER ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="CLUSTER NAME",priority=1,type="string",JSONPath=".spec.forProvider.displayName"
// +kubebuilder:printcolumn:name="DATACENTER ID",priority=1,type="string",JSONPath=".spec.forProvider.connections[0].datacenterConfig.datacenterId"
// +kubebuilder:printcolumn:name="LAN ID",priority=1,type="string",JSONPath=".spec.forProvider.connections[0].lanConfig.lanId"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud}
type PostgresCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec"`
	Status ClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PostgresClusterList contains a list of Cluster
type PostgresClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgresCluster `json:"items"`
}

// Cluster type metadata.
var (
	PostgresClusterKind             = reflect.TypeOf(PostgresCluster{}).Name()
	PostgresClusterGroupKind        = schema.GroupKind{Group: Group, Kind: PostgresClusterKind}.String()
	PostgresClusterKindAPIVersion   = PostgresClusterKind + "." + SchemeGroupVersion.String()
	PostgresClusterGroupVersionKind = SchemeGroupVersion.WithKind(PostgresClusterKind)
)

func init() {
	SchemeBuilder.Register(&PostgresCluster{}, &PostgresClusterList{})
}
