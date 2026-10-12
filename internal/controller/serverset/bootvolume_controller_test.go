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
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/ccpatch/substitution"
)

func fakeKubeClientFuncs(functions interceptor.Funcs) client.WithWatch {
	scheme, _ := computev1alpha1.SchemeBuilder.Build()
	return fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(functions).Build()
}

func getVolumePopulateStatus(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	vol := obj.(*v1alpha1.Volume)
	vol.Status.AtProvider.State = "AVAILABLE"
	vol.Status.AtProvider.VolumeID = "uuid"
	return nil
}

func createVolumeReturnsError(ctx context.Context, client client.WithWatch, obj client.Object,
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
			name: "expect success and status is populated",
			fields: fields{
				kube: fakeKubeClientFuncs(interceptor.Funcs{Get: getVolumePopulateStatus}),
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
			name: "kube object creation failed expect error",
			fields: fields{
				kube: fakeKubeClientFuncs(interceptor.Funcs{Create: createVolumeReturnsError}),
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

// Test_kubeBootVolumeController_setPatcher_withSubstitutions exercises the loop that reads a
// pre-existing substitution value out of the global state and writes it into the substitution
// configmap - the path used when a value was already computed on an earlier reconcile.
func Test_kubeBootVolumeController_setPatcher_withSubstitutions(t *testing.T) {
	cr := createBasicServerSet()
	cr.Name = "sset-setpatcher-test"
	cr.Spec.ForProvider.BootVolumeTemplate.Spec.Substitutions = []v1alpha1.Substitution{
		{Type: "unregistered-test-type", Key: "MY_KEY"},
	}

	identifier := substitution.Identifier("boot-1-1")
	state := getOrInitGlobalState(cr.Name)
	state.Set(identifier, "MY_KEY", "10.0.0.5")

	k := &kubeBootVolumeController{
		log:           logging.NewNopLogger(),
		mapController: newTestConfigmapController(),
	}

	_, _ = k.setPatcher(context.Background(), cr, 1, 1, "boot-1-1", "boot-1-0", fakeKubeClientObjs())

	cfgMap, err := k.mapController.Get(context.Background(), "sset-setpatcher-test", "default")
	if err != nil {
		t.Fatalf("expected substitution configmap to have been created, got error: %v", err)
	}
	if got := cfgMap.Data["1.1.MY_KEY"]; got != "10.0.0.5" {
		t.Errorf("substitution configmap has %q for key 1.1.MY_KEY, want %q", got, "10.0.0.5")
	}
}
