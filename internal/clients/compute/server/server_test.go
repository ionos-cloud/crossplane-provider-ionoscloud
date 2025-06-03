package server

import (
	"testing"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"

	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
)

func TestIsServerUpToDate(t *testing.T) {
	type args struct {
		cr     *v1alpha1.Server
		Server ionoscloud.Server
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "both empty",
			args: args{
				cr:     nil,
				Server: ionoscloud.Server{},
			},
			want: true,
		},
		{
			name: "cr empty",
			args: args{
				cr: nil,
				Server: ionoscloud.Server{Properties: &ionoscloud.ServerProperties{
					Name: ionoscloud.PtrString("foo"),
				}},
			},
			want: false,
		},
		{
			name: "api response empty",
			args: args{
				cr: &v1alpha1.Server{
					Spec: v1alpha1.ServerSpec{
						ForProvider: v1alpha1.ServerParameters{
							Name: "not empty",
						},
					},
				},
				Server: ionoscloud.Server{Properties: nil},
			},
			want: false,
		},
		{
			name: "all equal",
			args: args{
				cr: &v1alpha1.Server{
					Spec: v1alpha1.ServerSpec{
						ForProvider: v1alpha1.ServerParameters{
							Name:             "not empty",
							CPUFamily:        "super fast",
							AvailabilityZone: "AUTO",
							Cores:            4,
							RAM:              2048,
							PlacementGroupID: "testPlacementGroup",
						},
					},
				},
				Server: ionoscloud.Server{Properties: &ionoscloud.ServerProperties{
					Name:             ionoscloud.PtrString("not empty"),
					CpuFamily:        ionoscloud.PtrString("super fast"),
					AvailabilityZone: ionoscloud.PtrString("AUTO"),
					Cores:            ionoscloud.PtrInt32(4),
					Ram:              ionoscloud.PtrInt32(2048),
					PlacementGroupId: ionoscloud.PtrString("testPlacementGroup"),
				}},
			},
			want: true,
		},
		{
			name: "all different",
			args: args{
				cr: &v1alpha1.Server{
					Spec: v1alpha1.ServerSpec{
						ForProvider: v1alpha1.ServerParameters{
							Name:             "different",
							Cores:            4,
							RAM:              2048,
							AvailabilityZone: "1",
							CPUFamily:        "super slow",
							PlacementGroupID: "testPlacementGroup",
						},
					},
				},
				Server: ionoscloud.Server{Properties: &ionoscloud.ServerProperties{
					Name:             ionoscloud.PtrString("not empty"),
					Cores:            ionoscloud.PtrInt32(8),
					Ram:              ionoscloud.PtrInt32(4086),
					AvailabilityZone: ionoscloud.PtrString("AUTO"),
					CpuFamily:        ionoscloud.PtrString("super fast"),
					PlacementGroupId: ionoscloud.PtrString("testPlacementGroupUpdated"),
				}},
			},
			want: false,
		},
		{
			name: "only placementGroup differ",
			args: args{
				cr: &v1alpha1.Server{
					Spec: v1alpha1.ServerSpec{
						ForProvider: v1alpha1.ServerParameters{
							Name:             "not empty",
							CPUFamily:        "super fast",
							AvailabilityZone: "AUTO",
							PlacementGroupID: "testPlacementGroup",
						},
					},
				},
				Server: ionoscloud.Server{Properties: &ionoscloud.ServerProperties{
					Name:             ionoscloud.PtrString("not empty"),
					CpuFamily:        ionoscloud.PtrString("super fast"),
					AvailabilityZone: ionoscloud.PtrString("AUTO"),
					PlacementGroupId: ionoscloud.PtrString("testPlacementGroupUpdated"),
				}},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUpToDate(tt.args.cr, tt.args.Server); got != tt.want {
				t.Errorf("isServerUpToDate() = %v, want %v", got, tt.want)
			}
		})
	}
}
