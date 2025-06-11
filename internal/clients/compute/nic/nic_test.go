package nic

import (
	"testing"

	"github.com/ionos-cloud/sdk-go-bundle/shared"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/assert"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
)

func TestIsNicUpToDate(t *testing.T) {
	type args struct {
		cr  *v1alpha1.Nic
		Nic ionoscloud.Nic
		ips []string
	}
	tests := []struct {
		name     string
		args     args
		want     bool
		wantDiff string
	}{
		{
			name: "both empty",
			args: args{
				cr:  nil,
				Nic: ionoscloud.Nic{},
			},
			want:     true,
			wantDiff: "Nic is not created yet, no properties to compare",
		},
		{
			name: "CR empty",
			args: args{
				cr: nil,
				Nic: ionoscloud.Nic{Properties: &ionoscloud.NicProperties{
					Name: shared.ToPtr("foo"),
				}},
			},
			want:     false,
			wantDiff: "Nic is not created yet, but properties are set in the Nic object",
		},
		{
			name: "api response empty",
			args: args{
				cr: &v1alpha1.Nic{
					Spec: v1alpha1.NicSpec{
						ForProvider: v1alpha1.NicParameters{
							Name: "not empty",
						},
					},
				},
				Nic: ionoscloud.Nic{Properties: nil},
			},
			want:     false,
			wantDiff: "Nic properties are not set in the Nic object, but CR is set",
		},
		{
			name: "all equal",
			args: args{
				cr: &v1alpha1.Nic{
					Spec: v1alpha1.NicSpec{
						ForProvider: v1alpha1.NicParameters{
							Name:           "not empty",
							Dhcp:           false,
							FirewallActive: false,
							FirewallType:   "INGRESS",
							Vnet:           "1",
						},
					},
				},
				Nic: ionoscloud.Nic{
					Properties: &ionoscloud.NicProperties{
						Name:           shared.ToPtr("not empty"),
						FirewallActive: shared.ToPtr(false),
						FirewallType:   shared.ToPtr("INGRESS"),
						Vnet:           shared.ToPtr("1"),
					},
				},
			},
			want:     true,
			wantDiff: "Nic is up-to-date",
		},
		{
			name: "all different",
			args: args{
				cr: &v1alpha1.Nic{
					Spec: v1alpha1.NicSpec{
						ForProvider: v1alpha1.NicParameters{
							Name:           "different",
							Dhcp:           false,
							FirewallActive: false,
							FirewallType:   "INGRESS",
							Vnet:           "1",
						},
					},
				},
				Nic: ionoscloud.Nic{Properties: &ionoscloud.NicProperties{
					Name:           shared.ToPtr("not empty"),
					FirewallActive: shared.ToPtr(true),
					FirewallType:   shared.ToPtr("EGRESS"),
					Vnet:           shared.ToPtr("2"),
				}},
			},
			want:     false,
			wantDiff: "Nic name does not match the one in the CR not empty != different",
		},
		{
			name: "only vnet differs",
			args: args{
				cr: &v1alpha1.Nic{
					Spec: v1alpha1.NicSpec{
						ForProvider: v1alpha1.NicParameters{
							Name:           "not empty",
							Dhcp:           false,
							FirewallActive: false,
							FirewallType:   "INGRESS",
							Vnet:           "1",
						},
					},
				},
				Nic: ionoscloud.Nic{Properties: &ionoscloud.NicProperties{
					Name:           shared.ToPtr("not empty"),
					FirewallActive: shared.ToPtr(false),
					FirewallType:   shared.ToPtr("INGRESS"),
					Vnet:           shared.ToPtr("2"),
				}},
			},
			want:     false,
			wantDiff: "Nic Vnet does not match the one in the CR 2 != 1",
		},
		{
			name: "metadata state 'Busy', different name",
			args: args{
				cr: &v1alpha1.Nic{
					Spec: v1alpha1.NicSpec{
						ForProvider: v1alpha1.NicParameters{
							Name: "not empty",
						},
					},
					Status: v1alpha1.NicStatus{},
				},
				Nic: ionoscloud.Nic{
					Metadata: &ionoscloud.DatacenterElementMetadata{
						State: shared.ToPtr(ionoscloud.Busy),
					},
					Properties: &ionoscloud.NicProperties{
						Name: shared.ToPtr("empty"),
					},
				},
			},
			want:     true,
			wantDiff: "Nic is busy, cannot update it now",
		},
		{
			name: "IPs different from previous IPs",
			args: args{
				cr: &v1alpha1.Nic{
					Spec: v1alpha1.NicSpec{
						ForProvider: v1alpha1.NicParameters{
							Name: "empty",
							IpsCfg: v1alpha1.IPsConfigs{
								IPs: []string{
									"10.10.10.10",
									"10.10.10.11",
									"2001:0db8:85a3::8a2e:0370:7335",
								},
							},
						},
					},
					Status: v1alpha1.NicStatus{},
				},
				ips: []string{
					"10.10.10.10",
					"10.10.10.11",
					"2001:0db8:85a3::8a2e:0370:7335",
				},
				Nic: ionoscloud.Nic{
					Properties: &ionoscloud.NicProperties{
						Name: shared.ToPtr("empty"),
						Ips: &[]string{
							"10.11.12.13",
							"192.168.8.14",
						},
						Ipv6Ips: &[]string{
							"2001:0db8:85a3::8a2e:0370:7334",
						},
					},
				},
			},
			want:     false,
			wantDiff: "Nic IPv4s do not match the ones in the CR [10.11.12.13 192.168.8.14] != [10.10.10.10 10.10.10.11]",
		},
		{
			name: "IPs equal",
			args: args{
				cr: &v1alpha1.Nic{
					Spec: v1alpha1.NicSpec{
						ForProvider: v1alpha1.NicParameters{
							Name: "empty",
							IpsCfg: v1alpha1.IPsConfigs{
								IPs: []string{
									"10.10.10.10",
									"10.10.10.11",
									"2001:0db8:85a3::8a2e:0370:7335",
								},
							},
						},
					},
					Status: v1alpha1.NicStatus{},
				},
				ips: []string{
					"10.10.10.10",
					"10.10.10.11",
					"2001:0db8:85a3::8a2e:0370:7335",
				},
				Nic: ionoscloud.Nic{
					Properties: &ionoscloud.NicProperties{
						Name: shared.ToPtr("empty"),
						Ips: &[]string{
							"10.10.10.10",
							"10.10.10.11",
						},
						Ipv6Ips: &[]string{
							"2001:0db8:85a3::8a2e:0370:7335",
						},
					},
				},
			},
			want:     true,
			wantDiff: "Nic is up-to-date",
		},
		{
			name: "NIC dhcpv6 is nil",
			args: args{
				cr: &v1alpha1.Nic{
					Spec: v1alpha1.NicSpec{
						ForProvider: v1alpha1.NicParameters{
							DhcpV6: ionoscloud.ToPtr(true),
						},
					},
				},
				Nic: ionoscloud.Nic{
					Properties: &ionoscloud.NicProperties{
						Dhcpv6: nil,
					},
				},
			},
			want:     true,
			wantDiff: "Nic is up-to-date",
		},
		{
			name: "CR dhcpv6 is nil",
			args: args{
				cr: &v1alpha1.Nic{
					Spec: v1alpha1.NicSpec{
						ForProvider: v1alpha1.NicParameters{
							DhcpV6: nil,
						},
					},
				},
				Nic: ionoscloud.Nic{
					Properties: &ionoscloud.NicProperties{
						Dhcpv6: ionoscloud.ToPtr(true),
					},
				},
			},
			want:     true,
			wantDiff: "Nic is up-to-date",
		},
		{
			name: "CR and NIC dhcpv6 are nil",
			args: args{
				cr: &v1alpha1.Nic{
					Spec: v1alpha1.NicSpec{
						ForProvider: v1alpha1.NicParameters{
							DhcpV6: nil,
						},
					},
				},
				Nic: ionoscloud.Nic{
					Properties: &ionoscloud.NicProperties{
						Dhcpv6: nil,
					},
				},
			},
			want:     true,
			wantDiff: "Nic is up-to-date",
		},
		{
			name: "CR and NIC dhcpv6 are equal",
			args: args{
				cr: &v1alpha1.Nic{
					Spec: v1alpha1.NicSpec{
						ForProvider: v1alpha1.NicParameters{
							DhcpV6: ionoscloud.ToPtr(true),
						},
					},
				},
				Nic: ionoscloud.Nic{
					Properties: &ionoscloud.NicProperties{
						Dhcpv6: ionoscloud.ToPtr(true),
					},
				},
			},
			want:     true,
			wantDiff: "Nic is up-to-date",
		},
		{
			name: "CR and NIC dhcpv6 are different",
			args: args{
				cr: &v1alpha1.Nic{
					Spec: v1alpha1.NicSpec{
						ForProvider: v1alpha1.NicParameters{
							DhcpV6: ionoscloud.ToPtr(false),
						},
					},
				},
				Nic: ionoscloud.Nic{
					Properties: &ionoscloud.NicProperties{
						Dhcpv6: ionoscloud.ToPtr(true),
					},
				},
			},
			want:     false,
			wantDiff: "Nic DHCPv6 does not match the one in the CR true != false",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotDiff := IsUpToDateWithDiff(tt.args.cr, tt.args.Nic, tt.args.ips)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantDiff, gotDiff)
		})
	}
}
