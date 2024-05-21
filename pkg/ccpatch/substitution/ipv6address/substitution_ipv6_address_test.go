package ccpatch_test

import (
	"fmt"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/ccpatch/substitution"
)

func TestIPv6AddressSuccess(t *testing.T) {
	handler := substitution.GetSubstitution("ipv6Address")
	if handler == nil {
		t.Errorf("ipv6Address handler not found")
		return
	}

	total := 10

	state := &substitution.GlobalState{}
	for i := 0; i < total; i++ {
		err := handler.WriteState(substitution.Identifier(fmt.Sprintf("machine-%v", i)), state, substitution.Substitution{
			Type:   "ipv6Address",
			Key:    "$ipv6Address",
			Unique: true,
			AdditionalProperties: map[string]string{
				"cidr": "fc00:1::/64",
			},
		})
		require.NoError(t, err)
	}

	require.Equal(t, total, state.Len())
	state.Each(func(identifier substitution.Identifier, state []substitution.State) {
		require.Len(t, state, 1)
		require.Contains(t, state[0].Value, "fc00:1::")
		require.Equal(t, state[0].Key, "$ipv6Address")
	})

	spew.Dump(state)
}
