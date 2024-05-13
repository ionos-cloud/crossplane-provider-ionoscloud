package ccpatch_test

import (
	"testing"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/ccpatch/substitution"
	"github.com/stretchr/testify/require"
)

func TestIPv4AddressSuccess(t *testing.T) {
	handler := substitution.GetSubstitution("ipv4Address")
	state := &substitution.GlobalState{}
	err := handler.WriteState(substitution.Identifier("machine-0"), state, substitution.Substitution{
		Type:   "ipv4Address",
		Key:    "$ipv4Address",
		Unique: true,
		AdditionalProperties: map[string]string{
			"cidr": "10.0.0.0/31",
		},
	})

	expectedState := state.GetByIdentifier("machine-0")

	require.NoError(t, err)
	require.Equal(t, expectedState[0].Value, "10.0.0.1")
}
