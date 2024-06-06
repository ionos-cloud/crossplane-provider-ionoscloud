package serverset

import (
	"context"
	"fmt"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/google/go-cmp/cmp"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/assert"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	vnetID                    = "679070ab-1ebc-46ef-b9f7-c43c1ed9f6e9"
	serverSetNicIndexLabel    = "ionoscloud.com/serverset-nic-index"
	serverSetNicNicIndexLabel = "ionoscloud.com/serverset-nic-nicindex"
	serverSetNicVersionLabel  = "ionoscloud.com/serverset-nic-version"
	nicName                   = "nic-0-0-0"
	nicWithVNetName           = "nic1-1-0-0"
	nicWithoutVNetName        = "nic2-1-0-0"
	nicID                     = "bc59d87e-17cc-4313-b55b-6603884f9d97"
	serverID                  = "07a7e712-fc36-43ca-bc8f-76c05861ff8b"
	lanName                   = "lan1"
	lanID                     = "1"
	dataLAN                   = "data"
	dataLANIpv6CIDR           = "fd00:0:0:1::/64"
)

func Test_kubeNicController_Create(t *testing.T) {
	type fields struct {
		kube client.Client
		log  logging.Logger
	}
	type args struct {
		ctx          context.Context
		cr           *v1alpha1.ServerSet
		serverID     string
		lanName      string
		replicaIndex int
		nicIndex     int
		version      int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    v1alpha1.Nic
		wantErr bool
	}{
		{
			name: "The fields are populated correctly",
			fields: fields{
				kube: fakeKubeClientFuncs(interceptor.Funcs{Create: createWithVNetAndIPV4ReturnsErrorForUnexpectedNICs, Get: getPopulatesFieldsAndReturnsNoError}),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx:          context.Background(),
				cr:           createServerSetWithVNet(),
				serverID:     serverID,
				lanName:      lanName,
				replicaIndex: 1,
				nicIndex:     0,
				version:      0,
			},
			want:    *createWantedNicWithVNet(),
			wantErr: false,
		},
		{
			name: "The fields vnet and ipv4 are optional",
			fields: fields{
				kube: fakeKubeClientFuncs(interceptor.Funcs{Create: createWithoutVNetReturnsErrorForUnexpectedNICs, Get: getPopulatesFieldsAndReturnsNoError}),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx:          context.Background(),
				cr:           createServerSetWithoutVNetAndIPV4(),
				serverID:     serverID,
				lanName:      lanName,
				replicaIndex: 1,
				nicIndex:     0,
				version:      0,
			},
			want:    *createWantedNicWithoutVNet(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &kubeNicController{
				kube: tt.fields.kube,
				log:  tt.fields.log,
			}
			got, err := k.Create(tt.args.ctx, tt.args.cr, tt.args.serverID, tt.args.lanName, tt.args.replicaIndex, tt.args.nicIndex, tt.args.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_fromServerSetToNic(t *testing.T) {
	type args struct {
		cr           *v1alpha1.ServerSet
		name         string
		serverID     string
		lan          v1alpha1.Lan
		replicaIndex int
		nicIndex     int
		version      int
	}

	tests := []struct {
		name string
		args args
		want v1alpha1.Nic
	}{
		{
			name: "The fields are populated correctly",
			args: args{
				cr:       createServerSetWithVNet(),
				name:     nicName,
				serverID: serverID,
				lan:      *createLan(v1alpha1.LanParameters{}),
			},
			want: *createNic(v1alpha1.NicParameters{
				Name:      nicName,
				ServerCfg: v1alpha1.ServerConfig{ServerID: serverID},
				LanCfg:    v1alpha1.LanConfig{LanID: lanID},
				Vnet:      vnetID,
			}),
		},
		{
			name: "DhcpV6 not set if Ipv6Cidr not set on LAN",
			args: args{
				cr:       createServerSetWithDhcpV6(),
				name:     nicName,
				serverID: serverID,
				lan: *createLan(v1alpha1.LanParameters{
					Name:     dataLAN,
					Ipv6Cidr: "",
				}),
			},
			want: *createNic(v1alpha1.NicParameters{
				Name:      nicName,
				ServerCfg: v1alpha1.ServerConfig{ServerID: serverID},
				LanCfg:    v1alpha1.LanConfig{LanID: lanID},
				DhcpV6:    ionoscloud.PtrBool(false),
			}),
		},
		{
			name: "DhcpV6 set if Ipv6Cidr (AUTO) set on LAN",
			args: args{
				cr:       createServerSetWithDhcpV6(),
				name:     nicName,
				serverID: serverID,
				lan: *createLan(v1alpha1.LanParameters{
					Name:     dataLAN,
					Ipv6Cidr: v1alpha1.LANAuto,
				}),
			},
			want: *createNic(v1alpha1.NicParameters{
				Name:      nicName,
				ServerCfg: v1alpha1.ServerConfig{ServerID: serverID},
				LanCfg:    v1alpha1.LanConfig{LanID: lanID},
				DhcpV6:    ionoscloud.PtrBool(true),
			}),
		},
		{
			name: "DhcpV6 set if Ipv6Cidr (IPV6 CIDR block) set on LAN",
			args: args{
				cr:       createServerSetWithDhcpV6(),
				name:     nicName,
				serverID: serverID,
				lan: *createLan(v1alpha1.LanParameters{
					Name:     dataLAN,
					Ipv6Cidr: dataLANIpv6CIDR,
				}),
			},
			want: *createNic(v1alpha1.NicParameters{
				Name:      nicName,
				ServerCfg: v1alpha1.ServerConfig{ServerID: serverID},
				LanCfg:    v1alpha1.LanConfig{LanID: lanID},
				DhcpV6:    ionoscloud.PtrBool(true),
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := kubeNicController{
				log:  logging.NewNopLogger(),
				kube: fakeKubeClientObjs(),
			}

			got := k.fromServerSetToNic(tt.args.cr, tt.args.name, tt.args.serverID, tt.args.lan, tt.args.replicaIndex, tt.args.nicIndex, tt.args.version)
			assert.Equal(t, tt.want, got)
		})
	}
}

func createServerSetWithVNet() *v1alpha1.ServerSet {
	s := createBasicServerSet()
	for nicIndex := range s.Spec.ForProvider.Template.Spec.NICs {
		s.Spec.ForProvider.Template.Spec.NICs[nicIndex].VNetID = vnetID
	}
	return s
}

func createServerSetWithoutVNetAndIPV4() *v1alpha1.ServerSet {
	s := createBasicServerSet()
	s.Spec.ForProvider.Template.Spec.NICs = []v1alpha1.ServerSetTemplateNIC{
		{
			Name:         "nic2",
			DHCP:         false,
			LanReference: "data",
		},
	}
	return s
}

func createServerSetWithDhcpV6() *v1alpha1.ServerSet {
	s := createBasicServerSet()
	for nicIndex := range s.Spec.ForProvider.Template.Spec.NICs {
		s.Spec.ForProvider.Template.Spec.NICs[nicIndex].DHCPv6 = ionoscloud.PtrBool(true)
		s.Spec.ForProvider.Template.Spec.NICs[nicIndex].LanReference = dataLAN
	}
	return s
}

func populateBasicNicMetadataAndSpec(nic *v1alpha1.Nic, nicName string) {
	nic.ObjectMeta.Name = nicName
	nic.ObjectMeta.Labels = map[string]string{
		serverSetLabel:            serverSetName,
		serverSetNicIndexLabel:    "1",
		serverSetNicVersionLabel:  "0",
		serverSetNicNicIndexLabel: "0",
	}
	nic.Spec.ForProvider.Name = nicName
	nic.Spec.ForProvider.ServerCfg.ServerID = serverID
	nic.Spec.ForProvider.LanCfg.LanID = lanID
	nic.Spec.ForProvider.Dhcp = false
}

func makeNicAvailable(nic *v1alpha1.Nic) {
	nic.Status.AtProvider.NicID = nicID
	nic.Status.AtProvider.State = ionoscloud.Available
}

func createWantedNicWithVNet() *v1alpha1.Nic {
	nic := &v1alpha1.Nic{}
	populateBasicNicMetadataAndSpec(nic, nicWithVNetName)
	nic.Spec.ForProvider.Vnet = vnetID
	makeNicAvailable(nic)
	return nic
}

func createWantedNicWithoutVNet() *v1alpha1.Nic {
	nic := &v1alpha1.Nic{}
	populateBasicNicMetadataAndSpec(nic, nicWithoutVNetName)
	makeNicAvailable(nic)
	return nic
}

func createWithVNetAndIPV4ReturnsErrorForUnexpectedNICs(_ context.Context, _ client.WithWatch, obj client.Object, _ ...client.CreateOption) error {
	expectedNic := &v1alpha1.Nic{}
	populateBasicNicMetadataAndSpec(expectedNic, nicWithVNetName)
	expectedNic.Spec.ForProvider.Vnet = vnetID
	if diff := cmp.Diff(obj.(*v1alpha1.Nic), expectedNic); diff != "" {
		return fmt.Errorf("create was called with an unexpected NIC.\n Expected %#v.\n Got %#v", expectedNic, obj.(*v1alpha1.Nic))
	}
	return nil
}

func createWithoutVNetReturnsErrorForUnexpectedNICs(_ context.Context, _ client.WithWatch, obj client.Object, _ ...client.CreateOption) error {
	expectedNic := &v1alpha1.Nic{}
	populateBasicNicMetadataAndSpec(expectedNic, nicWithoutVNetName)
	if diff := cmp.Diff(obj.(*v1alpha1.Nic), expectedNic); diff != "" {
		return fmt.Errorf("create was called with an unexpected NIC.\n Expected %#v \n Got %#v", expectedNic, obj.(*v1alpha1.Nic))
	}
	return nil
}

func getPopulatesFieldsAndReturnsNoError(_ context.Context, _ client.WithWatch, key client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	switch key.Name {
	case lanName:
		lan := obj.(*v1alpha1.Lan)
		lan.Status.AtProvider.LanID = lanID
	case nicWithVNetName:
		nic := obj.(*v1alpha1.Nic)
		populateBasicNicMetadataAndSpec(nic, nicWithVNetName)
		nic.Spec.ForProvider.Vnet = vnetID
		makeNicAvailable(nic)
	case nicWithoutVNetName:
		nic := obj.(*v1alpha1.Nic)
		populateBasicNicMetadataAndSpec(nic, nicWithoutVNetName)
		makeNicAvailable(nic)
	}
	return nil
}

func createLan(params v1alpha1.LanParameters) *v1alpha1.Lan {
	lan := &v1alpha1.Lan{
		Status: v1alpha1.LanStatus{
			AtProvider: v1alpha1.LanObservation{
				LanID: lanID,
			},
		},
		Spec: v1alpha1.LanSpec{
			ForProvider: v1alpha1.LanParameters{
				DatacenterCfg: v1alpha1.DatacenterConfig{},
				Name:          "test-lan",
				Pcc:           v1alpha1.PccConfig{},
				Public:        false,
				Ipv6Cidr:      "",
			},
		},
	}

	if params.DatacenterCfg != (v1alpha1.DatacenterConfig{}) {
		lan.Spec.ForProvider.DatacenterCfg = params.DatacenterCfg
	}
	if params.Name != "" {
		lan.Spec.ForProvider.Name = params.Name
	}
	if params.Pcc != (v1alpha1.PccConfig{}) {
		lan.Spec.ForProvider.Pcc = params.Pcc
	}
	if params.Public != false {
		lan.Spec.ForProvider.Public = params.Public
	}
	if params.Ipv6Cidr != "" {
		lan.Spec.ForProvider.Ipv6Cidr = params.Ipv6Cidr
	}

	return lan
}
