package serverset

import (
	"context"
	"reflect"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	_ "sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	computev1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
)

func GetVolumeTest(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	vol := obj.(*v1alpha1.Volume)
	vol.Status.AtProvider.State = "AVAILABLE"
	vol.Status.AtProvider.VolumeID = "uuid"
	return nil
}
func Test_kubeBootVolumeController_Create(t *testing.T) {
	scheme, _ := computev1alpha1.SchemeBuilder.Build()
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(interceptor.Funcs{Get: GetVolumeTest}).Build()
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
			name: "TestCreateBootVolumeExpectSuccess",
			fields: fields{
				kube: fakeClient,
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr: &v1alpha1.ServerSet{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ServerSet",
						APIVersion: "v1alpha1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "serverset",
						Namespace: "",
					},
					Spec:   v1alpha1.ServerSetSpec{},
					Status: v1alpha1.ServerSetStatus{},
				},
				replicaIndex: 0,
				version:      0,
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
