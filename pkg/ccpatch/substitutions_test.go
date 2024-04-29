package ccpatch_test

import (
	"encoding/base64"
	"testing"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/ccpatch"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/ccpatch/substitution"
	"github.com/stretchr/testify/require"
)

var (
	substitutions = []substitution.Substitution{
		{
			Type:   "ipv6Address",
			Key:    "$ipv6Address",
			Unique: true,
			AdditionalProperties: map[string]string{
				"cidr": "fc00:1::1/64",
			},
		},
	}

	substitionInput = `#cloud-config
hostname: $ipv6Address
`
	substitionOutput = "#cloud-config\nhostname: fc00:1::2\n"
)

func TestSubstitutionManager(t *testing.T) {
	// Identifier is used to lookup the state of the current replica
	identifier := substitution.Identifier("replica-1")
	replica2 := substitution.Identifier("replica-2")

	// Global state of the substitutions
	globalState := &substitution.GlobalState{
		identifier: []substitution.State{},
		replica2: []substitution.State{
			{
				Key:   "$ipv6Address",
				Value: "fc00:1::1",
			},
		},
	}

	// Contents of the cloud-init configuration

	encoded := base64.StdEncoding.EncodeToString([]byte(substitionInput))

	cp, err := ccpatch.NewCloudInitPatcherWithSubstitutions(
		encoded,
		identifier,
		substitutions, globalState,
	)
	require.NoError(t, err)
	require.Equal(t, cp.String(), substitionOutput)
}
