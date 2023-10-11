package v1alpha1

import (
	"reflect"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// S3KeyParameters are the observable fields of an S3Key.
// Required values when creating an S3Key:
// UserID
type S3KeyParameters struct {
	// The UUID of the user owning the S3 Key.
	//
	// +kubebuilder:validation:Required
	UserID string `json:"userID"`
	// The S3 Secret key.
	//
	// +immutable
	// +kubebuilder:validation:Optional
	SecretKey string `json:"secretKey"`
	// Whether the S3 is active / enabled or not. Can only be updated to false, by default the key will be created as active. Default value is true.
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:default=true
	Active bool `json:"active,omitempty"`
}

// +kubebuilder:object:root=true

// A S3Key is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="S3Key ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="UserID",priority=1,type="string",JSONPath=".spec.forProvider.userID"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud}
type S3Key struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec               S3KeySpec               `json:"spec"`
	Status             S3KeyStatus             `json:"status,omitempty"`
	ManagementPolicies xpv1.ManagementPolicies `json:"managementPolicies"`
}

// SetManagementPolicies implement managed interface
func (mg *S3Key) SetManagementPolicies(p xpv1.ManagementPolicies) {
	mg.ManagementPolicies = p
}

// GetManagementPolicies implement managed interface
func (mg *S3Key) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.ManagementPolicies
}

// A S3KeySpec defines the desired state of a S3Key.
type S3KeySpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       S3KeyParameters `json:"forProvider"`
}

// A S3KeyStatus represents the observed state of a S3Key.
type S3KeyStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          S3KeyObservation `json:"atProvider,omitempty"`
}

// S3KeyObservation are the observable fields of a S3Key.
type S3KeyObservation struct {
	SecretKey string `json:"secretKey,omitempty"`
	S3KeyID   string `json:"s3KeyID,omitempty"`
}

// +kubebuilder:object:root=true

// S3KeyList contains a list of S3Key
type S3KeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []S3Key `json:"items"`
}

// S3Key type metadata.
var (
	S3KeyKind             = reflect.TypeOf(S3Key{}).Name()
	S3KeyGroupKind        = schema.GroupKind{Group: Group, Kind: S3KeyKind}.String()
	S3KeyKindAPIVersion   = S3KeyKind + "." + SchemeGroupVersion.String()
	S3KeyGroupVersionKind = SchemeGroupVersion.WithKind(S3KeyKind)
)

func init() {
	SchemeBuilder.Register(&S3Key{}, &S3KeyList{})
}
