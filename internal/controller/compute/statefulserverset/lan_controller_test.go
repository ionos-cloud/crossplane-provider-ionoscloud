package statefulserverset

import (
	"context"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
)

func Test_kubeLANController_Update(t *testing.T) {

	type fields struct {
		kube client.Client
		log  logging.Logger
	}
	type args struct {
		ctx      context.Context
		cr       *v1alpha1.StatefulServerSet
		lanIndex int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    v1alpha1.Lan
		wantErr error
	}{
		{
			name: "do not consider ipv6cidr as a change when value on spec is AUTO",
			fields: fields{
				kube: fakeKubeClientWithObjs(createCustomerLAN()),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr:  createSSSet(),
			},
			want:    v1alpha1.Lan{},
			wantErr: nil,
		},
		{
			name: "consider ipv6cidr as a change when value is not AUTO",
			fields: fields{
				kube: fakeKubeClientWithObjs(createLAN(v1alpha1.LanParameters{
					Name:     customerLanName,
					Ipv6Cidr: "2001:db8::/64",
				})),
				log: logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr: createSSSetWithCustomerLanUpdated(v1alpha1.StatefulServerSetLanSpec{
					IPv6cidr: "1000:db8::/64",
				}),
			},
			want:    v1alpha1.Lan{},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &kubeLANControllerWithIsAvailableStubbed{
				KubeLANController: kubeLANController{
					kube: tt.fields.kube,
					log:  tt.fields.log,
				},
			}
			got, err := k.KubeLANController.Update(tt.args.ctx, tt.args.cr, tt.args.lanIndex)
			assert.Equal(t, tt.wantErr, err)
			assert.Equalf(t, tt.want, got, "Update(%v, %v, %v)", tt.args.ctx, tt.args.cr, tt.args.lanIndex)
		})
	}
}
