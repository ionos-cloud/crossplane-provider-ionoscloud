package lan_test

import (
	"testing"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/assert"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/lan"
)

func TestIsUpToDateWithDiff(t *testing.T) {
	name := "test-lan"
	ipv6Cidr := "2001:db8::/64"
	pccID := "pcc-123"

	tests := []struct {
		name         string
		cr           *v1alpha1.Lan
		lan          sdkgo.Lan
		wantUpToDate bool
		wantReason   string
	}{
		{
			name:         "Both nil",
			cr:           nil,
			lan:          sdkgo.Lan{},
			wantUpToDate: true,
			wantReason:   "Lan does not exist, no update needed",
		},
		{
			name:         "Lan exists but not managed by Crossplane",
			cr:           nil,
			lan:          sdkgo.Lan{Properties: &sdkgo.LanProperties{}},
			wantUpToDate: false,
			wantReason:   "Lan exists but not managed by Crossplane, no update needed",
		},
		{
			name:         "Lan does not exist, update needed",
			cr:           &v1alpha1.Lan{},
			lan:          sdkgo.Lan{},
			wantUpToDate: false,
			wantReason:   "Lan properties are nil, but custom resource is not nil",
		},
		{
			name:         "Lan is busy",
			cr:           &v1alpha1.Lan{},
			lan:          sdkgo.Lan{Properties: sdkgo.NewLanProperties(), Metadata: &sdkgo.DatacenterElementMetadata{State: sdkgo.ToPtr("BUSY")}},
			wantUpToDate: true,
			wantReason:   "Lan is busy, cannot update it now",
		},
		{
			name:         "Name mismatch",
			cr:           &v1alpha1.Lan{Spec: v1alpha1.LanSpec{ForProvider: v1alpha1.LanParameters{Name: "foo"}}},
			lan:          sdkgo.Lan{Properties: &sdkgo.LanProperties{Name: sdkgo.ToPtr("bar")}},
			wantUpToDate: false,
			wantReason:   "Lan name does not match: bar != foo",
		},
		{
			name:         "Name not set but expected",
			cr:           &v1alpha1.Lan{Spec: v1alpha1.LanSpec{ForProvider: v1alpha1.LanParameters{Name: "foo"}}},
			lan:          sdkgo.Lan{Properties: &sdkgo.LanProperties{}},
			wantUpToDate: false,
			wantReason:   "Lan name is not set, expected: foo, got: nil",
		},
		{
			name:         "Public mismatch",
			cr:           &v1alpha1.Lan{Spec: v1alpha1.LanSpec{ForProvider: v1alpha1.LanParameters{Public: false}}},
			lan:          sdkgo.Lan{Properties: &sdkgo.LanProperties{Public: sdkgo.ToPtr(true)}},
			wantUpToDate: false,
			wantReason:   "Lan public property does not match: true != false",
		},
		{
			name:         "Ipv6CidrBlock mismatch",
			cr:           &v1alpha1.Lan{Spec: v1alpha1.LanSpec{ForProvider: v1alpha1.LanParameters{Ipv6Cidr: "2001:db8::/64"}}},
			lan:          sdkgo.Lan{Properties: &sdkgo.LanProperties{Ipv6CidrBlock: sdkgo.ToPtr("2001:db8::/65")}},
			wantUpToDate: false,
			wantReason:   "Lan Ipv6CidrBlock does not match: 2001:db8::/64 != 2001:db8::/65",
		},
		{
			name:         "Pcc mismatch",
			cr:           &v1alpha1.Lan{Spec: v1alpha1.LanSpec{ForProvider: v1alpha1.LanParameters{Pcc: v1alpha1.PccConfig{PrivateCrossConnectID: "pcc-abc"}}}},
			lan:          sdkgo.Lan{Properties: &sdkgo.LanProperties{Pcc: sdkgo.ToPtr("pcc-def")}},
			wantUpToDate: false,
			wantReason:   "Lan Pcc does not match: pcc-abc != pcc-def",
		},
		{
			name: "Up to date",
			cr: &v1alpha1.Lan{Spec: v1alpha1.LanSpec{ForProvider: v1alpha1.LanParameters{
				Name:     name,
				Public:   true,
				Ipv6Cidr: ipv6Cidr,
				Pcc:      v1alpha1.PccConfig{PrivateCrossConnectID: pccID},
			}}},
			lan: sdkgo.Lan{Properties: &sdkgo.LanProperties{
				Name:          &name,
				Public:        sdkgo.ToPtr(true),
				Ipv6CidrBlock: &ipv6Cidr,
				Pcc:           &pccID,
			}},
			wantUpToDate: true,
			wantReason:   "Lan is up-to-date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, diff := lan.IsUpToDateWithDiff(tt.cr, tt.lan)
			assert.Equal(t, tt.wantUpToDate, got, "IsUpToDateWithDiff() returned unexpected result")
			assert.Equal(t, tt.wantReason, diff, "IsUpToDateWithDiff() returned unexpected diff result")
		})
	}
}
