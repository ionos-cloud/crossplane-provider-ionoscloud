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
	"reflect"
	"regexp"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	serverSetName   = "serverset"
	bootVolumeSize  = 100
	bootVolumeType  = "HDD"
	bootVolumeImage = "image"
)

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
				State: "AVAILABLE",
			},
		},
	}
}

func createConfigMap(name string) v1.ConfigMap {
	regexPatternServer1 := ".*server-0-0"
	regexPatternServer2 := ".*server-1-0"
	regexS1 := regexp.MustCompile(regexPatternServer1)
	regexS2 := regexp.MustCompile(regexPatternServer2)

	role := "UNKNOWN"
	if regexS1.MatchString(name) {
		role = "ACTIVE"
	} else if regexS2.MatchString(name) {
		role = "PASSIVE"
	}

	return v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Data: map[string]string{"role": role},
	}
}

func createEmptyConfigMap(name string) v1.ConfigMap {
	return v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Data: map[string]string{},
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

func fakeKubeClientObjs(objs ...client.Object) client.WithWatch {
	scheme := runtime.NewScheme()
	v1.AddToScheme(scheme)       // Add the core k8s types to the Scheme
	v1alpha1.AddToScheme(scheme) // Add our custom types from v1alpha to the Scheme
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
}

func createBasicServerSet() *v1alpha1.ServerSet {
	return &v1alpha1.ServerSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServerSet",
			APIVersion: "v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serverSetName,
			Namespace: "",
			Annotations: map[string]string{
				"crossplane.io/external-name": serverSetName,
			},
		},
		Spec: v1alpha1.ServerSetSpec{
			ForProvider: v1alpha1.ServerSetParameters{
				Replicas: 2,
			},
		},
		Status: v1alpha1.ServerSetStatus{},
	}
}

func createServerSetWithUpdatedServerSpec() *v1alpha1.ServerSet {
	return &v1alpha1.ServerSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServerSet",
			APIVersion: "v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serverSetName,
			Namespace: "",
			Annotations: map[string]string{
				"crossplane.io/external-name": serverSetName,
			},
		},
		Spec: v1alpha1.ServerSetSpec{
			ForProvider: v1alpha1.ServerSetParameters{
				Replicas: 2,
				Template: v1alpha1.ServerSetTemplate{
					Spec: v1alpha1.ServerSetTemplateSpec{
						Cores: 10,
						RAM:   20480,
					},
				},
			},
		},
		Status: v1alpha1.ServerSetStatus{},
	}
}

func createServerSetWithUpdatedBootVolume(updatedSpec v1alpha1.ServerSetBootVolumeSpec) *v1alpha1.ServerSet {
	return &v1alpha1.ServerSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServerSet",
			APIVersion: "v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serverSetName,
			Namespace: "",
			Annotations: map[string]string{
				"crossplane.io/external-name": serverSetName,
			},
		},
		Spec: v1alpha1.ServerSetSpec{
			ForProvider: v1alpha1.ServerSetParameters{
				Replicas: 2,
				BootVolumeTemplate: v1alpha1.BootVolumeTemplate{
					Spec: updatedSpec,
				},
			},
		},
		Status: v1alpha1.ServerSetStatus{},
	}
}

func areEqual(t *testing.T, want, got v1alpha1.ServerSetObservation) {
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(v1alpha1.ServerSetReplicaStatus{}, "LastModified")); diff != "" {
		t.Errorf("ServerSetObservation() mismatch (-want +got):\n%s", diff)
	}
}

func createServerSetWhichUpdatesFrom1ReplicaTo2(serverName string) *v1alpha1.ServerSet {
	return &v1alpha1.ServerSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServerSet",
			APIVersion: "v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serverSetName,
			Namespace: "",
			Annotations: map[string]string{
				"crossplane.io/external-name": serverSetName,
			},
		},
		Spec: v1alpha1.ServerSetSpec{
			ForProvider: v1alpha1.ServerSetParameters{
				Replicas: 2,
			},
		},
		Status: v1alpha1.ServerSetStatus{
			AtProvider: v1alpha1.ServerSetObservation{
				Replicas: 1,
				ReplicaStatuses: []v1alpha1.ServerSetReplicaStatus{
					{
						Name:         serverName,
						Role:         "ACTIVE",
						Status:       "READY",
						ErrorMessage: "",
					},
				},
			},
		},
	}
}

func createServerSetWhichUpdatesFrom2ReplicasTo1(serverName1, serverName2 string) *v1alpha1.ServerSet {
	return &v1alpha1.ServerSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServerSet",
			APIVersion: "v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serverSetName,
			Namespace: "",
			Annotations: map[string]string{
				"crossplane.io/external-name": serverSetName,
			},
		},
		Spec: v1alpha1.ServerSetSpec{
			ForProvider: v1alpha1.ServerSetParameters{
				Replicas: 1,
			},
		},
		Status: v1alpha1.ServerSetStatus{
			AtProvider: v1alpha1.ServerSetObservation{
				Replicas: 2,
				ReplicaStatuses: []v1alpha1.ServerSetReplicaStatus{
					{
						Name:         serverName1,
						Role:         "ACTIVE",
						Status:       "READY",
						ErrorMessage: "",
					},
					{
						Name:         serverName2,
						Role:         "ACTIVE",
						Status:       "READY",
						ErrorMessage: "",
					},
				},
			},
		},
	}
}

func Test_serverSetController_Observe(t *testing.T) {
	type fields struct {
		kube client.Client
	}
	type args struct {
		ctx context.Context
		cr  *v1alpha1.ServerSet
	}
	server1 := createServer("serverset-server-0-0")
	server2 := createServer("serverset-server-1-0")
	configMap1 := createConfigMap("configs-" + server1.Name)
	configMap2 := createConfigMap("configs-" + server2.Name)
	nic1 := createNic(server1.Name)
	nic2 := createNic(server2.Name)
	bootVolume1 := createBootVolume("boot-volume-" + server1.Name)
	bootVolume2 := createBootVolume("boot-volume-" + server2.Name)

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    managed.ExternalObservation
		wantErr bool
	}{
		{
			name: "servers, nics and configMap for reading the role created, then resource exists and it is up to date",
			fields: fields{
				kube: fakeKubeClientObjs(&server1, &server2, &configMap1, &configMap2, &nic1, &nic2),
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
			name: "servers not created, then resource does not exist and is not up to date",
			fields: fields{
				kube: fakeKubeClientObjs(),
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSet(),
			},
			want: managed.ExternalObservation{
				ResourceExists:    false,
				ResourceUpToDate:  false,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr: false,
		},
		{
			name: "servers not up to date, then resource exists and is not up to date",
			fields: fields{
				kube: fakeKubeClientObjs(&server1, &server2, &configMap1, &configMap2),
			},
			args: args{
				ctx: context.Background(),
				cr:  createServerSetWithUpdatedServerSpec(),
			},
			want: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  false,
				ConnectionDetails: managed.ConnectionDetails{},
				Diff:              "servers are not up to date",
			},
			wantErr: false,
		},
		{
			name: "boot volume image is not up to date, then resource exists and is not up to date",
			fields: fields{
				kube: fakeKubeClientObjs(&server1, &server2, &configMap1, &configMap2, &bootVolume1, &bootVolume2),
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
				Diff:              "servers are not up to date",
			},
			wantErr: false,
		},
		{
			name: "boot volume size is not up to date, then resource exists and is not up to date",
			fields: fields{
				kube: fakeKubeClientObjs(&server1, &server2, &configMap1, &configMap2, &bootVolume1, &bootVolume2),
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
				Diff:              "servers are not up to date",
			},
			wantErr: false,
		},
		{
			name: "boot volume type is not up to date, then resource exists and is not up to date",
			fields: fields{
				kube: fakeKubeClientObjs(&server1, &server2, &configMap1, &configMap2, &bootVolume1, &bootVolume2),
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
				Diff:              "servers are not up to date",
			},
			wantErr: false,
		},
		{
			name: "servers < replica count, then resource does not exist and is not up to date",
			fields: fields{
				kube: fakeKubeClientObjs(&server1, &configMap1),
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSet(),
			},
			want: managed.ExternalObservation{
				ResourceExists:    false,
				ResourceUpToDate:  false,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr: false,
		},
		{
			name: "nics not created, then resource does not exist and is not up to date",
			fields: fields{
				kube: fakeKubeClientObjs(&server1, &server2, &configMap1, &configMap2),
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSet(),
			},
			want: managed.ExternalObservation{
				ResourceExists:    false,
				ResourceUpToDate:  false,
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
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Observe() got = %v, want %v", got, tt.want)
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

	server1 := createServer("serverset-server-0-0")
	server2 := createServer("serverset-server-1-0")
	configMap1 := createConfigMap("configs-" + server1.Name)
	configMap2 := createConfigMap("configs-" + server2.Name)
	nic1 := createNic(server1.Name)
	nic2 := createNic(server2.Name)
	emptyConfigMap1 := createEmptyConfigMap("configs-" + server1.Name)
	emptyConfigMap2 := createEmptyConfigMap("configs-" + server2.Name)

	fakeKubeClient := fakeKubeClientObjs(&server1, &server2, &configMap1, &configMap2, &nic1, &nic2)
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
				kube: fakeKubeClient,
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
						Role:         "ACTIVE",
						Status:       "READY",
						ErrorMessage: "",
					},
					{
						Name:         server2.Name,
						Role:         "PASSIVE",
						Status:       "READY",
						ErrorMessage: "",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "config map for role not found, then status is not populated and error is returned",
			fields: fields{
				kube: fakeKubeClientObjs(&server1, &server2),
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSet(),
			},
			want: v1alpha1.ServerSetObservation{
				Replicas:        2,
				ReplicaStatuses: make([]v1alpha1.ServerSetReplicaStatus, 2),
			},
			wantErr: true,
		},
		{
			name: "no role found in config, then status is populated but role is set to UNKNOWN",
			fields: fields{
				kube: fakeKubeClientObjs(&server1, &server2, &emptyConfigMap1, &emptyConfigMap2, &nic1, &nic2),
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
						Role:         "UNKNOWN",
						Status:       "READY",
						ErrorMessage: "Role not found in configmap. Will default to UNKNOWN",
					},
					{
						Name:         server2.Name,
						Role:         "UNKNOWN",
						Status:       "READY",
						ErrorMessage: "Role not found in configmap. Will default to UNKNOWN",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "replica count increases, then number of replica status is increased",
			fields: fields{
				kube: fakeKubeClientObjs(&server1, &server2, &configMap1, &configMap2, &nic1, &nic2),
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
						Role:         "ACTIVE",
						Status:       "READY",
						ErrorMessage: "",
					},
					{
						Name:         server2.Name,
						Role:         "PASSIVE",
						Status:       "READY",
						ErrorMessage: "",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "replica count decreases, then number of replica status is decreased",
			fields: fields{
				kube: fakeKubeClientObjs(&server1, &configMap1, &nic1),
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
						Role:         "ACTIVE",
						Status:       "READY",
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
