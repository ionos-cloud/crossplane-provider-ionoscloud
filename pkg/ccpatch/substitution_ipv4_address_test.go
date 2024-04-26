package ccpatch_test

import (
	"testing"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/ccpatch"
)

func TestIPv4AddressSuccess(t *testing.T) {
	handler := ccpatch.GetSubstitution("ipv4Address")
	state := &ccpatch.GlobalState{}
	handler.WriteState(ccpatch.Identifier("machine-0"), state, ccpatch.Substitution{
		Type:   "ipv4Address",
		Key:    "$ipv4Address",
		Unique: true,
		AdditionalProperties: map[string]string{
			"cidr": "10.0.0.0/24",
		},
	})

}
