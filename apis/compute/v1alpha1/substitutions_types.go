package v1alpha1

type Substitution struct {
	// +kubebuilder:validation:Required
	// +immutable
	Type string `json:"type" yaml:"type"`
	// +kubebuilder:validation:Required
	// +immutable
	Key                  string            `json:"key" yaml:"key"`
	Unique               bool              `json:"unique" yaml:"unique"`
	AdditionalProperties map[string]string `json:"additionalProperties" yaml:"additionalProperties"`
}
