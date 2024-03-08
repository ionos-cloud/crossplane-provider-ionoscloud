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
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func createBasicServerSet() *v1alpha1.ServerSet {
	return &v1alpha1.ServerSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServerSet",
			APIVersion: "v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "serverset",
			Namespace: "",
			Annotations: map[string]string{
				"crossplane.io/external-name": "serverset",
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

func fakeKubeClient(objs ...client.Object) client.WithWatch {
	scheme, _ := v1alpha1.SchemeBuilder.Build()
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
}

func createServer(name string) v1alpha1.Server {
	return v1alpha1.Server{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Status: v1alpha1.ServerStatus{
			AtProvider: v1alpha1.ServerObservation{
				State: "AVAILABLE",
			},
		},
	}
}

func getConfigMap(name string) v1.ConfigMap {
	role := "UNKNOWN"
	if name == "serverset-server-0-0" {
		role = "ACTIVE"
	} else if name == "serverset-server-1-0" {
		role = "PASSIVE"
	}

	return v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: map[string]string{"role": role},
	}
}

func Test_serverSetController_Observe(t *testing.T) {
	type fields struct {
		kubeWrapper wrapper
	}
	type args struct {
		ctx          context.Context
		cr           *v1alpha1.ServerSet
		replicaIndex int
		version      int
	}
	server1 := createServer("serverset-server-0-0")
	server2 := createServer("serverset-server-1-0")
	configMap1 := getConfigMap("serverset-server-0-0")
	configMap2 := getConfigMap("serverset-server-1-0")
	kubeClient := fakeKubeClient(&server1, &server2, &configMap1, &configMap2)
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    v1alpha1.ServerSetReplicaStatus
		wantErr bool
	}{
		{
			name: "ReplicaStatusesPopulatedCorrectly",
			fields: fields{
				kubeWrapper: wrapper{
					kube: kubeClient,
					log:  logging.NewNopLogger(),
				},
			},
			args: args{
				ctx: context.Background(),
				cr:  createBasicServerSet(),
			},
			want:    v1alpha1.ServerSetReplicaStatus{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &external{
				kubeWrapper: tt.fields.kubeWrapper,
			}
			// WHEN
			got, err := e.Observe(tt.args.ctx, tt.args.cr)

			// THEN
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
