package serverset

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	_ "sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	computev1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
)

func fakeKubeClient(functions interceptor.Funcs) client.WithWatch {
	scheme, _ := computev1alpha1.SchemeBuilder.Build()
	return fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(functions).Build()
}

func getVolumePopulateStatusTest(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	vol := obj.(*v1alpha1.Volume)
	vol.Status.AtProvider.State = "AVAILABLE"
	vol.Status.AtProvider.VolumeID = "uuid"
	return nil
}

func createVolumeReturnsErrorTest(ctx context.Context, client client.WithWatch, obj client.Object,
	opts ...client.CreateOption) error {
	return errors.New("something went wrong")
}

func Test_kubeBootVolumeController_Create(t *testing.T) {
	type fields struct {
		kube client.Client
		log  logging.Logger
	}
	type args struct {
		ctx          context.Context
		cr           *v1alpha1.ServerSet
		replicaIndex int
		version      int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    v1alpha1.Volume
		wantErr bool
	}{
		{
			name: "expect success",
			fields: fields{
				kube: fakeKubeClient(interceptor.Funcs{Get: getVolumePopulateStatusTest}),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr:  &v1alpha1.ServerSet{},
			},
			want: v1alpha1.Volume{
				Status: v1alpha1.VolumeStatus{
					AtProvider: computev1alpha1.VolumeObservation{
						VolumeID: "uuid",
						State:    "AVAILABLE",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "expect error",
			fields: fields{
				kube: fakeKubeClient(interceptor.Funcs{Create: createVolumeReturnsErrorTest}),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr:  &v1alpha1.ServerSet{},
			},
			want:    v1alpha1.Volume{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &kubeBootVolumeController{
				kube: tt.fields.kube,
				log:  tt.fields.log,
			}
			got, err := k.Create(tt.args.ctx, tt.args.cr, tt.args.replicaIndex, tt.args.version)
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
