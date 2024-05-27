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

	substitutionInput = `#cloud-config
ipv6: $ipv6Address
ip: $ipv4
`
	substitutionReplica1Output = "#cloud-config\nip: 192.0.2.1\nipv6: fc00:1::1\n"
	substitutionReplica2Output = "#cloud-config\nip: 192.0.2.2\nipv6: 'fc00:1::'\n"
)

func TestSubstitutionManager(t *testing.T) {
	// Identifier is used to look up the state of the current replica
	replica1 := substitution.Identifier("replica-1")
	replica2 := substitution.Identifier("replica-2")

	// Global state of the substitutions
	globalState := &substitution.GlobalState{
		replica1: []substitution.State{
			{
				Key:   "$ipv4Address",
				Value: "192.0.2.224",
			},
		},
		replica2: []substitution.State{
			{
				Key:   "$ipv6Address",
				Value: "fc00:1::",
			},
		},
	}

	// Contents of the cloud-init configuration
	encoded := base64.StdEncoding.EncodeToString([]byte(substitutionInput))

	cp, err := ccpatch.NewCloudInitPatcherWithSubstitutions(
		encoded,
		replica1,
		substitutions, globalState,
	)
	require.NoError(t, err)
	require.Equalf(t, substitutionReplica1Output, cp.String(), "expected equality for replica-1")
	cp, err = ccpatch.NewCloudInitPatcherWithSubstitutions(
		encoded,
		replica2,
		substitutions, globalState,
	)
	require.NoError(t, err)
	require.Equalf(t, substitutionReplica2Output, cp.String(), "expected equality for replica-2")
}
