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

	ensure     = "Ensure"
	ensureNICs = "EnsureNICs"

	noReplicas = 2

	server1Name        = "serverset-server-0-0"
	server2Name        = "serverset-server-1-0"
	serverSetCPUFamily = "AMD_OPTERON"
	serverSetCores     = 2
	serverSetRAM       = 4096
	serverSetName      = "serverset"

	reconcileErrorMsg = "some reconcile error happened"
)

var errAnErrorWasReceived = errors.New("an error was received")

type fakeKubeBootVolumeControlManager struct {
	kubeBootVolumeControlManager
	mock.Mock
}

type fakeKubeServerControlManager struct {
	kubeServerControlManager
	mock.Mock
}

type fakeKubeNicControlManager struct {
	kubeNicControlManager
	mock.Mock
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
	nic1 := createNic(server1.Name)
	nic2 := createNic(server2.Name)
	bootVolume1 := createBootVolume(bootVolumeNamePrefix + server1.Name)
	bootVolume2 := createBootVolume(bootVolumeNamePrefix + server2.Name)

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
				kube: fakeKubeClientObjs(&server1, &server2, &bootVolume1, &bootVolume2, &nic1, &nic2),
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
				kube: fakeKubeClientObjs(&server1, &server2, &bootVolume1, &bootVolume2, &nic1, &nic2),
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
				kube: fakeKubeClientObjs(&server1, &server2, &bootVolume1, &bootVolume2, &nic1, &nic2),
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
				kube: fakeKubeClientObjs(&server1, &server2, &bootVolume1, &bootVolume2, &nic1, &nic2),
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
				kube: fakeKubeClientObjs(&server1, &server2, &bootVolume1, &bootVolume2, &nic1, &nic2),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedBootVolume(v1alpha1.ServerSetBootVolumeSpec{
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
				kube: fakeKubeClientObjs(&server1, &server2, &bootVolume1, &bootVolume2, &nic1, &nic2),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedBootVolume(v1alpha1.ServerSetBootVolumeSpec{
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
				kube: fakeKubeClientObjs(&server1, &server2, &bootVolume1, &bootVolume2, &nic1, &nic2),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedBootVolume(v1alpha1.ServerSetBootVolumeSpec{
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
				kube: fakeKubeClientObjs(&server1),
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
				kube: fakeKubeClientObjs(&server1, &server2, &bootVolume1, &bootVolume2),
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
				kube: fakeKubeClientObjs(&server1, &server2, &bootVolume1, &bootVolume2, &nic1, &nic2),
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
				kube: fakeKubeClientObjs(&server1, &server2, &nic1, &nic2, createConfigLeaseMap()),
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
				kube: fakeKubeClientObjs(&server1, &server2, &nic1, &nic2),
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
				kube: fakeKubeClientObjs(&server1, &server2, &nic1, &nic2, createConfigLeaseMapDoesNotContainAnyReplica()),
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
				kube: fakeKubeClientObjs(&server1, &server2, &nic1, &nic2),
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
				kube: fakeKubeClientObjs(&server1, &nic1),
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
				kube: fakeKubeClientObjs(&serverWithErrorStatus),
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
				kube: fakeKubeClientObjs(&serverWithUnknownStatus, &nic1),
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
			name: "serverset successfully created",
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
			fakeBootVolumeCtrl := tt.fields.bootVolumeController.(*fakeKubeBootVolumeControlManager)
			fakeBootVolumeCtrl.AssertExpectations(t)

			fakeServerCtrl := tt.fields.serverController.(*fakeKubeServerControlManager)
			fakeServerCtrl.AssertExpectations(t)

			fakeNicCtrl := tt.fields.nicController.(*fakeKubeNicControlManager)
			fakeNicCtrl.AssertExpectations(t)

			assertions.Equalf(tt.wantErr, err, "Wrong error")
			assertions.Equalf(tt.want, got, "Wrong response")
			assertions.Equalf(1, len(tt.args.cr.Status.Conditions), "ServerSet should have one condition")
			assertCondition(t, xpv1.Creating(), tt.args.cr.Status.Conditions[0], "ServerSet has wrong condition")
		})
	}
}

func fakeBootVolumeCtrlEnsureMethod(timesCalled int) kubeBootVolumeControlManager {
	bootVolumeCtrl := new(fakeKubeBootVolumeControlManager)
	if timesCalled > 0 {
		bootVolumeCtrl.
			On(ensure, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil).
			Times(timesCalled)
	}
	return bootVolumeCtrl

}

func fakeBootVolumeCtrlEnsureMethodReturnsErr() kubeBootVolumeControlManager {
	bootVolumeCtrl := new(fakeKubeBootVolumeControlManager)
	bootVolumeCtrl.
		On(ensure, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(errAnErrorWasReceived).
		Times(1)
	return bootVolumeCtrl
}

func fakeServerCtrlEnsureMethod(timesCalled int) kubeServerControlManager {
	serverCtrl := new(fakeKubeServerControlManager)
	if timesCalled > 0 {
		serverCtrl.
			On(ensure, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil).
			Times(timesCalled)
	}
	return serverCtrl
}

func fakeServerCtrlEnsureMethodReturnsErr() kubeServerControlManager {
	serverCtrl := new(fakeKubeServerControlManager)
	serverCtrl.
		On(ensure, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(errAnErrorWasReceived).
		Times(1)
	return serverCtrl
}

func fakeNicCtrlEnsureNICsMethod(timesCalled int) kubeNicControlManager {
	nicCtrl := new(fakeKubeNicControlManager)
	if timesCalled > 0 {
		nicCtrl.
			On(ensureNICs, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil).
			Times(noReplicas)
	}
	return nicCtrl
}

func fakeNicCtrlEnsureNICsMethodReturnsErr() kubeNicControlManager {
	nicCtrl := new(fakeKubeNicControlManager)
	nicCtrl.
		On(ensureNICs, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(errAnErrorWasReceived).
		Times(1)

	return nicCtrl
}

func assertCondition(t *testing.T, expected xpv1.Condition, actual xpv1.Condition, msg string) {
	ignoreFields := cmpopts.IgnoreFields(xpv1.Condition{}, "LastTransitionTime")
	if diff := cmp.Diff(expected, actual, ignoreFields); diff != "" {
		t.Errorf("%s (-want +got):\n%s", msg, diff)
	}
}

func (f *fakeKubeBootVolumeControlManager) Ensure(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int) error {
	args := f.Called(ctx, cr, replicaIndex, version)
	return args.Error(0)
}

func (f *fakeKubeServerControlManager) Ensure(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version, volumeVersion int) error {
	args := f.Called(ctx, cr, replicaIndex, version, volumeVersion)
	return args.Error(0)
}

func (f *fakeKubeNicControlManager) EnsureNICs(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int) error {
	args := f.Called(ctx, cr, replicaIndex, version)
	return args.Error(0)
}

func createServer(name string) v1alpha1.Server {
	return v1alpha1.Server{
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

func createNic(name string) v1alpha1.Nic {
	return v1alpha1.Nic{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				serverSetLabel: serverSetName,
			},
		},
	}
}

func createBootVolume(name string) v1alpha1.Volume {
	return v1alpha1.Volume{
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
	return &volume
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
								DHCP:      true,
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

func createServerSetWithUpdatedBootVolume(updatedSpec v1alpha1.ServerSetBootVolumeSpec) *v1alpha1.ServerSet {
	sset := createBasicServerSet()
	sset.Spec.ForProvider.BootVolumeTemplate = v1alpha1.BootVolumeTemplate{
		Spec: updatedSpec,
	}
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
			DHCP:      true,
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
	return &server
}

func createServerWithIndex(name string, index int) *v1alpha1.Server {
	server := createServer(name)
	indexLabelBootVolume := fmt.Sprintf(indexLabel, serverSetName, ResourceServer)
	server.ObjectMeta.Labels[indexLabelBootVolume] = fmt.Sprintf("%d", index)
	return &server
}
