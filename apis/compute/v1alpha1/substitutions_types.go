package v1alpha1

// Substitution defines the substitution configuration
// Example:
// substitutions:
//   - type: ipv4Address
//     key: $keyToReplace
//     unique: true
//     options:
//     cidr: "10.0.0.0/24"
type Substitution struct {
	// +kubebuilder:validation:Required
	// +immutable
	// +kubebuilder:validation:Enum=ipv4Address;ipv6Address
	// The type of the handler that will be used for this substitution. The handler will
	// be responsible for computing the value we put in place of te key (e.g. in the above
	// example the value for $keyToReplace)
	Type string `json:"type" yaml:"type"`
	// +kubebuilder:validation:Required
	// +immutable
	Key string `json:"key" yaml:"key"`
	// Unique means that the value is unique across multiple ServerSets
	// +immutable
	Unique bool `json:"unique" yaml:"unique"`
	// The options for the handler
	Options map[string]string `json:"options" yaml:"options"`
}
