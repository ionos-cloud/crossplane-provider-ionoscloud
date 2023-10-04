/*
Copyright 2020 The Crossplane Authors.

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

package k8scluster

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/golang/mock/gomock"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/assert"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/k8s/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/k8s"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/mock/clients/k8s/k8scluster"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

// Unlike many Kubernetes projects Crossplane does not use third party testing
// libraries, per the common Go test review comments. Crossplane encourages the
// use of table driven unit tests. The tests of the crossplane-runtime project
// are representative of the testing style Crossplane encourages.
//
// https://github.com/golang/go/wiki/TestComments
// https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#contributing-code

func TestExternalControlPlaneClientObserve(t *testing.T) {
	tests := []struct {
		name                    string
		setupControlPlaneClient func(client *k8scluster.MockClient)
		args                    resource.Managed
		want                    managed.ExternalObservation
		wantErr                 bool
	}{
		{
			name: "Wrong Type, don't expect any calls",
			setupControlPlaneClient: func(client *k8scluster.MockClient) {
			},
			args: &v1alpha1.NodePool{},
			want: managed.ExternalObservation{
				ResourceExists:          false,
				ResourceUpToDate:        false,
				ResourceLateInitialized: false,
				ConnectionDetails:       nil,
				Diff:                    "",
			},
			wantErr: true,
		},
		{
			name: "No external name set",
			setupControlPlaneClient: func(client *k8scluster.MockClient) {
			},
			args: &v1alpha1.Cluster{},
			want: managed.ExternalObservation{
				ResourceExists:          false,
				ResourceUpToDate:        false,
				ResourceLateInitialized: false,
				ConnectionDetails:       nil,
				Diff:                    "",
			},
			wantErr: false,
		},
		{
			name: "Cluster does not exist",
			setupControlPlaneClient: func(client *k8scluster.MockClient) {
				client.EXPECT().
					GetK8sCluster(context.Background(), "cluster-id").
					Return(ionoscloud.KubernetesCluster{}, &ionoscloud.APIResponse{
						Response: &http.Response{
							StatusCode: http.StatusNotFound,
						},
					}, errors.New("resource not found"))
			},
			args: func() *v1alpha1.Cluster {
				np := &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{
						ForProvider: v1alpha1.ClusterParameters{},
					},
				}
				meta.SetExternalName(np, "cluster-id")
				return np
			}(),
			want: managed.ExternalObservation{
				ResourceExists:          false,
				ResourceUpToDate:        false,
				ResourceLateInitialized: false,
				ConnectionDetails:       nil,
				Diff:                    "",
			},
			wantErr: false,
		},
		{
			name: "Internal Client Error",
			setupControlPlaneClient: func(client *k8scluster.MockClient) {
				client.EXPECT().
					GetK8sCluster(context.Background(), "cluster-id").
					Return(
						ionoscloud.KubernetesCluster{},
						&ionoscloud.APIResponse{Response: nil},
						errors.New("some error in the client, no response"),
					)
			},
			args: func() *v1alpha1.Cluster {
				np := &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{
						ForProvider: v1alpha1.ClusterParameters{},
					},
				}
				meta.SetExternalName(np, "cluster-id")
				return np
			}(),
			want: managed.ExternalObservation{
				ResourceExists:          false,
				ResourceUpToDate:        false,
				ResourceLateInitialized: false,
				ConnectionDetails:       nil,
				Diff:                    "",
			},
			wantErr: true,
		},
		{
			name: "Cluster up to date",
			setupControlPlaneClient: func(client *k8scluster.MockClient) {
				client.EXPECT().GetK8sCluster(context.Background(), "cluster-id").Return(ionoscloud.KubernetesCluster{
					Properties: &ionoscloud.KubernetesClusterProperties{
						Name: ionoscloud.PtrString("cluster-name"),
					},
					Metadata: &ionoscloud.DatacenterElementMetadata{
						State: ionoscloud.PtrString(k8s.ACTIVE),
					},
				}, nil, nil)
				rets := "{\"apiVersion\":\"v1\",\"kind\":\"Config\",\"preferences\":{},\"current-context\":\"cluster-admin@exampleK8sCluster\",\"clusters\":[{\"cluster\":{\"certificate-authority-data\":\"cadata\",\"server\":\"https://domain.example\"},\"name\":\"exampleK8sCluster\"}],\"contexts\":[{\"context\":{\"cluster\":\"cluster-name\",\"user\":\"cluster-admin\"},\"name\":\"cluster-admin@exampleK8sCluster\"}],\"users\":[{\"name\":\"cluster-admin\",\"user\":{\"token\":\"longtoken\"}}]}"
				client.EXPECT().GetKubeConfig(context.Background(), "cluster-id").Return(rets, nil, nil)
			},
			args: func() *v1alpha1.Cluster {
				np := &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{
						ForProvider: v1alpha1.ClusterParameters{
							Name: "cluster-name",
						},
					},
				}
				meta.SetExternalName(np, "cluster-id")
				return np
			}(),
			want: managed.ExternalObservation{
				ResourceExists:          true,
				ResourceUpToDate:        true,
				ResourceLateInitialized: false,
				ConnectionDetails: managed.ConnectionDetails{
					"kubeconfig": []byte("{\"apiVersion\":\"v1\",\"kind\":\"Config\",\"preferences\":{},\"current-context\":\"cluster-admin@exampleK8sCluster\",\"clusters\":[{\"cluster\":{\"certificate-authority-data\":\"cadata\",\"server\":\"https://domain.example\"},\"name\":\"exampleK8sCluster\"}],\"contexts\":[{\"context\":{\"cluster\":\"cluster-name\",\"user\":\"cluster-admin\"},\"name\":\"cluster-admin@exampleK8sCluster\"}],\"users\":[{\"name\":\"cluster-admin\",\"user\":{\"token\":\"longtoken\"}}]}"),
					"name":       []byte(""),
					"server":     []byte("https://domain.example"),
					"token":      []byte("longtoken"),
				},
				Diff: "",
			},
			wantErr: false,
		},
		{
			name: "Cluster is destroying",
			setupControlPlaneClient: func(client *k8scluster.MockClient) {
				client.EXPECT().GetK8sCluster(context.Background(), "cluster-id").Return(ionoscloud.KubernetesCluster{
					Properties: &ionoscloud.KubernetesClusterProperties{
						Name: ionoscloud.PtrString("cluster-name"),
					},
					Metadata: &ionoscloud.DatacenterElementMetadata{
						State: ionoscloud.PtrString(k8s.DESTROYING),
					},
				}, nil, nil)
				client.EXPECT().GetKubeConfig(context.Background(), "cluster-id").Return("kubeconfig-base64", nil, nil)
			},
			args: func() *v1alpha1.Cluster {
				np := &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{
						ForProvider: v1alpha1.ClusterParameters{
							Name: "cluster-name",
						},
					},
				}
				meta.SetExternalName(np, "cluster-id")
				return np
			}(),
			want: managed.ExternalObservation{
				ResourceExists:          true,
				ResourceUpToDate:        true,
				ResourceLateInitialized: false,
				ConnectionDetails: managed.ConnectionDetails{
					"kubeconfig": []byte("kubeconfig-base64"),
					"name":       []byte(""),
					"server":     []byte(""),
					"token":      []byte(""),
				},
			},
			wantErr: false,
		},
		{
			name: "Cluster must be updated",
			setupControlPlaneClient: func(client *k8scluster.MockClient) {
				client.EXPECT().GetK8sCluster(context.Background(), "cluster-id").Return(ionoscloud.KubernetesCluster{
					Properties: &ionoscloud.KubernetesClusterProperties{
						Name:       ionoscloud.PtrString("node-pool-name"),
						K8sVersion: ionoscloud.PtrString("1.22.33"),
					},
				}, nil, nil)
				client.EXPECT().GetKubeConfig(context.Background(), "cluster-id").Return("kubeconfig-base64", nil, nil)
			},
			args: func() *v1alpha1.Cluster {
				np := &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{
						ForProvider: v1alpha1.ClusterParameters{
							Name:       "cluster-name",
							K8sVersion: "1.33.44",
						},
					},
				}
				meta.SetExternalName(np, "cluster-id")
				return np
			}(),
			want: managed.ExternalObservation{
				ResourceExists:          true,
				ResourceUpToDate:        false,
				ResourceLateInitialized: false,
				ConnectionDetails: managed.ConnectionDetails{
					"kubeconfig": []byte("kubeconfig-base64"),
					"name":       []byte(""),
					"server":     []byte(""),
					"token":      []byte(""),
				},
				Diff: "",
			},
			wantErr: false,
		},
		{
			name: "API response is inconsistent",
			setupControlPlaneClient: func(client *k8scluster.MockClient) {
				client.EXPECT().GetK8sCluster(context.Background(), "cluster-id").Return(ionoscloud.KubernetesCluster{
					Properties: nil,
				}, nil, nil)
				client.EXPECT().GetKubeConfig(context.Background(), "cluster-id").Return("kubeconfig-base64", nil, nil)

			},
			args: func() *v1alpha1.Cluster {
				np := &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{
						ForProvider: v1alpha1.ClusterParameters{
							Name:       "cluster-name",
							K8sVersion: "1.33.44",
						},
					},
				}
				meta.SetExternalName(np, "cluster-id")
				return np
			}(),
			want: managed.ExternalObservation{
				ResourceExists:          true,
				ResourceUpToDate:        false,
				ResourceLateInitialized: false,
				ConnectionDetails: managed.ConnectionDetails{
					"kubeconfig": []byte("kubeconfig-base64"),
					"name":       []byte(""),
					"server":     []byte(""),
					"token":      []byte(""),
				},
				Diff: "",
			},
			wantErr: false,
		},
		{
			name: "Get Kubeconfig fails",
			setupControlPlaneClient: func(client *k8scluster.MockClient) {
				client.EXPECT().GetK8sCluster(context.Background(), "cluster-id").Return(ionoscloud.KubernetesCluster{
					Properties: nil,
				}, nil, nil)
				client.EXPECT().GetKubeConfig(context.Background(), "cluster-id").Return("", nil, errors.New("api broken"))

			},
			args: func() *v1alpha1.Cluster {
				np := &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{
						ForProvider: v1alpha1.ClusterParameters{
							Name:       "cluster-name",
							K8sVersion: "1.33.44",
						},
					},
				}
				meta.SetExternalName(np, "cluster-id")
				return np
			}(),
			want: managed.ExternalObservation{
				ResourceExists:          true,
				ResourceUpToDate:        false,
				ResourceLateInitialized: false,
				ConnectionDetails: managed.ConnectionDetails{
					"kubeconfig": []byte(""),
					"name":       []byte(""),
					"server":     []byte(""),
					"token":      []byte(""),
				},
				Diff: "",
			},
			wantErr: false,
		},
		{
			name: "Cluster in State Deploying is up to date",
			setupControlPlaneClient: func(client *k8scluster.MockClient) {
				client.EXPECT().GetK8sCluster(context.Background(), "cluster-id").Return(ionoscloud.KubernetesCluster{
					Metadata: &ionoscloud.DatacenterElementMetadata{
						State: ionoscloud.PtrString(k8s.DEPLOYING),
					},
					Properties: &ionoscloud.KubernetesClusterProperties{
						Name:       ionoscloud.PtrString("node-pool-name"),
						K8sVersion: ionoscloud.PtrString("1.22.33"),
					}}, nil, nil)
				client.EXPECT().
					GetKubeConfig(context.Background(), "cluster-id").
					Return("", nil, errors.New("resource is deploying"))

			},
			args: func() *v1alpha1.Cluster {
				np := &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{
						ForProvider: v1alpha1.ClusterParameters{
							Name:       "cluster-name",
							K8sVersion: "1.33.44",
						},
					},
				}
				meta.SetExternalName(np, "cluster-id")
				return np
			}(),
			want: managed.ExternalObservation{
				ResourceExists:          true,
				ResourceUpToDate:        true,
				ResourceLateInitialized: false,
				ConnectionDetails: managed.ConnectionDetails{
					"kubeconfig": []byte(""),
					"name":       []byte(""),
					"server":     []byte(""),
					"token":      []byte(""),
				},
				Diff: "",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mnps := k8scluster.NewMockClient(ctrl)
			tt.setupControlPlaneClient(mnps)
			c := &externalCluster{
				service: mnps,
				log:     utils.NewTestLogger(),
			}
			got, err := c.Observe(context.Background(), tt.args)
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExternalControlPlaneClientDelete(t *testing.T) {
	tests := []struct {
		name                    string
		setupControlPlaneClient func(client *k8scluster.MockClient)
		args                    resource.Managed
		wantErr                 bool
	}{
		{
			name: "Wrong Type, don't expect any calls",
			setupControlPlaneClient: func(client *k8scluster.MockClient) {
			},
			args:    &v1alpha1.NodePool{},
			wantErr: true,
		},
		{
			name: "Cluster does not exist",
			setupControlPlaneClient: func(client *k8scluster.MockClient) {
				client.EXPECT().HasActiveK8sNodePools(context.Background(), "cluster-id").Return(false, errors.New("not found"))
			},
			args: func() *v1alpha1.Cluster {
				np := &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{
						ForProvider: v1alpha1.ClusterParameters{
							Name: "foo",
						},
					},
					Status: v1alpha1.ClusterStatus{AtProvider: v1alpha1.ClusterObservation{ClusterID: "cluster-id"}},
				}
				meta.SetExternalName(np, "cluster-id")
				return np
			}(),
			wantErr: true,
		},
		{
			name: "Cluster delete",
			setupControlPlaneClient: func(client *k8scluster.MockClient) {
				nodepools := client.EXPECT().HasActiveK8sNodePools(context.Background(), "cluster-id").Return(false, nil)
				client.EXPECT().DeleteK8sCluster(context.Background(), "cluster-id").Return(&ionoscloud.APIResponse{}, nil).After(nodepools)
			},
			args: func() *v1alpha1.Cluster {
				np := &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{
						ForProvider: v1alpha1.ClusterParameters{
							Name: "foo",
						},
					},
					Status: v1alpha1.ClusterStatus{AtProvider: v1alpha1.ClusterObservation{ClusterID: "cluster-id", State: "ACTIVE"}},
				}
				meta.SetExternalName(np, "cluster-id")
				return np
			}(),
			wantErr: false,
		},
		{
			name: "ionos API error",
			setupControlPlaneClient: func(client *k8scluster.MockClient) {
				nodepools := client.EXPECT().HasActiveK8sNodePools(context.Background(), "cluster-id").Return(false, nil)
				client.EXPECT().
					DeleteK8sCluster(context.Background(), "cluster-id").
					Return(&ionoscloud.APIResponse{}, errors.New("API error")).
					After(nodepools)
			},
			args: func() *v1alpha1.Cluster {
				np := &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{
						ForProvider: v1alpha1.ClusterParameters{
							Name: "foo",
						},
					},
					Status: v1alpha1.ClusterStatus{AtProvider: v1alpha1.ClusterObservation{ClusterID: "cluster-id", State: "ACTIVE"}},
				}
				meta.SetExternalName(np, "cluster-id")
				return np
			}(),
			wantErr: true,
		},
		{
			name: "externalControlPlaneClient name not yet known",
			setupControlPlaneClient: func(client *k8scluster.MockClient) {
			},
			args: func() *v1alpha1.Cluster {
				np := &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{
						ForProvider: v1alpha1.ClusterParameters{
							Name: "foo",
						},
					},
					Status: v1alpha1.ClusterStatus{AtProvider: v1alpha1.ClusterObservation{ClusterID: ""}},
				}
				meta.SetExternalName(np, "")
				return np
			}(),
			wantErr: false,
		},
		{
			name: "already destroying",
			setupControlPlaneClient: func(client *k8scluster.MockClient) {
				client.EXPECT().HasActiveK8sNodePools(context.Background(), "cluster-id").Return(false, nil)
			},
			args: func() *v1alpha1.Cluster {
				np := &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{
						ForProvider: v1alpha1.ClusterParameters{
							Name: "foo",
						},
					},
					Status: v1alpha1.ClusterStatus{AtProvider: v1alpha1.ClusterObservation{ClusterID: "cluster-id", State: "DESTROYING"}},
				}
				meta.SetExternalName(np, "cluster-id")
				return np
			}(),
			wantErr: false,
		},
		{
			name: "already terminated",
			setupControlPlaneClient: func(client *k8scluster.MockClient) {
				client.EXPECT().HasActiveK8sNodePools(context.Background(), "cluster-id").Return(false, nil)
			},
			args: func() *v1alpha1.Cluster {
				np := &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{
						ForProvider: v1alpha1.ClusterParameters{
							Name: "foo",
						},
					},
					Status: v1alpha1.ClusterStatus{AtProvider: v1alpha1.ClusterObservation{ClusterID: "cluster-id", State: "TERMINATED"}},
				}
				meta.SetExternalName(np, "cluster-id")
				return np
			}(),
			wantErr: false,
		},
		{
			name: "Cluster has nodepools",
			setupControlPlaneClient: func(client *k8scluster.MockClient) {
				client.EXPECT().HasActiveK8sNodePools(context.Background(), "cluster-id").Return(true, nil)

			},
			args: func() *v1alpha1.Cluster {
				np := &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{
						ForProvider: v1alpha1.ClusterParameters{
							Name:       "cluster-name",
							K8sVersion: "1.33.44",
						},
					},
					Status: v1alpha1.ClusterStatus{AtProvider: v1alpha1.ClusterObservation{ClusterID: "cluster-id", State: "DESTROYING"}},
				}
				meta.SetExternalName(np, "cluster-id")
				return np
			}(),
			wantErr: true,
		},
		{
			name: "still deploying",
			setupControlPlaneClient: func(client *k8scluster.MockClient) {
				client.EXPECT().HasActiveK8sNodePools(context.Background(), "cluster-id").Return(false, nil)
			},
			args: func() *v1alpha1.Cluster {
				np := &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{
						ForProvider: v1alpha1.ClusterParameters{
							Name: "foo",
						},
					},
					Status: v1alpha1.ClusterStatus{AtProvider: v1alpha1.ClusterObservation{ClusterID: "cluster-id", State: "DEPLOYING"}},
				}
				meta.SetExternalName(np, "cluster-id")
				return np
			}(),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mnps := k8scluster.NewMockClient(ctrl)
			tt.setupControlPlaneClient(mnps)
			c := &externalCluster{
				service: mnps,
				log:     utils.NewTestLogger(),
			}
			err := c.Delete(context.Background(), tt.args)
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestExternalControlPlaneClientCreate(t *testing.T) {

	tests := []struct {
		name                    string
		setupControlPlaneClient func(client *k8scluster.MockClient)
		args                    resource.Managed
		want                    managed.ExternalCreation
		wantErr                 bool
		wantExternalName        string
		wantCondition           xpv1.Condition
	}{
		{
			name:                    "Wrong Type, don't expect any calls",
			setupControlPlaneClient: func(client *k8scluster.MockClient) {},
			args:                    &v1alpha1.NodePool{},
			wantErr:                 true,
			wantCondition: xpv1.Condition{
				Type:   "Ready",
				Status: "Unknown",
			},
		},
		{
			name:                    "Already deploying",
			setupControlPlaneClient: func(client *k8scluster.MockClient) {},
			args: func() *v1alpha1.Cluster {
				np := &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{
						ForProvider: v1alpha1.ClusterParameters{
							Name:       "testCluster",
							K8sVersion: "v1.22.33",
						},
					},
					Status: v1alpha1.ClusterStatus{AtProvider: v1alpha1.ClusterObservation{State: "DEPLOYING"}},
				}
				return np
			}(),
			want:             managed.ExternalCreation{},
			wantErr:          false,
			wantExternalName: "",
			wantCondition:    xpv1.Creating(),
		},
		{
			name: "Cluster creation",
			setupControlPlaneClient: func(client *k8scluster.MockClient) {
				expectedCluster := ionoscloud.KubernetesClusterForPost{
					Properties: &ionoscloud.KubernetesClusterPropertiesForPost{
						Name:       ionoscloud.PtrString("testCluster"),
						K8sVersion: ionoscloud.PtrString("v1.22.33"),
					},
				}
				returnedCluster := ionoscloud.KubernetesCluster{
					Id: ionoscloud.PtrString("1234"),
				}
				client.EXPECT().
					CreateK8sCluster(
						context.Background(),
						gomock.GotFormatterAdapter(clusterGotFormatter{},
							matchesCluster(expectedCluster)),
					).
					Return(returnedCluster, nil, nil)
			},
			args: func() *v1alpha1.Cluster {
				np := &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{
						ForProvider: v1alpha1.ClusterParameters{
							Name:       "testCluster",
							K8sVersion: "v1.22.33",
						},
					},
				}
				return np
			}(),
			want: managed.ExternalCreation{
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr:          false,
			wantExternalName: "1234",
			wantCondition:    xpv1.Creating(),
		},
		{
			name: "API errors",
			setupControlPlaneClient: func(client *k8scluster.MockClient) {
				client.EXPECT().
					CreateK8sCluster(context.Background(), gomock.AssignableToTypeOf(ionoscloud.KubernetesClusterForPost{})).
					Return(ionoscloud.KubernetesCluster{}, nil, errors.New("failed to execute"))
			},
			args: func() *v1alpha1.Cluster {
				np := &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{
						ForProvider: v1alpha1.ClusterParameters{
							Name:       "testCluster",
							K8sVersion: "v1.22.33",
						},
					},
				}
				return np
			}(),
			want: managed.ExternalCreation{
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr: true,
			wantCondition: xpv1.Condition{
				Type:   "Ready",
				Status: "False",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mnps := k8scluster.NewMockClient(ctrl)
			tt.setupControlPlaneClient(mnps)
			c := &externalCluster{
				service: mnps,
				log:     utils.NewTestLogger(),
			}
			got, err := c.Create(context.Background(), tt.args)
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantExternalName, meta.GetExternalName(tt.args))
			if np, ok := tt.args.(*v1alpha1.Cluster); ok {
				assert.Equal(t, tt.wantExternalName, np.Status.AtProvider.ClusterID)
			}
			assert.EqualValues(t, tt.wantCondition.Status, tt.args.GetCondition(xpv1.TypeReady).Status)
		})
	}
}

type clusterGotFormatter struct {
}

func (n clusterGotFormatter) Got(got interface{}) string {
	return mustMarshal(got)
}

type clusterMatcher struct {
	expected ionoscloud.KubernetesClusterForPost
}

func matchesCluster(expected ionoscloud.KubernetesClusterForPost) clusterMatcher {
	return clusterMatcher{
		expected: expected,
	}
}

func (n clusterMatcher) Matches(x interface{}) bool {
	np, ok := x.(ionoscloud.KubernetesClusterForPost)
	if !ok {
		return false
	}
	switch {
	case np.Id != nil && n.expected.Id == nil:
		return false
	case np.Id == nil && n.expected.Id != nil:
		return false
	case np.Id != nil && n.expected.Id != nil:
		if *np.Id != *n.expected.Id {
			return false
		}
	}
	return reflect.DeepEqual(*n.expected.Properties, *np.Properties)
}

func (n clusterMatcher) String() string {
	return mustMarshal(n.expected)
}

type clusterPutMatcher struct {
	expected ionoscloud.KubernetesClusterForPut
}

func matchesClusterPut(expected ionoscloud.KubernetesClusterForPut) clusterPutMatcher {
	return clusterPutMatcher{
		expected: expected,
	}
}

func (n clusterPutMatcher) Matches(x interface{}) bool {
	np, ok := x.(ionoscloud.KubernetesClusterForPut)
	if !ok {
		return false
	}
	switch {
	case np.Id != nil && n.expected.Id == nil:
		return false
	case np.Id == nil && n.expected.Id != nil:
		return false
	case np.Id != nil && n.expected.Id != nil:
		if *np.Id != *n.expected.Id {
			return false
		}
	}
	return reflect.DeepEqual(*n.expected.Properties, *np.Properties)
}

func (n clusterPutMatcher) String() string {
	return mustMarshal(n.expected)
}

func TestExternalControlPlaneClientUpdate(t *testing.T) {

	tests := []struct {
		name                    string
		setupControlPlaneClient func(client *k8scluster.MockClient)
		args                    resource.Managed
		want                    managed.ExternalUpdate
		wantErr                 bool
		wantCondition           xpv1.Condition
	}{
		{
			name: "Wrong Type, don't expect any calls",
			setupControlPlaneClient: func(client *k8scluster.MockClient) {
			},
			args:    &v1alpha1.NodePool{},
			wantErr: true,
			wantCondition: xpv1.Condition{
				Type:   "Ready",
				Status: "Unknown",
			},
		},
		{
			name: "Already updating",
			setupControlPlaneClient: func(client *k8scluster.MockClient) {
			},
			args: func() *v1alpha1.Cluster {
				np := &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{
						ForProvider: v1alpha1.ClusterParameters{
							Name:       "testCluster",
							K8sVersion: "v1.22.33",
						},
					},
					Status: v1alpha1.ClusterStatus{AtProvider: v1alpha1.ClusterObservation{State: "UPDATING"}},
				}
				return np
			}(),
			wantErr: false,
			wantCondition: xpv1.Condition{
				Type:   "Ready",
				Status: "Unknown",
			},
		},
		{
			name: "ID not set",
			setupControlPlaneClient: func(client *k8scluster.MockClient) {
			},
			args: func() *v1alpha1.Cluster {
				np := &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{
						ForProvider: v1alpha1.ClusterParameters{
							Name:       "testCluster",
							K8sVersion: "v1.22.33",
						},
					},
				}
				return np
			}(),
			wantErr: true,
			wantCondition: xpv1.Condition{
				Type:   "Ready",
				Status: "Unknown",
			},
		},
		{
			name: "Put Cluster Fails",
			setupControlPlaneClient: func(client *k8scluster.MockClient) {
				client.EXPECT().
					UpdateK8sCluster(
						context.Background(),
						"cluster-id",
						gomock.AssignableToTypeOf(ionoscloud.KubernetesClusterForPut{}),
					).
					Return(ionoscloud.KubernetesCluster{}, nil, errors.New("put failed"))
			},
			args: func() *v1alpha1.Cluster {
				np := &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{
						ForProvider: v1alpha1.ClusterParameters{
							Name:       "testCluster",
							K8sVersion: "v1.22.33",
						},
					},
					Status: v1alpha1.ClusterStatus{
						AtProvider: v1alpha1.ClusterObservation{
							State:     k8s.ACTIVE,
							ClusterID: "cluster-id",
						},
					},
				}
				meta.SetExternalName(np, "cluster-id")
				return np
			}(),
			wantErr: true,
			wantCondition: xpv1.Condition{
				Type:   "Ready",
				Status: "Unknown",
			},
		},
		{
			name: "Empty APISubnetAllowList",
			setupControlPlaneClient: func(client *k8scluster.MockClient) {
				client.EXPECT().UpdateK8sCluster(context.Background(), "cluster-id",
					gomock.GotFormatterAdapter(clusterGotFormatter{}, matchesClusterPut(ionoscloud.KubernetesClusterForPut{
						Properties: &ionoscloud.KubernetesClusterPropertiesForPut{
							ApiSubnetAllowList: nil,
							Name:               ionoscloud.PtrString("testCluster"),
							K8sVersion:         ionoscloud.PtrString("v1.22.33"),
							MaintenanceWindow: &ionoscloud.KubernetesMaintenanceWindow{
								DayOfTheWeek: ionoscloud.PtrString("Mon"),
								Time:         ionoscloud.PtrString("15:24:30Z"),
							},
						},
					})),
				).
					Return(ionoscloud.KubernetesCluster{}, nil, nil)
			},
			args: func() *v1alpha1.Cluster {
				np := &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{
						ForProvider: v1alpha1.ClusterParameters{
							APISubnetAllowList: []string{},
							Name:               "testCluster",
							K8sVersion:         "v1.22.33",
							MaintenanceWindow: v1alpha1.MaintenanceWindow{
								Time:         "15:24:30Z",
								DayOfTheWeek: "Mon",
							},
						},
					},
					Status: v1alpha1.ClusterStatus{
						AtProvider: v1alpha1.ClusterObservation{
							State:     k8s.ACTIVE,
							ClusterID: "cluster-id",
						},
					},
				}
				meta.SetExternalName(np, "cluster-id")
				return np
			}(),
			wantErr: false,
			wantCondition: xpv1.Condition{
				Type:   "Ready",
				Status: "Unknown",
			},
		},
		{
			name: "API success",
			setupControlPlaneClient: func(client *k8scluster.MockClient) {
				client.EXPECT().UpdateK8sCluster(context.Background(), "cluster-id",
					gomock.GotFormatterAdapter(clusterGotFormatter{}, matchesClusterPut(ionoscloud.KubernetesClusterForPut{
						Properties: &ionoscloud.KubernetesClusterPropertiesForPut{
							ApiSubnetAllowList: &[]string{"233.252.0.12"},
							Name:               ionoscloud.PtrString("testCluster"),
							K8sVersion:         ionoscloud.PtrString("v1.22.33"),
							MaintenanceWindow: &ionoscloud.KubernetesMaintenanceWindow{
								DayOfTheWeek: ionoscloud.PtrString("Mon"),
								Time:         ionoscloud.PtrString("15:24:30Z"),
							},
						},
					})),
				).
					Return(ionoscloud.KubernetesCluster{}, nil, nil)
			},
			args: func() *v1alpha1.Cluster {
				np := &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{
						ForProvider: v1alpha1.ClusterParameters{
							APISubnetAllowList: []string{"233.252.0.12"},
							Name:               "testCluster",
							K8sVersion:         "v1.22.33",
							MaintenanceWindow: v1alpha1.MaintenanceWindow{
								Time:         "15:24:30Z",
								DayOfTheWeek: "Mon",
							},
						},
					},
					Status: v1alpha1.ClusterStatus{
						AtProvider: v1alpha1.ClusterObservation{
							State:     k8s.ACTIVE,
							ClusterID: "cluster-id",
						},
					},
				}
				meta.SetExternalName(np, "cluster-id")
				return np
			}(),
			wantErr: false,
			wantCondition: xpv1.Condition{
				Type:   "Ready",
				Status: "Unknown",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mnps := k8scluster.NewMockClient(ctrl)
			tt.setupControlPlaneClient(mnps)
			c := &externalCluster{
				service: mnps,
				log:     utils.NewTestLogger(),
			}
			got, err := c.Update(context.Background(), tt.args)
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, tt.want, got)
			assert.EqualValues(t, tt.wantCondition.Status, tt.args.GetCondition(xpv1.TypeReady).Status)
		})
	}
}

func mustMarshal(data interface{}) string {
	bytes, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}
