/*
Copyright 2022 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package serverset

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"

	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
)

const (
	bootVolumeSize       = 100
	bootVolumeType       = "HDD"
	bootVolumeImage      = "image"
	bootVolumeNamePrefix = "boot-volume-"

	delete     = "Delete"
	ensure     = "Ensure"
	ensureNICs = "EnsureNICs"
	get        = "Get"
	update     = "Update"

	noReplicas = 2

	server1Name        = "serverset-server-0-0"
	server2Name        = "serverset-server-1-0"
	serverSetCPUFamily = "AMD_OPTERON"
	serverSetCores     = 2
	serverSetRAM       = 4096
	serverSetName      = "serverset"

	reconcileErrorMsg = "some reconcile error happened"
)

type ServiceMethodName string

const (
	kubeUpdate       ServiceMethodName = "Client.Update"
	serverEnsure     ServiceMethodName = "kubeServerController.Ensure"
	serverDelete     ServiceMethodName = "kubeServerController.Delete"
	serverGet        ServiceMethodName = "kubeServerController.Get"
	serverUpdate     ServiceMethodName = "kubeServerController.Update"
	bootVolumeEnsure ServiceMethodName = "kubeBootVolumeControlManager.Ensure"
	bootVolumeDelete ServiceMethodName = "kubeBootVolumeControlManager.Delete"
	bootVolumeGet    ServiceMethodName = "kubeBootVolumeControlManager.Get"
	nicEnsureNICs    ServiceMethodName = "kubeNicControlManager.EnsureNICs"
	nicDelete        ServiceMethodName = "kubeNicControlManager.Delete"
)

var errAnErrorWasReceived = errors.New("an error was received")

type kubeBootVolumeControlManagerFake struct {
	kubeBootVolumeControlManager
	mock.Mock
}

type kubeServerControlManagerFake struct {
	kubeServerControlManager
	mock.Mock
}

type kubeNicControlManagerFake struct {
	kubeNicControlManager
	mock.Mock
}

type fakeKubeClient struct {
	client.Client
	mock.Mock
	t *testing.T
}

func Test_serverSetController_Observe(t *testing.T) {
	type fields struct {
		kube client.Client
	}
	type args struct {
		ctx context.Context
		cr  *v1alpha1.ServerSet
	}

	server1 := createServer(server1Name)
	server2 := createServer(server2Name)
	nic1 := createNic(server1Name)
	nic2 := createNic(server2Name)
	bootVolume1 := createBootVolume(bootVolumeNamePrefix + server1Name)
	bootVolume2 := createBootVolume(bootVolumeNamePrefix + server2Name)

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    managed.ExternalObservation
		wantErr bool
	}{
		{
			name: "servers, nics and boot volumes created",
			fields: fields{
				kube: fakeKubeClientObjs(server1, server2, bootVolume1, bootVolume2, nic1, nic2),
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSet(),
			},
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  true,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr: false,
		},
		{
			name: "servers not created",
			fields: fields{
				kube: fakeKubeClientObjs(),
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSet(),
			},
			want: managed.ExternalObservation{
				ResourceExists:    false,
				ResourceUpToDate:  true,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr: false,
		},
		{
			name: "server CPU family not up to date",
			fields: fields{
				kube: fakeKubeClientObjs(server1, server2, bootVolume1, bootVolume2, nic1, nic2),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedServerSpec(v1alpha1.ServerSetTemplateSpec{
					CPUFamily: "INTEL_XEON",
					Cores:     serverSetCores,
					RAM:       serverSetRAM,
				}),
			},
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  false,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr: false,
		},
		{
			name: "server cores not up to date",
			fields: fields{
				kube: fakeKubeClientObjs(server1, server2, bootVolume1, bootVolume2, nic1, nic2),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedServerSpec(v1alpha1.ServerSetTemplateSpec{
					CPUFamily: serverSetCPUFamily,
					Cores:     10,
					RAM:       serverSetRAM,
				}),
			},
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  false,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr: false,
		},
		{
			name: "server RAM not up to date",
			fields: fields{
				kube: fakeKubeClientObjs(server1, server2, bootVolume1, bootVolume2, nic1, nic2),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedServerSpec(v1alpha1.ServerSetTemplateSpec{
					CPUFamily: serverSetCPUFamily,
					Cores:     serverSetCores,
					RAM:       8192,
				}),
			},
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  false,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr: false,
		},
		{
			name: "boot volume image is not up to date",
			fields: fields{
				kube: fakeKubeClientObjs(server1, server2, bootVolume1, bootVolume2, nic1, nic2),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedBootVolumeUsingDefaultStrategy(v1alpha1.ServerSetBootVolumeSpec{
					Size:  bootVolumeSize,
					Image: "newImage",
					Type:  bootVolumeType,
				}),
			},
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  false,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr: false,
		},
		{
			name: "boot volume size is not up to date",
			fields: fields{
				kube: fakeKubeClientObjs(server1, server2, bootVolume1, bootVolume2, nic1, nic2),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedBootVolumeUsingDefaultStrategy(v1alpha1.ServerSetBootVolumeSpec{
					Size:  300,
					Image: bootVolumeImage,
					Type:  bootVolumeType,
				}),
			},
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  false,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr: false,
		},
		{
			name: "boot volume type is not up to date",
			fields: fields{
				kube: fakeKubeClientObjs(server1, server2, bootVolume1, bootVolume2, nic1, nic2),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedBootVolumeUsingDefaultStrategy(v1alpha1.ServerSetBootVolumeSpec{
					Size:  bootVolumeSize,
					Image: bootVolumeImage,
					Type:  "SSD",
				}),
			},
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  false,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr: false,
		},
		{
			name: "servers < replica count",
			fields: fields{
				kube: fakeKubeClientObjs(server1),
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSet(),
			},
			want: managed.ExternalObservation{
				ResourceExists:    false,
				ResourceUpToDate:  true,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr: false,
		},
		{
			name: "nics not created",
			fields: fields{
				kube: fakeKubeClientObjs(server1, server2, bootVolume1, bootVolume2),
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSet(),
			},
			want: managed.ExternalObservation{
				ResourceExists:    false,
				ResourceUpToDate:  true,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr: false,
		},
		{
			name: "nr of nics not up to date",
			fields: fields{
				kube: fakeKubeClientObjs(server1, server2, bootVolume1, bootVolume2, nic1, nic2),
			},
			args: args{
				ctx: context.Background(),
				cr:  createServerSetWithNrOfNICsUpdated(),
			},
			want: managed.ExternalObservation{
				ResourceExists:    false,
				ResourceUpToDate:  true,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &external{
				kube: tt.fields.kube,
				log:  logging.NewNopLogger(),
			}

			got, err := e.Observe(tt.args.ctx, tt.args.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Observe() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equalf(t, tt.want, got, "Observe() mismatch")
		})
	}
}

func Test_serverSetController_ServerSetObservation(t *testing.T) {
	type fields struct {
		kube client.Client
	}
	type args struct {
		ctx context.Context
		cr  *v1alpha1.ServerSet
	}

	server1 := createServer("serverset-server-0-0")
	server2 := createServer("serverset-server-1-0")

	serverWithErrorStatus := createServer("serverset-server-1-0")
	serverWithErrorStatus.Status.AtProvider.State = ionoscloud.Failed

	serverWithUnknownStatus := createServer("serverset-server-1-0")
	serverWithUnknownStatus.Status.AtProvider.State = "new-state"

	nic1 := createNic(server1.Name)
	nic2 := createNic(server2.Name)

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    v1alpha1.ServerSetObservation
		wantErr bool
	}{
		{
			name: "serverset status is populated correctly",
			fields: fields{
				kube: fakeKubeClientObjs(server1, server2, nic1, nic2, createConfigLeaseMap()),
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSet(),
			},
			want: v1alpha1.ServerSetObservation{
				Replicas: 2,
				ReplicaStatuses: []v1alpha1.ServerSetReplicaStatus{
					{
						Name:         server1.Name,
						Status:       statusReady,
						Role:         "ACTIVE",
						ErrorMessage: "",
					},
					{
						Name:         server2.Name,
						Status:       statusReady,
						Role:         "PASSIVE",
						ErrorMessage: "",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "config-lease map missing, then roles default to PASSIVE",
			fields: fields{
				kube: fakeKubeClientObjs(server1, server2, nic1, nic2),
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSet(),
			},
			want: v1alpha1.ServerSetObservation{
				Replicas: 2,
				ReplicaStatuses: []v1alpha1.ServerSetReplicaStatus{
					{
						Name:         server1.Name,
						Status:       statusReady,
						Role:         "PASSIVE",
						ErrorMessage: "",
					},
					{
						Name:         server2.Name,
						Status:       statusReady,
						Role:         "PASSIVE",
						ErrorMessage: "",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "replicas not in config-lease, then roles default to PASSIVE",
			fields: fields{
				kube: fakeKubeClientObjs(server1, server2, nic1, nic2, createConfigLeaseMapDoesNotContainAnyReplica()),
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSet(),
			},
			want: v1alpha1.ServerSetObservation{
				Replicas: 2,
				ReplicaStatuses: []v1alpha1.ServerSetReplicaStatus{
					{
						Name:         server1.Name,
						Status:       statusReady,
						Role:         "PASSIVE",
						ErrorMessage: "",
					},
					{
						Name:         server2.Name,
						Status:       statusReady,
						Role:         "PASSIVE",
						ErrorMessage: "",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "replica count increases, then number of replica status is increased",
			fields: fields{
				kube: fakeKubeClientObjs(server1, server2, nic1, nic2),
			},
			args: args{
				ctx: context.Background(),
				cr:  createServerSetWhichUpdatesFrom1ReplicaTo2(server1.Name),
			},
			want: v1alpha1.ServerSetObservation{
				Replicas: 2,
				ReplicaStatuses: []v1alpha1.ServerSetReplicaStatus{
					{
						Name:         server1.Name,
						Status:       statusReady,
						Role:         "PASSIVE",
						ErrorMessage: "",
					},
					{
						Name:         server2.Name,
						Status:       statusReady,
						Role:         "PASSIVE",
						ErrorMessage: "",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "replica count decreases, then number of replica status is decreased",
			fields: fields{
				kube: fakeKubeClientObjs(server1, nic1),
			},
			args: args{
				ctx: context.Background(),
				cr:  createServerSetWhichUpdatesFrom2ReplicasTo1(server1.Name, server2.Name),
			},
			want: v1alpha1.ServerSetObservation{
				Replicas: 1,
				ReplicaStatuses: []v1alpha1.ServerSetReplicaStatus{
					{
						Name:         server1.Name,
						Status:       statusReady,
						Role:         "PASSIVE",
						ErrorMessage: "",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "status of the server is failure, then status of replica is ERROR",
			fields: fields{
				kube: fakeKubeClientObjs(serverWithErrorStatus),
			},
			args: args{
				ctx: context.Background(),
				cr:  createServerSetWithOneReplica(),
			},
			want: v1alpha1.ServerSetObservation{
				Replicas: 1,
				ReplicaStatuses: []v1alpha1.ServerSetReplicaStatus{
					{
						Name:         serverWithErrorStatus.Name,
						Status:       statusError,
						Role:         "PASSIVE",
						ErrorMessage: "",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "error message on the server is, then status of replica is ERROR and error message is populated",
			fields: fields{
				kube: fakeKubeClientObjs(createServerWithReconcileErrorMsg()),
			},
			args: args{
				ctx: context.Background(),
				cr:  createServerSetWithOneReplica(),
			},
			want: v1alpha1.ServerSetObservation{
				Replicas: 1,
				ReplicaStatuses: []v1alpha1.ServerSetReplicaStatus{
					{
						Name:         serverWithErrorStatus.Name,
						Status:       statusError,
						Role:         "PASSIVE",
						ErrorMessage: reconcileErrorMsg,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "status of the server not among known ones, then status of replica is also UNKNOWN",
			fields: fields{
				kube: fakeKubeClientObjs(serverWithUnknownStatus, nic1),
			},
			args: args{
				ctx: context.Background(),
				cr:  createServerSetWhichUpdatesFrom2ReplicasTo1(serverWithUnknownStatus.Name, server2.Name),
			},
			want: v1alpha1.ServerSetObservation{
				Replicas: 1,
				ReplicaStatuses: []v1alpha1.ServerSetReplicaStatus{
					{
						Name:         serverWithUnknownStatus.Name,
						Status:       statusUnknown,
						Role:         "PASSIVE",
						ErrorMessage: "",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &external{
				kube: tt.fields.kube,
				log:  logging.NewNopLogger(),
			}

			_, err := e.Observe(tt.args.ctx, tt.args.cr)
			got := tt.args.cr.Status.AtProvider

			if (err != nil) != tt.wantErr {
				t.Errorf("Observe() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			areEqual(t, tt.want, got)
		})
	}
}

func Test_serverSetController_Create(t *testing.T) {
	type fields struct {
		kube                 client.Client
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
				log:                  logging.NewNopLogger(),
				kube:                 fakeKubeClientObjs(),
				bootVolumeController: fakeBootVolumeCtrlEnsureMethod(noReplicas),
				serverController:     fakeServerCtrlEnsureMethod(noReplicas),
				nicController:        fakeNicCtrlEnsureNICsMethod(noReplicas),
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
			name: "too many volumes returned for the same index",
			fields: fields{
				log: logging.NewNopLogger(),
				kube: fakeKubeClientObjs(
					createBootVolumeWithIndex("boot-volume1", 0),
					createBootVolumeWithIndex("boot-volume2", 0)),
				bootVolumeController: fakeBootVolumeCtrlEnsureMethod(0),
				serverController:     fakeServerCtrlEnsureMethod(0),
				nicController:        fakeNicCtrlEnsureNICsMethod(0),
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSet(),
			},
			want:    managed.ExternalCreation{},
			wantErr: errors.New("found too many volumes for index 0"),
		},
		{
			name: "error when ensuring boot volume",
			fields: fields{
				log:                  logging.NewNopLogger(),
				kube:                 fakeKubeClientObjs(),
				bootVolumeController: fakeBootVolumeCtrlEnsureMethodReturnsErr(),
				serverController:     fakeServerCtrlEnsureMethod(0),
				nicController:        fakeNicCtrlEnsureNICsMethod(0),
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSet(),
			},
			want:    managed.ExternalCreation{},
			wantErr: errAnErrorWasReceived,
		},
		{
			name: "too many servers returned for the same index",
			fields: fields{
				log: logging.NewNopLogger(),
				kube: fakeKubeClientObjs(
					createServerWithIndex("server1", 0),
					createServerWithIndex("server2", 0)),
				bootVolumeController: fakeBootVolumeCtrlEnsureMethod(1),
				serverController:     fakeServerCtrlEnsureMethod(0),
				nicController:        fakeNicCtrlEnsureNICsMethod(0),
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSet(),
			},
			want:    managed.ExternalCreation{},
			wantErr: errors.New("found too many servers for index 0"),
		},
		{
			name: "error when ensuring server",
			fields: fields{
				log:                  logging.NewNopLogger(),
				kube:                 fakeKubeClientObjs(),
				bootVolumeController: fakeBootVolumeCtrlEnsureMethod(1),
				serverController:     fakeServerCtrlEnsureMethodReturnsErr(),
				nicController:        fakeNicCtrlEnsureNICsMethod(0),
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSet(),
			},
			want:    managed.ExternalCreation{},
			wantErr: errAnErrorWasReceived,
		},
		{
			name: "error when ensuring NICs",
			fields: fields{
				log:                  logging.NewNopLogger(),
				kube:                 fakeKubeClientObjs(),
				bootVolumeController: fakeBootVolumeCtrlEnsureMethod(1),
				serverController:     fakeServerCtrlEnsureMethod(1),
				nicController:        fakeNicCtrlEnsureNICsMethodReturnsErr(),
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSet(),
			},
			want:    managed.ExternalCreation{},
			wantErr: errAnErrorWasReceived,
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

			assertions := assert.New(t)
			fakeBootVolumeCtrl := tt.fields.bootVolumeController.(*kubeBootVolumeControlManagerFake)
			fakeBootVolumeCtrl.AssertExpectations(t)

			fakeServerCtrl := tt.fields.serverController.(*kubeServerControlManagerFake)
			fakeServerCtrl.AssertExpectations(t)

			fakeNicCtrl := tt.fields.nicController.(*kubeNicControlManagerFake)
			fakeNicCtrl.AssertExpectations(t)

			assertions.Equalf(tt.wantErr, err, "Wrong error")
			assertions.Equalf(tt.want, got, "Wrong response")
			assertions.Equalf(1, len(tt.args.cr.Status.Conditions), "ServerSet should have one condition")
			assertCondition(t, xpv1.Creating(), tt.args.cr.Status.Conditions[0], "ServerSet has wrong condition")
		})
	}
}

func Test_serverSetController_Update(t *testing.T) {
	type fields struct {
		kube                 client.Client
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
		name            string
		fields          fields
		args            args
		wantErr         error
		want            managed.ExternalUpdate
		wantUpdateCalls int
	}{
		{
			name: "server set successfully updated (no changes)",
			fields: fields{
				kube: fakeKubeClientUpdateMethod(&v1alpha1.Server{}),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSet(),
			},
			wantErr: nil,
			want: managed.ExternalUpdate{
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantUpdateCalls: 0,
		},
		{
			name: "server set successfully updated (CPU Family changed)",
			fields: fields{
				kube: fakeKubeClientUpdateMethod(&v1alpha1.Server{}),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedServerSpec(v1alpha1.ServerSetTemplateSpec{
					CPUFamily: "INTEL_XEON",
					Cores:     serverSetCores,
					RAM:       serverSetRAM,
				}),
			},
			wantErr: nil,
			want: managed.ExternalUpdate{
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantUpdateCalls: 2,
		},
		{
			name: "server set successfully updated (Cores changed)",
			fields: fields{
				kube: fakeKubeClientUpdateMethod(&v1alpha1.Server{}),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedServerSpec(v1alpha1.ServerSetTemplateSpec{
					CPUFamily: serverSetCPUFamily,
					Cores:     10,
					RAM:       serverSetRAM,
				}),
			},
			wantErr: nil,
			want: managed.ExternalUpdate{
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantUpdateCalls: 2,
		},
		{
			name: "server set successfully updated (RAM changed)",
			fields: fields{
				kube: fakeKubeClientUpdateMethod(&v1alpha1.Server{}),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedServerSpec(v1alpha1.ServerSetTemplateSpec{
					CPUFamily: serverSetCPUFamily,
					Cores:     serverSetCores,
					RAM:       8192,
				}),
			},
			wantErr: nil,
			want: managed.ExternalUpdate{
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantUpdateCalls: 2,
		},
		{
			name: "server set successfully updated (boot volume size changed)",
			fields: fields{
				kube: fakeKubeClientUpdateMethod(&v1alpha1.Volume{}),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedBootVolumeUsingDefaultStrategy(v1alpha1.ServerSetBootVolumeSpec{
					Size:  200,
					Image: bootVolumeImage,
					Type:  bootVolumeType,
				}),
			},
			wantErr: nil,
			want: managed.ExternalUpdate{
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantUpdateCalls: 2,
		},
		{
			name: "failure in kube client when updating server",
			fields: fields{
				kube: fakeKubeClientUpdateMethodReturnsError(),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedServerSpec(v1alpha1.ServerSetTemplateSpec{
					CPUFamily: "INTEL_XEON",
					Cores:     serverSetCores,
					RAM:       serverSetRAM,
				}),
			},
			wantErr:         fmt.Errorf("error updating server %w", errAnErrorWasReceived),
			want:            managed.ExternalUpdate{},
			wantUpdateCalls: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields.kube.(*fakeKubeClient).t = t
			e := &external{
				kube:                 tt.fields.kube,
				bootVolumeController: tt.fields.bootVolumeController,
				nicController:        tt.fields.nicController,
				serverController:     tt.fields.serverController,
				log:                  tt.fields.log,
			}

			got, err := e.Update(tt.args.ctx, tt.args.cr)

			assertions := assert.New(t)
			assertions.Equalf(tt.wantErr, err, "Wrong error")
			assertions.Equalf(tt.want, got, "Wrong response")
			assertions.Equalf(0, len(tt.args.cr.Status.Conditions), "ServerSet should not have any conditions")
			kubeClient := tt.fields.kube.(*fakeKubeClient)
			kubeClient.AssertNumberOfCalls(t, "Update", tt.wantUpdateCalls)
		})
	}
}

func Test_serverSetController_BootVolumeUpdate(t *testing.T) {
	type fields struct {
		kube                 client.Client
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
		name      string
		fields    fields
		args      args
		wantErr   error
		want      managed.ExternalUpdate
		wantCalls map[ServiceMethodName]int
	}{
		{
			name: "boot volume updated using default strategy (boot volume image changed)",
			fields: fields{
				kube:                 fakeKubeClientUpdateMethodForBootVolume(),
				bootVolumeController: fakeBootVolumeCtrl(),
				serverController:     fakeServerCtrl(),
				nicController:        fakeNicCtrl(),
				log:                  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedBootVolumeUsingDefaultStrategy(v1alpha1.ServerSetBootVolumeSpec{
					Size:  bootVolumeSize,
					Image: "newImage",
					Type:  bootVolumeType,
				}),
			},
			wantErr: nil,
			want: managed.ExternalUpdate{
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantCalls: map[ServiceMethodName]int{
				kubeUpdate:       0,
				serverEnsure:     0,
				serverDelete:     0,
				serverGet:        2,
				serverUpdate:     2,
				bootVolumeEnsure: 2,
				bootVolumeDelete: 2,
				bootVolumeGet:    2,
				nicEnsureNICs:    0,
				nicDelete:        0,
			},
		},
		{
			name: "boot volume updated using default strategy (boot volume type changed)",
			fields: fields{
				kube:                 fakeKubeClientUpdateMethodForBootVolume(),
				bootVolumeController: fakeBootVolumeCtrl(),
				serverController:     fakeServerCtrl(),
				nicController:        fakeNicCtrl(),
				log:                  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedBootVolumeUsingDefaultStrategy(v1alpha1.ServerSetBootVolumeSpec{
					Size:  bootVolumeSize,
					Image: bootVolumeImage,
					Type:  "SSD",
				}),
			},
			wantErr: nil,
			want: managed.ExternalUpdate{
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantCalls: map[ServiceMethodName]int{
				kubeUpdate:       0,
				serverEnsure:     0,
				serverDelete:     0,
				serverGet:        2,
				serverUpdate:     2,
				bootVolumeEnsure: 2,
				bootVolumeDelete: 2,
				bootVolumeGet:    2,
				nicEnsureNICs:    0,
				nicDelete:        0,
			},
		},
		{
			name: "boot volume updated using createAllBeforeDestroy strategy (boot volume type changed)",
			fields: fields{
				kube:                 fakeKubeClientUpdateMethodForBootVolume(),
				bootVolumeController: fakeBootVolumeCtrl(),
				nicController:        fakeNicCtrl(),
				serverController:     fakeServerCtrl(),
				log:                  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedBootVolumeUsingStrategy(v1alpha1.ServerSetBootVolumeSpec{
					Size:  bootVolumeSize,
					Image: bootVolumeImage,
					Type:  "SSD",
				}, v1alpha1.UpdateStrategy{Stype: v1alpha1.CreateAllBeforeDestroy}),
			},
			wantErr: nil,
			want: managed.ExternalUpdate{
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantCalls: map[ServiceMethodName]int{
				kubeUpdate:       0,
				serverEnsure:     2,
				serverDelete:     2,
				serverGet:        0,
				serverUpdate:     0,
				bootVolumeEnsure: 2,
				bootVolumeDelete: 2,
				bootVolumeGet:    0,
				nicEnsureNICs:    2,
				nicDelete:        2,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields.kube.(*fakeKubeClient).t = t
			e := &external{
				kube:                 tt.fields.kube,
				bootVolumeController: tt.fields.bootVolumeController,
				nicController:        tt.fields.nicController,
				serverController:     tt.fields.serverController,
				log:                  tt.fields.log,
			}

			got, err := e.Update(tt.args.ctx, tt.args.cr)

			assertions := assert.New(t)
			assertions.Equalf(tt.wantErr, err, "Wrong error")
			assertions.Equalf(tt.want, got, "Wrong response")
			assertions.Equalf(0, len(tt.args.cr.Status.Conditions), "ServerSet should not have any conditions")

			kubeClient := tt.fields.kube.(*fakeKubeClient)
			kubeClient.AssertNumberOfCalls(t, update, tt.wantCalls[kubeUpdate])

			bootVolumeCtrl := tt.fields.bootVolumeController.(*kubeBootVolumeControlManagerFake)
			bootVolumeCtrl.AssertNumberOfCalls(t, ensure, tt.wantCalls[bootVolumeEnsure])
			bootVolumeCtrl.AssertNumberOfCalls(t, delete, tt.wantCalls[bootVolumeDelete])
			bootVolumeCtrl.AssertNumberOfCalls(t, get, tt.wantCalls[bootVolumeGet])

			serverController := tt.fields.serverController.(*kubeServerControlManagerFake)
			serverController.AssertNumberOfCalls(t, ensure, tt.wantCalls[serverEnsure])
			serverController.AssertNumberOfCalls(t, delete, tt.wantCalls[serverDelete])
			serverController.AssertNumberOfCalls(t, get, tt.wantCalls[serverGet])
			serverController.AssertNumberOfCalls(t, update, tt.wantCalls[serverUpdate])

			nicCtrl := tt.fields.nicController.(*kubeNicControlManagerFake)
			nicCtrl.AssertNumberOfCalls(t, ensureNICs, tt.wantCalls[nicEnsureNICs])
			nicCtrl.AssertNumberOfCalls(t, delete, tt.wantCalls[nicDelete])
		})
	}
}

func fakeKubeClientUpdateMethodReturnsError() client.Client {
	kubeClient := fakeKubeClient{
		Client: fakeKubeClientObjs(createServer("server1"), createServer("server2")),
	}
	kubeClient.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(errAnErrorWasReceived)
	return &kubeClient
}

func fakeKubeClientUpdateMethod(expectedObj client.Object) client.Client {
	kubeClient := fakeKubeClient{
		Client: fakeKubeClientObjs(
			createServer("server1"), createServer("server2"),
			createBootVolume("boot-volume-server1"), createBootVolume("boot-volume-server2"),
			createNic("nic-server1"), createNic("nic-server2")),
	}
	kubeClient.On("Update", mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		arg1 := args.Get(1)
		if reflect.TypeOf(arg1) != reflect.TypeOf(expectedObj) {
			kubeClient.t.Errorf("Update called with unexpeted type: want=%v, got=%v", reflect.TypeOf(expectedObj), reflect.TypeOf(arg1))
		}
	}).Return(nil)

	return &kubeClient
}

func fakeKubeClientUpdateMethodForBootVolume() client.Client {
	zero := "0"
	one := "1"

	server1 := createServer("server1")
	server1.Labels[computeIndexLabel(ResourceServer)] = zero
	server1.Labels[computeVersionLabel(ResourceServer)] = zero

	server2 := createServer("server2")
	server2.Labels[computeIndexLabel(ResourceServer)] = one
	server2.Labels[computeVersionLabel(ResourceServer)] = zero

	bootVolume1 := createBootVolume("boot-volume-server1")
	bootVolume1.Labels[computeIndexLabel(resourceBootVolume)] = zero
	bootVolume1.Labels[computeVersionLabel(resourceBootVolume)] = zero

	bootVolume2 := createBootVolume("boot-volume-server2")
	bootVolume2.Labels[computeIndexLabel(resourceBootVolume)] = one
	bootVolume2.Labels[computeVersionLabel(resourceBootVolume)] = zero

	kubeClient := fakeKubeClient{
		Client: fakeKubeClientObjs(server1, server2, bootVolume1, bootVolume2,
			createNic("nic-server1"), createNic("nic-server2")),
	}

	kubeClient.On("Update", mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		arg1 := args.Get(1)

		expectedType := reflect.TypeOf(v1alpha1.Volume{})
		actualType := reflect.TypeOf(arg1)
		if actualType != expectedType {
			kubeClient.t.Errorf("Update called with unexpeted type: want=%v, got=%v", expectedType, actualType)
		}
	}).Return(nil)

	return &kubeClient
}

func computeIndexLabel(resourceType string) string {
	return fmt.Sprintf(indexLabel, serverSetName, resourceType)
}

func computeVersionLabel(resourceType string) string {
	return fmt.Sprintf(versionLabel, serverSetName, resourceType)
}

func (f *fakeKubeClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	args := f.Called(ctx, obj, opts)
	return args.Error(0)
}

func fakeBootVolumeCtrlEnsureMethod(timesCalled int) kubeBootVolumeControlManager {
	bootVolumeCtrl := new(kubeBootVolumeControlManagerFake)
	if timesCalled > 0 {
		bootVolumeCtrl.
			On(ensure, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil).
			Times(timesCalled)
	}
	return bootVolumeCtrl
}

func fakeBootVolumeCtrl() kubeBootVolumeControlManager {
	bootVolumeCtrl := new(kubeBootVolumeControlManagerFake)
	bootVolumeCtrl.
		On(ensure, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).
		On(get, mock.Anything, mock.Anything, mock.Anything).Return(&v1alpha1.Volume{}, nil).
		On(delete, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	return bootVolumeCtrl

}

func fakeBootVolumeCtrlEnsureMethodReturnsErr() kubeBootVolumeControlManager {
	bootVolumeCtrl := new(kubeBootVolumeControlManagerFake)
	bootVolumeCtrl.
		On(ensure, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(errAnErrorWasReceived).
		Times(1)
	return bootVolumeCtrl
}

func fakeServerCtrlEnsureMethod(timesCalled int) kubeServerControlManager {
	serverCtrl := new(kubeServerControlManagerFake)
	if timesCalled > 0 {
		serverCtrl.
			On(ensure, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil).
			Times(timesCalled)
	}
	return serverCtrl
}

func fakeServerCtrl() kubeServerControlManager {
	serverCtrl := new(kubeServerControlManagerFake)
	serverCtrl.
		On(ensure, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).
		On(delete, mock.Anything, mock.Anything, mock.Anything).Return(nil).
		On(update, mock.Anything, mock.Anything).Return(nil).
		On(get, mock.Anything, mock.Anything, mock.Anything).Return(&v1alpha1.Server{}, nil)
	return serverCtrl
}

func fakeServerCtrlEnsureMethodReturnsErr() kubeServerControlManager {
	serverCtrl := new(kubeServerControlManagerFake)
	serverCtrl.
		On(ensure, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(errAnErrorWasReceived).
		Times(1)
	return serverCtrl
}

func fakeNicCtrlEnsureNICsMethod(timesCalled int) kubeNicControlManager {
	nicCtrl := new(kubeNicControlManagerFake)
	if timesCalled > 0 {
		nicCtrl.
			On(ensureNICs, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil).
			Times(noReplicas)
	}
	return nicCtrl
}

func fakeNicCtrlEnsureNICsMethodReturnsErr() kubeNicControlManager {
	nicCtrl := new(kubeNicControlManagerFake)
	nicCtrl.
		On(ensureNICs, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(errAnErrorWasReceived).
		Times(1)

	return nicCtrl
}
func fakeNicCtrl() kubeNicControlManager {
	nicCtrl := new(kubeNicControlManagerFake)
	nicCtrl.
		On(ensureNICs, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).
		On(delete, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	return nicCtrl
}

func assertCondition(t *testing.T, expected xpv1.Condition, actual xpv1.Condition, msg string) {
	ignoreFields := cmpopts.IgnoreFields(xpv1.Condition{}, "LastTransitionTime")
	if diff := cmp.Diff(expected, actual, ignoreFields); diff != "" {
		t.Errorf("%s (-want +got):\n%s", msg, diff)
	}
}

func (f *kubeBootVolumeControlManagerFake) Ensure(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int) error {
	args := f.Called(ctx, cr, replicaIndex, version)
	return args.Error(0)
}

func (f *kubeBootVolumeControlManagerFake) Get(ctx context.Context, name, ns string) (*v1alpha1.Volume, error) {
	args := f.Called(ctx, name, ns)
	return args.Get(0).(*v1alpha1.Volume), args.Error(1)
}

func (f *kubeBootVolumeControlManagerFake) Delete(ctx context.Context, name, ns string) error {
	args := f.Called(ctx, name, ns)
	return args.Error(0)
}

func (f *kubeServerControlManagerFake) Ensure(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version, volumeVersion int) error {
	args := f.Called(ctx, cr, replicaIndex, version, volumeVersion)
	return args.Error(0)
}

func (f *kubeServerControlManagerFake) Get(ctx context.Context, name, ns string) (*v1alpha1.Server, error) {
	args := f.Called(ctx, name, ns)
	return args.Get(0).(*v1alpha1.Server), args.Error(1)
}

func (f *kubeServerControlManagerFake) Update(ctx context.Context, cr *v1alpha1.Server) error {
	args := f.Called(ctx, cr)
	return args.Error(0)
}

func (f *kubeServerControlManagerFake) Delete(ctx context.Context, name, ns string) error {
	args := f.Called(ctx, name, ns)
	return args.Error(0)
}

func (f *kubeNicControlManagerFake) EnsureNICs(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int) error {
	args := f.Called(ctx, cr, replicaIndex, version)
	return args.Error(0)
}

func (f *kubeNicControlManagerFake) Delete(ctx context.Context, name, ns string) error {
	args := f.Called(ctx, name, ns)
	return args.Error(0)
}

func createServer(name string) *v1alpha1.Server {
	return &v1alpha1.Server{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				serverSetLabel: serverSetName,
			},
		},
		Status: v1alpha1.ServerStatus{
			AtProvider: v1alpha1.ServerObservation{
				State: ionoscloud.Available,
			},
		},
		Spec: v1alpha1.ServerSpec{
			ForProvider: v1alpha1.ServerParameters{
				Cores:     serverSetCores,
				RAM:       serverSetRAM,
				CPUFamily: serverSetCPUFamily,
			},
		},
	}
}

func createNic(name string) *v1alpha1.Nic {
	return &v1alpha1.Nic{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				serverSetLabel: serverSetName,
			},
		},
	}
}

func createBootVolume(name string) *v1alpha1.Volume {
	return &v1alpha1.Volume{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				serverSetLabel: serverSetName,
			},
		},
		Spec: v1alpha1.VolumeSpec{
			ForProvider: v1alpha1.VolumeParameters{
				Image: bootVolumeImage,
				Type:  bootVolumeType,
				Size:  bootVolumeSize,
			},
		},
	}
}

func createBootVolumeWithIndex(name string, index int) *v1alpha1.Volume {
	volume := createBootVolume(name)
	indexLabelBootVolume := fmt.Sprintf(indexLabel, serverSetName, resourceBootVolume)
	volume.ObjectMeta.Labels[indexLabelBootVolume] = fmt.Sprintf("%d", index)
	return volume
}

func fakeKubeClientObjs(objs ...client.Object) client.WithWatch {
	scheme := runtime.NewScheme()
	v1.AddToScheme(scheme)       // Add the core k8s types to the Scheme
	v1alpha1.AddToScheme(scheme) // Add our custom types from v1alpha to the Scheme
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
}

func createBasicServerSet() *v1alpha1.ServerSet {
	return &v1alpha1.ServerSet{
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
					Spec: v1alpha1.ServerSetTemplateSpec{
						Cores:     serverSetCores,
						RAM:       serverSetRAM,
						CPUFamily: serverSetCPUFamily,
						NICs: []v1alpha1.ServerSetTemplateNIC{
							{
								Name:      "nic1",
								IPv4:      "10.0.0.2/24",
								Reference: "data",
							},
						},
					},
				},
				BootVolumeTemplate: v1alpha1.BootVolumeTemplate{
					Spec: v1alpha1.ServerSetBootVolumeSpec{
						Size:  bootVolumeSize,
						Image: bootVolumeImage,
						Type:  bootVolumeType,
					},
				},
			},
		},
		Status: v1alpha1.ServerSetStatus{},
	}
}

func createServerSetWithUpdatedServerSpec(spec v1alpha1.ServerSetTemplateSpec) *v1alpha1.ServerSet {
	sset := createBasicServerSet()
	sset.Spec.ForProvider.Template.Spec.Cores = spec.Cores
	sset.Spec.ForProvider.Template.Spec.RAM = spec.RAM
	sset.Spec.ForProvider.Template.Spec.CPUFamily = spec.CPUFamily
	return sset
}

func createServerSetWithUpdatedBootVolumeUsingDefaultStrategy(updatedSpec v1alpha1.ServerSetBootVolumeSpec) *v1alpha1.ServerSet {
	sset := createBasicServerSet()
	sset.Spec.ForProvider.BootVolumeTemplate = v1alpha1.BootVolumeTemplate{
		Spec: updatedSpec,
	}
	return sset
}

func createServerSetWithUpdatedBootVolumeUsingStrategy(updatedSpec v1alpha1.ServerSetBootVolumeSpec, strategy v1alpha1.UpdateStrategy) *v1alpha1.ServerSet {
	sset := createServerSetWithUpdatedBootVolumeUsingDefaultStrategy(updatedSpec)
	sset.Spec.ForProvider.BootVolumeTemplate.Spec.UpdateStrategy = strategy
	return sset
}

func createServerSetWhichUpdatesFrom1ReplicaTo2(serverName string) *v1alpha1.ServerSet {
	sset := createBasicServerSet()
	sset.Status = v1alpha1.ServerSetStatus{
		AtProvider: v1alpha1.ServerSetObservation{
			Replicas: 1,
			ReplicaStatuses: []v1alpha1.ServerSetReplicaStatus{
				{
					Name:         serverName,
					Status:       statusReady,
					ErrorMessage: "",
				},
			},
		},
	}
	return sset
}

func createServerSetWhichUpdatesFrom2ReplicasTo1(serverName1, serverName2 string) *v1alpha1.ServerSet {
	sset := createServerSetWithOneReplica()
	sset.Status = v1alpha1.ServerSetStatus{
		AtProvider: v1alpha1.ServerSetObservation{
			Replicas: 2,
			ReplicaStatuses: []v1alpha1.ServerSetReplicaStatus{
				{
					Name:         serverName1,
					Status:       statusReady,
					ErrorMessage: "",
				},
				{
					Name:         serverName2,
					Status:       statusReady,
					ErrorMessage: "",
				},
			},
		},
	}
	return sset
}

func createServerSetWithOneReplica() *v1alpha1.ServerSet {
	sset := createBasicServerSet()
	sset.Spec.ForProvider.Replicas = 1
	return sset
}

func createServerSetWithNrOfNICsUpdated() *v1alpha1.ServerSet {
	sset := createBasicServerSet()
	sset.Spec.ForProvider.Template.Spec.NICs = append(
		sset.Spec.ForProvider.Template.Spec.NICs, v1alpha1.ServerSetTemplateNIC{
			Name:      "nic2",
			IPv4:      "10.0.0.3/24",
			Reference: "management",
		})

	return sset
}

func areEqual(t *testing.T, want, got v1alpha1.ServerSetObservation) {
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(v1alpha1.ServerSetReplicaStatus{}, "LastModified")); diff != "" {
		t.Errorf("ServerSetObservation() mismatch (-want +got):\n%s", diff)
	}
}

func createConfigLeaseMapDoesNotContainAnyReplica() *v1.ConfigMap {
	cm := createConfigLeaseMap()
	cm.Data = map[string]string{
		"identity": "some-other-server",
	}
	return cm
}

func createConfigLeaseMap() *v1.ConfigMap {
	return &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "config-lease",
			Namespace: "default",
		},
		Data: map[string]string{
			"identity": "serverset-server-0-0",
		},
	}
}

func createServerWithReconcileErrorMsg() *v1alpha1.Server {
	server := createServer("serverset-server-1-0")
	server.Status.AtProvider.State = ionoscloud.Failed
	server.Status.ResourceStatus.Conditions = []xpv1.Condition{
		{
			Reason:  xpv1.ReasonReconcileError,
			Message: reconcileErrorMsg,
		},
	}
	return server
}

func createServerWithIndex(name string, index int) *v1alpha1.Server {
	server := createServer(name)
	indexLabelBootVolume := fmt.Sprintf(indexLabel, serverSetName, ResourceServer)
	server.ObjectMeta.Labels[indexLabelBootVolume] = fmt.Sprintf("%d", index)
	return server
}
