package statefulserverset

import (
	"context"
	"testing"

	cv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
)

func Test_statefulServerSetController_Create(t *testing.T) {
	SSetCtrl := &fakeKubeServerSetController{
		methodCallCount: map[string]int{
			create: 0,
			ensure: 0,
		},
	}
	type fields struct {
		kube           client.Client
		log            logging.Logger
		SSetController kubeSSetControlManager
	}
	type args struct {
		ctx context.Context
		mg  resource.Managed
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    managed.ExternalCreation
		wantErr bool
	}{
		{
			name: "stateful server set is created successfully",
			fields: fields{
				kube:           fakeKubeClientWithObjs(),
				log:            logging.NewNopLogger(),
				SSetController: SSetCtrl,
			},
			args: args{
				ctx: context.Background(),
				mg:  &v1alpha1.StatefulServerSet{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			},
			want:    managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &external{
				kube:           tt.fields.kube,
				log:            tt.fields.log,
				SSetController: tt.fields.SSetController,
			}
			got, err := c.Create(tt.args.ctx, tt.args.mg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}

			assert.Equal(t, tt.want, got)

			cr := tt.args.mg.(*v1alpha1.StatefulServerSet)
			assert.Equal(t, cv1.ReasonCreating, cr.Status.ConditionedStatus.Conditions[0].Reason)
			assert.Equal(t, cv1.TypeReady, cr.Status.ConditionedStatus.Conditions[0].Type)
			assert.Equal(t, v1.ConditionFalse, cr.Status.ConditionedStatus.Conditions[0].Status)
			assert.Equal(t, "test", cr.ObjectMeta.Name)
			assert.Equal(t, SSetCtrl.methodCallCount[create], 0)
			assert.Equal(t, SSetCtrl.methodCallCount[ensure], 1)
		})
	}
}

func Test_statefulServerSetController_Observe(t *testing.T) {
	type fields struct {
		kube                 client.Client
		log                  logging.Logger
		LANController        kubeLANControlManager
		dataVolumeController kubeDataVolumeControlManager
	}
	type args struct {
		ctx context.Context
		mg  resource.Managed
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    managed.ExternalObservation
		wantErr bool
	}{
		{
			name: "external name not set on StatefulServerSet, then return empty ExternalObservation",
			fields: fields{
				kube: fakeKubeClientWithObjs(),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				mg:  &v1alpha1.StatefulServerSet{},
			},
			want:    managed.ExternalObservation{},
			wantErr: false,
		},
		{
			name: "LANs and Data Volumes not yet created, then StatefulServerSet does not exist and is not up to date",
			fields: fields{
				kube:                 fakeKubeClientWithSSetRelatedObjects(),
				log:                  logging.NewNopLogger(),
				LANController:        fakeKubeLANController{LanList: v1alpha1.LanList{}},
				dataVolumeController: fakeKubeDataVolumeController{VolumeList: v1alpha1.VolumeList{}},
			},
			args: args{
				ctx: context.Background(),
				mg:  createSSSet(),
			},
			want: managed.ExternalObservation{
				ResourceExists:    false,
				ResourceUpToDate:  false,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr: false,
		},
		{
			name: "LANs, Data Volumes and ServerSet up to date, then StatefulServerSet exists and is up to date",
			fields: fields{
				kube:                 fakeKubeClientWithSSetRelatedObjects(),
				log:                  logging.NewNopLogger(),
				LANController:        fakeKubeLANController{LanList: createLanList()},
				dataVolumeController: fakeKubeDataVolumeController{VolumeList: createVolumeList()},
			},
			args: args{
				ctx: context.Background(),
				mg:  createSSSet(),
			},
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  true,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr: false,
		},
		{
			name: "LANs not up to date (field=Public), then StatefulServerSet exists and is not up to date",
			fields: fields{
				kube: fakeKubeClientWithSSetRelatedObjects(),
				log:  logging.NewNopLogger(),
				LANController: fakeKubeLANController{
					LanList: createLanListNotUpToDate(
						LANFieldsUpToDate{isIpv6CidrUpToDate: true},
					),
				},
				dataVolumeController: fakeKubeDataVolumeController{
					VolumeList: createVolumeList(),
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  createSSSet(),
			},
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  false,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr: false,
		},
		{
			name: "LANs not up to date (field=Ipv6Cidr), then StatefulServerSet exists and is not up to date",
			fields: fields{
				kube: fakeKubeClientWithSSetRelatedObjects(),
				log:  logging.NewNopLogger(),
				LANController: fakeKubeLANController{
					LanList: createLanListNotUpToDate(
						LANFieldsUpToDate{isPublicUpToDate: true},
					),
				},
				dataVolumeController: fakeKubeDataVolumeController{
					VolumeList: createVolumeList(),
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  createSSSet(),
			},
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  false,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr: false,
		},
		{
			name: "LANs not up to date (all fields), then StatefulServerSet exists and is not up to date",
			fields: fields{
				kube: fakeKubeClientWithSSetRelatedObjects(),
				log:  logging.NewNopLogger(),
				LANController: fakeKubeLANController{
					LanList: createLanListNotUpToDate(LANFieldsUpToDate{}),
				},
				dataVolumeController: fakeKubeDataVolumeController{
					VolumeList: createVolumeList(),
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  createSSSet(),
			},
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  false,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr: false,
		},
		{
			name: "Data Volumes not up to date (field=size), then StatefulServerSet exists and is not up to date",
			fields: fields{
				kube: fakeKubeClientWithSSetRelatedObjects(),
				log:  logging.NewNopLogger(),
				LANController: fakeKubeLANController{
					LanList: createLanList(),
				},
				dataVolumeController: fakeKubeDataVolumeController{
					VolumeList: createVolumeListNotUpToDate(VolumeFieldUpToDate{}),
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  createSSSet(),
			},
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  false,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr: false,
		},
		{
			name: "LANs and Data Volumes not up to date, then StatefulServerSet exists and is not up to date",
			fields: fields{
				kube: fakeKubeClientWithObjs(createSSet()),
				log:  logging.NewNopLogger(),
				LANController: fakeKubeLANController{
					LanList: createLanListNotUpToDate(LANFieldsUpToDate{}),
				},
				dataVolumeController: fakeKubeDataVolumeController{
					VolumeList: createVolumeListNotUpToDate(VolumeFieldUpToDate{}),
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  createSSSet(),
			},
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  false,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr: false,
		},
		{
			name: "ServerSet not up to date (BootVolume), then StatefulServerSet exists and is not up to date",
			fields: fields{
				kube: fakeKubeClientWithObjs(
					createSSet(), createServer1(), createServer2(),
					createBootVolume1NotUpToDate(), createBootVolume2NotUpToDate(),
					createNIC1(), createNIC2(),
				),
				log: logging.NewNopLogger(),
				LANController: fakeKubeLANController{
					LanList: createLanList(),
				},
				dataVolumeController: fakeKubeDataVolumeController{
					VolumeList: createVolumeList(),
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  createSSSet(),
			},
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  false,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr: false,
		},
		{
			name: "ServerSet not up to date (Servers), then StatefulServerSet exists and is not up to date",
			fields: fields{
				kube: fakeKubeClientWithObjs(
					createSSet(), createServer1NotUpToDate(), createServer2NotUpToDate(),
					createBootVolume1(), createBootVolume2(),
					createNIC1(), createNIC2(),
				),
				log: logging.NewNopLogger(),
				LANController: fakeKubeLANController{
					LanList: createLanList(),
				},
				dataVolumeController: fakeKubeDataVolumeController{
					VolumeList: createVolumeList(),
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  createSSSet(),
			},
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  false,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr: false,
		},
		{
			name: "ServerSet not up to date (NICs), then StatefulServerSet exists and is not up to date",
			fields: fields{
				kube: fakeKubeClientWithObjs(
					createSSet(), createServer1(), createServer2(),
					createBootVolume1(), createBootVolume2(),
					createNIC1(),
				),
				log: logging.NewNopLogger(),
				LANController: fakeKubeLANController{
					LanList: createLanList(),
				},
				dataVolumeController: fakeKubeDataVolumeController{
					VolumeList: createVolumeList(),
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  createSSSet(),
			},
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  false,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &external{
				kube:                 tt.fields.kube,
				log:                  tt.fields.log,
				LANController:        tt.fields.LANController,
				dataVolumeController: tt.fields.dataVolumeController,
			}
			got, err := c.Observe(tt.args.ctx, tt.args.mg)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Observer() error = %v, wantErr = %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_areLansUpToDate(t *testing.T) {
	type want = struct {
		creationUpToDate bool
		areUpToDate      bool
	}

	tests := []struct {
		name  string
		sSSet *v1alpha1.StatefulServerSet
		lans  []v1alpha1.Lan
		want  want
	}{
		{
			name:  "No LANs",
			sSSet: createSSSet(),
			lans:  []v1alpha1.Lan{},
			want: want{
				creationUpToDate: false,
				areUpToDate:      false,
			},
		},
		{
			name:  "Different number of LANs",
			sSSet: createSSSet(),
			lans:  []v1alpha1.Lan{*createLanDefault()},
			want: want{
				creationUpToDate: false,
				areUpToDate:      false,
			},
		},
		{
			name:  "Empty StatefulServerSet",
			sSSet: &v1alpha1.StatefulServerSet{},
			lans:  createLanList().Items,
			want: want{
				creationUpToDate: false,
				areUpToDate:      false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creationUpToDate, areUpToDate := areLansUpToDate(tt.sSSet, tt.lans)
			assert.Equal(t, tt.want.creationUpToDate, creationUpToDate)
			assert.Equal(t, tt.want.areUpToDate, areUpToDate)
		})
	}
}
