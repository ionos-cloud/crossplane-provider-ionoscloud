package serverset

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/google/go-cmp/cmp"

	ionoscloud "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	ip                        = "10.0.0.2/24"
	vnetID                    = "679070ab-1ebc-46ef-b9f7-c43c1ed9f6e9"
	serverSetNicIndexLabel    = "ionoscloud.com/serverset-nic-index"
	serverSetNicVersionLabel  = "ionoscloud.com/serverset-nic-version"
	nicWithVNetAndIPV4Name    = "serverset-nic1-nic-1-0-0"
	nicWithoutVNetAndIPV4Name = "serverset-nic2-nic-1-0-0"
	nicID                     = "bc59d87e-17cc-4313-b55b-6603884f9d97"
	serverID                  = "07a7e712-fc36-43ca-bc8f-76c05861ff8b"
	lanName                   = "lan1"
	lanID                     = "1"
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
				kube: fakeKubeClient(interceptor.Funcs{Create: createWithVNetAndIPV4ReturnsErrorForUnexpectedNICs, Get: getPopulatesFieldsAndReturnsNoError}),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx:          context.Background(),
				cr:           createServerSetWithVNetAndIPV4(),
				serverID:     serverID,
				lanName:      lanName,
				replicaIndex: 1,
				nicIndex:     0,
				version:      0,
			},
			want:    *createWantedNicWithVNetAndIPV4(),
			wantErr: false,
		},
		{
			name: "The fields vnet and ipv4 are optional",
			fields: fields{
				kube: fakeKubeClient(interceptor.Funcs{Create: createWithoutVNetAndIPV4ReturnsErrorForUnexpectedNICs, Get: getPopulatesFieldsAndReturnsNoError}),
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
			want:    *createWantedNicWithoutVNetAndIPV4(),
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

func createServerSetWithVNetAndIPV4() *v1alpha1.ServerSet {
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
	nic.Spec.ForProvider.ServerCfg.ServerID = serverID
	nic.Spec.ForProvider.LanCfg.LanID = lanID
	nic.Spec.ForProvider.Dhcp = true
}

func populateVNetAndIPV4(nic *v1alpha1.Nic) {
	nic.Spec.ForProvider.Vnet = vnetID
	nic.Spec.ForProvider.IpsCfg.IPs = []string{ip}
}

func makeNicAvailable(nic *v1alpha1.Nic) {
	nic.Status.AtProvider.NicID = nicID
	nic.Status.AtProvider.State = ionoscloud.Available
}

func createWantedNicWithVNetAndIPV4() *v1alpha1.Nic {
	nic := &v1alpha1.Nic{}
	populateBasicNicMetadataAndSpec(nic, nicWithVNetAndIPV4Name)
	populateVNetAndIPV4(nic)
	makeNicAvailable(nic)
	return nic
}

func createWantedNicWithoutVNetAndIPV4() *v1alpha1.Nic {
	nic := &v1alpha1.Nic{}
	populateBasicNicMetadataAndSpec(nic, nicWithoutVNetAndIPV4Name)
	makeNicAvailable(nic)
	return nic
}

func createWithVNetAndIPV4ReturnsErrorForUnexpectedNICs(_ context.Context, _ client.WithWatch, obj client.Object, _ ...client.CreateOption) error {
	expectedNic := &v1alpha1.Nic{}
	populateBasicNicMetadataAndSpec(expectedNic, nicWithVNetAndIPV4Name)
	populateVNetAndIPV4(expectedNic)
	if diff := cmp.Diff(obj.(*v1alpha1.Nic), expectedNic); diff != "" {
		return fmt.Errorf("create was called with an unexpected NIC.\n Expected %#v.\n Got %#v", expectedNic, obj.(*v1alpha1.Nic))
	}
	return nil
}

func createWithoutVNetAndIPV4ReturnsErrorForUnexpectedNICs(_ context.Context, _ client.WithWatch, obj client.Object, _ ...client.CreateOption) error {
	expectedNic := &v1alpha1.Nic{}
	populateBasicNicMetadataAndSpec(expectedNic, nicWithoutVNetAndIPV4Name)
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
	case nicWithVNetAndIPV4Name:
		nic := obj.(*v1alpha1.Nic)
		populateBasicNicMetadataAndSpec(nic, nicWithVNetAndIPV4Name)
		populateVNetAndIPV4(nic)
		makeNicAvailable(nic)
	case nicWithoutVNetAndIPV4Name:
		nic := obj.(*v1alpha1.Nic)
		populateBasicNicMetadataAndSpec(nic, nicWithoutVNetAndIPV4Name)
		makeNicAvailable(nic)
	}
	return nil
}
