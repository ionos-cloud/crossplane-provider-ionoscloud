package v1alpha1

// Substitution defines the substitution configuration. Can be used to replace a key in the cloud-init user data with a value computed by a handler
// given in the options field.
// Example:
// substitutions:
//   - type: ipv4Address
//     key: $keyToReplace
//     unique: true
//     options:
//     cidr: "10.0.0.0/24"
type Substitution struct {
	// The type of the handler that will be used for this substitution. The handler will
	// be responsible for computing the value we put in place of te key
	// +kubebuilder:validation:Required
	// +immutable
	// +kubebuilder:validation:Enum=ipv4Address;ipv6Address
	Type string `json:"type" yaml:"type"`
	// The key that will be replaced by the value computed by the handler
	// +kubebuilder:validation:Required
	// +immutable
	Key string `json:"key" yaml:"key"`
	// The value is unique across multiple ServerSets
	// +immutable
	Unique bool `json:"unique" yaml:"unique"`
	// The options for the handler. For example, for ipv4Address and ipv6Address handlers, we need to specify cidr as an option
	Options map[string]string `json:"options" yaml:"options"`
}
