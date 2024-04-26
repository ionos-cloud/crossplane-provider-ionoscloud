package ccpatch

type Identifier string

// GlobalState defines the global state of the substitutions
// it's a map of index to a list of substitutions
type GlobalState map[Identifier][]State

func (gs GlobalState) Each(f func(Identifier, []State)) {
	for k, v := range gs {
		f(k, v)
	}
}

func (gs GlobalState) Set(identifier Identifier, key string, value string) {
	if gs[identifier] == nil {
		gs[identifier] = []State{}
	}

	gs[identifier] = append(gs[identifier], State{
		Key:   key,
		Value: value,
	})
}

func (gs GlobalState) Len() int {
	return len(gs)
}

// State keeps track of "Key" to generated "Value".
type State struct {
	Key   string
	Value string
}

// Substitution defines a substitution that can be used to replace
// placeholders in the cloud-init configuration.
type Substitution struct {
	Type                 string            `json:"type" yaml:"type"` // Type of the substitution, ipv4Address, ipv6Address, etc
	Key                  string            `json:"key" yaml:"key"`   // Name of the substitution to be replaced $ipv4Address for example
	Unique               bool              `json:"unique" yaml:"unique"`
	AdditionalProperties map[string]string `json:"additionalProperties" yaml:"additionalProperties"`
}

func NewSubstitutionManager(identifier string, substitutions []Substitution, globalState GlobalState, contents string) {
	// Identifier is used to lookup the state of the current replica

	for _, substitution := range substitutions {
		// Check if the substitution is unique
		// if it is, check if it already exists in the global state
		// if it does, use the value from the global state
		// if it doesn't, generate a new value and add it to the global state

		// Replace the placeholders in the cloud-init configuration
		// with the generated values

		_ = substitution
	}
}
