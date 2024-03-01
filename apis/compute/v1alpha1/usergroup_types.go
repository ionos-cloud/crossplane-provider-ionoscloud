package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

const (
	CreateDataCenter            = "createdatacenter"
	CreateSnapshot              = "createsnapshot"
	ReserveIp                   = "reserveip"
	AccessActivityLog           = "accessactivitylog"
	CreatePcc                   = "createpcc"
	S3Privilege                 = "s3privilege"
	CreateBackupUnit            = "createbackupunit"
	CreateInternetAccess        = "createinternetaccess"
	CreateK8sCluster            = "createk8scluster"
	CreateFlowLog               = "createflowlog"
	AccessAndManageMonitoring   = "accessandmanagemonitoring"
	AccessAndManageCertificates = "accessandmanagecertificates"
	ManageDBaaS                 = "managedbaas"
)

// Resource is an IONOS resource like datacenter.
type Resource struct {
	// ID represents the resource id
	ID string `json:"id,omitempty"`

	// EditPrivilege group will have an edit privilege on the resource
	EditPrivilege bool `json:"editPrivilege,omitempty"`

	// SharePrivilege group will have a share privilege on the resource
	SharePrivilege bool `json:"sharePrivilege,omitempty"`
}

// GroupParameters defines the desired state of a UserGroup.
// Required values when creating a UserGroup:
// Name
type GroupParameters struct {
	// Name a list of group permissions
	//
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Privileges a list of group permissions
	//
	// +kubebuilder:validation:Optional
	Privileges []string `json:"privileges"`

	// Users a list of user ids
	//
	// +kubebuilder:validation:Optional
	Users []string `json:"users"`

	// Resources a list of resources
	//
	// +kubebuilder:validation:Optional
	Resources []Resource `json:"resources"`
}

// +kubebuilder:object:root=true

// UserGroup is our managed resource.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="USERGROUP_ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="GROUP NAME",type="string",JSONPath=".spec.forProvider.name"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud}
type UserGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserGroupSpec   `json:"spec"`
	Status UserGroupStatus `json:"status,omitempty"`
}

// A UserGroupSpec defines the desired state of a UserGroup.
type UserGroupSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       GroupParameters `json:"forProvider"`
}

// A UserGroupStatus represents the observed state of a UserGroup.
type UserGroupStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          UserGroupObservation `json:"atProvider,omitempty"`
}

// UserGroupObservation are the observable fields of a UserGroup.
type UserGroupObservation struct {
	// UserGroupID is the user group id.
	// +kubebuilder:validation:Format=uuid
	UserGroupID string `json:"userGroupID,omitempty"`
}

// +kubebuilder:object:root=true

// UserGroupList contains a list of UserGroup
type UserGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UserGroup `json:"items"`
}

// User type metadata.
var (
	UserGroupTypeKind         = reflect.TypeOf(UserGroup{}).Name()
	UserGroupGroupKind        = schema.GroupKind{Group: Group, Kind: UserGroupTypeKind}.String()
	UserGroupKindAPIVersion   = UserGroupTypeKind + "." + SchemeGroupVersion.String()
	UserGroupGroupVersionKind = SchemeGroupVersion.WithKind(UserGroupTypeKind)
)

func init() {
	SchemeBuilder.Register(&UserGroup{}, &UserGroupList{})
}
