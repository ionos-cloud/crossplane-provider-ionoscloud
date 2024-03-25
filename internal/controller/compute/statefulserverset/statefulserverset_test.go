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
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
)

func fakeKubeClientWithObjs(objs ...client.Object) client.WithWatch {
	scheme := runtime.NewScheme()
	v1.AddToScheme(scheme)       // Add the core k8s types to the Scheme
	v1alpha1.AddToScheme(scheme) // Add our custom types from v1alpha to the Scheme
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
}

func fakeKubeClientWithFunc(funcs interceptor.Funcs) client.WithWatch {
	scheme := runtime.NewScheme()
	v1.AddToScheme(scheme)       // Add the core k8s types to the Scheme
	v1alpha1.AddToScheme(scheme) // Add our custom types from v1alpha to the Scheme
	return fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(funcs).Build()
}

const create = "Create"
const ensure = "Ensure"

type fakeKubeServerSetController struct {
	methodCallCount map[string]int
}

func (f *fakeKubeServerSetController) Create(ctx context.Context, cr *v1alpha1.StatefulServerSet) (*v1alpha1.ServerSet, error) {
	f.methodCallCount[create]++
	return nil, nil
}

func (f *fakeKubeServerSetController) Ensure(ctx context.Context, cr *v1alpha1.StatefulServerSet) error {
	f.methodCallCount[ensure]++
	return nil
}

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
