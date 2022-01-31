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

// Location The physical location where the cluster will be created. This will be where all of your instances live.
// Property cannot be modified after datacenter creation.
type Location string

// BackupLocation The S3 location where the backups will be stored.
type BackupLocation string

// StorageType The storage type used in your cluster.
type StorageType string

// Connection Details about the network connection for your cluster.
type Connection struct {
	// The datacenter to connect your cluster to.
	DatacenterID string `json:"datacenterID,omitempty"`
	// The numeric LAN ID to connect your cluster to.
	LanID string `json:"lanID,omitempty"`
	// The IP and subnet for your cluster. Note the following unavailable IP ranges: 10.233.64.0/18 10.233.0.0/18 10.233.114.0/24
	Cidr string `json:"cidr,omitempty"`
}

// MaintenanceWindow A weekly 4 hour-long window, during which maintenance might occur
type MaintenanceWindow struct {
	Time         string       `json:"time,omitempty"`
	DayOfTheWeek DayOfTheWeek `json:"dayOfTheWeek,omitempty"`
}

// DayOfTheWeek The name of the week day.
type DayOfTheWeek string

// SynchronizationMode Represents different modes of replication.
type SynchronizationMode string

// DBUser Credentials for the database user to be created.
type DBUser struct {
	// The username for the initial postgres user. some system usernames are restricted (e.g. \"postgres\", \"admin\", \"standby\").
	Username string `json:"username"`
	Password string `json:"password"`
}

// CreateRestoreRequest The restore request.
type CreateRestoreRequest struct {
	// The unique ID of the backup you want to restore.
	BackupID string `json:"backupID"`
	// If this value is supplied as ISO 8601 timestamp, the backup will be replayed up until the given timestamp. If empty, the backup will be applied completely.
	RecoveryTargetTime string `json:"RecoveryTargetTime,omitempty"`
}

// ClusterParameters are the observable fields of a Cluster.
type ClusterParameters struct {
	// The PostgreSQL version of your cluster.
	PostgresVersion string `json:"postgresVersion"`
	// The total number of instances in the cluster (one master and n-1 standbys).
	Instances int32 `json:"instances"`
	// The number of CPU cores per instance.
	Cores int32 `json:"cores"`
	// The amount of memory per instance in megabytes. Has to be a multiple of 1024.
	RAM int32 `json:"ram"`
	// The amount of storage per instance in megabytes.
	StorageSize int32        `json:"storageSize"`
	StorageType StorageType  `json:"storageType"`
	Connections []Connection `json:"connections"`
	Location    Location     `json:"location"`
	// The friendly name of your cluster.
	DisplayName         string               `json:"displayName"`
	MaintenanceWindow   MaintenanceWindow    `json:"maintenanceWindow,omitempty"`
	Credentials         DBUser               `json:"credentials"`
	SynchronizationMode SynchronizationMode  `json:"synchronizationMode"`
	FromBackup          CreateRestoreRequest `json:"fromBackup,omitempty"`
}

// ClusterObservation are the observable fields of a Cluster.
type ClusterObservation struct {
	ClusterID string `json:"clusterID,omitempty"`
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

// A Cluster is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,template}
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
