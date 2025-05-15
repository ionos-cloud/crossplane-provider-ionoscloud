//nolint:testifylint
package forwardingrule

import (
	"cmp"
	"context"
	"errors"
	"reflect"
	"slices"
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
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/nlb/forwardingrule"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/mock/clients/compute/ipblock"
	forwardingrulemock "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/mock/clients/nlb/forwardingrule"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

func TestNLBForwardingRuleObserve(t *testing.T) {

	notAForwardingRule := struct{ v1alpha1.ForwardingRule }{}
	tests := []struct {
		name    string
		mg      resource.Managed
		want    managed.ExternalObservation
		wantErr bool
		mock    func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient)
	}{
		{
			name:    "Wrong managed type",
			mg:      &notAForwardingRule,
			want:    managed.ExternalObservation{},
			wantErr: true,
			mock: func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient) {
			},
		},
		{
			name:    "ForwardingRule does not exist",
			mg:      &v1alpha1.ForwardingRule{},
			want:    managed.ExternalObservation{},
			wantErr: false,
			mock: func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient) {
			},
		},
		{
			name: "ForwardingRule not found in ionoscloud",
			mg: func() *v1alpha1.ForwardingRule {
				mg := &v1alpha1.ForwardingRule{
					Spec: v1alpha1.ForwardingRuleSpec{
						ForProvider: v1alpha1.ForwardingRuleParameters{
							DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: "nlb-dc-id"},
							NLBCfg:        v1alpha1.NetworkLoadBalancerConfig{NetworkLoadBalancerID: "nlb-id"},
						},
					},
				}
				meta.SetExternalName(mg, "fr-id")
				return mg
			}(),
			want:    managed.ExternalObservation{},
			wantErr: false,
			mock: func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient) {
				fwRuleClient.EXPECT().
					GetForwardingRuleByID(ctx, "nlb-dc-id", "nlb-id", "fr-id").
					Return(ionoscloud.NetworkLoadBalancerForwardingRule{}, forwardingrule.ErrNotFound)
			},
		},
		{
			name: "Client error - get forwarding rule by id",
			mg: func() *v1alpha1.ForwardingRule {
				mg := &v1alpha1.ForwardingRule{
					Spec: v1alpha1.ForwardingRuleSpec{
						ForProvider: v1alpha1.ForwardingRuleParameters{
							DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: "nlb-dc-id"},
							NLBCfg:        v1alpha1.NetworkLoadBalancerConfig{NetworkLoadBalancerID: "nlb-id"},
						},
					},
				}
				meta.SetExternalName(mg, "fr-id")
				return mg
			}(),
			want:    managed.ExternalObservation{},
			wantErr: true,
			mock: func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient) {
				fwRuleClient.EXPECT().
					GetForwardingRuleByID(ctx, "nlb-dc-id", "nlb-id", "fr-id").
					Return(ionoscloud.NetworkLoadBalancerForwardingRule{}, errors.New("internal client error"))
			},
		},
		{
			name: "Client error - get listener ip",
			mg: func() *v1alpha1.ForwardingRule {
				mg := &v1alpha1.ForwardingRule{
					Spec: v1alpha1.ForwardingRuleSpec{
						ForProvider: v1alpha1.ForwardingRuleParameters{
							DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: "nlb-dc-id"},
							NLBCfg:        v1alpha1.NetworkLoadBalancerConfig{NetworkLoadBalancerID: "nlb-id"},
							ListenerIP:    v1alpha1.IPConfig{IPBlockConfig: v1alpha1.IPBlockConfig{IPBlockID: "fr-ip-id"}, Index: 0},
						},
					},
				}
				meta.SetExternalName(mg, "fr-id")
				return mg
			}(),
			want:    managed.ExternalObservation{},
			wantErr: true,
			mock: func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient) {
				fwRuleClient.EXPECT().
					GetForwardingRuleByID(ctx, "nlb-dc-id", "nlb-id", "fr-id").
					Return(ionoscloud.NetworkLoadBalancerForwardingRule{}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "fr-ip-id", 0).
					Return(nil, errors.New("internal ip block client error"))
			},
		},
		{
			name: "Client error - get target ip",
			mg: func() *v1alpha1.ForwardingRule {
				mg := &v1alpha1.ForwardingRule{
					Spec: v1alpha1.ForwardingRuleSpec{
						ForProvider: v1alpha1.ForwardingRuleParameters{
							DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: "nlb-dc-id"},
							NLBCfg:        v1alpha1.NetworkLoadBalancerConfig{NetworkLoadBalancerID: "nlb-id"},
							ListenerIP:    v1alpha1.IPConfig{IPBlockConfig: v1alpha1.IPBlockConfig{IPBlockID: "fr-ip-id"}, Index: 0},
							Targets: []v1alpha1.ForwardingRuleTarget{
								{
									IPCfg:         v1alpha1.IPConfig{IPBlockConfig: v1alpha1.IPBlockConfig{IPBlockID: "frt-ip-id"}, Index: 1},
									Port:          888,
									Weight:        15,
									ProxyProtocol: "v1",
								},
							},
						},
					},
				}
				meta.SetExternalName(mg, "fr-id")
				return mg
			}(),
			want:    managed.ExternalObservation{},
			wantErr: true,
			mock: func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient) {
				fwRuleClient.EXPECT().
					GetForwardingRuleByID(ctx, "nlb-dc-id", "nlb-id", "fr-id").
					Return(ionoscloud.NetworkLoadBalancerForwardingRule{}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "fr-ip-id", 0).
					Return([]string{"some-ip"}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "frt-ip-id", 1).
					Return(nil, errors.New("internal ip block client error"))
			},
		},
		{
			name: "ForwardingRule is up to date",
			mg: func() *v1alpha1.ForwardingRule {
				mg := &v1alpha1.ForwardingRule{
					Spec: v1alpha1.ForwardingRuleSpec{
						ForProvider: v1alpha1.ForwardingRuleParameters{
							DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: "nlb-dc-id"},
							NLBCfg:        v1alpha1.NetworkLoadBalancerConfig{NetworkLoadBalancerID: "nlb-id"},
							Name:          "fr-name",
							ListenerIP:    v1alpha1.IPConfig{IPBlockConfig: v1alpha1.IPBlockConfig{IPBlockID: "fr-ip-id"}, Index: 0},
							ListenerPort:  1000,
							Protocol:      "TCP",
							Algorithm:     "ROUND_ROBIN",
							HealthCheck:   defaultFwRuleHealthCheckCR,
							Targets: []v1alpha1.ForwardingRuleTarget{
								{
									IPCfg:         v1alpha1.IPConfig{IP: "192.100.100.1"},
									Port:          888,
									Weight:        10,
									ProxyProtocol: "v1",
									HealthCheck:   defaultFwRuleTargetHealthCheckCR,
								}, {
									IPCfg:         v1alpha1.IPConfig{IPBlockConfig: v1alpha1.IPBlockConfig{IPBlockID: "frt-ip-id"}, Index: 1},
									Port:          888,
									Weight:        15,
									ProxyProtocol: "v1",
									HealthCheck:   defaultFwRuleTargetHealthCheckCR,
								},
							},
						},
					},
				}
				meta.SetExternalName(mg, "fr-id")
				return mg
			}(),
			want: managed.ExternalObservation{
				ResourceExists:          true,
				ResourceUpToDate:        true,
				ResourceLateInitialized: false,
				ConnectionDetails:       managed.ConnectionDetails{},
			},
			wantErr: false,
			mock: func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient) {
				fwRuleClient.EXPECT().
					GetForwardingRuleByID(ctx, "nlb-dc-id", "nlb-id", "fr-id").
					Return(ionoscloud.NetworkLoadBalancerForwardingRule{
						Properties: &ionoscloud.NetworkLoadBalancerForwardingRuleProperties{
							Name:         pstr("fr-name"),
							ListenerIp:   pstr("10.20.30.40"),
							ListenerPort: pi32(1000),
							Protocol:     pstr("TCP"),
							Algorithm:    pstr("ROUND_ROBIN"),
							HealthCheck:  &defaultFwRuleHealthCheck,
							Targets: &[]ionoscloud.NetworkLoadBalancerForwardingRuleTarget{
								{
									Ip:            pstr("192.100.100.1"),
									Port:          pi32(888),
									Weight:        pi32(10),
									ProxyProtocol: pstr("v1"),
									HealthCheck:   &defaultFwRuleTargetHealthCheck,
								}, {
									Ip:            pstr("192.100.100.2"),
									Port:          pi32(888),
									Weight:        pi32(15),
									ProxyProtocol: pstr("v1"),
									HealthCheck:   &defaultFwRuleTargetHealthCheck,
								},
							},
						},
						Metadata: &ionoscloud.DatacenterElementMetadata{State: pstr(compute.AVAILABLE)},
					}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "fr-ip-id", 0).
					Return([]string{"10.20.30.40"}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "frt-ip-id", 1).
					Return([]string{"192.100.100.2"}, nil)

			},
		},
		{
			name: "ForwardingRule requires update",
			mg: func() *v1alpha1.ForwardingRule {
				mg := &v1alpha1.ForwardingRule{
					Spec: v1alpha1.ForwardingRuleSpec{
						ForProvider: v1alpha1.ForwardingRuleParameters{
							DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: "nlb-dc-id"},
							NLBCfg:        v1alpha1.NetworkLoadBalancerConfig{NetworkLoadBalancerID: "nlb-id"},
							Name:          "fr-name",
							ListenerIP:    v1alpha1.IPConfig{IPBlockConfig: v1alpha1.IPBlockConfig{IPBlockID: "fr-ip-id"}, Index: 0},
							ListenerPort:  1000,
							Protocol:      "TCP",
							Algorithm:     "ROUND_ROBIN",
							HealthCheck:   defaultFwRuleHealthCheckCR,
							Targets: []v1alpha1.ForwardingRuleTarget{
								{
									IPCfg:         v1alpha1.IPConfig{IP: "192.100.100.1"},
									Port:          888,
									Weight:        10,
									ProxyProtocol: "v1",
									HealthCheck:   defaultFwRuleTargetHealthCheckCR,
								}, {
									IPCfg:         v1alpha1.IPConfig{IPBlockConfig: v1alpha1.IPBlockConfig{IPBlockID: "frt-ip-id"}, Index: 2},
									Port:          888,
									Weight:        15,
									ProxyProtocol: "v1",
									HealthCheck:   defaultFwRuleTargetHealthCheckCR,
								},
							},
						},
					},
				}
				meta.SetExternalName(mg, "fr-id")
				return mg
			}(),
			want: managed.ExternalObservation{
				ResourceExists:          true,
				ResourceUpToDate:        false,
				ResourceLateInitialized: false,
				ConnectionDetails:       managed.ConnectionDetails{},
			},
			wantErr: false,
			mock: func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient) {
				fwRuleClient.EXPECT().
					GetForwardingRuleByID(ctx, "nlb-dc-id", "nlb-id", "fr-id").
					Return(ionoscloud.NetworkLoadBalancerForwardingRule{
						Properties: &ionoscloud.NetworkLoadBalancerForwardingRuleProperties{
							Name:         pstr("fr-name"),
							ListenerIp:   pstr("10.20.30.40"),
							ListenerPort: pi32(1000),
							Protocol:     pstr("TCP"),
							Algorithm:    pstr("ROUND_ROBIN"),
							HealthCheck:  &defaultFwRuleHealthCheck,
							Targets: &[]ionoscloud.NetworkLoadBalancerForwardingRuleTarget{
								{
									Ip:            pstr("192.100.100.1"),
									Port:          pi32(888),
									Weight:        pi32(10),
									ProxyProtocol: pstr("v1"),
									HealthCheck:   &defaultFwRuleTargetHealthCheck,
								}, {
									Ip:            pstr("192.100.100.222"),
									Port:          pi32(888),
									Weight:        pi32(15),
									ProxyProtocol: pstr("v1"),
									HealthCheck:   &defaultFwRuleTargetHealthCheck,
								},
							},
						},
						Metadata: &ionoscloud.DatacenterElementMetadata{State: pstr(compute.AVAILABLE)},
					}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "fr-ip-id", 0).
					Return([]string{"10.20.30.40"}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "frt-ip-id", 2).
					Return([]string{"192.100.100.333"}, nil)

			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			fwRuleClient := forwardingrulemock.NewMockClient(ctrl)
			ipBlockClient := ipblock.NewMockClient(ctrl)
			tt.mock(ctx, fwRuleClient, ipBlockClient)
			external := externalForwardingRule{
				service:        fwRuleClient,
				ipBlockService: ipBlockClient,
				log:            logging.NewNopLogger(),
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

func TestNLBForwardingRuleCreate(t *testing.T) {
	testCreateInput := func() *v1alpha1.ForwardingRule {
		mg := &v1alpha1.ForwardingRule{
			Spec: v1alpha1.ForwardingRuleSpec{
				ForProvider: v1alpha1.ForwardingRuleParameters{
					DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: "nlb-dc-id"},
					NLBCfg:        v1alpha1.NetworkLoadBalancerConfig{NetworkLoadBalancerID: "nlb-id"},
					Name:          "fr-name",
					ListenerIP:    v1alpha1.IPConfig{IPBlockConfig: v1alpha1.IPBlockConfig{IPBlockID: "fr-ip-id"}, Index: 0},
					ListenerPort:  1000,
					Protocol:      "TCP",
					Algorithm:     "ROUND_ROBIN",
					HealthCheck:   defaultFwRuleHealthCheckCR,
					Targets: []v1alpha1.ForwardingRuleTarget{
						{
							IPCfg:         v1alpha1.IPConfig{IP: "192.100.100.1"},
							Port:          888,
							Weight:        10,
							ProxyProtocol: "v1",
							HealthCheck:   defaultFwRuleTargetHealthCheckCR,
						}, {
							IPCfg:         v1alpha1.IPConfig{IPBlockConfig: v1alpha1.IPBlockConfig{IPBlockID: "frt-ip-id"}, Index: 1},
							Port:          888,
							Weight:        15,
							ProxyProtocol: "v1",
							HealthCheck:   defaultFwRuleTargetHealthCheckCR,
						},
					},
				},
			},
		}
		return mg
	}
	notAForwardingRule := struct{ v1alpha1.ForwardingRule }{}
	tests := []struct {
		name             string
		mg               resource.Managed
		want             managed.ExternalCreation
		wantErr          bool
		wantExternalName string
		mock             func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient)
	}{
		{
			name:             "Wrong managed type",
			mg:               &notAForwardingRule,
			want:             managed.ExternalCreation{},
			wantErr:          true,
			wantExternalName: "",
			mock: func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient) {
			},
		},
		{
			name: "ForwardingRule already created",
			mg: func() *v1alpha1.ForwardingRule {
				mg := &v1alpha1.ForwardingRule{
					Spec: v1alpha1.ForwardingRuleSpec{ForProvider: v1alpha1.ForwardingRuleParameters{}},
				}
				meta.SetExternalName(mg, "fr-id")
				return mg
			}(),
			want:             managed.ExternalCreation{},
			wantErr:          false,
			wantExternalName: "fr-id",
			mock: func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient) {
			},
		},
		{
			name: "ForwardingRule is being provisioned",
			mg: func() *v1alpha1.ForwardingRule {
				mg := &v1alpha1.ForwardingRule{
					Spec:   v1alpha1.ForwardingRuleSpec{ForProvider: v1alpha1.ForwardingRuleParameters{}},
					Status: v1alpha1.ForwardingRuleStatus{AtProvider: v1alpha1.ForwardingRuleObservation{State: compute.BUSY}},
				}
				return mg
			}(),
			want:             managed.ExternalCreation{},
			wantErr:          false,
			wantExternalName: "",
			mock: func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient) {
			},
		},
		{
			name: "Imported duplicate name forwarding rule",
			mg: func() *v1alpha1.ForwardingRule {
				mg := &v1alpha1.ForwardingRule{
					Spec: v1alpha1.ForwardingRuleSpec{
						ForProvider: v1alpha1.ForwardingRuleParameters{
							DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: "nlb-dc-id"},
							NLBCfg:        v1alpha1.NetworkLoadBalancerConfig{NetworkLoadBalancerID: "nlb-id"},
							Name:          "fr-name",
						},
					},
				}
				return mg
			}(),
			want:             managed.ExternalCreation{},
			wantErr:          false,
			wantExternalName: "fr-id",
			mock: func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient) {
				fwRuleClient.EXPECT().
					CheckDuplicateForwardingRule(ctx, "nlb-dc-id", "nlb-id", "fr-name").
					Return("fr-id", nil)
			},
		},
		{
			name: "Failed to perform duplicate forwarding rule check",
			mg: func() *v1alpha1.ForwardingRule {
				mg := &v1alpha1.ForwardingRule{
					Spec: v1alpha1.ForwardingRuleSpec{
						ForProvider: v1alpha1.ForwardingRuleParameters{
							DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: "nlb-dc-id"},
							NLBCfg:        v1alpha1.NetworkLoadBalancerConfig{NetworkLoadBalancerID: "nlb-id"},
							Name:          "fr-name",
						},
					},
				}
				return mg
			}(),
			want:             managed.ExternalCreation{},
			wantErr:          true,
			wantExternalName: "",
			mock: func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient) {
				fwRuleClient.EXPECT().
					CheckDuplicateForwardingRule(ctx, "nlb-dc-id", "nlb-id", "fr-name").
					Return("", errors.New("duplicate forwarding rule check error"))
			},
		},
		{
			name:             "Create new forwarding rule",
			mg:               testCreateInput(),
			want:             managed.ExternalCreation{},
			wantErr:          false,
			wantExternalName: "new-fr-id",
			mock: func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient) {
				fwRuleClient.EXPECT().
					CheckDuplicateForwardingRule(ctx, "nlb-dc-id", "nlb-id", "fr-name").
					Return("", nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "fr-ip-id", 0).
					Return([]string{"10.20.30.40"}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "frt-ip-id", 1).
					Return([]string{"192.100.100.2"}, nil)
				crInput := testCreateInput()
				targetIps := map[string]v1alpha1.ForwardingRuleTarget{
					crInput.Spec.ForProvider.Targets[0].IPCfg.IP: crInput.Spec.ForProvider.Targets[0],
					"192.100.100.2": crInput.Spec.ForProvider.Targets[1],
				}
				createInput := forwardingrule.GenerateCreateInput(crInput, "10.20.30.40", targetIps)
				fwRuleClient.EXPECT().
					CreateForwardingRule(ctx, "nlb-dc-id", "nlb-id", utils.MatchFuncDefaultFormatter(createInput, matches)).
					Return(ionoscloud.NetworkLoadBalancerForwardingRule{Id: pstr("new-fr-id")}, nil)
			},
		},
		{
			name:             "Failed to create new forwarding rule - listener ip client error",
			mg:               testCreateInput(),
			want:             managed.ExternalCreation{},
			wantErr:          true,
			wantExternalName: "",
			mock: func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient) {
				fwRuleClient.EXPECT().
					CheckDuplicateForwardingRule(ctx, "nlb-dc-id", "nlb-id", "fr-name").
					Return("", nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "fr-ip-id", 0).
					Return([]string{}, errors.New("get listener ip client error"))
			},
		},
		{
			name:             "Failed to create new forwarding rule - target ip client error",
			mg:               testCreateInput(),
			want:             managed.ExternalCreation{},
			wantErr:          true,
			wantExternalName: "",
			mock: func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient) {
				fwRuleClient.EXPECT().
					CheckDuplicateForwardingRule(ctx, "nlb-dc-id", "nlb-id", "fr-name").
					Return("", nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "fr-ip-id", 0).
					Return([]string{"10.20.30.40"}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "frt-ip-id", 1).
					Return([]string{}, errors.New("get target ip client error"))
			},
		},
		{
			name:             "Failed to create new forwarding rule",
			mg:               testCreateInput(),
			want:             managed.ExternalCreation{},
			wantErr:          true,
			wantExternalName: "",
			mock: func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient) {
				fwRuleClient.EXPECT().
					CheckDuplicateForwardingRule(ctx, "nlb-dc-id", "nlb-id", "fr-name").
					Return("", nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "fr-ip-id", 0).
					Return([]string{"10.20.30.40"}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "frt-ip-id", 1).
					Return([]string{"192.100.100.2"}, nil)
				crInput := testCreateInput()
				targetIps := map[string]v1alpha1.ForwardingRuleTarget{
					crInput.Spec.ForProvider.Targets[0].IPCfg.IP: crInput.Spec.ForProvider.Targets[0],
					"192.100.100.2": crInput.Spec.ForProvider.Targets[1],
				}
				createInput := forwardingrule.GenerateCreateInput(crInput, "10.20.30.40", targetIps)
				fwRuleClient.EXPECT().
					CreateForwardingRule(ctx, "nlb-dc-id", "nlb-id", utils.MatchFuncDefaultFormatter(createInput, matches)).
					Return(ionoscloud.NetworkLoadBalancerForwardingRule{}, errors.New("forwarding rule creation error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			fwRuleClient := forwardingrulemock.NewMockClient(ctrl)
			ipBlockClient := ipblock.NewMockClient(ctrl)
			tt.mock(ctx, fwRuleClient, ipBlockClient)
			external := externalForwardingRule{
				service:              fwRuleClient,
				ipBlockService:       ipBlockClient,
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

func TestNLBForwardingRuleUpdate(t *testing.T) {
	testUpdateInput := func() *v1alpha1.ForwardingRule {
		mg := &v1alpha1.ForwardingRule{
			Status: v1alpha1.ForwardingRuleStatus{
				AtProvider: v1alpha1.ForwardingRuleObservation{
					ForwardingRuleID: "fr-id",
				},
			},
			Spec: v1alpha1.ForwardingRuleSpec{
				ForProvider: v1alpha1.ForwardingRuleParameters{
					DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: "nlb-dc-id"},
					NLBCfg:        v1alpha1.NetworkLoadBalancerConfig{NetworkLoadBalancerID: "nlb-id"},
					Name:          "fr-name-updated",
					ListenerIP:    v1alpha1.IPConfig{IPBlockConfig: v1alpha1.IPBlockConfig{IPBlockID: "fr-ip-id"}, Index: 0},
					ListenerPort:  1000,
					Protocol:      "TCP",
					Algorithm:     "ROUND_ROBIN",
					HealthCheck:   defaultFwRuleHealthCheckCR,
					Targets: []v1alpha1.ForwardingRuleTarget{
						{
							IPCfg:         v1alpha1.IPConfig{IP: "192.100.100.1"},
							Port:          888,
							Weight:        10,
							ProxyProtocol: "v1",
							HealthCheck:   defaultFwRuleTargetHealthCheckCR,
						}, {
							IPCfg:         v1alpha1.IPConfig{IPBlockConfig: v1alpha1.IPBlockConfig{IPBlockID: "frt-ip-id"}, Index: 1},
							Port:          888,
							Weight:        15,
							ProxyProtocol: "v1",
							HealthCheck:   defaultFwRuleTargetHealthCheckCR,
						},
					},
				},
			},
		}
		return mg
	}
	notAForwardingRule := struct{ v1alpha1.ForwardingRule }{}
	tests := []struct {
		name    string
		mg      resource.Managed
		want    managed.ExternalUpdate
		wantErr bool
		mock    func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient)
	}{
		{
			name:    "Wrong managed type",
			mg:      &notAForwardingRule,
			want:    managed.ExternalUpdate{},
			wantErr: true,
			mock: func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient) {
			},
		},
		{
			name: "ForwardingRule is busy",
			mg: func() *v1alpha1.ForwardingRule {
				mg := &v1alpha1.ForwardingRule{
					Status: v1alpha1.ForwardingRuleStatus{
						AtProvider: v1alpha1.ForwardingRuleObservation{
							State: compute.BUSY,
						},
					},
				}
				return mg
			}(),
			want:    managed.ExternalUpdate{},
			wantErr: false,
			mock: func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient) {
			},
		},
		{
			name:    "ForwardingRule update requested",
			mg:      testUpdateInput(),
			want:    managed.ExternalUpdate{},
			wantErr: false,
			mock: func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient) {
				ipBlockClient.EXPECT().
					GetIPs(ctx, "fr-ip-id", 0).
					Return([]string{"10.20.30.40"}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "frt-ip-id", 1).
					Return([]string{"192.100.100.222"}, nil)
				crInput := testUpdateInput()
				targetIps := map[string]v1alpha1.ForwardingRuleTarget{
					crInput.Spec.ForProvider.Targets[0].IPCfg.IP: crInput.Spec.ForProvider.Targets[0],
					"192.100.100.222": crInput.Spec.ForProvider.Targets[1],
				}
				updateInput := forwardingrule.GenerateUpdateInput(crInput, "10.20.30.40", targetIps)
				fwRuleClient.EXPECT().
					UpdateForwardingRule(ctx, "nlb-dc-id", "nlb-id", "fr-id", utils.MatchFuncDefaultFormatter(updateInput, matchesProperties)).
					Return(ionoscloud.NetworkLoadBalancerForwardingRule{}, nil)
			},
		},
		{
			name:    "Failed to update forwarding rule - listener ip client error",
			mg:      testUpdateInput(),
			want:    managed.ExternalUpdate{},
			wantErr: true,
			mock: func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient) {
				ipBlockClient.EXPECT().
					GetIPs(ctx, "fr-ip-id", 0).
					Return([]string{}, errors.New("get listener ip client error"))
			},
		},
		{
			name:    "Failed to create new forwarding rule - target ip client error",
			mg:      testUpdateInput(),
			want:    managed.ExternalUpdate{},
			wantErr: true,
			mock: func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient) {
				ipBlockClient.EXPECT().
					GetIPs(ctx, "fr-ip-id", 0).
					Return([]string{"10.20.30.40"}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "frt-ip-id", 1).
					Return([]string{}, errors.New("get target ip client error"))
			},
		},
		{
			name:    "Failed to update forwarding rule",
			mg:      testUpdateInput(),
			want:    managed.ExternalUpdate{},
			wantErr: true,
			mock: func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient) {
				ipBlockClient.EXPECT().
					GetIPs(ctx, "fr-ip-id", 0).
					Return([]string{"10.20.30.40"}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "frt-ip-id", 1).
					Return([]string{"192.100.100.222"}, nil)
				crInput := testUpdateInput()
				targetIps := map[string]v1alpha1.ForwardingRuleTarget{
					crInput.Spec.ForProvider.Targets[0].IPCfg.IP: crInput.Spec.ForProvider.Targets[0],
					"192.100.100.222": crInput.Spec.ForProvider.Targets[1],
				}
				updateInput := forwardingrule.GenerateUpdateInput(crInput, "10.20.30.40", targetIps)
				fwRuleClient.EXPECT().
					UpdateForwardingRule(ctx, "nlb-dc-id", "nlb-id", "fr-id", utils.MatchFuncDefaultFormatter(updateInput, matchesProperties)).
					Return(ionoscloud.NetworkLoadBalancerForwardingRule{}, errors.New("forwarding rule update error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			fwRuleClient := forwardingrulemock.NewMockClient(ctrl)
			ipBlockClient := ipblock.NewMockClient(ctrl)
			tt.mock(ctx, fwRuleClient, ipBlockClient)
			external := externalForwardingRule{
				service:              fwRuleClient,
				ipBlockService:       ipBlockClient,
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

func TestNLBForwardingRuleDelete(t *testing.T) {
	testDeleteInput := func() *v1alpha1.ForwardingRule {
		mg := &v1alpha1.ForwardingRule{
			Spec: v1alpha1.ForwardingRuleSpec{
				ForProvider: v1alpha1.ForwardingRuleParameters{
					DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: "nlb-dc-id"},
					NLBCfg:        v1alpha1.NetworkLoadBalancerConfig{NetworkLoadBalancerID: "nlb-id"},
				},
			},
			Status: v1alpha1.ForwardingRuleStatus{
				AtProvider: v1alpha1.ForwardingRuleObservation{
					ForwardingRuleID: "fr-id"},
			},
		}
		return mg
	}

	notAForwardingRule := struct{ v1alpha1.ForwardingRule }{}
	tests := []struct {
		name    string
		mg      resource.Managed
		wantErr bool
		mock    func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient)
	}{
		{
			name:    "Wrong managed type",
			mg:      &notAForwardingRule,
			wantErr: true,
			mock: func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient) {
			},
		},
		{
			name: "ForwardingRule already deleting",
			mg: func() *v1alpha1.ForwardingRule {
				mg := &v1alpha1.ForwardingRule{
					Status: v1alpha1.ForwardingRuleStatus{
						AtProvider: v1alpha1.ForwardingRuleObservation{
							State: compute.DESTROYING,
						},
					},
				}
				return mg
			}(),
			wantErr: false,
			mock: func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient) {
			},
		},
		{
			name:    "ForwardingRule delete requested",
			mg:      testDeleteInput(),
			wantErr: false,
			mock: func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient) {
				fwRuleClient.EXPECT().
					DeleteForwardingRule(ctx, "nlb-dc-id", "nlb-id", "fr-id").
					Return(nil)
			},
		},
		{
			name:    "ForwardingRule not found",
			mg:      testDeleteInput(),
			wantErr: false,
			mock: func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient) {
				fwRuleClient.EXPECT().
					DeleteForwardingRule(ctx, "nlb-dc-id", "nlb-id", "fr-id").
					Return(forwardingrule.ErrNotFound)
			},
		},
		{
			name:    "ForwardingRule delete failed",
			mg:      testDeleteInput(),
			wantErr: true,
			mock: func(ctx context.Context, fwRuleClient *forwardingrulemock.MockClient, ipBlockClient *ipblock.MockClient) {
				fwRuleClient.EXPECT().
					DeleteForwardingRule(ctx, "nlb-dc-id", "nlb-id", "fr-id").
					Return(errors.New("forwarding rule deletion error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			fwRuleClient := forwardingrulemock.NewMockClient(ctrl)
			ipBlockClient := ipblock.NewMockClient(ctrl)
			tt.mock(ctx, fwRuleClient, ipBlockClient)
			external := externalForwardingRule{
				service:              fwRuleClient,
				ipBlockService:       ipBlockClient,
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

var pi32 = ionoscloud.PtrInt32
var pbool = ionoscloud.PtrBool
var pstr = ionoscloud.PtrString

var defaultFwRuleHealthCheck = ionoscloud.NetworkLoadBalancerForwardingRuleHealthCheck{
	ClientTimeout:  pi32(50000),
	ConnectTimeout: pi32(5000),
	TargetTimeout:  pi32(50000),
	Retries:        pi32(3),
}
var defaultFwRuleHealthCheckCR = v1alpha1.ForwardingRuleHealthCheck{
	ClientTimeout:  50000,
	ConnectTimeout: 5000,
	TargetTimeout:  50000,
	Retries:        3,
}
var defaultFwRuleTargetHealthCheck = ionoscloud.NetworkLoadBalancerForwardingRuleTargetHealthCheck{
	Check:         pbool(true),
	CheckInterval: pi32(2000),
	Maintenance:   pbool(false),
}
var defaultFwRuleTargetHealthCheckCR = v1alpha1.ForwardingRuleTargetHealthCheck{
	Check:         true,
	CheckInterval: 2000,
	Maintenance:   false,
}

func matchesProperties(want, got any) bool {
	w, ok := want.(ionoscloud.NetworkLoadBalancerForwardingRuleProperties)
	if !ok {
		return false
	}
	g, ok := got.(ionoscloud.NetworkLoadBalancerForwardingRuleProperties)
	if !ok {
		return false
	}

	// sort Targets to ensure elements are ordered since the lists are built from a map iteration
	if w.Targets != nil && g.Targets != nil {
		slices.SortStableFunc(
			*w.Targets,
			func(a, b ionoscloud.NetworkLoadBalancerForwardingRuleTarget) int {
				return cmp.Compare(*a.Ip, *b.Ip)
			},
		)
		slices.SortStableFunc(
			*g.Targets,
			func(a, b ionoscloud.NetworkLoadBalancerForwardingRuleTarget) int {
				return cmp.Compare(*a.Ip, *b.Ip)
			},
		)
	}
	return reflect.DeepEqual(w, g)
}

func matches(want, got any) bool {
	w, ok := want.(ionoscloud.NetworkLoadBalancerForwardingRule)
	if !ok {
		return false
	}
	g, ok := got.(ionoscloud.NetworkLoadBalancerForwardingRule)
	if !ok {
		return false
	}

	switch {
	case w.Properties != nil && g.Properties == nil:
		return false
	case w.Properties == nil && g.Properties != nil:
		return false
	case w.Properties == nil && g.Properties == nil:
		return true
	}

	return matchesProperties(*w.Properties, *g.Properties)
}
