package serverset

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/resource/fake"
	"github.com/crossplane/crossplane-runtime/pkg/test"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/kube"
)

func Test_serverSet_Create(t *testing.T) {
	objList := func(obj client.ObjectList) error { return nil }
	fakeObj := func(obj client.Object) error {
		switch obj.(type) {
		case *v1alpha1.Server:
			res := obj.(*v1alpha1.Server)
			res.Status.AtProvider.State = "AVAILABLE"
			res.Status.AtProvider.ServerID = "test-id"
		case *v1alpha1.Volume:
			res := obj.(*v1alpha1.Volume)
			res.Status.AtProvider.State = "AVAILABLE"
			res.Status.AtProvider.VolumeID = "test-id"
		case *v1alpha1.Nic:
			res := obj.(*v1alpha1.Nic)
			res.Status.AtProvider.State = "AVAILABLE"
			res.Status.AtProvider.NicID = "test-id"
		}
		return nil
	}
	type fields struct {
		kube                 client.Client
		secretRefFieldPath   string
		toggleFieldPath      string
		mg                   resource.Managed
		bootVolumeController kubeBootVolumeControlManager
		nicController        kubeNicControlManager
		serverController     kubeServerControlManager
		log                  logging.Logger
	}
	type args struct {
		ctx context.Context
		cr  *v1alpha1.ServerSet
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    managed.ExternalCreation
		wantErr error
	}{
		{
			name: "server set successfully created",
			fields: fields{
				kube: &test.MockClient{
					MockList: test.NewMockListFn(nil, objList),
				},
				log: logging.NewNopLogger(),
				mg:  &fake.Managed{},
				bootVolumeController: &kubeBootVolumeController{
					kube: &test.MockClient{
						MockCreate: test.NewMockCreateFn(nil),
						MockList:   test.NewMockListFn(nil, objList),
						MockGet:    test.NewMockGetFn(nil, fakeObj),
					},
					log: logging.NewNopLogger(),
				},
				serverController: &kubeServerController{
					kube: &test.MockClient{
						MockCreate: test.NewMockCreateFn(nil),
						MockList:   test.NewMockListFn(nil, objList),
						MockGet:    test.NewMockGetFn(nil, fakeObj),
					},
					log: logging.NewNopLogger(),
				},
				nicController: &kubeNicController{
					kube: &test.MockClient{
						MockCreate: test.NewMockCreateFn(nil),
						MockList:   test.NewMockListFn(nil, objList),
						MockGet:    test.NewMockGetFn(nil, fakeObj),
					},
					log: logging.NewNopLogger(),
				},
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSet(),
			},
			want: managed.ExternalCreation{
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr: nil,
		},
		{
			name: "failed to create, too many volumes",
			fields: fields{
				kube: &test.MockClient{
					MockList: test.NewMockListFn(nil, func(obj client.ObjectList) error {
						vols := obj.(*v1alpha1.VolumeList)
						vols.Items = append(vols.Items, v1alpha1.Volume{})
						vols.Items = append(vols.Items, v1alpha1.Volume{})
						return nil
					}),
				},
				log: logging.NewNopLogger(),
				mg:  &fake.Managed{},
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSet(),
			},
			want: managed.ExternalCreation{
				ConnectionDetails: nil,
			},
			wantErr: errors.New("found too many volumes for index 0"),
		},
		{
			name: "external create failed",
			fields: fields{
				kube: &test.MockClient{
					MockList: test.NewMockListFn(nil, objList),
				},
				log: logging.NewNopLogger(),
				mg:  &fake.Managed{},
				bootVolumeController: &kubeBootVolumeController{
					kube: &test.MockClient{
						MockCreate: test.NewMockCreateFn(nil),
						MockList:   test.NewMockListFn(nil, objList),
						MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
							volume := obj.(*v1alpha1.Volume)
							annotations := make(map[string]string)
							annotations[meta.AnnotationKeyExternalCreateFailed] = "alas, we failed"
							volume.SetAnnotations(annotations)
							return nil
						}),
						MockDelete: test.NewMockDeleteFn(nil),
					},
					log: logging.NewNopLogger(),
				},
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSet(),
			},
			wantErr: fmt.Errorf("while waiting for BootVolume to be populated %w ", kube.ErrExternalCreateFailed),
		},
		{
			name: "bad cloud init",
			fields: fields{
				kube: &test.MockClient{
					MockList: test.NewMockListFn(nil, objList),
				},
				log: logging.NewNopLogger(),
				mg:  &fake.Managed{},
				bootVolumeController: &kubeBootVolumeController{
					kube: &test.MockClient{
						MockCreate: test.NewMockCreateFn(nil),
						MockList:   test.NewMockListFn(nil, objList),
						MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
							volume := obj.(*v1alpha1.Volume)
							annotations := make(map[string]string)
							annotations[meta.AnnotationKeyExternalCreateFailed] = "alas, we failed"
							volume.SetAnnotations(annotations)
							return nil
						}),
						MockDelete: test.NewMockDeleteFn(nil),
					},
					log: logging.NewNopLogger(),
				},
			},
			args: args{
				ctx: context.Background(),
				cr: &v1alpha1.ServerSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: serverSetName,
						Annotations: map[string]string{
							"crossplane.io/external-name": serverSetName,
						},
					},
					Spec: v1alpha1.ServerSetSpec{
						ForProvider: v1alpha1.ServerSetParameters{
							Replicas: noReplicas,
							Template: v1alpha1.ServerSetTemplate{
								Metadata: v1alpha1.ServerSetMetadata{
									Name: "servername",
								},
								Spec: v1alpha1.ServerSetTemplateSpec{
									Cores:     serverSetCores,
									RAM:       serverSetRAM,
									CPUFamily: serverSetCPUFamily,
									NICs: []v1alpha1.ServerSetTemplateNIC{
										{
											Name:      "nic1",
											IPv4:      "10.0.0.2/24",
											DHCP:      true,
											Reference: "data",
										},
									},
								},
							},
							BootVolumeTemplate: v1alpha1.BootVolumeTemplate{
								Metadata: v1alpha1.ServerSetBootVolumeMetadata{
									Name: "bootvolumename",
								},
								Spec: v1alpha1.ServerSetBootVolumeSpec{
									Size:     bootVolumeSize,
									Image:    bootVolumeImage,
									Type:     bootVolumeType,
									UserData: "malformed",
								},
							},
						},
					},
				},
			},
			wantErr: fmt.Errorf("while creating cloud init patcher for BootVolume bootvolumename-0-0 failed to decode base64 (illegal base64 data at input byte 8)"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &external{
				kube:                 tt.fields.kube,
				bootVolumeController: tt.fields.bootVolumeController,
				nicController:        tt.fields.nicController,
				serverController:     tt.fields.serverController,
				log:                  tt.fields.log,
			}

			got, err := e.Create(tt.args.ctx, tt.args.cr)
			if tt.wantErr != nil {
				assert.ErrorContains(t, err, tt.wantErr.Error())
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
