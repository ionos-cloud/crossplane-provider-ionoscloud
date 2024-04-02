package statefulserverset

import (
	"context"
	"testing"
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
)

var ErrSomethingWentWrong = errors.New("something went wrong")

func createReturnsError(ctx context.Context, client client.WithWatch, obj client.Object,
	opts ...client.CreateOption) error {
	return ErrSomethingWentWrong
}

func getReturnsError(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object,
	opts ...client.GetOption) error {
	return ErrSomethingWentWrong
}

func getReturnsSSet(ctx context.Context, watch client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	SSet := obj.(*v1alpha1.ServerSet)
	SSet.ObjectMeta.ResourceVersion = "1"
	return nil
}

func Test_kubeServerSetController_Ensure(t *testing.T) {
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	type fields struct {
		kube client.Client
		log  logging.Logger
	}
	type args struct {
		ctx context.Context
		cr  *v1alpha1.StatefulServerSet
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr error
	}{
		{
			name: "server set not yet created, then return no error",
			fields: fields{
				kube: fakeKubeClientWithObjs(),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx: ctxWithTimeout,
				cr:  &v1alpha1.StatefulServerSet{ObjectMeta: metav1.ObjectMeta{Name: statefulServerSetName}},
			},
			wantErr: context.DeadlineExceeded,
		},
		{
			name: "error received on server set creation, then return error",
			fields: fields{
				kube: fakeKubeClientWithFunc(interceptor.Funcs{Get: getReturnsError}),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr:  &v1alpha1.StatefulServerSet{ObjectMeta: metav1.ObjectMeta{Name: statefulServerSetName}},
			},
			wantErr: ErrSomethingWentWrong,
		},
		{
			name: "error received on reading the server set, then return error",
			fields: fields{
				kube: fakeKubeClientWithFunc(interceptor.Funcs{Create: createReturnsError}),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr:  &v1alpha1.StatefulServerSet{ObjectMeta: metav1.ObjectMeta{Name: statefulServerSetName}},
			},
			wantErr: ErrSomethingWentWrong,
		},
		{
			name: "server set already exists, then return no error",
			fields: fields{
				kube: fakeKubeClientWithFunc(interceptor.Funcs{Get: getReturnsSSet}),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr:  &v1alpha1.StatefulServerSet{ObjectMeta: metav1.ObjectMeta{Name: statefulServerSetName}},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &kubeServerSetController{
				kube: tt.fields.kube,
				log:  tt.fields.log,
			}
			err := k.Ensure(tt.args.ctx, tt.args.cr)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func Test_kubeServerSetController_Create(t *testing.T) {
	type fields struct {
		kube client.Client
		log  logging.Logger
	}
	type args struct {
		ctx context.Context
		cr  *v1alpha1.StatefulServerSet
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *v1alpha1.ServerSet
		wantErr error
	}{
		{
			name: "server set created successfully, then server set is returned and no error",
			fields: fields{
				kube: fakeKubeClientWithObjs(),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr: &v1alpha1.StatefulServerSet{
					ObjectMeta: metav1.ObjectMeta{Name: statefulServerSetName},
					Spec: v1alpha1.StatefulServerSetSpec{
						ResourceSpec: xpv1.ResourceSpec{
							ManagementPolicies:      []xpv1.ManagementAction{"*"},
							ProviderConfigReference: &xpv1.Reference{Name: "example"},
						},
						ForProvider: v1alpha1.StatefulServerSetParameters{
							Replicas: serverSetNrOfReplicas,
							DatacenterCfg: v1alpha1.DatacenterConfig{
								DatacenterIDRef: &xpv1.Reference{Name: datacenterName},
							},
							Template: createSSetTemplate(),
						},
					},
				},
			},
			want: createSSet(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &kubeServerSetController{
				kube: tt.fields.kube,
				log:  tt.fields.log,
			}
			got, err := k.Create(tt.args.ctx, tt.args.cr)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
