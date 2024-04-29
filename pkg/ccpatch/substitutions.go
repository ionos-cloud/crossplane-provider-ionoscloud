package ccpatch

import (
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/ccpatch/substitution"
)

func NewSubstitutionManager(identifier string, substitutions []substitution.Substitution, globalState substitution.GlobalState, contents string) {
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
