package server

import (
	"testing"

	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/assert"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
)

func TestIsServerUpToDate(t *testing.T) {
	type args struct {
		cr     *v1alpha1.Server
		Server ionoscloud.Server
	}
	tests := []struct {
		name           string
		args           args
		wantIsUpToDate bool
		wantDiff       string
	}{
		{
			name: "both empty",
			args: args{
				cr:     nil,
				Server: ionoscloud.Server{},
			},
			wantIsUpToDate: true,
			wantDiff:       "Server is nil",
		},
		{
			name: "cr empty",
			args: args{
				cr: nil,
				Server: ionoscloud.Server{Properties: &ionoscloud.ServerProperties{
					Name: ionoscloud.ToPtr("foo"),
				}},
			},
			wantIsUpToDate: false,
			wantDiff:       "Server is nil, but server properties are not nil",
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
			wantIsUpToDate: false,
			wantDiff:       "Server properties are nil, but server is not nil",
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
							NicMultiQueue:    ionoscloud.ToPtr(true),
						},
					},
				},
				Server: ionoscloud.Server{Properties: &ionoscloud.ServerProperties{
					Name:             ionoscloud.ToPtr("not empty"),
					CpuFamily:        ionoscloud.ToPtr("super fast"),
					AvailabilityZone: ionoscloud.ToPtr("AUTO"),
					Cores:            ionoscloud.PtrInt32(4),
					Ram:              ionoscloud.PtrInt32(2048),
					PlacementGroupId: ionoscloud.ToPtr("testPlacementGroup"),
					NicMultiQueue:    ionoscloud.ToPtr(true),
				}},
			},
			wantIsUpToDate: true,
			wantDiff:       "Server is up-to-date",
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
					Name:             ionoscloud.ToPtr("not empty"),
					Cores:            ionoscloud.PtrInt32(8),
					Ram:              ionoscloud.PtrInt32(4086),
					AvailabilityZone: ionoscloud.ToPtr("AUTO"),
					CpuFamily:        ionoscloud.ToPtr("super fast"),
					PlacementGroupId: ionoscloud.ToPtr("testPlacementGroupUpdated"),
				}},
			},
			wantIsUpToDate: false,
			wantDiff:       "Server name does not match the CR name: not empty != different",
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
					Name:             ionoscloud.ToPtr("not empty"),
					CpuFamily:        ionoscloud.ToPtr("super fast"),
					AvailabilityZone: ionoscloud.ToPtr("AUTO"),
					PlacementGroupId: ionoscloud.ToPtr("testPlacementGroupUpdated"),
				}},
			},
			wantIsUpToDate: false,
			wantDiff:       "Server placement group ID does not match the CR placement group ID: testPlacementGroupUpdated != testPlacementGroup",
		},
		{
			name: "only nicMultiQueue different",
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
							NicMultiQueue:    ionoscloud.ToPtr(true),
						},
					},
				},
				Server: ionoscloud.Server{Properties: &ionoscloud.ServerProperties{
					Name:             ionoscloud.ToPtr("not empty"),
					CpuFamily:        ionoscloud.ToPtr("super fast"),
					AvailabilityZone: ionoscloud.ToPtr("AUTO"),
					Cores:            ionoscloud.PtrInt32(4),
					Ram:              ionoscloud.PtrInt32(2048),
					PlacementGroupId: ionoscloud.ToPtr("testPlacementGroup"),
					NicMultiQueue:    ionoscloud.ToPtr(false),
				}},
			},
			wantIsUpToDate: true,
			wantDiff:       "NicMultiQueue do not match the CR NicMultiQueue: ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isUpToDate, wantDiff := IsUpToDateWithDiff(tt.args.cr, tt.args.Server)
			assert.Equal(t, tt.wantIsUpToDate, isUpToDate)
			assert.Equal(t, tt.wantDiff, wantDiff)
		})
	}
}
