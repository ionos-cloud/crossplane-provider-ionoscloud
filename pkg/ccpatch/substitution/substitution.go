package substitution

import (
	"strings"
)

// Substitution defines a substitution that can be used to replace
// placeholders in the cloud-init configuration.
type Substitution struct {
	Type                 string            `json:"type" yaml:"type"` // Type of the substitution, ipv4Address, ipv6Address, etc
	Key                  string            `json:"key" yaml:"key"`   // Name of the substitution to be replaced $ipv4Address for example
	Unique               bool              `json:"unique" yaml:"unique"`
	AdditionalProperties map[string]string `json:"additionalProperties" yaml:"additionalProperties"`
}

// ReplaceByState replaces the placeholders in the target string with the values from the state
func ReplaceByState(identifier Identifier, globalState *GlobalState, target string) (string, error) {
	output := target

	stateMap := map[string]string{}

	states := globalState.GetByIdentifier(identifier)
	for _, state := range states {
		stateMap[state.Key] = "'" + state.Value + "'"
	}

	for k, v := range stateMap {
		output = strings.ReplaceAll(output, k, v)
	}

	return output, nil
}
