package ccpatch_test

import (
	"testing"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/ccpatch"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/ccpatch/substitution"
)

var (
	substitutions = []substitution.Substitution{
		{
			Type:   "ipv4Address",
			Key:    "$ipv4Address",
			Unique: true,
			AdditionalProperties: map[string]string{
				"cidr": "10.0.0.0/24",
			},
		},
	}
)

func TestSubstitutionManager(t *testing.T) {
	// Identifier is used to lookup the state of the current replica
	identifier := substitution.Identifier("replica-1")
	replica2 := substitution.Identifier("replica-2")

	// Global state of the substitutions
	globalState := substitution.GlobalState{
		identifier: []substitution.State{},
		replica2: []substitution.State{
			{
				Key:   "ipv4Address",
				Value: "10.0.0.1",
			},
		},
	}

	// Contents of the cloud-init configuration
	contents := `#cloud-config
hostname: $ipv4Address
`

	ccpatch.NewSubstitutionManager(string(identifier), substitutions, globalState, contents)
}
