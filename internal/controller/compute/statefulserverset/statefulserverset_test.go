package statefulserverset

import (
	"context"
	"reflect"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
)

func fakeKubeClientObjs(objs ...client.Object) client.WithWatch {
	scheme := runtime.NewScheme()
	v1.AddToScheme(scheme)       // Add the core k8s types to the Scheme
	v1alpha1.AddToScheme(scheme) // Add our custom types from v1alpha to the Scheme
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
}

func Test_statefulServerSetController_Create(t *testing.T) {
	type fields struct {
		kube client.Client
		log  logging.Logger
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
			name: "stateful server set is created succesfully",
			fields: fields{
				kube: fakeKubeClientObjs(),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				mg:  &v1alpha1.StatefulServerSet{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &external{
				kube: tt.fields.kube,
				log:  tt.fields.log,
			}
			got, err := c.Create(tt.args.ctx, tt.args.mg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Create() got = %v, want = %v", got, tt.want)
			}
		})
	}
}