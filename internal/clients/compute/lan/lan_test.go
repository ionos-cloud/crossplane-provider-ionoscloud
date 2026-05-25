package lan_test

import (
	"testing"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/assert"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/lan"
)

func TestIsUpToDate(t *testing.T) {
	name := "test-lan"
	ipv6Cidr := "2001:db8::/64"
	pccID := "pcc-123"

	tests := []struct {
		name         string
		cr           *v1alpha1.Lan
		lan          sdkgo.Lan
		wantUpToDate bool
		wantDiff     string
	}{
		{
			name:         "Both nil",
			cr:           nil,
			lan:          sdkgo.Lan{},
			wantUpToDate: true,
			wantDiff:     "",
		},
		{
			name:         "Lan exists but not managed by Crossplane",
			cr:           nil,
			lan:          sdkgo.Lan{Properties: &sdkgo.LanProperties{}},
			wantUpToDate: false,
			wantDiff:     "lan properties presence mismatch",
		},
		{
			name:         "Lan does not exist, update needed",
			cr:           &v1alpha1.Lan{},
			lan:          sdkgo.Lan{},
			wantUpToDate: false,
			wantDiff:     "lan properties presence mismatch",
		},
		{
			name:         "Lan is busy",
			cr:           &v1alpha1.Lan{},
			lan:          sdkgo.Lan{Properties: sdkgo.NewLanProperties(), Metadata: &sdkgo.DatacenterElementMetadata{State: sdkgo.ToPtr("BUSY")}},
			wantUpToDate: true,
			wantDiff:     "",
		},
		{
			name:         "Name mismatch",
			cr:           &v1alpha1.Lan{Spec: v1alpha1.LanSpec{ForProvider: v1alpha1.LanParameters{Name: "foo"}}},
			lan:          sdkgo.Lan{Properties: &sdkgo.LanProperties{Name: sdkgo.ToPtr("bar")}},
			wantUpToDate: false,
			wantDiff:     "name exp=foo act=bar",
		},
		{
			name:         "Name not set but expected",
			cr:           &v1alpha1.Lan{Spec: v1alpha1.LanSpec{ForProvider: v1alpha1.LanParameters{Name: "foo"}}},
			lan:          sdkgo.Lan{Properties: &sdkgo.LanProperties{}},
			wantUpToDate: false,
			wantDiff:     "name exp=foo act=<nil>",
		},
		{
			name:         "Public mismatch",
			cr:           &v1alpha1.Lan{Spec: v1alpha1.LanSpec{ForProvider: v1alpha1.LanParameters{Public: false}}},
			lan:          sdkgo.Lan{Properties: &sdkgo.LanProperties{Public: sdkgo.ToPtr(true)}},
			wantUpToDate: false,
			wantDiff:     "public exp=false act=true",
		},
		{
			name:         "Ipv6CidrBlock mismatch",
			cr:           &v1alpha1.Lan{Spec: v1alpha1.LanSpec{ForProvider: v1alpha1.LanParameters{Ipv6Cidr: "2001:db8::/64"}}},
			lan:          sdkgo.Lan{Properties: &sdkgo.LanProperties{Ipv6CidrBlock: sdkgo.ToPtr("2001:db8::/65")}},
			wantUpToDate: false,
			wantDiff:     "ipv6CidrBlock exp=2001:db8::/64 act=2001:db8::/65",
		},
		{
			name:         "Pcc mismatch",
			cr:           &v1alpha1.Lan{Spec: v1alpha1.LanSpec{ForProvider: v1alpha1.LanParameters{Pcc: v1alpha1.PccConfig{PrivateCrossConnectID: "pcc-abc"}}}},
			lan:          sdkgo.Lan{Properties: &sdkgo.LanProperties{Pcc: sdkgo.ToPtr("pcc-def")}},
			wantUpToDate: false,
			wantDiff:     "pcc exp=pcc-abc act=pcc-def",
		},
		{
			name:         "Pcc not set but expected",
			cr:           &v1alpha1.Lan{Spec: v1alpha1.LanSpec{ForProvider: v1alpha1.LanParameters{Pcc: v1alpha1.PccConfig{PrivateCrossConnectID: "pcc-123"}}}},
			lan:          sdkgo.Lan{Properties: &sdkgo.LanProperties{}},
			wantUpToDate: false,
			wantDiff:     "pcc exp=pcc-123 act=<nil>",
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
			wantDiff:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotDiff := lan.IsUpToDate(tt.cr, tt.lan)
			assert.Equal(t, tt.wantUpToDate, got, "IsUpToDate() upToDate")
			assert.Equal(t, tt.wantDiff, gotDiff, "IsUpToDate() diff")
		})
	}
}
