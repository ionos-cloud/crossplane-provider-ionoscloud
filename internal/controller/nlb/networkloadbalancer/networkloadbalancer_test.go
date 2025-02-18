//nolint:testifylint
package networkloadbalancer

import (
	"context"
	"errors"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/golang/mock/gomock"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/assert"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/nlb/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/nlb/networkloadbalancer"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/mock/clients/compute/ipblock"
	networkloadbalancermock "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/mock/clients/nlb/networkloadbalancer"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

func TestNetworkLoadBalancerObserve(t *testing.T) {
	notANetworkLoadBalancer := struct{ v1alpha1.NetworkLoadBalancer }{}
	tests := []struct {
		name    string
		mg      resource.Managed
		want    managed.ExternalObservation
		wantErr bool
		mock    func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient)
	}{
		{
			name:    "Wrong managed type",
			mg:      &notANetworkLoadBalancer,
			want:    managed.ExternalObservation{},
			wantErr: true,
			mock: func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient) {
			},
		},
		{
			name:    "NetworkLoadBalancer does not exist",
			mg:      &v1alpha1.NetworkLoadBalancer{},
			want:    managed.ExternalObservation{},
			wantErr: false,
			mock: func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient) {
			},
		},
		{
			name: "NetworkLoadBalancer not found in ionoscloud",
			mg: func() *v1alpha1.NetworkLoadBalancer {
				mg := &v1alpha1.NetworkLoadBalancer{
					Spec: v1alpha1.NetworkLoadBalancerSpec{
						ForProvider: v1alpha1.NetworkLoadBalancerParameters{
							DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: "dc-id"},
						},
					},
				}
				meta.SetExternalName(mg, "nlb-id")
				return mg
			}(),
			want:    managed.ExternalObservation{},
			wantErr: false,
			mock: func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient) {
				nlbClient.EXPECT().
					GetNetworkLoadBalancerByID(ctx, "dc-id", "nlb-id").
					Return(ionoscloud.NetworkLoadBalancer{}, networkloadbalancer.ErrNotFound)
			},
		},
		{
			name: "Client error - get network load balancer by id",
			mg: func() *v1alpha1.NetworkLoadBalancer {
				mg := &v1alpha1.NetworkLoadBalancer{
					Spec: v1alpha1.NetworkLoadBalancerSpec{
						ForProvider: v1alpha1.NetworkLoadBalancerParameters{
							DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: "dc-id"},
						},
					},
				}
				meta.SetExternalName(mg, "nlb-id")
				return mg
			}(),
			want:    managed.ExternalObservation{},
			wantErr: true,
			mock: func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient) {
				nlbClient.EXPECT().
					GetNetworkLoadBalancerByID(ctx, "dc-id", "nlb-id").
					Return(ionoscloud.NetworkLoadBalancer{}, errors.New("internal client error"))
			},
		},
		{
			name: "Client error - get listener ips",
			mg: func() *v1alpha1.NetworkLoadBalancer {
				mg := &v1alpha1.NetworkLoadBalancer{
					Spec: v1alpha1.NetworkLoadBalancerSpec{
						ForProvider: v1alpha1.NetworkLoadBalancerParameters{
							DatacenterCfg:  v1alpha1.DatacenterConfig{DatacenterID: "dc-id"},
							TargetLanCfg:   v1alpha1.LanConfig{LanID: "0"},
							ListenerLanCfg: v1alpha1.LanConfig{LanID: "1"},
							IpsCfg: v1alpha1.IPsConfig{IPsBlocksCfg: []v1alpha1.IPsBlockConfig{
								{IPBlock: v1alpha1.IPBlockConfig{IPBlockID: "ipblock1-id"}, Indexes: []int{0, 1, 2}},
								{IPBlock: v1alpha1.IPBlockConfig{IPBlockID: "ipblock2-id"}, Indexes: []int{0}},
								{IPBlock: v1alpha1.IPBlockConfig{IPBlockID: "ipblock3-id"}, Indexes: []int{0, 2, 4}},
							},
							},
						},
					},
				}
				meta.SetExternalName(mg, "nlb-id")
				return mg
			}(),
			want:    managed.ExternalObservation{},
			wantErr: true,
			mock: func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient) {
				nlbClient.EXPECT().
					GetNetworkLoadBalancerByID(ctx, "dc-id", "nlb-id").
					Return(ionoscloud.NetworkLoadBalancer{}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock1-id", 0, 1, 2).
					Return([]string{"some-ips"}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock2-id", 0).
					Return([]string{}, errors.New("internal ip block client error"))
			},
		},
		{
			name: "Bad lan ids",
			mg: func() *v1alpha1.NetworkLoadBalancer {
				mg := &v1alpha1.NetworkLoadBalancer{
					Spec: v1alpha1.NetworkLoadBalancerSpec{
						ForProvider: v1alpha1.NetworkLoadBalancerParameters{
							DatacenterCfg:  v1alpha1.DatacenterConfig{DatacenterID: "dc-id"},
							TargetLanCfg:   v1alpha1.LanConfig{LanID: "0"},
							ListenerLanCfg: v1alpha1.LanConfig{LanID: "bad-lan-id"},
						},
					},
				}
				meta.SetExternalName(mg, "nlb-id")
				return mg
			}(),
			want:    managed.ExternalObservation{},
			wantErr: true,
			mock: func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient) {
				nlbClient.EXPECT().
					GetNetworkLoadBalancerByID(ctx, "dc-id", "nlb-id").
					Return(ionoscloud.NetworkLoadBalancer{}, nil)
			},
		},
		{
			name: "Network Load Balancer is up to date",
			mg: func() *v1alpha1.NetworkLoadBalancer {
				mg := &v1alpha1.NetworkLoadBalancer{
					Spec: v1alpha1.NetworkLoadBalancerSpec{
						ForProvider: v1alpha1.NetworkLoadBalancerParameters{
							DatacenterCfg:  v1alpha1.DatacenterConfig{DatacenterID: "dc-id"},
							TargetLanCfg:   v1alpha1.LanConfig{LanID: "0"},
							ListenerLanCfg: v1alpha1.LanConfig{LanID: "1"},
							IpsCfg: v1alpha1.IPsConfig{IPsBlocksCfg: []v1alpha1.IPsBlockConfig{
								{IPBlock: v1alpha1.IPBlockConfig{IPBlockID: "ipblock1-id"}, Indexes: []int{0, 1, 2}},
								{IPBlock: v1alpha1.IPBlockConfig{IPBlockID: "ipblock2-id"}, Indexes: []int{0}},
								{IPBlock: v1alpha1.IPBlockConfig{IPBlockID: "ipblock3-id"}, Indexes: []int{0, 2, 4}},
							},
							},
							Name:         "nlb-name",
							LbPrivateIps: []string{"10.10.10.10", "20.20.20.20"},
						},
					},
				}
				meta.SetExternalName(mg, "nlb-id")
				return mg
			}(),
			want: managed.ExternalObservation{
				ResourceExists:          true,
				ResourceUpToDate:        true,
				ResourceLateInitialized: false,
				ConnectionDetails:       managed.ConnectionDetails{},
			},
			wantErr: false,
			mock: func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient) {
				nlbClient.EXPECT().
					GetNetworkLoadBalancerByID(ctx, "dc-id", "nlb-id").
					Return(ionoscloud.NetworkLoadBalancer{
						Properties: &ionoscloud.NetworkLoadBalancerProperties{
							Name:         pstr("nlb-name"),
							TargetLan:    pi32(0),
							ListenerLan:  pi32(1),
							Ips:          &[]string{"111.100.100.0", "111.100.100.1", "111.100.100.2", "122.100.100.0", "133.100.100.0", "133.100.100.2", "133.100.100.4"},
							LbPrivateIps: &[]string{"10.10.10.10", "20.20.20.20"},
						},
					}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock1-id", 0, 1, 2).
					Return([]string{"111.100.100.0", "111.100.100.1", "111.100.100.2"}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock2-id", 0).
					Return([]string{"122.100.100.0"}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock3-id", 0, 2, 4).
					Return([]string{"133.100.100.0", "133.100.100.2", "133.100.100.4"}, nil)
			},
		},
		{
			name: "Network Load Balancer is up to date and late initialized",
			mg: func() *v1alpha1.NetworkLoadBalancer {
				mg := &v1alpha1.NetworkLoadBalancer{
					Spec: v1alpha1.NetworkLoadBalancerSpec{
						ForProvider: v1alpha1.NetworkLoadBalancerParameters{
							DatacenterCfg:  v1alpha1.DatacenterConfig{DatacenterID: "dc-id"},
							TargetLanCfg:   v1alpha1.LanConfig{LanID: "0"},
							ListenerLanCfg: v1alpha1.LanConfig{LanID: "1"},
							IpsCfg: v1alpha1.IPsConfig{IPsBlocksCfg: []v1alpha1.IPsBlockConfig{
								{IPBlock: v1alpha1.IPBlockConfig{IPBlockID: "ipblock1-id"}, Indexes: []int{0, 1, 2}},
								{IPBlock: v1alpha1.IPBlockConfig{IPBlockID: "ipblock2-id"}, Indexes: []int{0}},
								{IPBlock: v1alpha1.IPBlockConfig{IPBlockID: "ipblock3-id"}, Indexes: []int{0, 2, 4}},
							},
							},
							Name: "nlb-name",
						},
					},
				}
				meta.SetExternalName(mg, "nlb-id")
				return mg
			}(),
			want: managed.ExternalObservation{
				ResourceExists:          true,
				ResourceUpToDate:        true,
				ResourceLateInitialized: true,
				ConnectionDetails:       managed.ConnectionDetails{},
			},
			wantErr: false,
			mock: func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient) {
				nlbClient.EXPECT().
					GetNetworkLoadBalancerByID(ctx, "dc-id", "nlb-id").
					Return(ionoscloud.NetworkLoadBalancer{
						Properties: &ionoscloud.NetworkLoadBalancerProperties{
							Name:         pstr("nlb-name"),
							TargetLan:    pi32(0),
							ListenerLan:  pi32(1),
							Ips:          &[]string{"111.100.100.0", "111.100.100.1", "111.100.100.2", "122.100.100.0", "133.100.100.0", "133.100.100.2", "133.100.100.4"},
							LbPrivateIps: &[]string{"10.10.10.10", "20.20.20.20"},
						},
					}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock1-id", 0, 1, 2).
					Return([]string{"111.100.100.0", "111.100.100.1", "111.100.100.2"}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock2-id", 0).
					Return([]string{"122.100.100.0"}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock3-id", 0, 2, 4).
					Return([]string{"133.100.100.0", "133.100.100.2", "133.100.100.4"}, nil)
			},
		},
		{
			name: "Network Load Balancer requires update",
			mg: func() *v1alpha1.NetworkLoadBalancer {
				mg := &v1alpha1.NetworkLoadBalancer{
					Spec: v1alpha1.NetworkLoadBalancerSpec{
						ForProvider: v1alpha1.NetworkLoadBalancerParameters{
							DatacenterCfg:  v1alpha1.DatacenterConfig{DatacenterID: "dc-id"},
							TargetLanCfg:   v1alpha1.LanConfig{LanID: "0"},
							ListenerLanCfg: v1alpha1.LanConfig{LanID: "1"},
							IpsCfg: v1alpha1.IPsConfig{IPsBlocksCfg: []v1alpha1.IPsBlockConfig{
								{IPBlock: v1alpha1.IPBlockConfig{IPBlockID: "ipblock1-id"}, Indexes: []int{0, 1, 2}},
								{IPBlock: v1alpha1.IPBlockConfig{IPBlockID: "ipblock2-id"}, Indexes: []int{0}},
								{IPBlock: v1alpha1.IPBlockConfig{IPBlockID: "ipblock3-id"}, Indexes: []int{0, 2, 4}},
							},
							},
							LbPrivateIps: []string{"10.10.10.10", "20.20.20.20"},
							Name:         "nlb-name",
						},
					},
				}
				meta.SetExternalName(mg, "nlb-id")
				return mg
			}(),
			want: managed.ExternalObservation{
				ResourceExists:          true,
				ResourceUpToDate:        false,
				ResourceLateInitialized: false,
				ConnectionDetails:       managed.ConnectionDetails{},
			},
			wantErr: false,
			mock: func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient) {
				nlbClient.EXPECT().
					GetNetworkLoadBalancerByID(ctx, "dc-id", "nlb-id").
					Return(ionoscloud.NetworkLoadBalancer{
						Properties: &ionoscloud.NetworkLoadBalancerProperties{
							Name:         pstr("nlb-name"),
							TargetLan:    pi32(0),
							ListenerLan:  pi32(1),
							Ips:          &[]string{"111.100.100.0", "111.100.100.1", "111.100.100.2"},
							LbPrivateIps: &[]string{"10.10.10.10", "20.20.20.20"},
						},
					}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock1-id", 0, 1, 2).
					Return([]string{"111.100.100.0", "111.100.100.1", "111.100.100.2"}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock2-id", 0).
					Return([]string{"122.100.100.0"}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock3-id", 0, 2, 4).
					Return([]string{"133.100.100.0", "133.100.100.2", "133.100.100.4"}, nil)
			},
		},
		{
			name: "Network Load Balancer requires update and late initialized",
			mg: func() *v1alpha1.NetworkLoadBalancer {
				mg := &v1alpha1.NetworkLoadBalancer{
					Spec: v1alpha1.NetworkLoadBalancerSpec{
						ForProvider: v1alpha1.NetworkLoadBalancerParameters{
							DatacenterCfg:  v1alpha1.DatacenterConfig{DatacenterID: "dc-id"},
							TargetLanCfg:   v1alpha1.LanConfig{LanID: "0"},
							ListenerLanCfg: v1alpha1.LanConfig{LanID: "1"},
							IpsCfg: v1alpha1.IPsConfig{IPsBlocksCfg: []v1alpha1.IPsBlockConfig{
								{IPBlock: v1alpha1.IPBlockConfig{IPBlockID: "ipblock1-id"}, Indexes: []int{0, 1, 2}},
								{IPBlock: v1alpha1.IPBlockConfig{IPBlockID: "ipblock2-id"}, Indexes: []int{0}},
								{IPBlock: v1alpha1.IPBlockConfig{IPBlockID: "ipblock3-id"}, Indexes: []int{0, 2, 4}},
							},
							},
							Name: "nlb-name",
						},
					},
				}
				meta.SetExternalName(mg, "nlb-id")
				return mg
			}(),
			want: managed.ExternalObservation{
				ResourceExists:          true,
				ResourceUpToDate:        false,
				ResourceLateInitialized: true,
				ConnectionDetails:       managed.ConnectionDetails{},
			},
			wantErr: false,
			mock: func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient) {
				nlbClient.EXPECT().
					GetNetworkLoadBalancerByID(ctx, "dc-id", "nlb-id").
					Return(ionoscloud.NetworkLoadBalancer{
						Properties: &ionoscloud.NetworkLoadBalancerProperties{
							Name:         pstr("nlb-name"),
							TargetLan:    pi32(0),
							ListenerLan:  pi32(1),
							Ips:          &[]string{"111.100.100.0", "111.100.100.1", "111.100.100.2"},
							LbPrivateIps: &[]string{"10.10.10.10", "20.20.20.20"},
						},
					}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock1-id", 0, 1, 2).
					Return([]string{"111.100.100.0", "111.100.100.1", "111.100.100.2"}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock2-id", 0).
					Return([]string{"122.100.100.0"}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock3-id", 0, 2, 4).
					Return([]string{"133.100.100.0", "133.100.100.2", "133.100.100.4"}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			nlbClient := networkloadbalancermock.NewMockClient(ctrl)
			ipBlockClient := ipblock.NewMockClient(ctrl)
			tt.mock(ctx, nlbClient, ipBlockClient)
			external := externalNetworkLoadBalancer{
				service:        nlbClient,
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

func TestNetworkLoadBalancerCreate(t *testing.T) {
	testCreateInput := func() *v1alpha1.NetworkLoadBalancer {
		mg := &v1alpha1.NetworkLoadBalancer{
			Spec: v1alpha1.NetworkLoadBalancerSpec{
				ForProvider: v1alpha1.NetworkLoadBalancerParameters{
					DatacenterCfg:  v1alpha1.DatacenterConfig{DatacenterID: "nlb-dc-id"},
					TargetLanCfg:   v1alpha1.LanConfig{LanID: "0"},
					ListenerLanCfg: v1alpha1.LanConfig{LanID: "1"},
					IpsCfg: v1alpha1.IPsConfig{IPsBlocksCfg: []v1alpha1.IPsBlockConfig{
						{IPBlock: v1alpha1.IPBlockConfig{IPBlockID: "ipblock1-id"}, Indexes: []int{0, 1, 2}},
						{IPBlock: v1alpha1.IPBlockConfig{IPBlockID: "ipblock2-id"}, Indexes: []int{0}},
						{IPBlock: v1alpha1.IPBlockConfig{IPBlockID: "ipblock3-id"}, Indexes: []int{0, 2, 4}},
					},
					},
					Name:         "nlb-name",
					LbPrivateIps: []string{"10.10.10.10", "20.20.20.20"},
				},
			},
		}
		return mg
	}

	notANetworkLoadBalancer := struct{ v1alpha1.NetworkLoadBalancer }{}
	tests := []struct {
		name             string
		mg               resource.Managed
		want             managed.ExternalCreation
		wantErr          bool
		wantExternalName string
		mock             func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient)
	}{
		{
			name:             "Wrong managed type",
			mg:               &notANetworkLoadBalancer,
			want:             managed.ExternalCreation{},
			wantErr:          true,
			wantExternalName: "",
			mock: func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient) {
			},
		},
		{
			name: "NetworkLoadBalancer already created",
			mg: func() *v1alpha1.NetworkLoadBalancer {
				mg := &v1alpha1.NetworkLoadBalancer{}
				meta.SetExternalName(mg, "nlb-id")
				return mg
			}(),
			want:             managed.ExternalCreation{},
			wantErr:          false,
			wantExternalName: "nlb-id",
			mock: func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient) {
			},
		},
		{
			name: "NetworkLoadBalancer is being provisioned",
			mg: func() *v1alpha1.NetworkLoadBalancer {
				mg := &v1alpha1.NetworkLoadBalancer{Status: v1alpha1.NetworkLoadBalancerStatus{AtProvider: v1alpha1.NetworkLoadBalancerObservation{State: compute.BUSY}}}
				return mg
			}(),
			want:             managed.ExternalCreation{},
			wantErr:          false,
			wantExternalName: "",
			mock: func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient) {
			},
		},
		{
			name:             "Imported duplicate name network load balancer",
			mg:               testCreateInput(),
			want:             managed.ExternalCreation{},
			wantErr:          false,
			wantExternalName: "nlb-id",
			mock: func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient) {
				nlbClient.EXPECT().
					CheckDuplicateNetworkLoadBalancer(ctx, "nlb-dc-id", "nlb-name").
					Return("nlb-id", nil)
			},
		},
		{
			name:             "Failed to perform duplicate network load balancer check",
			mg:               testCreateInput(),
			want:             managed.ExternalCreation{},
			wantErr:          true,
			wantExternalName: "",
			mock: func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient) {
				nlbClient.EXPECT().
					CheckDuplicateNetworkLoadBalancer(ctx, "nlb-dc-id", "nlb-name").
					Return("", errors.New("duplicate network load balancer check error"))
			},
		},
		{
			name:             "Create new network load balancer",
			mg:               testCreateInput(),
			want:             managed.ExternalCreation{},
			wantErr:          false,
			wantExternalName: "new-nlb-id",
			mock: func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient) {
				nlbClient.EXPECT().
					CheckDuplicateNetworkLoadBalancer(ctx, "nlb-dc-id", "nlb-name").
					Return("", nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock1-id", 0, 1, 2).
					Return([]string{"111.100.100.0", "111.100.100.1", "111.100.100.2"}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock2-id", 0).
					Return([]string{"122.100.100.0"}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock3-id", 0, 2, 4).
					Return([]string{"133.100.100.0", "133.100.100.2", "133.100.100.4"}, nil)
				crInput := testCreateInput()
				createInput := networkloadbalancer.GenerateCreateInput(crInput, 1, 0,
					[]string{"111.100.100.0", "111.100.100.1", "111.100.100.2", "122.100.100.0",
						"133.100.100.0", "133.100.100.2", "133.100.100.4"},
				)
				nlbClient.EXPECT().
					CreateNetworkLoadBalancer(ctx, "nlb-dc-id", utils.MatchEqDefaultFormatter(createInput)).
					Return(ionoscloud.NetworkLoadBalancer{Id: pstr("new-nlb-id")}, nil)
			},
		},
		{
			name: "Failed to create new network load balancer - bad lan id",
			mg: func() *v1alpha1.NetworkLoadBalancer {
				mg := testCreateInput()
				mg.Spec.ForProvider.TargetLanCfg.LanID = "bad-id"
				return mg
			}(),
			want:             managed.ExternalCreation{},
			wantErr:          true,
			wantExternalName: "",
			mock: func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient) {
				nlbClient.EXPECT().
					CheckDuplicateNetworkLoadBalancer(ctx, "nlb-dc-id", "nlb-name").
					Return("", nil)
			},
		},
		{
			name:             "Failed to create new network load balancer - ips client error",
			mg:               testCreateInput(),
			want:             managed.ExternalCreation{},
			wantErr:          true,
			wantExternalName: "",
			mock: func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient) {
				nlbClient.EXPECT().
					CheckDuplicateNetworkLoadBalancer(ctx, "nlb-dc-id", "nlb-name").
					Return("", nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock1-id", 0, 1, 2).
					Return([]string{"111.100.100.0", "111.100.100.1", "111.100.100.2"}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock2-id", 0).
					Return([]string{""}, errors.New("ips client error"))
			},
		},
		{
			name:             "Failed to create new network load balancer",
			mg:               testCreateInput(),
			want:             managed.ExternalCreation{},
			wantErr:          true,
			wantExternalName: "",
			mock: func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient) {
				nlbClient.EXPECT().
					CheckDuplicateNetworkLoadBalancer(ctx, "nlb-dc-id", "nlb-name").
					Return("", nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock1-id", 0, 1, 2).
					Return([]string{"111.100.100.0", "111.100.100.1", "111.100.100.2"}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock2-id", 0).
					Return([]string{"122.100.100.0"}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock3-id", 0, 2, 4).
					Return([]string{"133.100.100.0", "133.100.100.2", "133.100.100.4"}, nil)
				crInput := testCreateInput()
				createInput := networkloadbalancer.GenerateCreateInput(crInput, 1, 0,
					[]string{"111.100.100.0", "111.100.100.1", "111.100.100.2", "122.100.100.0",
						"133.100.100.0", "133.100.100.2", "133.100.100.4"},
				)
				nlbClient.EXPECT().
					CreateNetworkLoadBalancer(ctx, "nlb-dc-id", createInput).
					Return(ionoscloud.NetworkLoadBalancer{}, errors.New("network load balancer creation error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			nlbClient := networkloadbalancermock.NewMockClient(ctrl)
			ipBlockClient := ipblock.NewMockClient(ctrl)
			tt.mock(ctx, nlbClient, ipBlockClient)
			external := externalNetworkLoadBalancer{
				service:              nlbClient,
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

func TestNetworkLoadBalancerUpdate(t *testing.T) {
	testUpdateInput := func() *v1alpha1.NetworkLoadBalancer {
		mg := &v1alpha1.NetworkLoadBalancer{
			Status: v1alpha1.NetworkLoadBalancerStatus{AtProvider: v1alpha1.NetworkLoadBalancerObservation{NetworkLoadBalancerID: "nlb-id"}},
			Spec: v1alpha1.NetworkLoadBalancerSpec{
				ForProvider: v1alpha1.NetworkLoadBalancerParameters{
					DatacenterCfg:  v1alpha1.DatacenterConfig{DatacenterID: "nlb-dc-id"},
					TargetLanCfg:   v1alpha1.LanConfig{LanID: "0"},
					ListenerLanCfg: v1alpha1.LanConfig{LanID: "1"},
					IpsCfg: v1alpha1.IPsConfig{IPsBlocksCfg: []v1alpha1.IPsBlockConfig{
						{IPBlock: v1alpha1.IPBlockConfig{IPBlockID: "ipblock1-id"}, Indexes: []int{0, 1, 2}},
						{IPBlock: v1alpha1.IPBlockConfig{IPBlockID: "ipblock2-id"}, Indexes: []int{0}},
						{IPBlock: v1alpha1.IPBlockConfig{IPBlockID: "ipblock3-id"}, Indexes: []int{0, 2, 4}},
					},
					},
					Name:         "nlb-name",
					LbPrivateIps: []string{"10.10.10.10", "20.20.20.20"},
				},
			},
		}
		return mg
	}

	notANetworkLoadBalancer := struct{ v1alpha1.NetworkLoadBalancer }{}
	tests := []struct {
		name    string
		mg      resource.Managed
		want    managed.ExternalUpdate
		wantErr bool
		mock    func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient)
	}{
		{
			name:    "Wrong managed type",
			mg:      &notANetworkLoadBalancer,
			want:    managed.ExternalUpdate{},
			wantErr: true,
			mock: func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient) {
			},
		},
		{
			name: "NetworkLoadBalancer is busy",
			mg: &v1alpha1.NetworkLoadBalancer{
				Status: v1alpha1.NetworkLoadBalancerStatus{AtProvider: v1alpha1.NetworkLoadBalancerObservation{State: compute.BUSY}},
			},
			want:    managed.ExternalUpdate{},
			wantErr: false,
			mock: func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient) {
			},
		},
		{
			name:    "NetworkLoadBalancer update requested",
			mg:      testUpdateInput(),
			want:    managed.ExternalUpdate{},
			wantErr: false,
			mock: func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient) {
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock1-id", 0, 1, 2).
					Return([]string{"111.100.100.0", "111.100.100.1", "111.100.100.2"}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock2-id", 0).
					Return([]string{"122.100.100.0"}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock3-id", 0, 2, 4).
					Return([]string{"133.100.100.0", "133.100.100.2", "133.100.100.4"}, nil)
				crInput := testUpdateInput()
				updateInput := networkloadbalancer.GenerateUpdateInput(crInput, 1, 0,
					[]string{"111.100.100.0", "111.100.100.1", "111.100.100.2", "122.100.100.0",
						"133.100.100.0", "133.100.100.2", "133.100.100.4"},
				)
				nlbClient.EXPECT().
					UpdateNetworkLoadBalancer(ctx, "nlb-dc-id", "nlb-id", utils.MatchEqDefaultFormatter(updateInput)).
					Return(ionoscloud.NetworkLoadBalancer{}, nil)
			},
		},
		{
			name: "Failed to update network load balancer - bad lan id",
			mg: func() *v1alpha1.NetworkLoadBalancer {
				mg := testUpdateInput()
				mg.Spec.ForProvider.ListenerLanCfg.LanID = "bad-lan-id"
				return mg
			}(),
			want:    managed.ExternalUpdate{},
			wantErr: true,
			mock: func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient) {
			},
		},
		{
			name:    "Failed to update network load balancer - ips client error",
			mg:      testUpdateInput(),
			want:    managed.ExternalUpdate{},
			wantErr: true,
			mock: func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient) {
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock1-id", 0, 1, 2).
					Return([]string{"111.100.100.0", "111.100.100.1", "111.100.100.2"}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock2-id", 0).
					Return([]string{"122.100.100.0"}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock3-id", 0, 2, 4).
					Return([]string{""}, errors.New("ips client error"))
			},
		},
		{
			name:    "Failed to update network load balancer",
			mg:      testUpdateInput(),
			want:    managed.ExternalUpdate{},
			wantErr: true,
			mock: func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient) {
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock1-id", 0, 1, 2).
					Return([]string{"111.100.100.0", "111.100.100.1", "111.100.100.2"}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock2-id", 0).
					Return([]string{"122.100.100.0"}, nil)
				ipBlockClient.EXPECT().
					GetIPs(ctx, "ipblock3-id", 0, 2, 4).
					Return([]string{"133.100.100.0", "133.100.100.2", "133.100.100.4"}, nil)
				crInput := testUpdateInput()
				updateInput := networkloadbalancer.GenerateUpdateInput(crInput, 1, 0,
					[]string{"111.100.100.0", "111.100.100.1", "111.100.100.2", "122.100.100.0",
						"133.100.100.0", "133.100.100.2", "133.100.100.4"},
				)
				nlbClient.EXPECT().
					UpdateNetworkLoadBalancer(ctx, "nlb-dc-id", "nlb-id", utils.MatchEqDefaultFormatter(updateInput)).
					Return(ionoscloud.NetworkLoadBalancer{}, errors.New("network load balancer update error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			nlbClient := networkloadbalancermock.NewMockClient(ctrl)
			ipBlockClient := ipblock.NewMockClient(ctrl)
			tt.mock(ctx, nlbClient, ipBlockClient)
			external := externalNetworkLoadBalancer{
				service:        nlbClient,
				ipBlockService: ipBlockClient,
				log:            logging.NewNopLogger(),
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

func TestNetworkLoadBalancerDelete(t *testing.T) {

	testDeleteInput := func() *v1alpha1.NetworkLoadBalancer {
		mg := &v1alpha1.NetworkLoadBalancer{
			Status: v1alpha1.NetworkLoadBalancerStatus{AtProvider: v1alpha1.NetworkLoadBalancerObservation{NetworkLoadBalancerID: "nlb-id"}},
			Spec: v1alpha1.NetworkLoadBalancerSpec{
				ForProvider: v1alpha1.NetworkLoadBalancerParameters{
					DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: "nlb-dc-id"},
				},
			},
		}
		return mg
	}

	notANetworkLoadBalancer := struct{ v1alpha1.NetworkLoadBalancer }{}
	tests := []struct {
		name    string
		mg      resource.Managed
		wantErr bool
		mock    func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient)
	}{
		{
			name:    "Wrong managed type",
			mg:      &notANetworkLoadBalancer,
			wantErr: true,
			mock: func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient) {
			},
		},
		{
			name:    "NetworkLoadBalancer already deleting",
			mg:      &v1alpha1.NetworkLoadBalancer{Status: v1alpha1.NetworkLoadBalancerStatus{AtProvider: v1alpha1.NetworkLoadBalancerObservation{State: compute.DESTROYING}}},
			wantErr: false,
			mock: func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient) {
			},
		},
		{
			name:    "NetworkLoadBalancer delete requested",
			mg:      testDeleteInput(),
			wantErr: false,
			mock: func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient) {
				nlbClient.EXPECT().
					DeleteNetworkLoadBalancer(ctx, "nlb-dc-id", "nlb-id").
					Return(nil)
			},
		},
		{
			name:    "NetworkLoadBalancer not found",
			mg:      testDeleteInput(),
			wantErr: false,
			mock: func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient) {
				nlbClient.EXPECT().
					DeleteNetworkLoadBalancer(ctx, "nlb-dc-id", "nlb-id").
					Return(networkloadbalancer.ErrNotFound)
			},
		},
		{
			name:    "NetworkLoadBalancer delete failed",
			mg:      testDeleteInput(),
			wantErr: true,
			mock: func(ctx context.Context, nlbClient *networkloadbalancermock.MockClient, ipBlockClient *ipblock.MockClient) {
				nlbClient.EXPECT().
					DeleteNetworkLoadBalancer(ctx, "nlb-dc-id", "nlb-id").
					Return(errors.New("network load balancer deletion error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			nlbClient := networkloadbalancermock.NewMockClient(ctrl)
			ipBlockClient := ipblock.NewMockClient(ctrl)
			tt.mock(ctx, nlbClient, ipBlockClient)
			external := externalNetworkLoadBalancer{
				service:        nlbClient,
				ipBlockService: ipBlockClient,
				log:            logging.NewNopLogger(),
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
var pstr = ionoscloud.PtrString
