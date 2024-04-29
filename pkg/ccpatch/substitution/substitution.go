package substitution

// Substitution defines a substitution that can be used to replace
// placeholders in the cloud-init configuration.
type Substitution struct {
	Type                 string            `json:"type" yaml:"type"` // Type of the substitution, ipv4Address, ipv6Address, etc
	Key                  string            `json:"key" yaml:"key"`   // Name of the substitution to be replaced $ipv4Address for example
	Unique               bool              `json:"unique" yaml:"unique"`
	AdditionalProperties map[string]string `json:"additionalProperties" yaml:"additionalProperties"`
}
