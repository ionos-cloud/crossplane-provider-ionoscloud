package ccpatch

import (
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/ccpatch/substitution"
)

// populateGlobalState is a helper function to build the state of the substitutions
func populateGlobalState(identifier substitution.Identifier, s []substitution.Substitution, gs *substitution.GlobalState) error {
	for _, sub := range s {
		handler := substitution.GetSubstitution(sub.Type)
		if handler == nil {
			continue
		}

		if err := handler.WriteState(identifier, gs, sub); err != nil {
			return err
		}
	}

	return nil
}
