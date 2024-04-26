package ccpatch_test

import (
	"fmt"
	"testing"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/ccpatch"
	"github.com/stretchr/testify/require"
)

func TestIPv6AddressSuccess(t *testing.T) {
	handler := ccpatch.GetSubstitution("ipv6Address")
	if handler == nil {
		t.Errorf("ipv6Address handler not found")
		return
	}

	total := 10

	state := &ccpatch.GlobalState{}
	for i := 0; i < total; i++ {
		handler.WriteState(ccpatch.Identifier(fmt.Sprintf("machine-%v", i)), state, ccpatch.Substitution{
			Type:   "ipv6Address",
			Key:    "$ipv6Address",
			Unique: true,
			AdditionalProperties: map[string]string{
				"cidr": "fc00:1::1/64",
			},
		})
	}

	require.Equal(t, total, state.Len())
	state.Each(func(identifier ccpatch.Identifier, state []ccpatch.State) {
		require.Len(t, state, 1)
		require.Contains(t, state[0].Value, "fc00:1::")
		require.Equal(t, state[0].Key, "$ipv6Address")
	})
}
