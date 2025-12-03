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
			wantIsUpToDate: false,
			wantDiff:       "NicMultiQueue do not match the CR NicMultiQueue: false != true",
		},
		{
			name: "vmState matches RUNNING",
			args: args{
				cr: &v1alpha1.Server{
					Spec: v1alpha1.ServerSpec{
						ForProvider: v1alpha1.ServerParameters{
							Name:             "not empty",
							CPUFamily:        "super fast",
							AvailabilityZone: "AUTO",
							Cores:            4,
							RAM:              2048,
							VmState:          "RUNNING",
						},
					},
				},
				Server: ionoscloud.Server{Properties: &ionoscloud.ServerProperties{
					Name:             ionoscloud.ToPtr("not empty"),
					CpuFamily:        ionoscloud.ToPtr("super fast"),
					AvailabilityZone: ionoscloud.ToPtr("AUTO"),
					Cores:            ionoscloud.PtrInt32(4),
					Ram:              ionoscloud.PtrInt32(2048),
					VmState:          ionoscloud.ToPtr("RUNNING"),
				}},
			},
			wantIsUpToDate: true,
			wantDiff:       "Server is up-to-date",
		},
		{
			name: "vmState matches SHUTOFF",
			args: args{
				cr: &v1alpha1.Server{
					Spec: v1alpha1.ServerSpec{
						ForProvider: v1alpha1.ServerParameters{
							Name:             "not empty",
							CPUFamily:        "super fast",
							AvailabilityZone: "AUTO",
							Cores:            4,
							RAM:              2048,
							VmState:          "SHUTOFF",
						},
					},
				},
				Server: ionoscloud.Server{Properties: &ionoscloud.ServerProperties{
					Name:             ionoscloud.ToPtr("not empty"),
					CpuFamily:        ionoscloud.ToPtr("super fast"),
					AvailabilityZone: ionoscloud.ToPtr("AUTO"),
					Cores:            ionoscloud.PtrInt32(4),
					Ram:              ionoscloud.PtrInt32(2048),
					VmState:          ionoscloud.ToPtr("SHUTOFF"),
				}},
			},
			wantIsUpToDate: true,
			wantDiff:       "Server is up-to-date",
		},
		{
			name: "vmState differs - desired RUNNING but actual SHUTOFF",
			args: args{
				cr: &v1alpha1.Server{
					Spec: v1alpha1.ServerSpec{
						ForProvider: v1alpha1.ServerParameters{
							Name:             "not empty",
							CPUFamily:        "super fast",
							AvailabilityZone: "AUTO",
							Cores:            4,
							RAM:              2048,
							VmState:          "RUNNING",
						},
					},
				},
				Server: ionoscloud.Server{Properties: &ionoscloud.ServerProperties{
					Name:             ionoscloud.ToPtr("not empty"),
					CpuFamily:        ionoscloud.ToPtr("super fast"),
					AvailabilityZone: ionoscloud.ToPtr("AUTO"),
					Cores:            ionoscloud.PtrInt32(4),
					Ram:              ionoscloud.PtrInt32(2048),
					VmState:          ionoscloud.ToPtr("SHUTOFF"),
				}},
			},
			wantIsUpToDate: false,
			wantDiff:       "Server vmState does not match the CR vmState: SHUTOFF != RUNNING",
		},
		{
			name: "vmState differs - desired SHUTOFF but actual RUNNING",
			args: args{
				cr: &v1alpha1.Server{
					Spec: v1alpha1.ServerSpec{
						ForProvider: v1alpha1.ServerParameters{
							Name:             "not empty",
							CPUFamily:        "super fast",
							AvailabilityZone: "AUTO",
							Cores:            4,
							RAM:              2048,
							VmState:          "SHUTOFF",
						},
					},
				},
				Server: ionoscloud.Server{Properties: &ionoscloud.ServerProperties{
					Name:             ionoscloud.ToPtr("not empty"),
					CpuFamily:        ionoscloud.ToPtr("super fast"),
					AvailabilityZone: ionoscloud.ToPtr("AUTO"),
					Cores:            ionoscloud.PtrInt32(4),
					Ram:              ionoscloud.PtrInt32(2048),
					VmState:          ionoscloud.ToPtr("RUNNING"),
				}},
			},
			wantIsUpToDate: false,
			wantDiff:       "Server vmState does not match the CR vmState: RUNNING != SHUTOFF",
		},
		{
			name: "vmState not specified in CR - should be up to date",
			args: args{
				cr: &v1alpha1.Server{
					Spec: v1alpha1.ServerSpec{
						ForProvider: v1alpha1.ServerParameters{
							Name:             "not empty",
							CPUFamily:        "super fast",
							AvailabilityZone: "AUTO",
							Cores:            4,
							RAM:              2048,
						},
					},
				},
				Server: ionoscloud.Server{Properties: &ionoscloud.ServerProperties{
					Name:             ionoscloud.ToPtr("not empty"),
					CpuFamily:        ionoscloud.ToPtr("super fast"),
					AvailabilityZone: ionoscloud.ToPtr("AUTO"),
					Cores:            ionoscloud.PtrInt32(4),
					Ram:              ionoscloud.PtrInt32(2048),
					VmState:          ionoscloud.ToPtr("RUNNING"),
				}},
			},
			wantIsUpToDate: true,
			wantDiff:       "Server is up-to-date",
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

func TestIsCubeServerUpToDate(t *testing.T) {
	type args struct {
		cr     *v1alpha1.CubeServer
		Server ionoscloud.Server
	}
	tests := []struct {
		name           string
		args           args
		wantIsUpToDate bool
	}{
		{
			name: "both empty",
			args: args{
				cr:     nil,
				Server: ionoscloud.Server{},
			},
			wantIsUpToDate: true,
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
		},
		{
			name: "api response empty",
			args: args{
				cr: &v1alpha1.CubeServer{
					Spec: v1alpha1.CubeServerSpec{
						ForProvider: v1alpha1.CubeServerProperties{
							Name: "not empty",
						},
					},
				},
				Server: ionoscloud.Server{Properties: nil},
			},
			wantIsUpToDate: false,
		},
		{
			name: "all equal",
			args: args{
				cr: &v1alpha1.CubeServer{
					Spec: v1alpha1.CubeServerSpec{
						ForProvider: v1alpha1.CubeServerProperties{
							Name:             "not empty",
							AvailabilityZone: "AUTO",
							Template: v1alpha1.Template{
								TemplateID: "template-123",
							},
						},
					},
				},
				Server: ionoscloud.Server{
					Metadata: &ionoscloud.DatacenterElementMetadata{
						State: ionoscloud.ToPtr("AVAILABLE"),
					},
					Properties: &ionoscloud.ServerProperties{
						Name:             ionoscloud.ToPtr("not empty"),
						AvailabilityZone: ionoscloud.ToPtr("AUTO"),
						TemplateUuid:     ionoscloud.ToPtr("template-123"),
						Type:             ionoscloud.ToPtr("CUBE"),
					},
				},
			},
			wantIsUpToDate: true,
		},
		{
			name: "name different",
			args: args{
				cr: &v1alpha1.CubeServer{
					Spec: v1alpha1.CubeServerSpec{
						ForProvider: v1alpha1.CubeServerProperties{
							Name:             "not empty",
							AvailabilityZone: "AUTO",
							Template: v1alpha1.Template{
								TemplateID: "template-123",
							},
						},
					},
				},
				Server: ionoscloud.Server{
					Metadata: &ionoscloud.DatacenterElementMetadata{
						State: ionoscloud.ToPtr("AVAILABLE"),
					},
					Properties: &ionoscloud.ServerProperties{
						Name:             ionoscloud.ToPtr("different name"),
						AvailabilityZone: ionoscloud.ToPtr("AUTO"),
						TemplateUuid:     ionoscloud.ToPtr("template-123"),
						Type:             ionoscloud.ToPtr("CUBE"),
					},
				},
			},
			wantIsUpToDate: false,
		},
		{
			name: "availability zone different",
			args: args{
				cr: &v1alpha1.CubeServer{
					Spec: v1alpha1.CubeServerSpec{
						ForProvider: v1alpha1.CubeServerProperties{
							Name:             "not empty",
							AvailabilityZone: "AUTO",
							Template: v1alpha1.Template{
								TemplateID: "template-123",
							},
						},
					},
				},
				Server: ionoscloud.Server{
					Metadata: &ionoscloud.DatacenterElementMetadata{
						State: ionoscloud.ToPtr("AVAILABLE"),
					},
					Properties: &ionoscloud.ServerProperties{
						Name:             ionoscloud.ToPtr("not empty"),
						AvailabilityZone: ionoscloud.ToPtr("ZONE_1"),
						TemplateUuid:     ionoscloud.ToPtr("template-123"),
						Type:             ionoscloud.ToPtr("CUBE"),
					},
				},
			},
			wantIsUpToDate: false,
		},
		{
			name: "server with nil metadata - should not panic",
			args: args{
				cr: &v1alpha1.CubeServer{
					Spec: v1alpha1.CubeServerSpec{
						ForProvider: v1alpha1.CubeServerProperties{
							Name:             "not empty",
							AvailabilityZone: "AUTO",
							Template: v1alpha1.Template{
								TemplateID: "template-123",
							},
						},
					},
				},
				Server: ionoscloud.Server{
					Metadata: nil,
					Properties: &ionoscloud.ServerProperties{
						Name:             ionoscloud.ToPtr("not empty"),
						AvailabilityZone: ionoscloud.ToPtr("AUTO"),
						TemplateUuid:     ionoscloud.ToPtr("template-123"),
						Type:             ionoscloud.ToPtr("CUBE"),
					},
				},
			},
			wantIsUpToDate: true,
		},
		{
			name: "vmState matches RUNNING",
			args: args{
				cr: &v1alpha1.CubeServer{
					Spec: v1alpha1.CubeServerSpec{
						ForProvider: v1alpha1.CubeServerProperties{
							Name:             "not empty",
							AvailabilityZone: "AUTO",
							Template: v1alpha1.Template{
								TemplateID: "template-123",
							},
							VmState: "RUNNING",
						},
					},
				},
				Server: ionoscloud.Server{
					Metadata: &ionoscloud.DatacenterElementMetadata{
						State: ionoscloud.ToPtr("AVAILABLE"),
					},
					Properties: &ionoscloud.ServerProperties{
						Name:             ionoscloud.ToPtr("not empty"),
						AvailabilityZone: ionoscloud.ToPtr("AUTO"),
						TemplateUuid:     ionoscloud.ToPtr("template-123"),
						Type:             ionoscloud.ToPtr("CUBE"),
						VmState:          ionoscloud.ToPtr("RUNNING"),
					},
				},
			},
			wantIsUpToDate: true,
		},
		{
			name: "vmState matches SUSPENDED",
			args: args{
				cr: &v1alpha1.CubeServer{
					Spec: v1alpha1.CubeServerSpec{
						ForProvider: v1alpha1.CubeServerProperties{
							Name:             "not empty",
							AvailabilityZone: "AUTO",
							Template: v1alpha1.Template{
								TemplateID: "template-123",
							},
							VmState: "SUSPENDED",
						},
					},
				},
				Server: ionoscloud.Server{
					Metadata: &ionoscloud.DatacenterElementMetadata{
						State: ionoscloud.ToPtr("AVAILABLE"),
					},
					Properties: &ionoscloud.ServerProperties{
						Name:             ionoscloud.ToPtr("not empty"),
						AvailabilityZone: ionoscloud.ToPtr("AUTO"),
						TemplateUuid:     ionoscloud.ToPtr("template-123"),
						Type:             ionoscloud.ToPtr("CUBE"),
						VmState:          ionoscloud.ToPtr("SUSPENDED"),
					},
				},
			},
			wantIsUpToDate: true,
		},
		{
			name: "vmState differs - desired RUNNING but actual SUSPENDED",
			args: args{
				cr: &v1alpha1.CubeServer{
					Spec: v1alpha1.CubeServerSpec{
						ForProvider: v1alpha1.CubeServerProperties{
							Name:             "not empty",
							AvailabilityZone: "AUTO",
							Template: v1alpha1.Template{
								TemplateID: "template-123",
							},
							VmState: "RUNNING",
						},
					},
				},
				Server: ionoscloud.Server{
					Metadata: &ionoscloud.DatacenterElementMetadata{
						State: ionoscloud.ToPtr("AVAILABLE"),
					},
					Properties: &ionoscloud.ServerProperties{
						Name:             ionoscloud.ToPtr("not empty"),
						AvailabilityZone: ionoscloud.ToPtr("AUTO"),
						TemplateUuid:     ionoscloud.ToPtr("template-123"),
						Type:             ionoscloud.ToPtr("CUBE"),
						VmState:          ionoscloud.ToPtr("SUSPENDED"),
					},
				},
			},
			wantIsUpToDate: false,
		},
		{
			name: "vmState differs - desired SUSPENDED but actual RUNNING",
			args: args{
				cr: &v1alpha1.CubeServer{
					Spec: v1alpha1.CubeServerSpec{
						ForProvider: v1alpha1.CubeServerProperties{
							Name:             "not empty",
							AvailabilityZone: "AUTO",
							Template: v1alpha1.Template{
								TemplateID: "template-123",
							},
							VmState: "SUSPENDED",
						},
					},
				},
				Server: ionoscloud.Server{
					Metadata: &ionoscloud.DatacenterElementMetadata{
						State: ionoscloud.ToPtr("AVAILABLE"),
					},
					Properties: &ionoscloud.ServerProperties{
						Name:             ionoscloud.ToPtr("not empty"),
						AvailabilityZone: ionoscloud.ToPtr("AUTO"),
						TemplateUuid:     ionoscloud.ToPtr("template-123"),
						Type:             ionoscloud.ToPtr("CUBE"),
						VmState:          ionoscloud.ToPtr("RUNNING"),
					},
				},
			},
			wantIsUpToDate: false,
		},
		{
			name: "vmState not specified in CR - should be up to date",
			args: args{
				cr: &v1alpha1.CubeServer{
					Spec: v1alpha1.CubeServerSpec{
						ForProvider: v1alpha1.CubeServerProperties{
							Name:             "not empty",
							AvailabilityZone: "AUTO",
							Template: v1alpha1.Template{
								TemplateID: "template-123",
							},
						},
					},
				},
				Server: ionoscloud.Server{
					Metadata: &ionoscloud.DatacenterElementMetadata{
						State: ionoscloud.ToPtr("AVAILABLE"),
					},
					Properties: &ionoscloud.ServerProperties{
						Name:             ionoscloud.ToPtr("not empty"),
						AvailabilityZone: ionoscloud.ToPtr("AUTO"),
						TemplateUuid:     ionoscloud.ToPtr("template-123"),
						Type:             ionoscloud.ToPtr("CUBE"),
						VmState:          ionoscloud.ToPtr("RUNNING"),
					},
				},
			},
			wantIsUpToDate: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isUpToDate := IsCubeServerUpToDate(tt.args.cr, tt.args.Server)
			assert.Equal(t, tt.wantIsUpToDate, isUpToDate)
		})
	}
}
