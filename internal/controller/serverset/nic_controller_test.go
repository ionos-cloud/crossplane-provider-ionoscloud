package serverset

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/google/go-cmp/cmp"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	ip                        = "10.0.0.2/24"
	vnetId                    = "679070ab-1ebc-46ef-b9f7-c43c1ed9f6e9"
	serverSetNicIndexLabel    = "ionoscloud.com/serverset-nic-index"
	serverSetNicVersionLabel  = "ionoscloud.com/serverset-nic-version"
	nicWithVnetAndIpV4Name    = "serverset-nic1-nic-1-0-0"
	nicWithoutVnetAndIpV4Name = "serverset-nic2-nic-1-0-0"
	nicId                     = "bc59d87e-17cc-4313-b55b-6603884f9d97"
	serverId                  = "07a7e712-fc36-43ca-bc8f-76c05861ff8b"
	lanName                   = "lan1"
	lanId                     = "1"
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
				kube: fakeKubeClient(interceptor.Funcs{Create: createWithVNetAndIpV4ReturnsErrorForUnexpectedNICs, Get: getPopulatesFieldsAndReturnsNoError}),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx:          context.Background(),
				cr:           createServerSetWithVNetAndIpV4(),
				serverID:     serverId,
				lanName:      lanName,
				replicaIndex: 1,
				nicIndex:     0,
				version:      0,
			},
			want:    *createWantedNicWithVNetAndIpV4(),
			wantErr: false,
		},
		{
			name: "The fields vnet and ipv4 are optional",
			fields: fields{
				kube: fakeKubeClient(interceptor.Funcs{Create: createWithoutVNetAndIpV4ReturnsErrorForUnexpectedNICs, Get: getPopulatesFieldsAndReturnsNoError}),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx:          context.Background(),
				cr:           createServerSetWithoutVNetAndIpV4(),
				serverID:     serverId,
				lanName:      lanName,
				replicaIndex: 1,
				nicIndex:     0,
				version:      0,
			},
			want:    *createWantedNicWithoutVNetAndIpV4(),
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
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Create() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func createServerSetWithVNetAndIpV4() *v1alpha1.ServerSet {
	s := createBasicServerSet()
	for nicIndex := range s.Spec.ForProvider.Template.Spec.NICs {
		s.Spec.ForProvider.Template.Spec.NICs[nicIndex].VNetID = vnetId
	}
	return s
}

func createServerSetWithoutVNetAndIpV4() *v1alpha1.ServerSet {
	s := createBasicServerSet()
	s.Spec.ForProvider.Template.Spec.NICs = []v1alpha1.ServerSetTemplateNIC{
		{
			Name:      "nic2",
			DHCP:      true,
			Reference: "data",
		},
	}
	return s
}

func populateBasicNicMetadataAndSpec(nic *v1alpha1.Nic, nicName string) {
	nic.ObjectMeta.Name = nicName
	nic.ObjectMeta.Labels = map[string]string{
		serverSetLabel:           serverSetName,
		serverSetNicIndexLabel:   "1",
		serverSetNicVersionLabel: "0",
	}
	nic.Spec.ForProvider.Name = nicName
	nic.Spec.ForProvider.ServerCfg.ServerID = serverId
	nic.Spec.ForProvider.LanCfg.LanID = lanId
	nic.Spec.ForProvider.Dhcp = true
}

func populateVNetAndIpV4(nic *v1alpha1.Nic) {
	nic.Spec.ForProvider.Vnet = vnetId
	nic.Spec.ForProvider.IpsCfg.IPs = []string{ip}
}

func makeNicAvailable(nic *v1alpha1.Nic) {
	nic.Status.AtProvider.NicID = nicId
	nic.Status.AtProvider.State = ionoscloud.Available
}

func createWantedNicWithVNetAndIpV4() *v1alpha1.Nic {
	nic := &v1alpha1.Nic{}
	populateBasicNicMetadataAndSpec(nic, nicWithVnetAndIpV4Name)
	populateVNetAndIpV4(nic)
	makeNicAvailable(nic)
	return nic
}

func createWantedNicWithoutVNetAndIpV4() *v1alpha1.Nic {
	nic := &v1alpha1.Nic{}
	populateBasicNicMetadataAndSpec(nic, nicWithoutVnetAndIpV4Name)
	makeNicAvailable(nic)
	return nic
}

func createWithVNetAndIpV4ReturnsErrorForUnexpectedNICs(_ context.Context, _ client.WithWatch, obj client.Object, _ ...client.CreateOption) error {
	expectedNic := &v1alpha1.Nic{}
	populateBasicNicMetadataAndSpec(expectedNic, nicWithVnetAndIpV4Name)
	populateVNetAndIpV4(expectedNic)
	if diff := cmp.Diff(obj.(*v1alpha1.Nic), expectedNic); diff != "" {
		return errors.New(fmt.Sprintf("create was called with an unexpected NIC.\n Expected %#v.\n Got %#v", expectedNic, obj.(*v1alpha1.Nic)))
	}
	return nil
}

func createWithoutVNetAndIpV4ReturnsErrorForUnexpectedNICs(_ context.Context, _ client.WithWatch, obj client.Object, _ ...client.CreateOption) error {
	expectedNic := &v1alpha1.Nic{}
	populateBasicNicMetadataAndSpec(expectedNic, nicWithoutVnetAndIpV4Name)
	if diff := cmp.Diff(obj.(*v1alpha1.Nic), expectedNic); diff != "" {
		return errors.New(fmt.Sprintf("create was called with an unexpected NIC.\n Expected %#v \n Got %#v", expectedNic, obj.(*v1alpha1.Nic)))
	}
	return nil
}

func getPopulatesFieldsAndReturnsNoError(_ context.Context, _ client.WithWatch, key client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	if key.Name == lanName {
		lan := obj.(*v1alpha1.Lan)
		lan.Status.AtProvider.LanID = lanId
	} else if key.Name == nicWithVnetAndIpV4Name {
		nic := obj.(*v1alpha1.Nic)
		populateBasicNicMetadataAndSpec(nic, nicWithVnetAndIpV4Name)
		populateVNetAndIpV4(nic)
		makeNicAvailable(nic)
	} else if key.Name == nicWithoutVnetAndIpV4Name {
		nic := obj.(*v1alpha1.Nic)
		populateBasicNicMetadataAndSpec(nic, nicWithoutVnetAndIpV4Name)
		makeNicAvailable(nic)
	}
	return nil
}
