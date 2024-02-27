package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// Resource is an IONOS resource like datacenter.
type Resource struct {
	// ID represents the resource id
	ID string `json:"id,omitempty"`

	// Type is the resource type like group, datacenter, etc.
	Type string `json:"type,omitempty"`
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
	privileges []string `json:"privileges"`

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
// +kubebuilder:printcolumn:name="NAME",type="string",JSONPath=".spec.forProvider.name"
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
	AtProvider          UserObservation `json:"atProvider,omitempty"`
}

// UserGroupObservation are the observable fields of a UserGroup.
type UserGroupObservation struct {
	// UserGroupID is the user group id.
	// +kubebuilder:validation:Format=uuid
	UserGroupID string `json:"userID,omitempty"`
}

// +kubebuilder:object:root=true

// UserGroupList contains a list of UserGroup
type UserGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []User `json:"items"`
}

// User type metadata.
var (
	UserGroupTypeKind         = reflect.TypeOf(UserGroup{}).Name()
	UserGroupGroupKind        = schema.GroupKind{Group: Group, Kind: UserGroupKind}.String()
	UserGroupKindAPIVersion   = UserGroupKind + "." + SchemeGroupVersion.String()
	UserGroupGroupVersionKind = SchemeGroupVersion.WithKind(UserGroupKind)
)

func init() {
	SchemeBuilder.Register(&User{}, &UserList{})
}
