package ccpatch_test

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/ccpatch"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/ccpatch/substitution"
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
		{
			Type:   "ipv4Address",
			Key:    "$ipv4",
			Unique: true,
			AdditionalProperties: map[string]string{
				"cidr": "192.0.2.0/24",
			},
		},
	}

	substitionInput = `#cloud-config
hostname: $ipv6Address
ip: $ipv4
`
	substitionOutput = "#cloud-config\nhostname: fc00:1::2\nip: 192.0.2.224\n"
)

func TestSubstitutionManager(t *testing.T) {
	// Identifier is used to lookup the state of the current replica
	identifier := substitution.Identifier("replica-1")
	replica2 := substitution.Identifier("replica-2")

	// Global state of the substitutions
	globalState := &substitution.GlobalState{
		identifier: []substitution.State{
			{
				Key:   "$ipv4",
				Value: "192.0.2.224",
			},
		},
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

func TestExampleReadme(t *testing.T) {
	gs := substitution.NewGlobalState()
	identifier := substitution.Identifier("machine-0")

	encoded := base64.StdEncoding.EncodeToString([]byte(substitionInput))

	ccpatch.NewCloudInitPatcherWithSubstitutions(encoded, identifier, substitutions, gs)
}
