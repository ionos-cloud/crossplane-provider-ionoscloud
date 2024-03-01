package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// UserParameters defines the desired state of a User.
// Required values when creating a User:
// Administrator
// Email
// FirstName
// ForceSecAuth
// LastName
// Password
type UserParameters struct {
	// Administrator The group has permission to edit privileges on this resource.
	//
	// +kubebuilder:validation:Required
	Administrator bool `json:"administrator"`
	// Email An e-mail address for the user.
	//
	// +kubebuilder:validation:Required
	Email string `json:"email"`
	// FirstName A first name for the user.
	//
	// +kubebuilder:validation:Required
	FirstName string `json:"firstName"`
	// ForceSecAuth Indicates if secure (two-factor) authentication should be enabled for the user (true) or not (false).
	//
	// +kubebuilder:validation:Required
	ForceSecAuth bool `json:"forceSecAuth"`
	// LastName A last name for the user.
	//
	// +kubebuilder:validation:Required
	LastName string `json:"lastName"`
	// Password A password for the user.
	//
	// +kubebuilder:validation:Required
	Password string `json:"password"`
	// SecAuthActive Indicates if secure authentication is active for the user or not.
	// It can not be used in create requests - can be used in update. Default: false.
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:default=false
	SecAuthActive bool `json:"secAuthActive"`
	// Active Indicates if the user is active. Default: true.
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:default=true
	Active bool `json:"active"`
}

// UserConfig is used by resources that need to link Users via id or via reference.
type UserConfig struct {
	// UserID is the ID of the User on which the resource should have access.
	// It needs to be provided directly or via reference.
	//
	// +immutable
	// +kubebuilder:validation:Format=uuid
	// +crossplane:generate:reference:type=github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1.User
	// +crossplane:generate:reference:extractor=github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1.ExtractUserID()
	UserID string `json:"userId,omitempty"`
	// UserIDRef references to a User to retrieve its ID.
	//
	// +optional
	// +immutable
	UserIDRef *xpv1.Reference `json:"userIdRef,omitempty"`
	// UserIDSelector selects reference to a User to retrieve its UserID.
	//
	// +optional
	UserIDSelector *xpv1.Selector `json:"userIdSelector,omitempty"`
}

// +kubebuilder:object:root=true

// User is our managed resource.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="USER_ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="ACTIVE",type="string",JSONPath=".status.atProvider.active"
// +kubebuilder:printcolumn:name="EMAIL",type="string",JSONPath=".spec.forProvider.email"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud}
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserSpec   `json:"spec"`
	Status UserStatus `json:"status,omitempty"`
}

// A UserSpec defines the desired state of a User.
type UserSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       UserParameters `json:"forProvider"`
}

// A UserStatus represents the observed state of a User.
type UserStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          UserObservation `json:"atProvider,omitempty"`
}

// UserObservation are the observable fields of a User.
type UserObservation struct {
	// UserID is the user id.
	// +kubebuilder:validation:Format=uuid
	UserID string `json:"userID,omitempty"`
	// S3CanonicalUserID Canonical (S3) id of the user for a given identity.
	S3CanonicalUserID string `json:"s3CanonicalUserID,omitempty"`
	// Active Indicates if the user is active.
	Active bool `json:"active"`
	// SecAuthActive Indicates if secure authentication is active for the user or not.
	SecAuthActive bool `json:"secAuthActive"`
}

// +kubebuilder:object:root=true

// UserList contains a list of User
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []User `json:"items"`
}

// User type metadata.
var (
	UserKind             = reflect.TypeOf(User{}).Name()
	UserGroupKind        = schema.GroupKind{Group: APIGroup, Kind: UserKind}.String()
	UserKindAPIVersion   = UserKind + "." + SchemeGroupVersion.String()
	UserGroupVersionKind = SchemeGroupVersion.WithKind(UserKind)
)

func init() {
	SchemeBuilder.Register(&User{}, &UserList{})
}
