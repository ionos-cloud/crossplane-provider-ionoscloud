//nolint:testifylint
package flowlog

import (
	"context"
	"errors"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/nlb/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/flowlog"
	flowlogmock "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/mock/clients/flowlog/networkloadbalancer"
)

func TestNLBFlowLogObserve(t *testing.T) {

	obviouslyNotAFlowLog := struct{ v1alpha1.FlowLog }{}
	pstr := ionoscloud.PtrString
	tests := []struct {
		name    string
		mg      resource.Managed
		want    managed.ExternalObservation
		wantErr bool
		mock    func(ctx context.Context, client *flowlogmock.MockNLBFlowLog)
	}{
		{
			name:    "Wrong managed type",
			mg:      &obviouslyNotAFlowLog,
			want:    managed.ExternalObservation{},
			wantErr: true,
			mock:    func(ctx context.Context, client *flowlogmock.MockNLBFlowLog) {},
		},
		{
			name:    "FlowLog does not exist",
			mg:      &v1alpha1.FlowLog{},
			want:    managed.ExternalObservation{},
			wantErr: false,
			mock:    func(ctx context.Context, client *flowlogmock.MockNLBFlowLog) {},
		},
		{
			name: "FlowLog not found in ionoscloud",
			mg: func() *v1alpha1.FlowLog {
				mg := &v1alpha1.FlowLog{
					Spec: v1alpha1.FlowLogSpec{
						ForProvider: v1alpha1.FlowLogParameters{
							DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: "nlb-dc-id"},
							NLBCfg:        v1alpha1.NetworkLoadBalancerConfig{NetworkLoadBalancerID: "nlb-id"},
						},
					},
				}
				meta.SetExternalName(mg, "fl-id")
				return mg
			}(),
			want:    managed.ExternalObservation{},
			wantErr: false,
			mock: func(ctx context.Context, client *flowlogmock.MockNLBFlowLog) {
				client.EXPECT().
					GetFlowLogByID(ctx, "nlb-dc-id", "nlb-id", "fl-id").
					Return(ionoscloud.FlowLog{}, flowlog.ErrNotFound)
			},
		},
		{
			name: "Client error",
			mg: func() *v1alpha1.FlowLog {
				mg := &v1alpha1.FlowLog{
					Spec: v1alpha1.FlowLogSpec{
						ForProvider: v1alpha1.FlowLogParameters{
							DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: "nlb-dc-id"},
							NLBCfg:        v1alpha1.NetworkLoadBalancerConfig{NetworkLoadBalancerID: "nlb-id"},
						},
					},
				}
				meta.SetExternalName(mg, "fl-id")
				return mg
			}(),
			want:    managed.ExternalObservation{},
			wantErr: true,
			mock: func(ctx context.Context, client *flowlogmock.MockNLBFlowLog) {
				client.EXPECT().
					GetFlowLogByID(ctx, "nlb-dc-id", "nlb-id", "fl-id").
					Return(ionoscloud.FlowLog{}, errors.New("internal client error"))
			},
		},
		{
			name: "FlowLog is up to date",
			mg: func() *v1alpha1.FlowLog {
				mg := &v1alpha1.FlowLog{
					Spec: v1alpha1.FlowLogSpec{
						ForProvider: v1alpha1.FlowLogParameters{
							DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: "nlb-dc-id"},
							NLBCfg:        v1alpha1.NetworkLoadBalancerConfig{NetworkLoadBalancerID: "nlb-id"},
							Name:          "fl-name",
							Action:        "fl-action",
							Direction:     "fl-direction",
							Bucket:        "fl-bucket",
						},
					},
				}
				meta.SetExternalName(mg, "fl-id")
				return mg
			}(),
			want: managed.ExternalObservation{
				ResourceExists:          true,
				ResourceUpToDate:        true,
				ResourceLateInitialized: false,
				ConnectionDetails:       managed.ConnectionDetails{},
			},
			wantErr: false,
			mock: func(ctx context.Context, client *flowlogmock.MockNLBFlowLog) {
				client.EXPECT().
					GetFlowLogByID(ctx, "nlb-dc-id", "nlb-id", "fl-id").
					Return(ionoscloud.FlowLog{
						Properties: &ionoscloud.FlowLogProperties{
							Name:      pstr("fl-name"),
							Action:    pstr("fl-action"),
							Direction: pstr("fl-direction"),
							Bucket:    pstr("fl-bucket"),
						},
						Metadata: &ionoscloud.DatacenterElementMetadata{State: pstr(compute.AVAILABLE)},
					}, nil)

			},
		},
		{
			name: "FlowLog requires update",
			mg: func() *v1alpha1.FlowLog {
				mg := &v1alpha1.FlowLog{
					Spec: v1alpha1.FlowLogSpec{
						ForProvider: v1alpha1.FlowLogParameters{
							DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: "nlb-dc-id"},
							NLBCfg:        v1alpha1.NetworkLoadBalancerConfig{NetworkLoadBalancerID: "nlb-id"},
							Name:          "fl-name-updated",
							Action:        "fl-action",
							Direction:     "fl-direction",
							Bucket:        "fl-bucket",
						},
					},
				}
				meta.SetExternalName(mg, "fl-id")
				return mg
			}(),
			want: managed.ExternalObservation{
				ResourceExists:          true,
				ResourceUpToDate:        false,
				ResourceLateInitialized: false,
				ConnectionDetails:       managed.ConnectionDetails{},
			},
			wantErr: false,
			mock: func(ctx context.Context, client *flowlogmock.MockNLBFlowLog) {
				client.EXPECT().
					GetFlowLogByID(ctx, "nlb-dc-id", "nlb-id", "fl-id").
					Return(ionoscloud.FlowLog{
						Properties: &ionoscloud.FlowLogProperties{
							Name:      pstr("fl-name"),
							Action:    pstr("fl-action"),
							Direction: pstr("fl-direction"),
							Bucket:    pstr("fl-bucket"),
						},
						Metadata: &ionoscloud.DatacenterElementMetadata{State: pstr(compute.AVAILABLE)},
					}, nil)

			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			client := flowlogmock.NewMockNLBFlowLog(ctrl)
			tt.mock(ctx, client)
			external := externalFlowLog{
				service: client,
				log:     logging.NewNopLogger(),
			}
			got, err := external.Observe(ctx, tt.mg)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNLBFlowLogCreate(t *testing.T) {

	obviouslyNotAFlowLog := struct{ v1alpha1.FlowLog }{}
	pstr := ionoscloud.PtrString

	testCreateInput := func() *v1alpha1.FlowLog {
		mg := &v1alpha1.FlowLog{
			Spec: v1alpha1.FlowLogSpec{
				ForProvider: v1alpha1.FlowLogParameters{
					DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: "nlb-dc-id"},
					NLBCfg:        v1alpha1.NetworkLoadBalancerConfig{NetworkLoadBalancerID: "nlb-id"},
					Name:          "fl-name",
					Action:        "fl-action",
					Direction:     "fl-direction",
					Bucket:        "fl-bucket",
				},
			},
		}
		return mg
	}

	tests := []struct {
		name             string
		mg               resource.Managed
		want             managed.ExternalCreation
		wantErr          bool
		wantExternalName string
		mock             func(ctx context.Context, client *flowlogmock.MockNLBFlowLog)
	}{
		{
			name:             "Wrong managed type",
			mg:               &obviouslyNotAFlowLog,
			want:             managed.ExternalCreation{},
			wantErr:          true,
			wantExternalName: "",
			mock:             func(ctx context.Context, client *flowlogmock.MockNLBFlowLog) {},
		},
		{
			name: "FlowLog already created",
			mg: func() *v1alpha1.FlowLog {
				mg := &v1alpha1.FlowLog{
					Spec: v1alpha1.FlowLogSpec{
						ForProvider: v1alpha1.FlowLogParameters{},
					},
				}
				meta.SetExternalName(mg, "fl-id")
				return mg
			}(),
			want:             managed.ExternalCreation{},
			wantErr:          false,
			wantExternalName: "fl-id",
			mock:             func(ctx context.Context, client *flowlogmock.MockNLBFlowLog) {},
		},
		{
			name: "FlowLog is being provisioned",
			mg: func() *v1alpha1.FlowLog {
				mg := &v1alpha1.FlowLog{
					Status: v1alpha1.FlowLogStatus{
						AtProvider: v1alpha1.FlowLogObservation{
							State: compute.BUSY,
						},
					},
				}
				return mg
			}(),
			want:             managed.ExternalCreation{},
			wantErr:          false,
			wantExternalName: "",
			mock:             func(ctx context.Context, client *flowlogmock.MockNLBFlowLog) {},
		},
		{
			name: "Imported duplicate name flow log",
			mg: func() *v1alpha1.FlowLog {
				mg := &v1alpha1.FlowLog{
					Spec: v1alpha1.FlowLogSpec{
						ForProvider: v1alpha1.FlowLogParameters{
							DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: "nlb-dc-id"},
							NLBCfg:        v1alpha1.NetworkLoadBalancerConfig{NetworkLoadBalancerID: "nlb-id"},
							Name:          "fl-name",
						},
					},
				}
				return mg
			}(),
			want:             managed.ExternalCreation{},
			wantErr:          false,
			wantExternalName: "fl-id",
			mock: func(ctx context.Context, client *flowlogmock.MockNLBFlowLog) {
				client.EXPECT().
					CheckDuplicateFlowLog(ctx, "nlb-dc-id", "nlb-id", "fl-name").
					Return("fl-id", nil)
			},
		},
		{
			name: "Failed to perform duplicate flow log check",
			mg: func() *v1alpha1.FlowLog {
				mg := &v1alpha1.FlowLog{
					Spec: v1alpha1.FlowLogSpec{
						ForProvider: v1alpha1.FlowLogParameters{
							DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: "nlb-dc-id"},
							NLBCfg:        v1alpha1.NetworkLoadBalancerConfig{NetworkLoadBalancerID: "nlb-id"},
							Name:          "fl-name",
						},
					},
				}
				return mg
			}(),
			want:             managed.ExternalCreation{},
			wantErr:          true,
			wantExternalName: "",
			mock: func(ctx context.Context, client *flowlogmock.MockNLBFlowLog) {
				client.EXPECT().
					CheckDuplicateFlowLog(ctx, "nlb-dc-id", "nlb-id", "fl-name").
					Return("", errors.New("duplicate flow log check error"))

			},
		},
		{
			name:             "Create new flow log",
			mg:               testCreateInput(),
			want:             managed.ExternalCreation{},
			wantErr:          false,
			wantExternalName: "new-fl-id",
			mock: func(ctx context.Context, client *flowlogmock.MockNLBFlowLog) {
				client.EXPECT().
					CheckDuplicateFlowLog(ctx, "nlb-dc-id", "nlb-id", "fl-name").
					Return("", nil)
				createInput := flowlog.GenerateCreateInput(testCreateInput())
				client.EXPECT().
					CreateFlowLog(ctx, "nlb-dc-id", "nlb-id", createInput).
					Return(ionoscloud.FlowLog{Id: pstr("new-fl-id")}, nil)
			},
		},
		{
			name:             "Failed to create new flow log",
			mg:               testCreateInput(),
			want:             managed.ExternalCreation{},
			wantErr:          true,
			wantExternalName: "",
			mock: func(ctx context.Context, client *flowlogmock.MockNLBFlowLog) {
				client.EXPECT().
					CheckDuplicateFlowLog(ctx, "nlb-dc-id", "nlb-id", "fl-name").
					Return("", nil)
				createInput := flowlog.GenerateCreateInput(testCreateInput())
				client.EXPECT().
					CreateFlowLog(ctx, "nlb-dc-id", "nlb-id", createInput).
					Return(ionoscloud.FlowLog{}, errors.New("failed to create"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			client := flowlogmock.NewMockNLBFlowLog(ctrl)
			tt.mock(ctx, client)
			external := externalFlowLog{
				service:              client,
				log:                  logging.NewNopLogger(),
				isUniqueNamesEnabled: true,
			}
			got, err := external.Create(ctx, tt.mg)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
			gotExternalName := meta.GetExternalName(tt.mg)
			assert.Equal(t, tt.wantExternalName, gotExternalName)
		})
	}
}

func TestNLBFlowLogUpdate(t *testing.T) {

	obviouslyNotAFlowLog := struct{ v1alpha1.FlowLog }{}

	testUpdateInput := func() *v1alpha1.FlowLog {
		mg := &v1alpha1.FlowLog{
			Spec: v1alpha1.FlowLogSpec{
				ForProvider: v1alpha1.FlowLogParameters{
					DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: "nlb-dc-id"},
					NLBCfg:        v1alpha1.NetworkLoadBalancerConfig{NetworkLoadBalancerID: "nlb-id"},
					Name:          "fl-name-updated",
					Action:        "fl-action-updated",
					Direction:     "fl-direction",
					Bucket:        "fl-bucket",
				},
			},
			Status: v1alpha1.FlowLogStatus{
				AtProvider: v1alpha1.FlowLogObservation{
					FlowLogID: "fl-id",
				},
			},
		}
		return mg
	}

	tests := []struct {
		name    string
		mg      resource.Managed
		want    managed.ExternalUpdate
		wantErr bool
		mock    func(ctx context.Context, client *flowlogmock.MockNLBFlowLog)
	}{
		{
			name:    "Wrong managed type",
			mg:      &obviouslyNotAFlowLog,
			want:    managed.ExternalUpdate{},
			wantErr: true,
			mock:    func(ctx context.Context, client *flowlogmock.MockNLBFlowLog) {},
		},
		{
			name: "FlowLog is busy",
			mg: func() *v1alpha1.FlowLog {
				mg := &v1alpha1.FlowLog{
					Status: v1alpha1.FlowLogStatus{
						AtProvider: v1alpha1.FlowLogObservation{
							State: compute.BUSY,
						},
					},
				}
				return mg
			}(),
			want:    managed.ExternalUpdate{},
			wantErr: false,
			mock:    func(ctx context.Context, client *flowlogmock.MockNLBFlowLog) {},
		},
		{
			name:    "FlowLog update requested",
			mg:      testUpdateInput(),
			want:    managed.ExternalUpdate{},
			wantErr: false,
			mock: func(ctx context.Context, client *flowlogmock.MockNLBFlowLog) {
				updateInput := flowlog.GenerateUpdateInput(testUpdateInput())
				client.EXPECT().
					UpdateFlowLog(ctx, "nlb-dc-id", "nlb-id", "fl-id", updateInput).
					Return(ionoscloud.FlowLog{}, nil)
			},
		},
		{
			name:    "FowLog update failed",
			mg:      testUpdateInput(),
			want:    managed.ExternalUpdate{},
			wantErr: true,
			mock: func(ctx context.Context, client *flowlogmock.MockNLBFlowLog) {
				updateInput := flowlog.GenerateUpdateInput(testUpdateInput())
				client.EXPECT().
					UpdateFlowLog(ctx, "nlb-dc-id", "nlb-id", "fl-id", updateInput).
					Return(ionoscloud.FlowLog{}, errors.New("failed to update flow log"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			client := flowlogmock.NewMockNLBFlowLog(ctrl)
			tt.mock(ctx, client)
			external := externalFlowLog{
				service:              client,
				log:                  logging.NewNopLogger(),
				isUniqueNamesEnabled: true,
			}
			got, err := external.Update(ctx, tt.mg)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNLBFlowLogDelete(t *testing.T) {

	obviouslyNotAFlowLog := struct{ v1alpha1.FlowLog }{}

	testDeleteInput := func() *v1alpha1.FlowLog {
		mg := &v1alpha1.FlowLog{
			Spec: v1alpha1.FlowLogSpec{
				ForProvider: v1alpha1.FlowLogParameters{
					DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: "nlb-dc-id"},
					NLBCfg:        v1alpha1.NetworkLoadBalancerConfig{NetworkLoadBalancerID: "nlb-id"},
				},
			},
			Status: v1alpha1.FlowLogStatus{
				AtProvider: v1alpha1.FlowLogObservation{
					FlowLogID: "fl-id",
				},
			},
		}
		return mg
	}

	tests := []struct {
		name    string
		mg      resource.Managed
		wantErr bool
		mock    func(ctx context.Context, client *flowlogmock.MockNLBFlowLog)
	}{
		{
			name:    "Wrong managed type",
			mg:      &obviouslyNotAFlowLog,
			wantErr: true,
			mock:    func(ctx context.Context, client *flowlogmock.MockNLBFlowLog) {},
		},
		{
			name: "FlowLog already deleting",
			mg: func() *v1alpha1.FlowLog {
				mg := &v1alpha1.FlowLog{
					Status: v1alpha1.FlowLogStatus{
						AtProvider: v1alpha1.FlowLogObservation{
							State: compute.DESTROYING,
						},
					},
				}
				return mg
			}(),
			wantErr: false,
			mock:    func(ctx context.Context, client *flowlogmock.MockNLBFlowLog) {},
		},
		{
			name:    "FlowLog delete requested",
			mg:      testDeleteInput(),
			wantErr: false,
			mock: func(ctx context.Context, client *flowlogmock.MockNLBFlowLog) {
				client.EXPECT().
					DeleteFlowLog(ctx, "nlb-dc-id", "nlb-id", "fl-id").
					Return(nil)
			},
		},
		{
			name:    "FlowLog not found",
			mg:      testDeleteInput(),
			wantErr: false,
			mock: func(ctx context.Context, client *flowlogmock.MockNLBFlowLog) {
				client.EXPECT().
					DeleteFlowLog(ctx, "nlb-dc-id", "nlb-id", "fl-id").
					Return(flowlog.ErrNotFound)
			},
		},
		{
			name:    "FlowLog delete failed",
			mg:      testDeleteInput(),
			wantErr: true,
			mock: func(ctx context.Context, client *flowlogmock.MockNLBFlowLog) {
				client.EXPECT().
					DeleteFlowLog(ctx, "nlb-dc-id", "nlb-id", "fl-id").
					Return(errors.New("flow log delete failed"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			client := flowlogmock.NewMockNLBFlowLog(ctrl)
			tt.mock(ctx, client)
			external := externalFlowLog{
				service:              client,
				log:                  logging.NewNopLogger(),
				isUniqueNamesEnabled: true,
			}
			_, err := external.Delete(ctx, tt.mg)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
