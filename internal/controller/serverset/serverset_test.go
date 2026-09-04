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
	"strconv"
	"testing"
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	ionoscloud "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	serverctrl "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/server"
)

const (
	bootVolumeSize       = 100
	bootVolumeType       = "HDD"
	bootVolumeImage      = "image"
	bootVolumeNamePrefix = "boot-volume-"

	deleteMethod              = "Delete"
	ensureMethod              = "Ensure"
	ensureNICsMethod          = "EnsureNICs"
	ensureFirewallRulesMethod = "EnsureFirewallRules"
	getMethod                 = "Get"
	updateMethod              = "Update"

	noReplicas = 2
	nic1UUID   = "nic1UUID"
	nic2UUID   = "nic2UUID"

	server1Name        = "serverset-server-0-0"
	server2Name        = "serverset-server-1-0"
	serverNotReadyName = "server-not-ready"
	serverSetCPUFamily = "INTEL_XEON"
	serverSetCores     = 2
	serverSetRAM       = 4096
	serverSetName      = "serverset"
	serverName         = "server-name"

	stateMapName      = "state-map"
	stateMapNamespace = "default"
	vmNotRunningState = "VM-NOT-RUNNING"

	reconcileErrorMsg = "some reconcile error happened"

	pciSlot = 2
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

type crType string

const (
	nic            crType = "Nic"
	server         crType = "Server"
	volume         crType = "Volume"
	volumeSelector crType = "VolumeSelector"
)

var errAnErrorWasReceived = errors.New("an error was received")

type kubeBootVolumeControlManagerFake struct {
	kubeBootVolumeControlManager
	mock.Mock
}

type kubeBootVolumeCallTracker struct {
	kubeBootVolumeControlManager
	lastMethodCall map[ServiceMethodName][]any
}

type kubeServerControlManagerFake struct {
	kubeServerControlManager
	mock.Mock
}

type kubeServerCallTracker struct {
	kubeServerControlManager
	lastMethodCall map[ServiceMethodName][]any
}

type kubeNicControlManagerFake struct {
	kubeNicControlManager
	mock.Mock
}

type kubeNicCallTracker struct {
	kubeNicControlManager
	lastMethodCall map[ServiceMethodName][]any
}

type kubeFirewallRuleControlManagerFake struct {
	kubeFirewallRuleControlManager
	mock.Mock
}

type kubeFirewallRuleCallTracker struct {
	kubeFirewallRuleControlManager
	lastMethodCall map[ServiceMethodName][]any
}

type kubeClientFake struct {
	client.Client
	mock.Mock
	crShouldReturnErr map[crType]bool
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
	nic1 := createNic(v1alpha1.NicParameters{Name: server1Name})
	nic2 := createNic(v1alpha1.NicParameters{Name: server2Name})
	bootVolume1 := createBootVolumeWithHotPlug(bootVolumeNamePrefix + server1Name)
	bootVolume2 := createBootVolumeWithHotPlug(bootVolumeNamePrefix + server2Name)

	tests := []struct {
		name                   string
		fields                 fields
		args                   args
		want                   managed.ExternalObservation
		wantErr                bool
		wantAvailableCondition bool
		wantCreatingCondition  bool
	}{
		{
			name: "servers, nics and boot volumes created without state map",
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
			name: "servers, nics and boot volumes created with state map, all servers in VM-RUNNING state",
			fields: fields{
				kube: fakeKubeClientObjs(server1, server2, bootVolume1, bootVolume2, nic1, nic2, createStateMapRunning()),
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSetWithStateMap(),
			},
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  true,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr:                false,
			wantAvailableCondition: true,
		},
		{
			name: "servers, nics and boot volumes created, but with server in VM-ERROR state in state map",
			fields: fields{
				kube: fakeKubeClientObjs(server1, server2, bootVolume1, bootVolume2, nic1, nic2, createStateMapOneVMError()),
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSetWithStateMap(),
			},
			want:    managed.ExternalObservation{},
			wantErr: true,
		},
		{
			name: "servers, nics and boot volumes created, but with server in VM-NOT-RUNNING state in state map",
			fields: fields{
				kube: fakeKubeClientObjs(server1, server2, bootVolume1, bootVolume2, nic1, nic2, createStateMapOneVMNotRunning()),
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSetWithStateMap(),
			},
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  true,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr:               false,
			wantCreatingCondition: true,
		},
		{
			name: "servers, nics and boot volumes created, state map is defined, but not created yet",
			fields: fields{
				kube: fakeKubeClientObjs(server1, server2, bootVolume1, bootVolume2, nic1, nic2),
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSetWithStateMap(),
			},
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  true,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr:               false,
			wantCreatingCondition: true,
		},
		{
			name: "servers, nics and boot volumes created, but one server has wrong timestamp format in state map",
			fields: fields{
				kube: fakeKubeClientObjs(server1, server2, bootVolume1, bootVolume2, nic1, nic2, createStateMapOneVMWrongTimestampFormat()),
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSetWithStateMap(),
			},
			want:    managed.ExternalObservation{},
			wantErr: true,
		},
		{
			name: "servers, nics and boot volumes created, but one server is missing state in state map",
			fields: fields{
				kube: fakeKubeClientObjs(server1, server2, bootVolume1, bootVolume2, nic1, nic2, createStateMapOneVMMissingState()),
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSetWithStateMap(),
			},
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  true,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr:               false,
			wantCreatingCondition: true,
		},
		{
			name: "servers, nics and boot volumes created, but one server is missing timestamp in state map",
			fields: fields{
				kube: fakeKubeClientObjs(server1, server2, bootVolume1, bootVolume2, nic1, nic2, createStateMapOneVMMissingStateTimestamp()),
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSetWithStateMap(),
			},
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  true,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr:               false,
			wantCreatingCondition: true,
		},
		{
			name: "servers, nics and boot volumes created, but state map is empty",
			fields: fields{
				kube: fakeKubeClientObjs(server1, server2, bootVolume1, bootVolume2, nic1, nic2, createStateMapEmpty()),
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSetWithStateMap(),
			},
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  true,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr:               false,
			wantCreatingCondition: true,
		},
		{
			name: "servers, nics and boot volumes created with state map, zero timestamp bypasses staleness check",
			fields: fields{
				kube: fakeKubeClientObjs(server1, server2, bootVolume1, bootVolume2, nic1, nic2, createStateMapZeroTimestamp()),
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSetWithStateMap(),
			},
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  true,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr:                false,
			wantAvailableCondition: true,
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
				Diff:              "servers: expected=2 actual=0 | bootVolumes: expected=2 actual=0 | nics: expected=2 actual=0",
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
					CPUFamily: "INTEL_SKYLAKE",
					Cores:     serverSetCores,
					RAM:       serverSetRAM,
				}),
			},
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  false,
				Diff:              "server[0](serverset-server-0-0): cpuFamily exp=INTEL_SKYLAKE act=INTEL_XEON | server[1](serverset-server-1-0): cpuFamily exp=INTEL_SKYLAKE act=INTEL_XEON",
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
				Diff:              "server[0](serverset-server-0-0): cores exp=10 act=2 | server[1](serverset-server-1-0): cores exp=10 act=2",
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
				Diff:              "server[0](serverset-server-0-0): ram exp=8192 act=4096 | server[1](serverset-server-1-0): ram exp=8192 act=4096",
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
				Diff:              "volume[0](boot-volume-serverset-server-0-0): image exp=newImage act=image | volume[1](boot-volume-serverset-server-1-0): image exp=newImage act=image",
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
				Diff:              "volume[0](boot-volume-serverset-server-0-0): size exp=300 act=100 | volume[1](boot-volume-serverset-server-1-0): size exp=300 act=100",
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
				Diff:              "volume[0](boot-volume-serverset-server-0-0): type exp=SSD act=HDD | volume[1](boot-volume-serverset-server-1-0): type exp=SSD act=HDD",
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
				Diff:              "servers: expected=2 actual=1 | bootVolumes: expected=2 actual=0 | nics: expected=2 actual=0",
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
				Diff:              "nics: expected=2 actual=0",
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
				Diff:              "nics: expected=4 actual=2",
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

			if tt.wantCreatingCondition {
				cond := tt.args.cr.GetCondition(xpv1.TypeReady)
				assert.Equalf(t, xpv1.Creating().Status, cond.Status, "Creating condition status mismatch")
			}

			if tt.wantAvailableCondition {
				cond := tt.args.cr.GetCondition(xpv1.TypeReady)
				assert.Equalf(t, xpv1.Available().Status, cond.Status, "Available condition status mismatch")
			}
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

	server1 := createServer("server-0-0")
	server2 := createServer("server-1-0")
	label := fmt.Sprintf(indexLabel, serverSetName, ResourceServer)
	server2.Labels[label] = "1"

	serverWithErrorStatus := createServer("serverset-server-0-0")
	serverWithErrorStatus.Status.AtProvider.State = ionoscloud.Failed

	serverWithUnknownStatus := createServer("server-0-0")
	serverWithUnknownStatus.Status.AtProvider.State = "new-state"

	nic1 := createNic(v1alpha1.NicParameters{Name: server1.Name})
	nic1.Status.AtProvider.NicID = nic1UUID
	nic1.Status.AtProvider.PCISlot = pciSlot
	nic1.Labels[serverSetNicIndexLabel] = "0"
	nic2 := createNic(v1alpha1.NicParameters{Name: server2.Name})
	nic2.Status.AtProvider.NicID = nic2UUID
	nic2.Status.AtProvider.PCISlot = pciSlot
	nic2.Labels[serverSetNicIndexLabel] = "1"

	tests := []struct {
		name   string
		fields fields
		args   args
		want   v1alpha1.ServerSetObservation
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
						Hostname:     getNameFrom(serverName, 0, 0),
						Status:       statusReady,
						Role:         "ACTIVE",
						ReplicaIndex: 0,
						NICStatuses: []v1alpha1.NicStatus{
							{
								AtProvider: v1alpha1.NicObservation{
									NicID:   nic1UUID,
									PCISlot: pciSlot,
								},
							},
						},
						ErrorMessage: "",
					},
					{
						Name:         server2.Name,
						Hostname:     getNameFrom(serverName, 1, 0),
						Status:       statusReady,
						Role:         "PASSIVE",
						ReplicaIndex: 1,
						NICStatuses: []v1alpha1.NicStatus{
							{
								AtProvider: v1alpha1.NicObservation{
									NicID:   nic2UUID,
									PCISlot: pciSlot,
								},
							},
						},
						ErrorMessage: "",
					},
				},
			},
		},
		{
			name: "configMap missing, then replica 0 defaults to ACTIVE",
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
						Hostname:     getNameFrom(serverName, 0, 0),
						Status:       statusReady,
						Role:         "ACTIVE",
						ReplicaIndex: 0,
						NICStatuses: []v1alpha1.NicStatus{
							{
								AtProvider: v1alpha1.NicObservation{
									NicID:   nic1UUID,
									PCISlot: pciSlot,
								},
							},
						},
						ErrorMessage: "",
					},
					{
						Name:         server2.Name,
						Hostname:     getNameFrom(serverName, 1, 0),
						Status:       statusReady,
						Role:         "PASSIVE",
						ReplicaIndex: 1,
						NICStatuses: []v1alpha1.NicStatus{
							{
								AtProvider: v1alpha1.NicObservation{
									NicID:   nic2UUID,
									PCISlot: pciSlot,
								},
							},
						},
						ErrorMessage: "",
					},
				},
			},
		},
		{
			name: "replicas not in configMap, then role for replica 0 defaults to ACTIVE",
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
						Hostname:     getNameFrom(serverName, 0, 0),
						Status:       statusReady,
						Role:         "ACTIVE",
						ReplicaIndex: 0,
						NICStatuses: []v1alpha1.NicStatus{
							{
								AtProvider: v1alpha1.NicObservation{
									NicID:   nic1UUID,
									PCISlot: pciSlot,
								},
							},
						},
						ErrorMessage: "",
					},
					{
						Name:         server2.Name,
						Hostname:     getNameFrom(serverName, 1, 0),
						Status:       statusReady,
						Role:         "PASSIVE",
						ReplicaIndex: 1,
						NICStatuses: []v1alpha1.NicStatus{
							{
								AtProvider: v1alpha1.NicObservation{
									NicID:   nic2UUID,
									PCISlot: pciSlot,
								},
							},
						},
						ErrorMessage: "",
					},
				},
			},
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
						Hostname:     getNameFrom(serverName, 0, 0),
						Status:       statusReady,
						Role:         "ACTIVE",
						ReplicaIndex: 0,
						NICStatuses: []v1alpha1.NicStatus{
							{
								AtProvider: v1alpha1.NicObservation{
									NicID:   nic1UUID,
									PCISlot: pciSlot,
								},
							},
						},
						ErrorMessage: "",
					},
					{
						Name:         server2.Name,
						Status:       statusReady,
						Role:         "PASSIVE",
						Hostname:     getNameFrom(serverName, 1, 0),
						ReplicaIndex: 1,
						NICStatuses: []v1alpha1.NicStatus{
							{
								AtProvider: v1alpha1.NicObservation{
									NicID:   nic2UUID,
									PCISlot: pciSlot,
								},
							},
						},
						ErrorMessage: "",
					},
				},
			},
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
						Hostname:     getNameFrom(serverName, 0, 0),
						Status:       statusReady,
						Role:         "ACTIVE",
						ReplicaIndex: 0,
						NICStatuses: []v1alpha1.NicStatus{
							{
								AtProvider: v1alpha1.NicObservation{
									NicID:   nic1UUID,
									PCISlot: pciSlot,
								},
							},
						},
						ErrorMessage: "",
					},
				},
			},
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
						Hostname:     getNameFrom(serverName, 0, 0),
						ReplicaIndex: 0,
						NICStatuses:  []v1alpha1.NicStatus{},
						ErrorMessage: "",
					},
				},
			},
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
						Hostname:     getNameFrom(serverName, 0, 0),
						Status:       statusError,
						Role:         "PASSIVE",
						ReplicaIndex: 0,
						NICStatuses:  []v1alpha1.NicStatus{},
						ErrorMessage: reconcileErrorMsg,
					},
				},
			},
		},
		{
			name: "status of the server not among known ones, then status of replica is also UNKNOWN",
			fields: fields{
				kube: fakeKubeClientObjs(serverWithUnknownStatus, nic1),
			},
			args: args{
				ctx: context.Background(),
				cr:  createServerSetWithOneReplica(),
			},
			want: v1alpha1.ServerSetObservation{
				Replicas: 1,
				ReplicaStatuses: []v1alpha1.ServerSetReplicaStatus{
					{
						Name:         serverWithUnknownStatus.Name,
						Hostname:     getNameFrom(serverName, 0, 0),
						Status:       statusUnknown,
						Role:         "PASSIVE",
						ReplicaIndex: 0,
						NICStatuses: []v1alpha1.NicStatus{
							{
								AtProvider: v1alpha1.NicObservation{
									NicID:   nic1UUID,
									PCISlot: pciSlot,
								},
							},
						},
						ErrorMessage: "",
					},
				},
			},
		},
		{
			name: "server not ready yet",
			fields: fields{
				kube: fakeKubeClientObjs(createServerNotReadyYet(), nic1),
			},
			args: args{
				ctx: context.Background(),
				cr:  createServerSetWithOneReplica(),
			},
			want: v1alpha1.ServerSetObservation{
				Replicas: 1,
				ReplicaStatuses: []v1alpha1.ServerSetReplicaStatus{
					{
						Name:     serverNotReadyName,
						Hostname: getNameFrom(serverName, 0, 0),
						Status:   statusBusy,
						Role:     "PASSIVE",
						NICStatuses: []v1alpha1.NicStatus{
							{
								AtProvider: v1alpha1.NicObservation{
									NicID:   nic1UUID,
									PCISlot: pciSlot,
								},
							},
						},
						ErrorMessage: "",
					},
				},
			},
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

			assert.NoError(t, err)
			areEqual(t, tt.want, got)
		})
	}
}

func Test_serverSetController_Create(t *testing.T) {
	type fields struct {
		kube                   client.Client
		bootVolumeController   kubeBootVolumeControlManager
		nicController          kubeNicControlManager
		serverController       kubeServerControlManager
		firewallRuleController kubeFirewallRuleControlManager
		log                    logging.Logger
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
				bootVolumeController: fakeBootVolumeCtrlGetEnsure(),
				serverController:     fakeServerCtrlGetEnsure(),
				// will not get called due to listresources by lavel where it doesn't find a server
				nicController:          fakeNicCtrlEnsureNICsMethod(0),
				firewallRuleController: fakeFirewallRuleCtrlEnsureMethod(0),
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
					createBootVolumeWithIndexWithHotPlug("boot-volume1", 0),
					createBootVolumeWithIndexWithHotPlug("boot-volume2", 0),
				),
				bootVolumeController:   new(kubeBootVolumeControlManagerFake),
				serverController:       fakeServerCtrlEnsureMethod(0),
				nicController:          fakeNicCtrlEnsureNICsMethod(0),
				firewallRuleController: fakeFirewallRuleCtrlEnsureMethod(0),
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
				log:                    logging.NewNopLogger(),
				kube:                   fakeKubeClientObjs(),
				bootVolumeController:   fakeBootVolumeCtrlEnsureMethodReturnsErr(),
				serverController:       fakeServerCtrlEnsureMethod(1),
				nicController:          fakeNicCtrlEnsureNICsMethod(0),
				firewallRuleController: fakeFirewallRuleCtrlEnsureMethod(0),
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSet(),
			},
			want:    managed.ExternalCreation{},
			wantErr: fmt.Errorf("while ensuring bootVolume (%w)", errAnErrorWasReceived),
		},
		{
			name: "too many servers returned for the same index",
			fields: fields{
				log: logging.NewNopLogger(),
				kube: fakeKubeClientObjs(
					createServerWithIndex("server1", 0),
					createServerWithIndex("server2", 0)),
				bootVolumeController:   new(kubeBootVolumeControlManagerFake),
				serverController:       fakeServerCtrlEnsureMethod(0),
				nicController:          fakeNicCtrlEnsureNICsMethod(0),
				firewallRuleController: fakeFirewallRuleCtrlEnsureMethod(0),
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
				log:                    logging.NewNopLogger(),
				kube:                   fakeKubeClientObjs(),
				bootVolumeController:   new(kubeBootVolumeControlManagerFake),
				serverController:       fakeServerCtrlEnsureMethodReturnsErr(),
				nicController:          fakeNicCtrlEnsureNICsMethod(0),
				firewallRuleController: fakeFirewallRuleCtrlEnsureMethod(0),
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
				log:                    logging.NewNopLogger(),
				kube:                   fakeKubeClientOneServer(),
				bootVolumeController:   new(kubeBootVolumeControlManagerFake),
				serverController:       new(kubeServerControlManagerFake),
				nicController:          fakeNicCtrlEnsureNICsMethodReturnsErr(),
				firewallRuleController: fakeFirewallRuleCtrlEnsureMethod(0),
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
				kube:                   tt.fields.kube,
				bootVolumeController:   tt.fields.bootVolumeController,
				nicController:          tt.fields.nicController,
				serverController:       tt.fields.serverController,
				firewallRuleController: tt.fields.firewallRuleController,
				log:                    tt.fields.log,
			}

			got, err := e.Create(tt.args.ctx, tt.args.cr)

			assertions := assert.New(t)
			fakeBootVolumeCtrl := tt.fields.bootVolumeController.(*kubeBootVolumeControlManagerFake)
			fakeBootVolumeCtrl.AssertExpectations(t)

			fakeServerCtrl := tt.fields.serverController.(*kubeServerControlManagerFake)
			fakeServerCtrl.AssertExpectations(t)

			fakeNicCtrl := tt.fields.nicController.(*kubeNicControlManagerFake)
			fakeNicCtrl.AssertExpectations(t)

			fakeFirewallRuleCtrl := tt.fields.firewallRuleController.(*kubeFirewallRuleControlManagerFake)
			fakeFirewallRuleCtrl.AssertExpectations(t)

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
		wantWrappedErr  error
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
					CPUFamily: "INTEL_SKYLAKE",
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
			name: "failure in kube client when updating server",
			fields: fields{
				kube: fakeKubeClientUpdateMethodReturnsError(),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedServerSpec(v1alpha1.ServerSetTemplateSpec{
					CPUFamily: "INTEL_SKYLAKE",
					Cores:     serverSetCores,
					RAM:       serverSetRAM,
				}),
			},
			wantErr:         fmt.Errorf("error updating server %w", errAnErrorWasReceived),
			want:            managed.ExternalUpdate{},
			wantUpdateCalls: 1,
		},
		{
			name: "update server with successful failover (CPU non-hotpluggable change) without state map",
			fields: fields{
				kube: fakeKubeClientUpdateMethodWithSuccessfulFailover(&v1alpha1.Server{}),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedServerSpec(
					v1alpha1.ServerSetTemplateSpec{
						CPUFamily: serverSetCPUFamily,
						Cores:     10,
						RAM:       serverSetRAM,
					},
				),
			},
			wantErr: nil,
			want: managed.ExternalUpdate{
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantUpdateCalls: 2,
		},
		{
			name: "update server with successful failover (RAM non-hotpluggable change) without state map",
			fields: fields{
				kube: fakeKubeClientUpdateMethodWithSuccessfulFailover(&v1alpha1.Server{}),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedServerSpec(
					v1alpha1.ServerSetTemplateSpec{
						CPUFamily: serverSetCPUFamily,
						Cores:     serverSetCores,
						RAM:       8192,
					},
				),
			},
			wantErr: nil,
			want: managed.ExternalUpdate{
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantUpdateCalls: 2,
		},
		{
			name: "update server with successful failover (RAM non-hotpluggable change) with state map",
			fields: fields{
				kube: fakeKubeClientUpdateMethodWithStateMapSuccessfulFailover(&v1alpha1.Server{}),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedServerSpecWithStateMap(
					v1alpha1.ServerSetTemplateSpec{
						CPUFamily: serverSetCPUFamily,
						Cores:     serverSetCores,
						RAM:       8192,
					},
				),
			},
			wantErr: nil,
			want: managed.ExternalUpdate{
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantUpdateCalls: 2,
		},
		{
			name: "update server with failed failover (CPU non-hotpluggable change) without state map",
			fields: fields{
				kube: fakeKubeClientUpdateMethodWithFailedFailover(&v1alpha1.Server{}),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedServerSpec(
					v1alpha1.ServerSetTemplateSpec{
						CPUFamily: serverSetCPUFamily,
						Cores:     10,
						RAM:       serverSetRAM,
					},
				),
			},
			wantWrappedErr:  fmt.Errorf("error waiting for server to be updated"),
			want:            managed.ExternalUpdate{},
			wantUpdateCalls: 1,
		},
		{
			name: "update server with failed failover (RAM non-hotpluggable change) without state map",
			fields: fields{
				kube: fakeKubeClientUpdateMethodWithFailedFailover(&v1alpha1.Server{}),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedServerSpecWithStateMap(
					v1alpha1.ServerSetTemplateSpec{
						CPUFamily: serverSetCPUFamily,
						Cores:     serverSetCores,
						RAM:       8192,
					},
				),
			},
			// When state map is missing, pre-reboot validation fails because VM is not ready
			wantWrappedErr:  fmt.Errorf("software is not yet running"),
			want:            managed.ExternalUpdate{},
			wantUpdateCalls: 0,
		},
		{
			name: "update server with failed failover (RAM non-hotpluggable change) with state map",
			fields: fields{
				kube: fakeKubeClientUpdateMethodWithStateMapFailedFailover(&v1alpha1.Server{}),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedServerSpecWithStateMap(
					v1alpha1.ServerSetTemplateSpec{
						CPUFamily: serverSetCPUFamily,
						Cores:     serverSetCores,
						RAM:       8192,
					},
				),
			},
			// Pre-reboot validation now fails fast when any VM reports VM-ERROR
			wantWrappedErr:  fmt.Errorf("VM-ERROR runtime state"),
			want:            managed.ExternalUpdate{},
			wantUpdateCalls: 0,
		},
		{
			name: "update server with successful failover (RAM non-hotpluggable change) with zero timestamp in state map",
			fields: fields{
				kube: fakeKubeClientUpdateMethodWithStateMapZeroTimestampSuccessfulFailover(&v1alpha1.Server{}),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedServerSpecWithStateMap(
					v1alpha1.ServerSetTemplateSpec{
						CPUFamily: serverSetCPUFamily,
						Cores:     serverSetCores,
						RAM:       8192,
					},
				),
			},
			wantErr: nil,
			want: managed.ExternalUpdate{
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantUpdateCalls: 2,
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

			got, err := e.Update(tt.args.ctx, tt.args.cr)

			assertions := require.New(t)

			if tt.wantWrappedErr != nil {
				assertions.ErrorContains(err, tt.wantWrappedErr.Error())
			} else {
				assertions.Equalf(tt.wantErr, err, "Wrong error")
			}
			assertions.Equalf(tt.want, got, "Wrong response")
			assertions.Equalf(0, len(tt.args.cr.Status.Conditions), "ServerSet should not have any conditions")
			kubeClient := tt.fields.kube.(*kubeClientFake)
			kubeClient.AssertNumberOfCalls(t, "Update", tt.wantUpdateCalls)
		})
	}
}

func Test_serverSetController_BootVolumeUpdate(t *testing.T) {
	type fields struct {
		kube                   client.Client
		bootVolumeController   kubeBootVolumeControlManager
		nicController          kubeNicControlManager
		serverController       kubeServerControlManager
		firewallRuleController kubeFirewallRuleControlManager
		log                    logging.Logger
	}
	type args struct {
		ctx context.Context
		cr  *v1alpha1.ServerSet
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantErr        error // exact matching
		wantWrappedErr error // substring matching
		want           managed.ExternalUpdate
		wantCalls      map[ServiceMethodName]int
	}{
		{
			name: "updated using default strategy (image changed)",
			fields: fields{
				kube:                   fakeKubeClientUpdateMethodForBootVolume(),
				bootVolumeController:   fakeBootVolumeCtrl(),
				serverController:       fakeServerCtrl(),
				nicController:          fakeNicCtrl(),
				firewallRuleController: fakeFirewallRuleCtrl(),
				log:                    logging.NewNopLogger(),
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
				serverGet:        1,
				serverUpdate:     1,
				bootVolumeEnsure: 1,
				bootVolumeDelete: 1,
				bootVolumeGet:    2,
			},
		},
		{
			name: "updated using default strategy (type changed)",
			fields: fields{
				kube:                   fakeKubeClientUpdateMethodForBootVolume(),
				bootVolumeController:   fakeBootVolumeCtrl(),
				serverController:       fakeServerCtrl(),
				nicController:          fakeNicCtrl(),
				firewallRuleController: fakeFirewallRuleCtrl(),
				log:                    logging.NewNopLogger(),
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
				serverGet:        1,
				serverUpdate:     1,
				bootVolumeEnsure: 1,
				bootVolumeDelete: 1,
				bootVolumeGet:    2,
			},
		},
		{
			name: "updated using createAllBeforeDestroy strategy (type changed)",
			fields: fields{
				kube:                   fakeKubeClientUpdateMethodForBootVolume(),
				bootVolumeController:   fakeBootVolumeCtrl(),
				nicController:          fakeNicCtrl(),
				serverController:       fakeServerCtrl(),
				firewallRuleController: fakeFirewallRuleCtrl(),
				log:                    logging.NewNopLogger(),
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
				serverEnsure:     1,
				serverDelete:     1,
				bootVolumeEnsure: 1,
				bootVolumeDelete: 1,
				nicEnsureNICs:    1,
				nicDelete:        1,
				bootVolumeGet:    1,
				serverGet:        1,
				serverUpdate:     1,
			},
		},
		{
			name: "updated (size changed)",
			fields: fields{
				kube:                   fakeKubeClientUpdateMethod(&v1alpha1.Volume{}),
				bootVolumeController:   fakeBootVolumeCtrl(),
				nicController:          fakeNicCtrl(),
				serverController:       fakeServerCtrl(),
				firewallRuleController: fakeFirewallRuleCtrl(),
				log:                    logging.NewNopLogger(),
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
			wantCalls: map[ServiceMethodName]int{
				kubeUpdate: 2,
			},
		},
		{
			name: "failed to update (size changed)",
			fields: fields{
				kube:                   fakeKubeClientUpdateMethodReturnsError(),
				bootVolumeController:   fakeBootVolumeCtrl(),
				nicController:          fakeNicCtrl(),
				serverController:       fakeServerCtrl(),
				firewallRuleController: fakeFirewallRuleCtrl(),
				log:                    logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedBootVolumeUsingDefaultStrategy(v1alpha1.ServerSetBootVolumeSpec{
					Size:  200,
					Image: bootVolumeImage,
					Type:  bootVolumeType,
				}),
			},
			wantErr: fmt.Errorf("while updating volumes for serverset serverset %w", fmt.Errorf("error updating volume %w", errAnErrorWasReceived)),
			want:    managed.ExternalUpdate{},
			wantCalls: map[ServiceMethodName]int{
				kubeUpdate: 1,
			},
		},
		{
			name: "failed to update using default strategy (type changed)",
			fields: fields{
				kube:                   fakeKubeClientUpdateMethodForBootVolume(),
				bootVolumeController:   fakeBootVolumeCtrlGetEnsureMethodReturnsErr(),
				nicController:          fakeNicCtrl(),
				serverController:       fakeServerCtrl(),
				firewallRuleController: fakeFirewallRuleCtrl(),
				log:                    logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedBootVolumeUsingDefaultStrategy(v1alpha1.ServerSetBootVolumeSpec{
					Size:  bootVolumeSize,
					Image: bootVolumeImage,
					Type:  "SSD",
				}),
			},
			wantErr: fmt.Errorf("while updating volumes for serverset serverset %w", errAnErrorWasReceived),
			want:    managed.ExternalUpdate{},
			wantCalls: map[ServiceMethodName]int{
				bootVolumeEnsure: 1,
				bootVolumeGet:    1,
			},
		},
		{
			name: "updated with state map - all VMs ready before boot volume update (image changed)",
			fields: fields{
				kube:                   fakeKubeClientUpdateMethodForBootVolumeWithStateMap(),
				bootVolumeController:   fakeBootVolumeCtrl(),
				serverController:       fakeServerCtrl(),
				nicController:          fakeNicCtrl(),
				firewallRuleController: fakeFirewallRuleCtrl(),
				log:                    logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedBootVolumeWithStateMap(v1alpha1.ServerSetBootVolumeSpec{
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
				serverGet:        1,
				serverUpdate:     1,
				bootVolumeEnsure: 1,
				bootVolumeDelete: 1,
				bootVolumeGet:    2,
			},
		},
		{
			name: "failed to update with state map - VM in error state before boot volume update",
			fields: fields{
				kube:                   fakeKubeClientUpdateMethodForBootVolumeWithStateMapVMError(),
				bootVolumeController:   fakeBootVolumeCtrl(),
				serverController:       fakeServerCtrl(),
				nicController:          fakeNicCtrl(),
				firewallRuleController: fakeFirewallRuleCtrl(),
				log:                    logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedBootVolumeWithStateMap(v1alpha1.ServerSetBootVolumeSpec{
					Size:  bootVolumeSize,
					Image: "newImage",
					Type:  bootVolumeType,
				}),
			},
			wantWrappedErr: fmt.Errorf("error validating all VMs are ready for boot volume update"),
			want:           managed.ExternalUpdate{},
			wantCalls:      map[ServiceMethodName]int{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &external{
				kube:                   tt.fields.kube,
				bootVolumeController:   tt.fields.bootVolumeController,
				nicController:          tt.fields.nicController,
				serverController:       tt.fields.serverController,
				firewallRuleController: tt.fields.firewallRuleController,
				log:                    tt.fields.log,
			}

			got, err := e.Update(tt.args.ctx, tt.args.cr)

			assertions := assert.New(t)
			if tt.wantWrappedErr != nil {
				assertions.ErrorContains(err, tt.wantWrappedErr.Error()) //nolint:testifylint // prefer to continue test execution
			} else {
				assertions.Equalf(tt.wantErr, err, "Wrong error")
			}
			assertions.Equalf(tt.want, got, "Wrong response")
			assertions.Equalf(0, len(tt.args.cr.Status.Conditions), "ServerSet should not have any conditions")

			kubeClient := tt.fields.kube.(*kubeClientFake)
			kubeClient.AssertNumberOfCalls(t, updateMethod, tt.wantCalls[kubeUpdate])

			bootVolumeCtrl := tt.fields.bootVolumeController.(*kubeBootVolumeControlManagerFake)
			bootVolumeCtrl.AssertNumberOfCalls(t, ensureMethod, tt.wantCalls[bootVolumeEnsure])
			bootVolumeCtrl.AssertNumberOfCalls(t, deleteMethod, tt.wantCalls[bootVolumeDelete])
			bootVolumeCtrl.AssertNumberOfCalls(t, getMethod, tt.wantCalls[bootVolumeGet])

			serverController := tt.fields.serverController.(*kubeServerControlManagerFake)
			serverController.AssertNumberOfCalls(t, ensureMethod, tt.wantCalls[serverEnsure])
			serverController.AssertNumberOfCalls(t, deleteMethod, tt.wantCalls[serverDelete])
			serverController.AssertNumberOfCalls(t, getMethod, tt.wantCalls[serverGet])
			serverController.AssertNumberOfCalls(t, updateMethod, tt.wantCalls[serverUpdate])

			nicCtrl := tt.fields.nicController.(*kubeNicControlManagerFake)
			nicCtrl.AssertNumberOfCalls(t, ensureNICsMethod, tt.wantCalls[nicEnsureNICs])
			nicCtrl.AssertNumberOfCalls(t, deleteMethod, tt.wantCalls[nicDelete])
		})
	}
}

func Test_serverSetController_updateOrRecreateVolumes_activeReplicaUpdatedLast_defaultUpdateStrategy(t *testing.T) {
	const thirdArg = 2
	const secondArg = 1
	ctx := context.Background()
	cr := createServerSetWithUpdatedBootVolumeUsingDefaultStrategy(v1alpha1.ServerSetBootVolumeSpec{
		Size:  bootVolumeSize,
		Image: bootVolumeImage,
		Type:  "SSD",
	})
	bootVolumes := []v1alpha1.Volume{
		*createBootVolumeWithIndexLabelsWithHotPlug("bootvolumename-0-0", 0),
		*createBootVolumeWithIndexLabelsWithHotPlug("bootvolumename-1-0", 1),
	}
	masterIndex := 0
	e := external{
		kube: fakeKubeClientUpdateMethodForBootVolume(),
		bootVolumeController: &kubeBootVolumeCallTracker{
			lastMethodCall: make(map[ServiceMethodName][]any),
		},
		serverController: &kubeServerCallTracker{
			lastMethodCall: make(map[ServiceMethodName][]any),
		},
		log: logging.NewNopLogger(),
	}

	err := e.updateOrRecreateVolumes(ctx, cr, bootVolumes, masterIndex)

	assertions := assert.New(t)
	assertions.NoError(err, "Expected no error")

	kubeClient := e.kube.(*kubeClientFake)
	kubeClient.AssertNumberOfCalls(t, updateMethod, 0)

	bootVolumeController := e.bootVolumeController.(*kubeBootVolumeCallTracker)
	assertions.Equal(1, bootVolumeController.lastMethodCall[ensureMethod][thirdArg])
	assertions.Equal("bootvolumename-1-1", bootVolumeController.lastMethodCall[getMethod][secondArg])
	assertions.Equal("bootvolumename-1-0", bootVolumeController.lastMethodCall[deleteMethod][secondArg])

	serverController := e.serverController.(*kubeServerCallTracker)
	assertions.Equal("server-name-1-0", serverController.lastMethodCall[getMethod][secondArg])
	actualServer := serverController.lastMethodCall[updateMethod][secondArg].(*v1alpha1.Server)
	assertions.Equal("bootvolumename-1-1-uuid", actualServer.Spec.ForProvider.VolumeCfg.VolumeID)
}

func Test_serverSetController_updateOrRecreateVolumes_activeReplicaUpdatedLast_createBeforeDestroyUpdateStrategy(t *testing.T) {
	const thirdArg = 2
	const secondArg = 1
	ctx := context.Background()
	cr := createServerSetWithUpdatedBootVolumeUsingStrategy(v1alpha1.ServerSetBootVolumeSpec{
		Size:  bootVolumeSize,
		Image: bootVolumeImage,
		Type:  "SSD"}, v1alpha1.UpdateStrategy{Stype: v1alpha1.CreateAllBeforeDestroy},
	)
	bootVolumes := []v1alpha1.Volume{
		*createBootVolumeWithIndexLabelsWithHotPlug("bootvolumename-0-0", 0),
		*createBootVolumeWithIndexLabelsWithHotPlug("bootvolumename-1-0", 1),
	}
	masterIndex := 0
	updatedIndex := 1
	e := external{
		kube: fakeKubeClientUpdateMethodForBootVolume(),
		bootVolumeController: &kubeBootVolumeCallTracker{
			lastMethodCall: make(map[ServiceMethodName][]any),
		},
		serverController: &kubeServerCallTracker{
			lastMethodCall: make(map[ServiceMethodName][]any),
		},
		nicController: &kubeNicCallTracker{
			lastMethodCall: make(map[ServiceMethodName][]any),
		},
		firewallRuleController: &kubeFirewallRuleCallTracker{
			lastMethodCall: make(map[ServiceMethodName][]any),
		},
		log: logging.NewNopLogger(),
	}

	err := e.updateOrRecreateVolumes(ctx, cr, bootVolumes, masterIndex)

	assertions := assert.New(t)
	assertions.NoError(err, "Expected no error")

	kubeClient := e.kube.(*kubeClientFake)
	kubeClient.AssertNumberOfCalls(t, updateMethod, 0)

	bootVolumeController := e.bootVolumeController.(*kubeBootVolumeCallTracker)
	assertions.Equal(updatedIndex, bootVolumeController.lastMethodCall[ensureMethod][thirdArg])
	assertions.Equal("bootvolumename-1-0", bootVolumeController.lastMethodCall[deleteMethod][secondArg])

	serverController := e.serverController.(*kubeServerCallTracker)
	assertions.Equal(updatedIndex, serverController.lastMethodCall[ensureMethod][thirdArg])
	assertions.Equal("server-name-1-0", serverController.lastMethodCall[deleteMethod][secondArg])

	nicController := e.nicController.(*kubeNicCallTracker)
	assertions.Equal(updatedIndex, nicController.lastMethodCall[ensureNICsMethod][thirdArg])
	assertions.Equal("nic1-1-0-0", nicController.lastMethodCall[deleteMethod][secondArg])
}

// Test_serverSetController_updateOrRecreateVolumes_usesReplicaIndexLabelNotListPosition
// reproduces ICNAS-854: updateOrRecreateVolumes() iterates the volumes slice using the raw
// `for idx := range volumes` loop position and passes that raw position downstream (as the
// argument to updateWithFailoverOrchestration/updateByIndex) as if it were the volume's real
// replica index. In production, the volumes slice comes from GetVolumesOfSSet(), which does an
// unsorted client.List() - list-position is not guaranteed to match each volume's own
// "<crName>-bv-ri" index label.
//
// This test calls updateOrRecreateVolumes() directly - like the two tests above it - and
// deliberately constructs the `volumes` slice with a list-position/label-index mismatch: the
// volume that actually needs recreating (its Type/Image differ from the new BootVolumeTemplate)
// carries the replica-index label "1", but is placed at slice position 0. The volume at slice
// position 1 carries replica-index label "0" and is already up to date with the new template.
//
// Bug behavior (current/unfixed code): the loop passes raw idx=0 downstream. That resolves
// (via the label-filtered getVersionsFromVolumeAndServer/ListResFromSSetWithIndex lookups) to
// bootvolumename-0-0 - the OTHER, already-up-to-date replica's volume - so
// createBeforeDestroyOnlyBootVolume.update's guard (recreate_only_bootvolume.go:31-35) sees a
// volume that already matches the template and silently `return nil`s. The real stale volume
// (bootvolumename-1-0, replica index 1) is never examined and bootVolumeController.Ensure is
// never called for it.
//
// Correct behavior (post-fix): the replica index should be derived from the volume's own index
// label (via ComputeReplicaIdx), so replica index 1 - the actually-stale volume - is the one
// passed downstream, and bootVolumeController.Ensure gets called for replica index 1 with the
// bumped volume version.
//
// This test asserts the CORRECT (post-fix) behavior, so it is expected to FAIL against the
// current, unfixed updateOrRecreateVolumes() implementation.
func Test_serverSetController_updateOrRecreateVolumes_usesReplicaIndexLabelNotListPosition(t *testing.T) {
	ctx := context.Background()

	newImage := "new-image"
	newType := "SSD"
	cr := createServerSetWithUpdatedBootVolumeUsingDefaultStrategy(v1alpha1.ServerSetBootVolumeSpec{
		Size:  bootVolumeSize,
		Image: newImage,
		Type:  newType,
	})

	// The volume that still needs to be recreated (Type/Image still match the OLD template),
	// labeled with replica index 1, but deliberately placed at slice position 0.
	staleVolumeLabeledIndex1 := createBootVolumeWithIndexLabelsWithHotPlug("bootvolumename-1-0", 1)

	// The volume that is already up to date with the NEW template, labeled with replica index 0,
	// placed at slice position 1.
	upToDateVolumeLabeledIndex0 := createBootVolumeWithIndexLabelsWithHotPlug("bootvolumename-0-0", 0)
	upToDateVolumeLabeledIndex0.Spec.ForProvider.Type = newType
	upToDateVolumeLabeledIndex0.Spec.ForProvider.Image = newImage

	// Deliberate list-position/label-index mismatch: position 0 holds the replica-index-1
	// (stale) volume; position 1 holds the replica-index-0 (already up to date) volume.
	bootVolumes := []v1alpha1.Volume{
		*staleVolumeLabeledIndex1,
		*upToDateVolumeLabeledIndex0,
	}

	// No active leader, so we isolate the loop-position-as-index bug from the separate
	// masterIndex==idx comparison bug (both are described in ICNAS-854, but this test targets
	// the primary raw-loop-position defect).
	masterIndex := -1

	bootVolumeController := new(kubeBootVolumeControlManagerFake)
	bootVolumeController.
		// Get() for the WRONG (list-position 0) volume - what the current buggy code inspects.
		// It already matches the new template, so the buggy code's guard silently no-ops here.
		On(getMethod, mock.Anything, "bootvolumename-0-0", mock.Anything).
		Return(&v1alpha1.Volume{
			Spec: v1alpha1.VolumeSpec{
				ForProvider: v1alpha1.VolumeParameters{
					Type:                 newType,
					Image:                newImage,
					SetHotPlugsFromImage: false,
				},
			},
		}, nil).
		// Get() for the REAL stale volume (replica index 1) - what the fixed code should
		// inspect. It still has the OLD Type/Image, so the guard should NOT short-circuit and
		// Ensure() should be called to recreate it.
		On(getMethod, mock.Anything, "bootvolumename-1-0", mock.Anything).
		Return(&v1alpha1.Volume{
			Spec: v1alpha1.VolumeSpec{
				ForProvider: v1alpha1.VolumeParameters{
					Type:                 bootVolumeType,
					Image:                bootVolumeImage,
					SetHotPlugsFromImage: false,
				},
			},
		}, nil).
		On(getMethod, mock.Anything, "bootvolumename-1-1", mock.Anything).
		Return(&v1alpha1.Volume{
			Status: v1alpha1.VolumeStatus{
				AtProvider: v1alpha1.VolumeObservation{VolumeID: "bootvolumename-1-1-uuid"},
			},
		}, nil).
		On(ensureMethod, mock.Anything, mock.Anything, 1, 1).Return(nil).
		On(deleteMethod, mock.Anything, "bootvolumename-1-0", mock.Anything).Return(nil)

	e := external{
		kube:                 fakeKubeClientUpdateMethodForBootVolume(),
		bootVolumeController: bootVolumeController,
		serverController: &kubeServerCallTracker{
			lastMethodCall: make(map[ServiceMethodName][]any),
		},
		log: logging.NewNopLogger(),
	}

	err := e.updateOrRecreateVolumes(ctx, cr, bootVolumes, masterIndex)
	require.NoError(t, err, "updateOrRecreateVolumes should not surface an error even when it (incorrectly) no-ops")

	// This is the crux of ICNAS-854: the volume actually needing recreation is labeled replica
	// index 1, so recreation must be driven for replica index 1, regardless of its position in
	// the volumes slice. Against the current, unfixed code this call is never made because the
	// raw loop position (0) is used instead of the volume's own index label (1), so the guard in
	// createBeforeDestroyOnlyBootVolume.update silently no-ops on the WRONG (already up to date)
	// volume instead.
	bootVolumeController.AssertCalled(t, ensureMethod, mock.Anything, cr, 1, 1)
}

// Test_getIdentityFromStatus_returnsReplicaIndexNotSlicePosition
// reproduces the second half of the ICNAS-854 index/position mismatch.
//
// populateReplicasStatuses() writes cr.Status.AtProvider.ReplicaStatuses[i] at the *list
// position* i of the unsorted GetServersOfSSet() result and stores the server's real replica
// index in the ReplicaIndex field. getIdentityFromStatus() currently returns the loop position
// instead of that field, so masterIndex is a slice position.
//
// That value is then used as if it were a replica index: updateOrRecreateVolumes() compares it
// against the label-derived replica index (serverset.go: `if masterIndex == replicaIdx`) and,
// when the leader's volume needs recreating, passes it to updateWithFailoverOrchestration().
// Whenever list order differs from replica-index order (the >=10 replica / mixed volume-version
// name-sorting case from ICNAS-854), the leader is either recreated inline - losing the
// "recreate the leader last" ordering - or the deferred recreation is driven for the wrong
// replica.
func Test_getIdentityFromStatus_returnsReplicaIndexNotSlicePosition(t *testing.T) {
	// Deliberate position/index mismatch, exactly as client.List() name-sorting produces it:
	// "server-name-10-0" sorts before "server-name-2-0", so the leader (replica index 2) ends up
	// at slice position 1.
	statuses := []v1alpha1.ServerSetReplicaStatus{
		{Name: "server-name-10-0", ReplicaIndex: 10, Role: v1alpha1.Passive},
		{Name: "server-name-2-0", ReplicaIndex: 22, Role: v1alpha1.Active},
	}

	assert.Equal(t, 22, getIdentityFromStatus(statuses),
		"the leader must be identified by its replica index, not by its position in the status slice")
}

// Test_serverSetController_updateWithFailoverOrchestration_usesServerMatchingReplicaIndexNotListPosition
// reproduces the same index/position mismatch inside updateWithFailoverOrchestration().
//
// The function receives a replica index but then picks the server object with
// `serverObj := servers[replicaIndex]`, indexing the unsorted GetServersOfSSet() slice by
// replica index. serverObj is what the post-update state-map wait polls, so a mismatch makes it
// watch the wrong VM - and when the replica index is not a valid position in the slice, the
// lookup panics with index out of range.
//
// The setup below is the ordinary mid-recreation state: replica 1's server has been deleted and
// not yet recreated, so the serverset temporarily has servers for replica indices 0 and 2 only.
// Driving a boot-volume recreation for replica 2 then indexes servers[2] on a 2-element slice.
//
// The serverset uses a state map, so that the post-update reboot wait - the only consumer of the
// server object resolved out of the list - is actually exercised.
//
// This test asserts the CORRECT (post-fix) behavior: the server has to be looked up by its index
// label, the way getServerVersion() does.
func Test_serverSetController_updateWithFailoverOrchestration_usesServerMatchingReplicaIndexNotListPosition(t *testing.T) {
	ctx := context.Background()
	cr := createBasicServerSetWithStateMap()

	server0 := createServer("server-name-0-0")
	server0.Labels[computeIndexLabel(ResourceServer)] = "0"
	server0.Labels[computeVersionLabel(ResourceServer)] = "0"
	server2 := createServer("server-name-2-0")
	server2.Labels[computeIndexLabel(ResourceServer)] = "2"
	server2.Labels[computeVersionLabel(ResourceServer)] = "0"

	// Both existing replicas report a healthy, freshly refreshed runtime state, so neither the
	// pre-update areAllVMsReadyForFailover() check nor the post-update reboot wait blocks.
	stateMap := &v1.ConfigMap{
		Name:      stateMapName,
		Namespace: stateMapNamespace,
		Data: map[string]string{
			fmt.Sprintf(stateKeyFormat, server0.Name):          statusVMRunning,
			fmt.Sprintf(stateTimestampKeyFormat, server0.Name): time.Now().Add(5 * time.Hour).Format(time.RFC3339),
			fmt.Sprintf(stateKeyFormat, server2.Name):          statusVMRunning,
			fmt.Sprintf(stateTimestampKeyFormat, server2.Name): time.Now().Add(5 * time.Hour).Format(time.RFC3339),
		},
	}

	kubeClient := &kubeClientFake{
		Client: fakeKubeClientObjs(server0, server2, stateMap,
			createBootVolumeWithIndexLabelsWithHotPlug("bootvolumename-0-0", 0),
			createBootVolumeWithIndexLabelsWithHotPlug("bootvolumename-2-0", 2)),
	}
	kubeClient.On(updateMethod, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	e := external{
		kube: kubeClient,
		bootVolumeController: &kubeBootVolumeCallTracker{
			lastMethodCall: make(map[ServiceMethodName][]any),
		},
		serverController: &kubeServerCallTracker{
			lastMethodCall: make(map[ServiceMethodName][]any),
		},
		log: logging.NewNopLogger(),
	}

	var err error
	require.NotPanics(t, func() {
		err = e.updateWithFailoverOrchestration(ctx, cr, 2)
	}, "the server must be resolved via its replica-index label, not by indexing the unsorted servers slice")
	require.NoError(t, err)
}

// Test_serverSetController_updateServersFromTemplate_usesReplicaIndexLabelNotListPosition
// reproduces the same index/position mismatch in updateServersFromTemplate().
//
// The loop variable `idx` is used both as a list position (servers[idx]) and as a replica index
// (getVolumeVersion(..., idx) and getNameFrom(..., idx, ...)), so a server is diffed against
// whatever boot volume happens to share its list position. Because checkServerDiff() reads the
// CPU/RAM hotplug flags off that boot volume to decide whether a failover-triggering reboot is
// needed, the wrong volume yields the wrong failover decision - and when no volume carries the
// list position as its index label, the whole update reconcile aborts.
//
// Setup: replica 1 is mid-recreation, so servers and boot volumes exist for replica indices 0
// and 2 only, and the template bumps Cores (hotplug is enabled, so this is an in-place update
// with no failover wait). Both remaining replicas must be updated.
func Test_serverSetController_updateServersFromTemplate_usesReplicaIndexLabelNotListPosition(t *testing.T) {
	ctx := context.Background()
	cr := createBasicServerSet()
	cr.Spec.ForProvider.Template.Spec.Cores = serverSetCores + 1

	server0 := createServer("server-name-0-0")
	server0.Labels[computeIndexLabel(ResourceServer)] = "0"
	server0.Labels[computeVersionLabel(ResourceServer)] = "0"
	server2 := createServer("server-name-2-0")
	server2.Labels[computeIndexLabel(ResourceServer)] = "2"
	server2.Labels[computeVersionLabel(ResourceServer)] = "0"

	kubeClient := &kubeClientFake{
		Client: fakeKubeClientObjs(server0, server2,
			createBootVolumeWithIndexLabelsWithHotPlug("bootvolumename-0-0", 0),
			createBootVolumeWithIndexLabelsWithHotPlug("bootvolumename-2-0", 2)),
	}
	kubeClient.On(updateMethod, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	e := external{
		kube: kubeClient,
		log:  logging.NewNopLogger(),
	}

	err := e.updateServersFromTemplate(ctx, cr)

	require.NoError(t, err, "each server must be paired with the boot volume carrying its own replica-index label")
	kubeClient.AssertNumberOfCalls(t, updateMethod, 2)
}

// func Test_serverSetController_Delete(t *testing.T) {
// 	type fields struct {
// 		kube client.Client
// 		log  logging.Logger
// 	}
// 	type args struct {
// 		ctx context.Context
// 		cr  *v1alpha1.ServerSet
// 	}
// 	tests := []struct {
// 		name        string
// 		fields      fields
// 		args        args
// 		wantErr     error
// 		wantNoCalls int
// 	}{
// 		{
// 			name: "success",
// 			fields: fields{
// 				kube: fakeKubeClientDeleteAllOfMethod(),
// 				log:  logging.NewNopLogger(),
// 			},
// 			args: args{
// 				ctx: context.Background(),
// 				cr:  createBasicServerSet(),
// 			},
// 			wantErr:     nil,
// 			wantNoCalls: 3,
// 		},
// 		{
// 			name: "failure (error when deleting the NICs)",
// 			fields: fields{
// 				kube: fakeKubeClientDeleteAllOfMethodReturnError(nic),
// 				log:  logging.NewNopLogger(),
// 			},
// 			args: args{
// 				ctx: context.Background(),
// 				cr:  createBasicServerSet(),
// 			},
// 			wantErr:     errAnErrorWasReceived,
// 			wantNoCalls: 1,
// 		},
// 		{
// 			name: "failure (error when deleting the Servers)",
// 			fields: fields{
// 				kube: fakeKubeClientDeleteAllOfMethodReturnError(server),
// 				log:  logging.NewNopLogger(),
// 			},
// 			args: args{
// 				ctx: context.Background(),
// 				cr:  createBasicServerSet(),
// 			},
// 			wantErr:     errAnErrorWasReceived,
// 			wantNoCalls: 2,
// 		},
// 		{
// 			name: "failure (error when deleting the BootVolumes)",
// 			fields: fields{
// 				kube: fakeKubeClientDeleteAllOfMethodReturnError(volume),
// 				log:  logging.NewNopLogger(),
// 			},
// 			args: args{
// 				ctx: context.Background(),
// 				cr:  createBasicServerSet(),
// 			},
// 			wantErr:     errAnErrorWasReceived,
// 			wantNoCalls: 3,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			e := &external{
// 				kube: tt.fields.kube,
// 				log:  tt.fields.log,
// 			}
//
// 			err := e.Delete(tt.args.ctx, tt.args.cr)
//
// 			assertions := assert.New(t)
// 			assertions.Equalf(tt.wantErr, err, "Wrong error")
// 			assertions.Equalf("", tt.args.cr.ObjectMeta.Annotations["AnnotationKeyExternalName"], "ExternalName annotation should be empty")
// 			assertions.Equalf(1, len(tt.args.cr.Status.Conditions), "ServerSet should have one condition")
// 			assertCondition(t, xpv1.Deleting(), tt.args.cr.Status.Conditions[0], "ServerSet has wrong condition")
//
// 			kubeClient := tt.fields.kube.(*kubeClientFake)
// 			kubeClient.AssertNumberOfCalls(t, "DeleteAllOf", tt.wantNoCalls)
// 			kubeClient.AssertExpectations(t)
// 		})
// 	}
// }

// func fakeKubeClientDeleteAllOfMethod() client.Client {
// 	kubeClient := kubeClientFake{}
// 	kubeClient.On("Delete",
// 		mock.Anything,
// 		mock.Anything,
// 		[]client.DeleteOption{},
// 	).Return(nil)
// 	return &kubeClient
// }

// func fakeKubeClientDeleteAllOfMethodReturnError(typeWhenToReturnErr crType) client.Client {
// 	kubeClient := fakeKubeClientDeleteAllOfMethod()
// 	kubeClient.(*kubeClientFake).crShouldReturnErr = make(map[crType]bool)
// 	kubeClient.(*kubeClientFake).crShouldReturnErr[typeWhenToReturnErr] = true
// 	return kubeClient
// }

func fakeKubeClientUpdateMethodReturnsError() client.Client {
	kubeClient := kubeClientFake{
		Client: fakeKubeClientObjs(
			createServer("server1"), createServer("server2"),
			createBootVolumeWithIndexLabelsWithHotPlug("bootvolumename-0-0", 0), createBootVolumeWithIndexLabelsWithHotPlug("bootvolumename-1-0", 1),
			createNic(v1alpha1.NicParameters{Name: "nic-server1"}),
			createNic(v1alpha1.NicParameters{Name: "nic-server2"}),
		),
	}
	kubeClient.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(errAnErrorWasReceived)
	return &kubeClient
}

func fakeKubeClientOneServer() client.Client {
	kubeClient := kubeClientFake{
		Client: fakeKubeClientObjs(
			createServer("server1"),
		),
	}
	return &kubeClient
}

func fakeKubeClientUpdateMethod(expectedObj client.Object) client.Client {
	kubeClient := kubeClientFake{
		Client: fakeKubeClientObjs(
			createServer("server1"), createServer("server2"),
			createBootVolumeWithIndexLabelsWithHotPlug("bootvolumename-0-0", 0), createBootVolumeWithIndexLabelsWithHotPlug("bootvolumename-1-0", 1),
			createNic(v1alpha1.NicParameters{Name: "nic-server1"}),
			createNic(v1alpha1.NicParameters{Name: "nic-server2"}),
		),
	}
	kubeClient.On("Update", mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		arg1 := args.Get(1)
		if reflect.TypeOf(arg1) != reflect.TypeOf(expectedObj) {
			panic(fmt.Sprintf("Update called with unexpected type: want=%v, got=%v", reflect.TypeOf(expectedObj), reflect.TypeOf(arg1)))
		}
	}).Return(nil)

	return &kubeClient
}

func fakeKubeClientUpdateMethodWithNicMultiQueueServers(nmq *bool, expectedObj client.Object) client.Client {
	kubeClient := kubeClientFake{
		Client: fakeKubeClientObjs(
			createServerWithNicMultiQueue("server1", nmq), createServerWithNicMultiQueue("server2", nmq),
			createBootVolumeWithIndexLabelsWithHotPlug("bootvolumename-0-0", 0), createBootVolumeWithIndexLabelsWithHotPlug("bootvolumename-1-0", 1),
			createNic(v1alpha1.NicParameters{Name: "nic-server1"}),
			createNic(v1alpha1.NicParameters{Name: "nic-server2"}),
		),
	}
	kubeClient.On("Update", mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		arg1 := args.Get(1)
		if reflect.TypeOf(arg1) != reflect.TypeOf(expectedObj) {
			panic(fmt.Sprintf("Update called with unexpected type: want=%v, got=%v", reflect.TypeOf(expectedObj), reflect.TypeOf(arg1)))
		}
	}).Return(nil)

	return &kubeClient
}

func fakeKubeClientUpdateMethodWithNicMultiQueueServersAndSuccessfulFailover(nmq *bool, expectedObj client.Object) client.Client {
	kubeClient := kubeClientFake{
		Client: fakeKubeClientObjs(
			createServerWithNicMultiQueueAndUpdateSucceededCondition("server1", nmq),
			createServerWithNicMultiQueueAndUpdateSucceededCondition("server2", nmq),
			createBootVolumeWithIndexLabelsWithoutHotPlug("bootvolumename-0-0", 0), createBootVolumeWithIndexLabelsWithoutHotPlug("bootvolumename-1-0", 1),
			createNic(v1alpha1.NicParameters{Name: "nic-server1"}),
			createNic(v1alpha1.NicParameters{Name: "nic-server2"}),
		),
	}
	kubeClient.On("Update", mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		arg1 := args.Get(1)
		if reflect.TypeOf(arg1) != reflect.TypeOf(expectedObj) {
			panic(fmt.Sprintf("Update called with unexpected type: want=%v, got=%v", reflect.TypeOf(expectedObj), reflect.TypeOf(arg1)))
		}
	}).Return(nil)

	return &kubeClient
}

func fakeKubeClientUpdateMethodWithNicMultiQueueServersAndFailedFailover(nmq *bool, expectedObj client.Object) client.Client {
	kubeClient := kubeClientFake{
		Client: fakeKubeClientObjs(
			createServerWithNicMultiQueueAndUpdateFailedCondition("server1", nmq),
			createServerWithNicMultiQueueAndUpdateFailedCondition("server2", nmq),
			createBootVolumeWithIndexLabelsWithoutHotPlug("bootvolumename-0-0", 0), createBootVolumeWithIndexLabelsWithoutHotPlug("bootvolumename-1-0", 1),
			createNic(v1alpha1.NicParameters{Name: "nic-server1"}),
			createNic(v1alpha1.NicParameters{Name: "nic-server2"}),
		),
	}
	kubeClient.On("Update", mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		arg1 := args.Get(1)
		if reflect.TypeOf(arg1) != reflect.TypeOf(expectedObj) {
			panic(fmt.Sprintf("Update called with unexpected type: want=%v, got=%v", reflect.TypeOf(expectedObj), reflect.TypeOf(arg1)))
		}
	}).Return(nil)

	return &kubeClient
}

func fakeKubeClientUpdateMethodWithSuccessfulFailover(expectedObject client.Object) client.Client {
	kubeClient := kubeClientFake{
		Client: fakeKubeClientObjs(
			createServerWithUpdateSucceededConditionSet("server1"), createServerWithUpdateSucceededConditionSet("server2"),
			createBootVolumeWithIndexLabelsWithoutHotPlug("bootvolumename-0-0", 0), createBootVolumeWithIndexLabelsWithoutHotPlug("bootvolumename-1-0", 1),
			createNic(v1alpha1.NicParameters{Name: "nic-server1"}),
			createNic(v1alpha1.NicParameters{Name: "nic-server2"}),
		),
	}
	kubeClient.On("Update", mock.Anything, mock.Anything, mock.Anything).Run(
		func(args mock.Arguments) {
			arg1 := args.Get(1)
			if reflect.TypeOf(arg1) != reflect.TypeOf(expectedObject) {
				panic(fmt.Sprintf("Update called with unexpected type: want=%v, got=%v", reflect.TypeOf(expectedObject), reflect.TypeOf(arg1)))
			}
		},
	).Return(nil)

	return &kubeClient
}

func fakeKubeClientUpdateMethodWithFailedFailover(expectedObject client.Object) client.Client {
	kubeClient := kubeClientFake{
		Client: fakeKubeClientObjs(
			createServerWithUpdateFailedConditionSet("server1"), createServerWithUpdateFailedConditionSet("server2"),
			createBootVolumeWithIndexLabelsWithoutHotPlug("bootvolumename-0-0", 0), createBootVolumeWithIndexLabelsWithoutHotPlug("bootvolumename-1-0", 1),
			createNic(v1alpha1.NicParameters{Name: "nic-server1"}),
			createNic(v1alpha1.NicParameters{Name: "nic-server2"}),
		),
	}
	kubeClient.On("Update", mock.Anything, mock.Anything, mock.Anything).Run(
		func(args mock.Arguments) {
			arg1 := args.Get(1)
			if reflect.TypeOf(arg1) != reflect.TypeOf(expectedObject) {
				panic(fmt.Sprintf("Update called with unexpected type: want=%v, got=%v", reflect.TypeOf(expectedObject), reflect.TypeOf(arg1)))
			}
		},
	).Return(nil)

	return &kubeClient
}

func fakeKubeClientUpdateMethodWithStateMapSuccessfulFailover(expectedObject client.Object) client.Client {
	kubeClient := kubeClientFake{
		Client: fakeKubeClientObjs(
			createServerWithUpdateSucceededConditionSet(server1Name), createServerWithUpdateSucceededConditionSet(server2Name),
			createBootVolumeWithIndexLabelsWithoutHotPlug("bootvolumename-0-0", 0), createBootVolumeWithIndexLabelsWithoutHotPlug("bootvolumename-1-0", 1),
			createNic(v1alpha1.NicParameters{Name: "nic-server1"}),
			createNic(v1alpha1.NicParameters{Name: "nic-server2"}),
			createStateMapRunning(),
		),
	}
	kubeClient.On("Update", mock.Anything, mock.Anything, mock.Anything).Run(
		func(args mock.Arguments) {
			arg1 := args.Get(1)
			if reflect.TypeOf(arg1) != reflect.TypeOf(expectedObject) {
				panic(fmt.Sprintf("Update called with unexpected type: want=%v, got=%v", reflect.TypeOf(expectedObject), reflect.TypeOf(arg1)))
			}
		},
	).Return(nil)

	return &kubeClient
}

func fakeKubeClientUpdateMethodWithStateMapFailedFailover(expectedObject client.Object) client.Client {
	kubeClient := kubeClientFake{
		Client: fakeKubeClientObjs(
			createServerWithUpdateSucceededConditionSet(server1Name), createServerWithUpdateSucceededConditionSet(server2Name),
			createBootVolumeWithIndexLabelsWithoutHotPlug("bootvolumename-0-0", 0), createBootVolumeWithIndexLabelsWithoutHotPlug("bootvolumename-1-0", 1),
			createNic(v1alpha1.NicParameters{Name: "nic-server1"}),
			createNic(v1alpha1.NicParameters{Name: "nic-server2"}),
			createStateMapOneVMError(),
		),
	}
	kubeClient.On("Update", mock.Anything, mock.Anything, mock.Anything).Run(
		func(args mock.Arguments) {
			arg1 := args.Get(1)
			if reflect.TypeOf(arg1) != reflect.TypeOf(expectedObject) {
				panic(fmt.Sprintf("Update called with unexpected type: want=%v, got=%v", reflect.TypeOf(expectedObject), reflect.TypeOf(arg1)))
			}
		},
	).Return(nil)

	return &kubeClient
}

func fakeKubeClientUpdateMethodWithStateMapZeroTimestampSuccessfulFailover(expectedObject client.Object) client.Client {
	kubeClient := kubeClientFake{
		Client: fakeKubeClientObjs(
			createServerWithUpdateSucceededConditionSet(server1Name), createServerWithUpdateSucceededConditionSet(server2Name),
			createBootVolumeWithIndexLabelsWithoutHotPlug("bootvolumename-0-0", 0), createBootVolumeWithIndexLabelsWithoutHotPlug("bootvolumename-1-0", 1),
			createNic(v1alpha1.NicParameters{Name: "nic-server1"}),
			createNic(v1alpha1.NicParameters{Name: "nic-server2"}),
			createStateMapZeroTimestamp(),
		),
	}
	kubeClient.On("Update", mock.Anything, mock.Anything, mock.Anything).Run(
		func(args mock.Arguments) {
			arg1 := args.Get(1)
			if reflect.TypeOf(arg1) != reflect.TypeOf(expectedObject) {
				panic(fmt.Sprintf("Update called with unexpected type: want=%v, got=%v", reflect.TypeOf(expectedObject), reflect.TypeOf(arg1)))
			}
		},
	).Return(nil)

	return &kubeClient
}

func fakeKubeClientUpdateMethodForBootVolume() client.Client {
	kubeClient := kubeClientFake{
		Client: kubeClientWithObjsForBootVolume(),
	}

	kubeClient.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	return &kubeClient
}

func fakeKubeClientUpdateMethodForBootVolumeWithStateMap() client.Client {
	kubeClient := kubeClientFake{
		Client: kubeClientWithObjsForBootVolumeWithStateMap(),
	}

	kubeClient.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	return &kubeClient
}

func fakeKubeClientUpdateMethodForBootVolumeWithStateMapVMError() client.Client {
	kubeClient := kubeClientFake{
		Client: kubeClientWithObjsForBootVolumeWithStateMapVMError(),
	}

	kubeClient.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	return &kubeClient
}

func kubeClientWithObjsForBootVolume() client.WithWatch {
	zero := "0"
	one := "1"

	server1 := createServer("server1")
	server1.Labels[computeIndexLabel(ResourceServer)] = zero
	server1.Labels[computeVersionLabel(ResourceServer)] = zero

	server2 := createServer("server2")
	server2.Labels[computeIndexLabel(ResourceServer)] = one
	server2.Labels[computeVersionLabel(ResourceServer)] = zero

	bootVolume1 := createBootVolumeWithIndexLabelsWithHotPlug("bootvolumename-0-0", 0)
	bootVolume2 := createBootVolumeWithIndexLabelsWithHotPlug("bootvolumename-1-0", 1)

	return fakeKubeClientObjs(server1, server2, bootVolume1, bootVolume2,
		createNic(v1alpha1.NicParameters{Name: "nic-server1"}),
		createNic(v1alpha1.NicParameters{Name: "nic-server2"}))
}

func kubeClientWithObjsForBootVolumeWithStateMap() client.WithWatch {
	zero := "0"
	one := "1"

	server1 := createServerWithUpdateSucceededConditionSet(server1Name)
	server1.Labels[computeIndexLabel(ResourceServer)] = zero
	server1.Labels[computeVersionLabel(ResourceServer)] = zero

	server2 := createServerWithUpdateSucceededConditionSet(server2Name)
	server2.Labels[computeIndexLabel(ResourceServer)] = one
	server2.Labels[computeVersionLabel(ResourceServer)] = zero

	bootVolume1 := createBootVolumeWithIndexLabelsWithHotPlug("bootvolumename-0-0", 0)
	bootVolume2 := createBootVolumeWithIndexLabelsWithHotPlug("bootvolumename-1-0", 1)

	return fakeKubeClientObjs(server1, server2, bootVolume1, bootVolume2,
		createNic(v1alpha1.NicParameters{Name: "nic-server1"}),
		createNic(v1alpha1.NicParameters{Name: "nic-server2"}),
		createStateMapRunning())
}

func kubeClientWithObjsForBootVolumeWithStateMapVMError() client.WithWatch {
	zero := "0"
	one := "1"

	server1 := createServerWithUpdateSucceededConditionSet(server1Name)
	server1.Labels[computeIndexLabel(ResourceServer)] = zero
	server1.Labels[computeVersionLabel(ResourceServer)] = zero

	server2 := createServerWithUpdateSucceededConditionSet(server2Name)
	server2.Labels[computeIndexLabel(ResourceServer)] = one
	server2.Labels[computeVersionLabel(ResourceServer)] = zero

	bootVolume1 := createBootVolumeWithIndexLabelsWithHotPlug("bootvolumename-0-0", 0)
	bootVolume2 := createBootVolumeWithIndexLabelsWithHotPlug("bootvolumename-1-0", 1)

	return fakeKubeClientObjs(server1, server2, bootVolume1, bootVolume2,
		createNic(v1alpha1.NicParameters{Name: "nic-server1"}),
		createNic(v1alpha1.NicParameters{Name: "nic-server2"}),
		createStateMapOneVMError())
}

func computeIndexLabel(resourceType string) string {
	return fmt.Sprintf(indexLabel, serverSetName, resourceType)
}

func computeVersionLabel(resourceType string) string {
	return fmt.Sprintf(versionLabel, serverSetName, resourceType)
}

func (f *kubeClientFake) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	args := f.Called(ctx, obj, opts)
	return args.Error(0)
}

func (f *kubeClientFake) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	args := f.Called(ctx, obj, opts)
	if f.shouldReturnError(obj) {
		return errAnErrorWasReceived
	}
	return args.Error(0)
}

func (f *kubeClientFake) shouldReturnError(obj client.Object) bool {
	switch obj.(type) {
	case *v1alpha1.Server:
		return f.crShouldReturnErr[server]
	case *v1alpha1.Nic:
		return f.crShouldReturnErr[nic]
	case *v1alpha1.Volume:
		return f.crShouldReturnErr[volume]
	case *v1alpha1.Volumeselector:
		return f.crShouldReturnErr[volumeSelector]
	default:
		return false
	}
}

func fakeBootVolumeCtrlEnsure() kubeBootVolumeControlManager {
	bootVolumeCtrl := new(kubeBootVolumeControlManagerFake)
	bootVolumeCtrl.
		On(ensureMethod, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).
		On(getMethod, mock.Anything, mock.Anything, mock.Anything).Return(&v1alpha1.Volume{}, nil)

	return bootVolumeCtrl

}

func fakeBootVolumeCtrlGetEnsure() kubeBootVolumeControlManager {
	bootVolumeCtrl := new(kubeBootVolumeControlManagerFake)
	bootVolumeCtrl.
		On(ensureMethod, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).
		On(getMethod, mock.Anything, mock.Anything, mock.Anything).Return(&v1alpha1.Volume{}, nil)
	return bootVolumeCtrl

}

func fakeBootVolumeCtrl() kubeBootVolumeControlManager {
	bootVolumeCtrl := new(kubeBootVolumeControlManagerFake)
	bootVolumeCtrl.
		On(ensureMethod, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).
		On(getMethod, mock.Anything, mock.Anything, mock.Anything).Return(&v1alpha1.Volume{}, nil).
		On(deleteMethod, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	return bootVolumeCtrl

}

func fakeBootVolumeCtrlGetEnsureMethodReturnsErr() kubeBootVolumeControlManager {
	bootVolumeCtrl := new(kubeBootVolumeControlManagerFake)
	bootVolumeCtrl.
		On(getMethod, mock.Anything, mock.Anything, mock.Anything).Return(&v1alpha1.Volume{}, nil).
		On(ensureMethod, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(errAnErrorWasReceived).
		Times(1)
	return bootVolumeCtrl
}

func fakeBootVolumeCtrlEnsureMethodReturnsErr() kubeBootVolumeControlManager {
	bootVolumeCtrl := new(kubeBootVolumeControlManagerFake)
	bootVolumeCtrl.
		On(ensureMethod, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(errAnErrorWasReceived).
		Times(1)
	return bootVolumeCtrl
}
func fakeServerCtrlEnsureMethod(timesCalled int) kubeServerControlManager {
	serverCtrl := new(kubeServerControlManagerFake)
	if timesCalled > 0 {
		serverCtrl.
			On(ensureMethod, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil).
			Times(timesCalled)
	}
	return serverCtrl
}
func fakeServerCtrlGetEnsure() kubeServerControlManager {
	serverCtrl := new(kubeServerControlManagerFake)
	serverCtrl.
		On(ensureMethod, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).
		On(updateMethod, mock.Anything, mock.Anything).Return(nil).
		On(getMethod, mock.Anything, mock.Anything, mock.Anything).Return(&v1alpha1.Server{}, nil)
	return serverCtrl
}

func fakeServerCtrl() kubeServerControlManager {
	serverCtrl := new(kubeServerControlManagerFake)
	serverCtrl.
		On(ensureMethod, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).
		On(deleteMethod, mock.Anything, mock.Anything, mock.Anything).Return(nil).
		On(updateMethod, mock.Anything, mock.Anything).Return(nil).
		On(getMethod, mock.Anything, mock.Anything, mock.Anything).Return(&v1alpha1.Server{}, nil)
	return serverCtrl
}

func fakeServerCtrlEnsureMethodReturnsErr() kubeServerControlManager {
	serverCtrl := new(kubeServerControlManagerFake)
	serverCtrl.
		On(ensureMethod, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(errAnErrorWasReceived).
		Times(1)
	return serverCtrl
}

func fakeNicCtrlEnsureNICsMethodBasic() kubeNicControlManager {
	nicCtrl := new(kubeNicControlManagerFake)
	nicCtrl.
		On(ensureNICsMethod, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Times(0)
	return nicCtrl
}

func fakeNicCtrlEnsureNICsMethod(timesCalled int) kubeNicControlManager {
	nicCtrl := new(kubeNicControlManagerFake)
	if timesCalled > 0 {
		nicCtrl.
			On(ensureNICsMethod, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil).
			Times(timesCalled)
	}
	return nicCtrl
}

func fakeNicCtrlEnsureNICsMethodReturnsErr() kubeNicControlManager {
	nicCtrl := new(kubeNicControlManagerFake)
	nicCtrl.
		On(ensureNICsMethod, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(errAnErrorWasReceived).
		Times(1)

	return nicCtrl
}

func fakeNicCtrl() kubeNicControlManager {
	nicCtrl := new(kubeNicControlManagerFake)
	nicCtrl.
		On(ensureNICsMethod, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		On(deleteMethod, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	return nicCtrl
}

func fakeFirewallRuleCtrlEnsureMethodBasic() kubeFirewallRuleControlManager {
	firewallRuleCtrl := new(kubeFirewallRuleControlManagerFake)
	firewallRuleCtrl.
		On(ensureFirewallRulesMethod, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Times(0)
	return firewallRuleCtrl
}

func fakeFirewallRuleCtrlEnsureMethod(timesCalled int) kubeFirewallRuleControlManager {
	firewallRuleCtrl := new(kubeFirewallRuleControlManagerFake)
	if timesCalled > 0 {
		firewallRuleCtrl.
			On(ensureFirewallRulesMethod, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil).
			Times(timesCalled)
	}
	return firewallRuleCtrl
}

func fakeFirewallRuleCtrl() kubeFirewallRuleControlManager {
	firewallRuleCtrl := new(kubeFirewallRuleControlManagerFake)
	firewallRuleCtrl.
		On(ensureFirewallRulesMethod, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		On(deleteMethod, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	return firewallRuleCtrl
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

func (f *kubeBootVolumeCallTracker) Ensure(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int) error {
	f.lastMethodCall[ensureMethod] = []any{ctx, cr, replicaIndex, version}
	return nil
}

func (f *kubeBootVolumeCallTracker) Get(ctx context.Context, name, ns string) (*v1alpha1.Volume, error) {
	f.lastMethodCall[getMethod] = []any{ctx, name, ns}
	volume := v1alpha1.Volume{}
	volume.Status.AtProvider.VolumeID = name + "-uuid"
	return &volume, nil
}

func (f *kubeBootVolumeCallTracker) Delete(ctx context.Context, name, ns string) error {
	f.lastMethodCall[deleteMethod] = []any{ctx, name, ns}
	return nil
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

func (f *kubeServerCallTracker) Ensure(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version, volumeVersion int) error {
	f.lastMethodCall[ensureMethod] = []any{ctx, cr, replicaIndex, version, volumeVersion}
	return nil
}

func (f *kubeServerCallTracker) Get(ctx context.Context, name, ns string) (*v1alpha1.Server, error) {
	f.lastMethodCall[getMethod] = []any{ctx, name, ns}
	return &v1alpha1.Server{}, nil
}

func (f *kubeServerCallTracker) Update(ctx context.Context, cr *v1alpha1.Server) error {
	f.lastMethodCall[updateMethod] = []any{ctx, cr}
	return nil
}

func (f *kubeServerCallTracker) Delete(ctx context.Context, name, ns string) error {
	f.lastMethodCall[deleteMethod] = []any{ctx, name, ns}
	return nil
}

func (f *kubeNicControlManagerFake) EnsureNICs(
	ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int, serverID string,
) error {
	args := f.Called(ctx, cr, replicaIndex, version, serverID)
	return args.Error(0)
}

func (f *kubeNicControlManagerFake) Delete(ctx context.Context, name, ns string) error {
	args := f.Called(ctx, name, ns)
	return args.Error(0)
}

func (f *kubeNicCallTracker) EnsureNICs(
	ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int, stringID string,
) error {
	f.lastMethodCall[ensureNICsMethod] = []any{ctx, cr, replicaIndex, version, serverID}
	return nil
}

func (f *kubeNicCallTracker) Delete(ctx context.Context, name, ns string) error {
	f.lastMethodCall[deleteMethod] = []any{ctx, name, ns}
	return nil
}

func (f *kubeFirewallRuleControlManagerFake) EnsureFirewallRules(
	ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int, serverID string,
) error {
	args := f.Called(ctx, cr, replicaIndex, version, serverID)
	return args.Error(0)
}

func (f *kubeFirewallRuleControlManagerFake) Delete(ctx context.Context, name, ns string) error {
	args := f.Called(ctx, name, ns)
	return args.Error(0)
}

func (f *kubeFirewallRuleCallTracker) EnsureFirewallRules(
	ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int, serverID string,
) error {
	f.lastMethodCall[ensureFirewallRulesMethod] = []any{ctx, cr, replicaIndex, version, serverID}
	return nil
}

func (f *kubeFirewallRuleCallTracker) Delete(ctx context.Context, name, ns string) error {
	f.lastMethodCall[deleteMethod] = []any{ctx, name, ns}
	return nil
}

func createServerNotReadyYet() *v1alpha1.Server {
	serverNotReady := createServer(serverNotReadyName)
	serverNotReady.Status.AtProvider.State = ionoscloud.Busy
	return serverNotReady
}

func createServer(name string) *v1alpha1.Server {
	return &v1alpha1.Server{
		Name: name,
		Labels: map[string]string{
			serverSetLabel: serverSetName,
			fmt.Sprintf(indexLabel, serverSetName, ResourceServer): "0",
		},
		Status: v1alpha1.ServerStatus{
			AtProvider: v1alpha1.ServerObservation{
				State:    ionoscloud.Available,
				ServerID: "serverID",
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

func createServerWithNicMultiQueue(name string, nmq *bool) *v1alpha1.Server {
	server := createServer(name)
	server.Spec.ForProvider.NicMultiQueue = nmq
	return server
}

// createServerWithNicMultiQueueAndUpdateSucceededCondition is for testing a NicMultiQueue
// change, which now goes through the failover path: it needs the pre-change NicMultiQueue
// value AND an already-succeeded update condition, or isUpdateFinished polls until timeout.
func createServerWithNicMultiQueueAndUpdateSucceededCondition(name string, nmq *bool) *v1alpha1.Server {
	server := createServerWithUpdateSucceededConditionSet(name)
	server.Spec.ForProvider.NicMultiQueue = nmq
	return server
}

// createServerWithNicMultiQueueAndUpdateFailedCondition is the failed-failover counterpart of
// createServerWithNicMultiQueueAndUpdateSucceededCondition, used to prove that a NicMultiQueue
// change actually goes through the failover/wait path rather than the non-failover path.
func createServerWithNicMultiQueueAndUpdateFailedCondition(name string, nmq *bool) *v1alpha1.Server {
	server := createServerWithUpdateFailedConditionSet(name)
	server.Spec.ForProvider.NicMultiQueue = nmq
	return server
}

func createServerWithUpdateSucceededConditionSet(name string) *v1alpha1.Server {
	return &v1alpha1.Server{
		Name: name,
		Labels: map[string]string{
			serverSetLabel: serverSetName,
			fmt.Sprintf(indexLabel, serverSetName, ResourceServer): "0",
		},
		Status: v1alpha1.ServerStatus{
			AtProvider: v1alpha1.ServerObservation{
				State:    ionoscloud.Available,
				ServerID: "serverID",
			},
			ResourceStatus: xpv1.ResourceStatus{
				ConditionedStatus: xpv1.ConditionedStatus{
					// Set the update succeeded condition an hour later to simulate that an update has occurred
					Conditions: []xpv1.Condition{serverctrl.UpdateSucceededCondition(metav1.NewTime(time.Now().Add(time.Hour)))},
				},
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

func createServerWithUpdateFailedConditionSet(name string) *v1alpha1.Server {
	return &v1alpha1.Server{
		Name: name,
		Labels: map[string]string{
			serverSetLabel: serverSetName,
			fmt.Sprintf(indexLabel, serverSetName, ResourceServer): "0",
		},
		Status: v1alpha1.ServerStatus{
			AtProvider: v1alpha1.ServerObservation{
				State:    ionoscloud.Available,
				ServerID: "serverID",
			},
			ResourceStatus: xpv1.ResourceStatus{
				ConditionedStatus: xpv1.ConditionedStatus{
					// Set the update succeeded condition an hour later to simulate that an update has occurred
					Conditions: []xpv1.Condition{serverctrl.UpdateFailedCondition(fmt.Errorf("an error of sorts"), metav1.NewTime(time.Now().Add(time.Hour)))},
				},
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

func createNic(params v1alpha1.NicParameters) *v1alpha1.Nic {
	nic := createBasicNic()

	if params.Name != "" {
		nic.Name = params.Name
		nic.Spec.ForProvider.Name = params.Name
	}
	if params.DatacenterCfg != (v1alpha1.DatacenterConfig{}) {
		nic.Spec.ForProvider.DatacenterCfg = params.DatacenterCfg
	}
	if params.ServerCfg != (v1alpha1.ServerConfig{}) {
		nic.Spec.ForProvider.ServerCfg = params.ServerCfg
	}
	if params.LanCfg != (v1alpha1.LanConfig{}) {
		nic.Spec.ForProvider.LanCfg = params.LanCfg
	}
	if !reflect.DeepEqual(params.IpsCfg, v1alpha1.IPsConfigs{}) {
		nic.Spec.ForProvider.IpsCfg = params.IpsCfg
	}
	if params.Dhcp != false {
		nic.Spec.ForProvider.Dhcp = params.Dhcp
	}
	if params.DhcpV6 != nil && *params.DhcpV6 != false {
		nic.Spec.ForProvider.DhcpV6 = params.DhcpV6
	}
	if params.FirewallActive != false {
		nic.Spec.ForProvider.FirewallActive = params.FirewallActive
	}
	if params.FirewallType != "" {
		nic.Spec.ForProvider.FirewallType = params.FirewallType
	}
	if params.Vnet != "" {
		nic.Spec.ForProvider.Vnet = params.Vnet
	}

	return nic
}

func createBasicNic() *v1alpha1.Nic {
	return &v1alpha1.Nic{
		Name: "test-nic",
		Labels: map[string]string{
			serverSetLabel:            serverSetName,
			serverSetNicIndexLabel:    "0",
			serverSetNicVersionLabel:  "0",
			serverSetNicNicIndexLabel: "0",
		},
		Spec: v1alpha1.NicSpec{
			ForProvider: v1alpha1.NicParameters{
				DatacenterCfg:  v1alpha1.DatacenterConfig{},
				ServerCfg:      v1alpha1.ServerConfig{},
				LanCfg:         v1alpha1.LanConfig{},
				Name:           "test-nic",
				IpsCfg:         v1alpha1.IPsConfigs{},
				Dhcp:           false,
				DhcpV6:         nil,
				FirewallActive: false,
				FirewallType:   "",
				Vnet:           "",
			},
		},
	}
}

func createBootVolumeWithHotPlug(name string) *v1alpha1.Volume {
	return &v1alpha1.Volume{
		Name: name,
		Labels: map[string]string{
			serverSetLabel: serverSetName,
		},
		Spec: v1alpha1.VolumeSpec{
			ForProvider: v1alpha1.VolumeParameters{
				Image:      bootVolumeImage,
				Type:       bootVolumeType,
				Size:       bootVolumeSize,
				CPUHotPlug: true,
				RAMHotPlug: true,
			},
		},
		Status: v1alpha1.VolumeStatus{
			AtProvider: v1alpha1.VolumeObservation{
				State: ionoscloud.Available,
			},
		},
	}
}

func createBootVolumeWithoutHotPlug(name string) *v1alpha1.Volume {
	return &v1alpha1.Volume{
		Name: name,
		Labels: map[string]string{
			serverSetLabel: serverSetName,
		},
		Spec: v1alpha1.VolumeSpec{
			ForProvider: v1alpha1.VolumeParameters{
				Image:      bootVolumeImage,
				Type:       bootVolumeType,
				Size:       bootVolumeSize,
				CPUHotPlug: false,
				RAMHotPlug: false,
			},
		},
		Status: v1alpha1.VolumeStatus{
			AtProvider: v1alpha1.VolumeObservation{
				State: ionoscloud.Available,
			},
		},
	}
}

func createBootVolumeWithIndexLabelsWithHotPlug(name string, index int) *v1alpha1.Volume {
	volume := createBootVolumeWithHotPlug(name)
	volume.Labels[computeIndexLabel(resourceBootVolume)] = strconv.Itoa(index)
	volume.Labels[computeVersionLabel(resourceBootVolume)] = "0"
	return volume
}

func createBootVolumeWithIndexWithHotPlug(name string, index int) *v1alpha1.Volume {
	volume := createBootVolumeWithHotPlug(name)
	indexLabelBootVolume := fmt.Sprintf(indexLabel, serverSetName, resourceBootVolume)
	volume.Labels[indexLabelBootVolume] = fmt.Sprintf("%d", index)
	return volume
}

func createBootVolumeWithIndexLabelsWithoutHotPlug(name string, index int) *v1alpha1.Volume {
	volume := createBootVolumeWithoutHotPlug(name)
	volume.Labels[computeIndexLabel(resourceBootVolume)] = strconv.Itoa(index)
	volume.Labels[computeVersionLabel(resourceBootVolume)] = "0"
	return volume
}

func createBootVolumeWithIndexWithoutHotPlug(name string, index int) *v1alpha1.Volume {
	volume := createBootVolumeWithoutHotPlug(name)
	indexLabelBootVolume := fmt.Sprintf(indexLabel, serverSetName, resourceBootVolume)
	volume.Labels[indexLabelBootVolume] = fmt.Sprintf("%d", index)
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
		Name: serverSetName,
		Annotations: map[string]string{
			"crossplane.io/external-name": serverSetName,
		},
		Spec: v1alpha1.ServerSetSpec{
			ForProvider: v1alpha1.ServerSetParameters{
				Replicas: noReplicas,
				Template: v1alpha1.ServerSetTemplate{
					Metadata: v1alpha1.ServerSetMetadata{
						Name: serverName,
					},
					Spec: v1alpha1.ServerSetTemplateSpec{
						Cores:     serverSetCores,
						RAM:       serverSetRAM,
						CPUFamily: serverSetCPUFamily,
						NICs: []v1alpha1.ServerSetTemplateNIC{
							{
								Name:         "nic1",
								DHCP:         false,
								LanReference: "user",
							},
						},
					},
				},
				BootVolumeTemplate: v1alpha1.BootVolumeTemplate{
					Metadata: v1alpha1.ServerSetBootVolumeMetadata{
						Name: "bootvolumename",
					},
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

func createBasicServerSetWithStateMap() *v1alpha1.ServerSet {
	return &v1alpha1.ServerSet{
		Name: serverSetName,
		Annotations: map[string]string{
			"crossplane.io/external-name": serverSetName,
		},
		Spec: v1alpha1.ServerSetSpec{
			ForProvider: v1alpha1.ServerSetParameters{
				Replicas: noReplicas,
				Template: v1alpha1.ServerSetTemplate{
					Metadata: v1alpha1.ServerSetMetadata{
						Name: serverName,
					},
					Spec: v1alpha1.ServerSetTemplateSpec{
						Cores:     serverSetCores,
						RAM:       serverSetRAM,
						CPUFamily: serverSetCPUFamily,
						NICs: []v1alpha1.ServerSetTemplateNIC{
							{
								Name:         "nic1",
								DHCP:         false,
								LanReference: "user",
							},
						},
						StateMap: &v1alpha1.StateConfigMap{
							Name:      stateMapName,
							Namespace: stateMapNamespace,
						},
					},
				},
				BootVolumeTemplate: v1alpha1.BootVolumeTemplate{
					Metadata: v1alpha1.ServerSetBootVolumeMetadata{
						Name: "bootvolumename",
					},
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
	sset.Spec.ForProvider.Template.Spec.NicMultiQueue = spec.NicMultiQueue
	return sset
}

func createServerSetWithUpdatedServerSpecWithStateMap(spec v1alpha1.ServerSetTemplateSpec) *v1alpha1.ServerSet {
	sset := createBasicServerSetWithStateMap()
	sset.Spec.ForProvider.Template.Spec.Cores = spec.Cores
	sset.Spec.ForProvider.Template.Spec.RAM = spec.RAM
	sset.Spec.ForProvider.Template.Spec.CPUFamily = spec.CPUFamily
	return sset
}

func createServerSetWithUpdatedBootVolumeUsingDefaultStrategy(updatedSpec v1alpha1.ServerSetBootVolumeSpec) *v1alpha1.ServerSet {
	sset := createBasicServerSet()
	sset.Spec.ForProvider.BootVolumeTemplate.Spec = updatedSpec
	return sset
}

func createServerSetWithUpdatedBootVolumeUsingStrategy(updatedSpec v1alpha1.ServerSetBootVolumeSpec, strategy v1alpha1.UpdateStrategy) *v1alpha1.ServerSet {
	sset := createServerSetWithUpdatedBootVolumeUsingDefaultStrategy(updatedSpec)
	sset.Spec.ForProvider.BootVolumeTemplate.Spec.UpdateStrategy = strategy
	return sset
}

func createServerSetWithUpdatedBootVolumeWithStateMap(updatedSpec v1alpha1.ServerSetBootVolumeSpec) *v1alpha1.ServerSet {
	sset := createBasicServerSetWithStateMap()
	sset.Spec.ForProvider.BootVolumeTemplate.Spec = updatedSpec
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
			Name:         "nic2",
			DHCP:         true,
			LanReference: "management",
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
		Name:      "config-lease",
		Namespace: "default",
		Data: map[string]string{
			"identity": "serverset-server-0-0",
		},
	}
}

func createServerWithReconcileErrorMsg() *v1alpha1.Server {
	server := createServer("serverset-server-0-0")
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
	server.Labels[indexLabelBootVolume] = fmt.Sprintf("%d", index)
	return server
}

func createStateMapRunning() *v1.ConfigMap {
	return &v1.ConfigMap{
		Name:      stateMapName,
		Namespace: stateMapNamespace,
		Data: map[string]string{
			fmt.Sprintf(stateKeyFormat, server1Name):          statusVMRunning,
			fmt.Sprintf(stateTimestampKeyFormat, server1Name): time.Now().Add(5 * time.Hour).Format(time.RFC3339),
			fmt.Sprintf(stateKeyFormat, server2Name):          statusVMRunning,
			fmt.Sprintf(stateTimestampKeyFormat, server2Name): time.Now().Add(5 * time.Hour).Format(time.RFC3339),
		},
	}
}

func createStateMapOneVMError() *v1.ConfigMap {
	return &v1.ConfigMap{
		Name:      stateMapName,
		Namespace: stateMapNamespace,
		Data: map[string]string{
			fmt.Sprintf(stateKeyFormat, server1Name):          statusVMError,
			fmt.Sprintf(stateTimestampKeyFormat, server1Name): time.Now().Add(5 * time.Hour).Format(time.RFC3339),
			fmt.Sprintf(stateKeyFormat, server2Name):          statusVMRunning,
			fmt.Sprintf(stateTimestampKeyFormat, server2Name): time.Now().Add(5 * time.Hour).Format(time.RFC3339),
		},
	}
}

func createStateMapOneVMNotRunning() *v1.ConfigMap {
	return &v1.ConfigMap{
		Name:      stateMapName,
		Namespace: stateMapNamespace,
		Data: map[string]string{
			fmt.Sprintf(stateKeyFormat, server1Name):          vmNotRunningState,
			fmt.Sprintf(stateTimestampKeyFormat, server1Name): time.Now().Add(5 * time.Hour).Format(time.RFC3339),
			fmt.Sprintf(stateKeyFormat, server2Name):          statusVMRunning,
			fmt.Sprintf(stateTimestampKeyFormat, server2Name): time.Now().Add(5 * time.Hour).Format(time.RFC3339),
		},
	}
}

func createStateMapOneVMWrongTimestampFormat() *v1.ConfigMap {
	return &v1.ConfigMap{
		Name:      stateMapName,
		Namespace: stateMapNamespace,
		Data: map[string]string{
			fmt.Sprintf(stateKeyFormat, server1Name):          vmNotRunningState,
			fmt.Sprintf(stateTimestampKeyFormat, server1Name): time.Now().Add(5 * time.Hour).Format(time.RFC822),
			fmt.Sprintf(stateKeyFormat, server2Name):          statusVMRunning,
			fmt.Sprintf(stateTimestampKeyFormat, server2Name): time.Now().Add(5 * time.Hour).Format(time.RFC3339),
		},
	}
}

func createStateMapOneVMMissingState() *v1.ConfigMap {
	return &v1.ConfigMap{
		Name:      stateMapName,
		Namespace: stateMapNamespace,
		Data: map[string]string{
			fmt.Sprintf(stateKeyFormat, server2Name):          statusVMRunning,
			fmt.Sprintf(stateTimestampKeyFormat, server2Name): time.Now().Add(5 * time.Hour).Format(time.RFC3339),
		},
	}
}

func createStateMapOneVMMissingStateTimestamp() *v1.ConfigMap {
	return &v1.ConfigMap{
		Name:      stateMapName,
		Namespace: stateMapNamespace,
		Data: map[string]string{
			fmt.Sprintf(stateKeyFormat, server1Name):          vmNotRunningState,
			fmt.Sprintf(stateKeyFormat, server2Name):          statusVMRunning,
			fmt.Sprintf(stateTimestampKeyFormat, server2Name): time.Now().Add(5 * time.Hour).Format(time.RFC3339),
		},
	}
}

func createStateMapEmpty() *v1.ConfigMap {
	return &v1.ConfigMap{
		Name:      stateMapName,
		Namespace: stateMapNamespace,
		Data:      map[string]string{},
	}
}

func createStateMapZeroTimestamp() *v1.ConfigMap {
	zeroTime := time.Time{}
	return &v1.ConfigMap{
		Name:      stateMapName,
		Namespace: stateMapNamespace,
		Data: map[string]string{
			fmt.Sprintf(stateKeyFormat, server1Name):          statusVMRunning,
			fmt.Sprintf(stateTimestampKeyFormat, server1Name): zeroTime.Format(time.RFC3339),
			fmt.Sprintf(stateKeyFormat, server2Name):          statusVMRunning,
			fmt.Sprintf(stateTimestampKeyFormat, server2Name): zeroTime.Format(time.RFC3339),
		},
	}
}

func Test_serverSetController_Observe_NicMultiQueue(t *testing.T) {
	type fields struct {
		kube client.Client
	}
	type args struct {
		ctx context.Context
		cr  *v1alpha1.ServerSet
	}

	nic1 := createNic(v1alpha1.NicParameters{Name: server1Name})
	nic2 := createNic(v1alpha1.NicParameters{Name: server2Name})
	bootVolume1 := createBootVolumeWithHotPlug(bootVolumeNamePrefix + server1Name)
	bootVolume2 := createBootVolumeWithHotPlug(bootVolumeNamePrefix + server2Name)

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    managed.ExternalObservation
		wantErr bool
	}{
		{
			name: "NicMultiQueue equal via distinct pointers - up to date",
			fields: fields{
				kube: fakeKubeClientObjs(
					createServerWithNicMultiQueue(server1Name, new(true)),
					createServerWithNicMultiQueue(server2Name, new(true)),
					bootVolume1, bootVolume2, nic1, nic2,
				),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedServerSpec(v1alpha1.ServerSetTemplateSpec{
					CPUFamily:     serverSetCPUFamily,
					Cores:         serverSetCores,
					RAM:           serverSetRAM,
					NicMultiQueue: new(true),
				}),
			},
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  true,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr: false,
		},
		{
			name: "NicMultiQueue both nil - up to date",
			fields: fields{
				kube: fakeKubeClientObjs(
					createServer(server1Name), createServer(server2Name),
					bootVolume1, bootVolume2, nic1, nic2,
				),
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
			name: "NicMultiQueue template true, server nil - not up to date",
			fields: fields{
				kube: fakeKubeClientObjs(
					createServer(server1Name), createServer(server2Name),
					bootVolume1, bootVolume2, nic1, nic2,
				),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedServerSpec(v1alpha1.ServerSetTemplateSpec{
					CPUFamily:     serverSetCPUFamily,
					Cores:         serverSetCores,
					RAM:           serverSetRAM,
					NicMultiQueue: new(true),
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
			name: "NicMultiQueue template true, server false - not up to date",
			fields: fields{
				kube: fakeKubeClientObjs(
					createServerWithNicMultiQueue(server1Name, new(false)),
					createServerWithNicMultiQueue(server2Name, new(false)),
					bootVolume1, bootVolume2, nic1, nic2,
				),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedServerSpec(v1alpha1.ServerSetTemplateSpec{
					CPUFamily:     serverSetCPUFamily,
					Cores:         serverSetCores,
					RAM:           serverSetRAM,
					NicMultiQueue: new(true),
				}),
			},
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  false,
				Diff:              "server[0](serverset-server-0-0): nicMultiQueue exp=true act=false | server[1](serverset-server-1-0): nicMultiQueue exp=true act=false",
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

func Test_serverSetController_Update_NicMultiQueue(t *testing.T) {
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
		wantWrappedErr  error
		want            managed.ExternalUpdate
		wantUpdateCalls int
	}{
		{
			name: "NicMultiQueue both equal (no update)",
			fields: fields{
				kube: fakeKubeClientUpdateMethodWithNicMultiQueueServers(new(true), &v1alpha1.Server{}),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedServerSpec(v1alpha1.ServerSetTemplateSpec{
					CPUFamily:     serverSetCPUFamily,
					Cores:         serverSetCores,
					RAM:           serverSetRAM,
					NicMultiQueue: new(true),
				}),
			},
			wantErr: nil,
			want: managed.ExternalUpdate{
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantUpdateCalls: 0,
		},
		{
			name: "NicMultiQueue both nil (no update)",
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
			name: "NicMultiQueue old nil, cr true (update)",
			fields: fields{
				kube: fakeKubeClientUpdateMethodWithNicMultiQueueServersAndSuccessfulFailover(nil, &v1alpha1.Server{}),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedServerSpec(v1alpha1.ServerSetTemplateSpec{
					CPUFamily:     serverSetCPUFamily,
					Cores:         serverSetCores,
					RAM:           serverSetRAM,
					NicMultiQueue: new(true),
				}),
			},
			wantErr: nil,
			want: managed.ExternalUpdate{
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantUpdateCalls: 2,
		},
		{
			name: "NicMultiQueue old true, cr false (update)",
			fields: fields{
				kube: fakeKubeClientUpdateMethodWithNicMultiQueueServersAndSuccessfulFailover(new(true), &v1alpha1.Server{}),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedServerSpec(v1alpha1.ServerSetTemplateSpec{
					CPUFamily:     serverSetCPUFamily,
					Cores:         serverSetCores,
					RAM:           serverSetRAM,
					NicMultiQueue: new(false),
				}),
			},
			wantErr: nil,
			want: managed.ExternalUpdate{
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantUpdateCalls: 2,
		},
		{
			name: "NicMultiQueue old nil, cr true, failed failover (update returns wait error)",
			fields: fields{
				kube: fakeKubeClientUpdateMethodWithNicMultiQueueServersAndFailedFailover(nil, &v1alpha1.Server{}),
				log:  logging.NewNopLogger(),
			},
			args: args{
				ctx: context.Background(),
				cr: createServerSetWithUpdatedServerSpec(v1alpha1.ServerSetTemplateSpec{
					CPUFamily:     serverSetCPUFamily,
					Cores:         serverSetCores,
					RAM:           serverSetRAM,
					NicMultiQueue: new(true),
				}),
			},
			wantWrappedErr:  fmt.Errorf("error waiting for server to be updated"),
			want:            managed.ExternalUpdate{},
			wantUpdateCalls: 1,
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

			got, err := e.Update(tt.args.ctx, tt.args.cr)

			assertions := require.New(t)
			if tt.wantWrappedErr != nil {
				assertions.ErrorContains(err, tt.wantWrappedErr.Error())
			} else {
				assertions.Equalf(tt.wantErr, err, "Wrong error")
			}
			assertions.Equalf(tt.want, got, "Wrong response")
			kubeClient := tt.fields.kube.(*kubeClientFake)
			kubeClient.AssertNumberOfCalls(t, "Update", tt.wantUpdateCalls)
		})
	}
}
