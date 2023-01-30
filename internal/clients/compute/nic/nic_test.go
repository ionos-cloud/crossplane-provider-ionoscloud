package nic

import (
	"testing"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"

	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
)

func TestIsNicUpToDate(t *testing.T) {
	type args struct {
		cr     *v1alpha1.Nic
		Nic    ionoscloud.Nic
		ips    []string
		oldIps []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "both empty",
			args: args{
				cr:  nil,
				Nic: ionoscloud.Nic{},
			},
			want: true,
		},
		{
			name: "cr empty",
			args: args{
				cr: nil,
				Nic: ionoscloud.Nic{Properties: &ionoscloud.NicProperties{
					Name: ionoscloud.PtrString("foo"),
				}},
			},
			want: false,
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
			want: false,
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
						Name:           ionoscloud.PtrString("not empty"),
						FirewallActive: ionoscloud.PtrBool(false),
						FirewallType:   ionoscloud.PtrString("INGRESS"),
						Vnet:           ionoscloud.PtrString("1"),
					}},
			},
			want: true,
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
					Name:           ionoscloud.PtrString("not empty"),
					FirewallActive: ionoscloud.PtrBool(true),
					FirewallType:   ionoscloud.PtrString("EGRESS"),
					Vnet:           ionoscloud.PtrString("2"),
				}},
			},
			want: false,
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
					Name:           ionoscloud.PtrString("not empty"),
					FirewallActive: ionoscloud.PtrBool(false),
					FirewallType:   ionoscloud.PtrString("INGRESS"),
					Vnet:           ionoscloud.PtrString("2"),
				}},
			},
			want: false,
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
						State: ionoscloud.PtrString(ionoscloud.Busy),
					},
					Properties: &ionoscloud.NicProperties{
						Name: ionoscloud.PtrString("empty"),
					}},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNicUpToDate(tt.args.cr, tt.args.Nic, tt.args.ips, tt.args.oldIps); got != tt.want {
				t.Errorf("isNicUpToDate() = %v, want %v", got, tt.want)
			}
		})
	}
}
