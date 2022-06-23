package k8snodepool

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/golang/mock/gomock"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/k8s/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/k8s"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/mock/clients/compute/datacenter"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/mock/clients/compute/ipblock"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/mock/clients/k8s/k8scluster"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/mock/clients/k8s/k8snodepool"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const (
	testClusterID  = "cluster-id"
	testNodePoolID = "nodepool-id"
	testIPBlockID  = "ipblock-id"
)

func TestExternalNodePoolObserve(t *testing.T) {

	basicObserveNodePool := &v1alpha1.NodePool{
		Spec: v1alpha1.NodePoolSpec{
			ForProvider: v1alpha1.NodePoolParameters{
				DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: "12345"},
				ClusterCfg: v1alpha1.ClusterConfig{
					ClusterID: testClusterID,
				},
			},
		},
	}
	meta.SetExternalName(basicObserveNodePool, testNodePoolID)

	tests := []struct {
		name string
		nodepoolMocker
		args    resource.Managed
		want    managed.ExternalObservation
		wantErr bool
	}{
		{
			name:           "Wrong Type, don't expect any calls",
			nodepoolMocker: nodepoolMocker{},
			args:           &v1alpha1.Cluster{},
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
			name:           "No external name set",
			nodepoolMocker: nodepoolMocker{},
			args:           &v1alpha1.NodePool{},
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
			name: "Fail to observe, if can't find IPBlocks",
			nodepoolMocker: nodepoolMocker{
				setupNodePoolService: func(service *k8snodepool.MockClient) {
					service.EXPECT().
						GetK8sNodePool(context.Background(), testClusterID, testNodePoolID).
						Return(ionoscloud.KubernetesNodePool{
							Properties: &ionoscloud.KubernetesNodePoolProperties{
								Name: ionoscloud.PtrString("node-pool-name"),
							},
						}, nil, nil)
				},
				setupIPBlockService: networkErrorIPBlockService,
			},
			args: func() *v1alpha1.NodePool {
				np := basicObserveNodePool.DeepCopy()
				np.Spec.ForProvider.PublicIPsCfg.IPBlockCfgs = []v1alpha1.IPsBlockConfig{{IPBlockID: testIPBlockID}}
				return np
			}(),
			wantErr: true,
		},
		{
			name: "Internal Client Error",
			nodepoolMocker: nodepoolMocker{
				setupNodePoolService: func(service *k8snodepool.MockClient) {
					service.EXPECT().
						GetK8sNodePool(context.Background(), testClusterID, testNodePoolID).
						Return(
							ionoscloud.KubernetesNodePool{},
							&ionoscloud.APIResponse{Response: nil},
							errors.New("some error in the client, no response"),
						)
				},
			},
			args: basicObserveNodePool.DeepCopy(),
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
			name: "Nodepool does not exist",
			nodepoolMocker: nodepoolMocker{
				setupNodePoolService: func(service *k8snodepool.MockClient) {
					service.EXPECT().
						GetK8sNodePool(context.Background(), testClusterID, testNodePoolID).
						Return(ionoscloud.KubernetesNodePool{}, &ionoscloud.APIResponse{
							Response: &http.Response{
								StatusCode: http.StatusNotFound,
							},
						}, errors.New("resource not found"))
				},
			},
			args: basicObserveNodePool.DeepCopy(),
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
			name: "Nodepool up to date",
			nodepoolMocker: nodepoolMocker{
				setupNodePoolService: func(service *k8snodepool.MockClient) {
					service.EXPECT().
						GetK8sNodePool(context.Background(), testClusterID, testNodePoolID).
						Return(ionoscloud.KubernetesNodePool{
							Properties: &ionoscloud.KubernetesNodePoolProperties{
								Name: ionoscloud.PtrString("node-pool-name"),
							},
						}, nil, nil)
				},
			},
			args: basicObserveNodePool.DeepCopy(),
			want: managed.ExternalObservation{
				ResourceExists:          true,
				ResourceUpToDate:        true,
				ResourceLateInitialized: false,
				ConnectionDetails:       managed.ConnectionDetails{},
				Diff:                    "",
			},
			wantErr: false,
		},
		{
			name: "Nodepool must be updated (NodeCount differs)",
			nodepoolMocker: nodepoolMocker{
				setupNodePoolService: func(service *k8snodepool.MockClient) {
					service.EXPECT().
						GetK8sNodePool(context.Background(), testClusterID, testNodePoolID).
						Return(ionoscloud.KubernetesNodePool{
							Properties: &ionoscloud.KubernetesNodePoolProperties{
								Name:      ionoscloud.PtrString("node-pool-name"),
								NodeCount: ionoscloud.PtrInt32(2),
							},
						}, nil, nil)
				},
			},
			args: basicObserveNodePool.DeepCopy(),
			want: managed.ExternalObservation{
				ResourceExists:          true,
				ResourceUpToDate:        false,
				ResourceLateInitialized: false,
				ConnectionDetails:       managed.ConnectionDetails{},
				Diff:                    "",
			},
			wantErr: false,
		},
		{
			name: "Nodepool must be updated (Labels differ)",
			nodepoolMocker: nodepoolMocker{
				setupNodePoolService: func(service *k8snodepool.MockClient) {
					service.EXPECT().
						GetK8sNodePool(context.Background(), testClusterID, testNodePoolID).
						Return(ionoscloud.KubernetesNodePool{
							Properties: &ionoscloud.KubernetesNodePoolProperties{
								Name: ionoscloud.PtrString("node-pool-name"),
								Labels: &map[string]string{
									"some": "old-label",
								},
							},
						}, nil, nil)
				},
			},
			args: func() *v1alpha1.NodePool {
				np := basicObserveNodePool.DeepCopy()
				np.Spec.ForProvider.Labels = map[string]string{
					"some": "new-label",
				}
				return np
			}(),
			want: managed.ExternalObservation{
				ResourceExists:          true,
				ResourceUpToDate:        false,
				ResourceLateInitialized: false,
				ConnectionDetails:       managed.ConnectionDetails{},
				Diff:                    "",
			},
			wantErr: false,
		},
		{
			name: "Nodepool must be updated (Annotations differ)",
			nodepoolMocker: nodepoolMocker{
				setupNodePoolService: func(service *k8snodepool.MockClient) {
					service.EXPECT().
						GetK8sNodePool(context.Background(), testClusterID, testNodePoolID).
						Return(ionoscloud.KubernetesNodePool{
							Properties: &ionoscloud.KubernetesNodePoolProperties{
								Name: ionoscloud.PtrString("node-pool-name"),
								Annotations: &map[string]string{
									"some": "old-annotation",
								},
							},
						}, nil, nil)
				},
			},
			args: func() *v1alpha1.NodePool {
				np := basicObserveNodePool.DeepCopy()
				np.Spec.ForProvider.Annotations = map[string]string{
					"some": "new-annotation",
				}
				return np
			}(),
			want: managed.ExternalObservation{
				ResourceExists:          true,
				ResourceUpToDate:        false,
				ResourceLateInitialized: false,
				ConnectionDetails:       managed.ConnectionDetails{},
				Diff:                    "",
			},
			wantErr: false,
		},
		{
			name: "API response is inconsistent",
			nodepoolMocker: nodepoolMocker{
				setupNodePoolService: func(service *k8snodepool.MockClient) {
					service.EXPECT().
						GetK8sNodePool(context.Background(), testClusterID, testNodePoolID).
						Return(ionoscloud.KubernetesNodePool{
							Properties: nil,
						}, nil, nil)
				},
			},
			args: basicObserveNodePool.DeepCopy(),
			want: managed.ExternalObservation{
				ResourceExists:          true,
				ResourceUpToDate:        false,
				ResourceLateInitialized: false,
				ConnectionDetails:       managed.ConnectionDetails{},
				Diff:                    "",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			c := tt.NewExternalNodepool(ctrl)
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

func TestExternalNodePoolDelete(t *testing.T) {

	basicDeleteNodepool := &v1alpha1.NodePool{
		Spec: v1alpha1.NodePoolSpec{
			ForProvider: v1alpha1.NodePoolParameters{
				Name: "node-pool-name",
				ClusterCfg: v1alpha1.ClusterConfig{
					ClusterID: testClusterID,
				},
			},
		},
		Status: v1alpha1.NodePoolStatus{AtProvider: v1alpha1.NodePoolObservation{NodePoolID: testNodePoolID}},
	}
	meta.SetExternalName(basicDeleteNodepool, testNodePoolID)

	tests := []struct {
		name string
		nodepoolMocker
		args    resource.Managed
		wantErr bool
	}{
		{
			name:           "Wrong Type, don't expect any calls",
			nodepoolMocker: nodepoolMocker{},
			args:           &v1alpha1.Cluster{},
			wantErr:        true,
		},
		{
			name: "Don't delete if cluster state is unknown due to network error",
			nodepoolMocker: nodepoolMocker{
				setupClusterService: networkErrorClusterService,
			},
			args:    basicDeleteNodepool.DeepCopy(),
			wantErr: true,
		},
		{
			name: "Don't delete if cluster doesn't exist",
			nodepoolMocker: nodepoolMocker{
				setupClusterService: clusterNotFoundService,
			},
			args:    basicDeleteNodepool.DeepCopy(),
			wantErr: true,
		},
		{
			name: "Dont' delete if Cluster is not active",
			nodepoolMocker: nodepoolMocker{
				setupClusterService: clusterNotActiveService,
			},
			args:    basicDeleteNodepool.DeepCopy(),
			wantErr: true,
		},
		{
			name: "Nodepool does not exist",
			nodepoolMocker: nodepoolMocker{
				setupNodePoolService: func(service *k8snodepool.MockClient) {
					service.EXPECT().
						DeleteK8sNodePool(context.Background(), testClusterID, testNodePoolID).
						Return(&ionoscloud.APIResponse{
							Response: &http.Response{
								StatusCode: http.StatusNotFound,
							},
						}, errors.New("not found"))
				},
				setupClusterService: clusterActiveService,
			},
			args:    basicDeleteNodepool,
			wantErr: false,
		},
		{
			name: "Nodepool delete",
			nodepoolMocker: nodepoolMocker{
				setupNodePoolService: func(service *k8snodepool.MockClient) {
					service.EXPECT().
						DeleteK8sNodePool(context.Background(), testClusterID, testNodePoolID).
						Return(nil, nil)
				},
				setupClusterService: clusterActiveService,
			},
			args:    basicDeleteNodepool,
			wantErr: false,
		},
		{
			name: "ionos API error",
			nodepoolMocker: nodepoolMocker{
				setupNodePoolService: func(service *k8snodepool.MockClient) {
					service.EXPECT().
						DeleteK8sNodePool(context.Background(), testClusterID, testNodePoolID).
						Return(nil, errors.New("API error"))
				},
				setupClusterService: clusterActiveService,
			},
			args:    basicDeleteNodepool,
			wantErr: true,
		},
		{
			name:           "externalControlPlaneClient name not yet known",
			nodepoolMocker: nodepoolMocker{},
			args: func() *v1alpha1.NodePool {
				np := basicDeleteNodepool.DeepCopy()
				np.Status.AtProvider.NodePoolID = ""
				meta.SetExternalName(np, "")
				return np
			}(),
			wantErr: false,
		},
		{
			name:           "already destroying",
			nodepoolMocker: nodepoolMocker{},
			args: func() *v1alpha1.NodePool {
				np := basicDeleteNodepool.DeepCopy()
				np.Status.AtProvider.State = k8s.DESTROYING
				return np
			}(),
			wantErr: false,
		},
		{
			name:           "already terminated",
			nodepoolMocker: nodepoolMocker{},
			args: func() *v1alpha1.NodePool {
				np := basicDeleteNodepool.DeepCopy()
				np.Status.AtProvider.State = k8s.TERMINATED
				return np
			}(),
			wantErr: false,
		},
		{
			name:           "still deploying",
			nodepoolMocker: nodepoolMocker{},
			args: func() *v1alpha1.NodePool {
				np := basicDeleteNodepool.DeepCopy()
				np.Status.AtProvider.State = k8s.DEPLOYING
				return np
			}(),
			wantErr: true,
		},
		{
			name: "Nodepool in FAILED state",
			nodepoolMocker: nodepoolMocker{
				setupNodePoolService: func(service *k8snodepool.MockClient) {
					service.EXPECT().
						DeleteK8sNodePool(context.Background(), testClusterID, testNodePoolID).
						Return(nil, nil)
				},
				setupClusterService: clusterActiveService,
			},
			args: func() *v1alpha1.NodePool {
				np := basicDeleteNodepool.DeepCopy()
				np.Status.AtProvider.State = k8s.FAILED
				return np
			}(),
			wantErr: false,
		},
		{
			name: "Nodepool in FAILED_TERMINATED state",
			nodepoolMocker: nodepoolMocker{
				setupNodePoolService: func(service *k8snodepool.MockClient) {
					service.EXPECT().
						DeleteK8sNodePool(context.Background(), testClusterID, testNodePoolID).
						Return(nil, nil)
				},
				setupClusterService: clusterActiveService,
			},
			args: func() *v1alpha1.NodePool {
				np := basicDeleteNodepool.DeepCopy()
				np.Status.AtProvider.State = k8s.FAILED_DESTROYING
				return np
			}(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			c := tt.NewExternalNodepool(ctrl)
			err := c.Delete(context.Background(), tt.args)
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestExternalNodePoolCreate(t *testing.T) {

	basicCreateNodepool := &v1alpha1.NodePool{
		Spec: v1alpha1.NodePoolSpec{
			ForProvider: v1alpha1.NodePoolParameters{
				DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: "12345"},
				ClusterCfg: v1alpha1.ClusterConfig{
					ClusterID: testClusterID,
				},
				Name:             "testNodePool",
				K8sVersion:       "v1.22.33",
				NodeCount:        2,
				CoresCount:       4,
				StorageSize:      15,
				StorageType:      "SSD",
				RAMSize:          5,
				CPUFamily:        "SUPER_FAST",
				AvailabilityZone: "AUTO",
			},
		},
	}

	tests := []struct {
		name string
		nodepoolMocker
		args             resource.Managed
		want             managed.ExternalCreation
		wantErr          bool
		wantExternalName string
		wantCondition    xpv1.Condition
	}{
		{
			name: "Wrong Type, don't expect any calls",
			nodepoolMocker: nodepoolMocker{
				setupNodePoolService: func(service *k8snodepool.MockClient) {},
			},
			args:    &v1alpha1.Cluster{},
			wantErr: true,
			wantCondition: xpv1.Condition{
				Type:   "Ready",
				Status: "Unknown",
			},
		},
		{
			name: "Dont' create if Cluster is not active",
			nodepoolMocker: nodepoolMocker{
				setupClusterService: clusterNotActiveService,
			},
			args:    basicCreateNodepool.DeepCopy(),
			wantErr: true,
			wantCondition: xpv1.Condition{
				Type:   "Ready",
				Status: "Unknown",
			},
		},
		{
			name: "Don't create if cluster state is unknown due to network error",
			nodepoolMocker: nodepoolMocker{
				setupClusterService: networkErrorClusterService,
			},
			args:    basicCreateNodepool.DeepCopy(),
			wantErr: true,
			wantCondition: xpv1.Condition{
				Type:   "Ready",
				Status: "Unknown",
			},
		},
		{
			name: "Don't create if cluster doesn't exist",
			nodepoolMocker: nodepoolMocker{
				setupClusterService: clusterNotFoundService,
			},
			args:    basicCreateNodepool.DeepCopy(),
			wantErr: true,
			wantCondition: xpv1.Condition{
				Type:   "Ready",
				Status: "Unknown",
			},
		},
		{
			name: "Don't create if nodepool is deploying",
			nodepoolMocker: nodepoolMocker{
				setupClusterService: clusterActiveService,
			},
			args: func() *v1alpha1.NodePool {
				np := basicCreateNodepool.DeepCopy()
				np.Status.AtProvider.State = k8s.DEPLOYING
				return np
			}(),
			wantErr: false,
			wantCondition: xpv1.Condition{
				Type:   "Ready",
				Status: "False",
			},
		},
		{
			name: "Don't create if can't find IPBlocks",
			nodepoolMocker: nodepoolMocker{
				setupClusterService: clusterActiveService,
				setupIPBlockService: networkErrorIPBlockService,
			},
			args: func() *v1alpha1.NodePool {
				np := basicCreateNodepool.DeepCopy()
				np.Spec.ForProvider.PublicIPsCfg.IPBlockCfgs = []v1alpha1.IPsBlockConfig{{IPBlockID: testIPBlockID}}
				return np
			}(),
			wantErr: true,
			wantCondition: xpv1.Condition{
				Type:   "Ready",
				Status: "False",
			},
		},
		{
			name: "Nodepool does not exist",
			nodepoolMocker: nodepoolMocker{
				setupNodePoolService: func(service *k8snodepool.MockClient) {
					expectedNodePool := ionoscloud.KubernetesNodePool{
						Properties: &ionoscloud.KubernetesNodePoolProperties{
							Name:             ionoscloud.PtrString("testNodePool"),
							DatacenterId:     ionoscloud.PtrString("12345"),
							NodeCount:        ionoscloud.PtrInt32(2),
							CpuFamily:        ionoscloud.PtrString("SUPER_FAST"),
							CoresCount:       ionoscloud.PtrInt32(4),
							RamSize:          ionoscloud.PtrInt32(5),
							AvailabilityZone: ionoscloud.PtrString("AUTO"),
							StorageType:      ionoscloud.PtrString("SSD"),
							StorageSize:      ionoscloud.PtrInt32(15),
							K8sVersion:       ionoscloud.PtrString("v1.22.33")},
					}
					expectedNodePoolForPost := ionoscloud.KubernetesNodePoolForPost{
						Properties: &ionoscloud.KubernetesNodePoolPropertiesForPost{
							AvailabilityZone: ionoscloud.PtrString("AUTO"),
							Annotations:      nil,
							CoresCount:       ionoscloud.PtrInt32(4),
							CpuFamily:        ionoscloud.PtrString("SUPER_FAST"),
							DatacenterId:     ionoscloud.PtrString("12345"),
							K8sVersion:       ionoscloud.PtrString("v1.22.33"),
							Lans:             &[]ionoscloud.KubernetesNodePoolLan{},
							Name:             ionoscloud.PtrString("testNodePool"),
							NodeCount:        ionoscloud.PtrInt32(2),
							RamSize:          ionoscloud.PtrInt32(5),
							StorageSize:      ionoscloud.PtrInt32(15),
							StorageType:      ionoscloud.PtrString("SSD"),
						},
					}
					returnedNodePool := expectedNodePool
					returnedNodePool.Id = ionoscloud.PtrString("1234")
					service.EXPECT().
						CreateK8sNodePool(
							context.Background(),
							testClusterID,
							gomock.GotFormatterAdapter(nodePoolGotFormatter{},
								matchesNodePoolPost(expectedNodePoolForPost)),
						).
						Return(returnedNodePool, nil, nil)
				},
				setupClusterService: clusterActiveService,
			},
			args: basicCreateNodepool.DeepCopy(),
			want: managed.ExternalCreation{
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr:          false,
			wantExternalName: "1234",
			wantCondition:    xpv1.Creating(),
		},
		{
			name: "API errors",
			nodepoolMocker: nodepoolMocker{
				setupNodePoolService: func(service *k8snodepool.MockClient) {
					service.EXPECT().
						CreateK8sNodePool(context.Background(),
							testClusterID,
							gomock.AssignableToTypeOf(ionoscloud.KubernetesNodePoolForPost{}),
						).
						Return(ionoscloud.KubernetesNodePool{}, nil, errors.New("failed to execute"))
				},
				setupClusterService: clusterActiveService,
			},
			args: basicCreateNodepool.DeepCopy(),
			want: managed.ExternalCreation{
				ConnectionDetails: managed.ConnectionDetails{},
			},
			wantErr: true,
			wantCondition: xpv1.Condition{
				Type:   "Ready",
				Status: "False",
			},
		},
		{
			name: "Nodepool with empty CPUFamily",
			nodepoolMocker: nodepoolMocker{
				setupNodePoolService: func(service *k8snodepool.MockClient) {
					expectedNodePool := ionoscloud.KubernetesNodePool{
						Properties: &ionoscloud.KubernetesNodePoolProperties{
							Name:             ionoscloud.PtrString("testNodePool"),
							DatacenterId:     ionoscloud.PtrString("12345"),
							NodeCount:        ionoscloud.PtrInt32(2),
							CpuFamily:        ionoscloud.PtrString("INTEL_XEON"),
							CoresCount:       ionoscloud.PtrInt32(4),
							RamSize:          ionoscloud.PtrInt32(5),
							AvailabilityZone: ionoscloud.PtrString("AUTO"),
							StorageType:      ionoscloud.PtrString("SSD"),
							StorageSize:      ionoscloud.PtrInt32(15),
							K8sVersion:       ionoscloud.PtrString("v1.22.33")},
					}
					expectedNodePoolForPost := ionoscloud.KubernetesNodePoolForPost{
						Properties: &ionoscloud.KubernetesNodePoolPropertiesForPost{
							Name:             ionoscloud.PtrString("testNodePool"),
							DatacenterId:     ionoscloud.PtrString("12345"),
							Lans:             &[]ionoscloud.KubernetesNodePoolLan{},
							NodeCount:        ionoscloud.PtrInt32(2),
							CpuFamily:        ionoscloud.PtrString("INTEL_XEON"),
							CoresCount:       ionoscloud.PtrInt32(4),
							RamSize:          ionoscloud.PtrInt32(5),
							AvailabilityZone: ionoscloud.PtrString("AUTO"),
							StorageType:      ionoscloud.PtrString("SSD"),
							StorageSize:      ionoscloud.PtrInt32(15),
							K8sVersion:       ionoscloud.PtrString("v1.22.33")},
					}
					returnedNodePool := expectedNodePool
					returnedNodePool.Id = ionoscloud.PtrString("1234")

					service.EXPECT().
						CreateK8sNodePool(
							context.Background(),
							testClusterID,
							gomock.GotFormatterAdapter(nodePoolGotFormatter{},
								matchesNodePoolPost(expectedNodePoolForPost)),
						).
						Return(returnedNodePool, nil, nil)
				},
				setupClusterService: clusterActiveService,
				setupDatacenterService: func(service *datacenter.MockClient) {
					service.EXPECT().GetCPUFamiliesForDatacenter(context.Background(), "12345").Return([]string{"INTEL_XEON", "AMD_OPTERON"}, nil)
				},
			},
			args: func() *v1alpha1.NodePool {
				np := basicCreateNodepool.DeepCopy()
				np.Spec.ForProvider.CPUFamily = ""
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
			name: "IONOS API doesn't returns error instead of CPUFamilies",
			nodepoolMocker: nodepoolMocker{
				setupClusterService: clusterActiveService,
				setupDatacenterService: func(service *datacenter.MockClient) {
					service.EXPECT().GetCPUFamiliesForDatacenter(context.Background(), "12345").Return(nil, errors.New("everything is broken"))
				},
			},
			args: func() *v1alpha1.NodePool {
				np := basicCreateNodepool.DeepCopy()
				np.Spec.ForProvider.CPUFamily = ""
				return np
			}(),
			want: managed.ExternalCreation{
				ConnectionDetails: nil,
			},
			wantErr:          true,
			wantExternalName: "",
			wantCondition: xpv1.Condition{
				Type:   "Ready",
				Status: "False",
			},
		},
		{
			name: "IONOS API returns empty list of CPU families",
			nodepoolMocker: nodepoolMocker{
				setupNodePoolService: func(service *k8snodepool.MockClient) {
					expectedNodePool := ionoscloud.KubernetesNodePool{
						Properties: &ionoscloud.KubernetesNodePoolProperties{
							Name:             ionoscloud.PtrString("testNodePool"),
							DatacenterId:     ionoscloud.PtrString("12345"),
							NodeCount:        ionoscloud.PtrInt32(2),
							CpuFamily:        ionoscloud.PtrString("INTEL_XEON"),
							CoresCount:       ionoscloud.PtrInt32(4),
							RamSize:          ionoscloud.PtrInt32(5),
							AvailabilityZone: ionoscloud.PtrString("AUTO"),
							StorageType:      ionoscloud.PtrString("SSD"),
							StorageSize:      ionoscloud.PtrInt32(15),
							K8sVersion:       ionoscloud.PtrString("v1.22.33")},
					}
					expectedNodePoolForPost := ionoscloud.KubernetesNodePoolForPost{
						Properties: &ionoscloud.KubernetesNodePoolPropertiesForPost{
							Name:             ionoscloud.PtrString("testNodePool"),
							DatacenterId:     ionoscloud.PtrString("12345"),
							Lans:             &[]ionoscloud.KubernetesNodePoolLan{},
							NodeCount:        ionoscloud.PtrInt32(2),
							CpuFamily:        ionoscloud.PtrString(""),
							CoresCount:       ionoscloud.PtrInt32(4),
							RamSize:          ionoscloud.PtrInt32(5),
							AvailabilityZone: ionoscloud.PtrString("AUTO"),
							StorageType:      ionoscloud.PtrString("SSD"),
							StorageSize:      ionoscloud.PtrInt32(15),
							K8sVersion:       ionoscloud.PtrString("v1.22.33")},
					}
					returnedNodePool := expectedNodePool
					returnedNodePool.Id = ionoscloud.PtrString("1234")

					service.EXPECT().
						CreateK8sNodePool(
							context.Background(),
							testClusterID,
							gomock.GotFormatterAdapter(nodePoolGotFormatter{},
								matchesNodePoolPost(expectedNodePoolForPost)),
						).
						Return(returnedNodePool, nil, nil)
				},
				setupClusterService: clusterActiveService,
				setupDatacenterService: func(service *datacenter.MockClient) {
					service.EXPECT().GetCPUFamiliesForDatacenter(context.Background(), "12345").Return([]string{}, nil)
				},
			},
			args: func() *v1alpha1.NodePool {
				np := &v1alpha1.NodePool{
					Spec: v1alpha1.NodePoolSpec{
						ForProvider: v1alpha1.NodePoolParameters{
							DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: "12345"},
							ClusterCfg: v1alpha1.ClusterConfig{
								ClusterID: testClusterID,
							},
							Name:             "testNodePool",
							K8sVersion:       "v1.22.33",
							NodeCount:        2,
							CoresCount:       4,
							StorageSize:      15,
							StorageType:      "SSD",
							RAMSize:          5,
							CPUFamily:        "",
							AvailabilityZone: "AUTO",
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			c := tt.NewExternalNodepool(ctrl)
			got, err := c.Create(context.Background(), tt.args)
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantExternalName, meta.GetExternalName(tt.args))
			if np, ok := tt.args.(*v1alpha1.NodePool); ok {
				assert.Equal(t, tt.wantExternalName, np.Status.AtProvider.NodePoolID)
			}
			assert.EqualValues(t, tt.wantCondition.Status, tt.args.GetCondition(xpv1.TypeReady).Status)
		})
	}
}

func TestExternalNodePoolUpdate(t *testing.T) {

	basicUpdateNodePool := &v1alpha1.NodePool{
		Spec: v1alpha1.NodePoolSpec{
			ForProvider: v1alpha1.NodePoolParameters{
				Name: "node-pool-name",
				ClusterCfg: v1alpha1.ClusterConfig{
					ClusterID: testClusterID,
				},
			},
		},
		Status: v1alpha1.NodePoolStatus{AtProvider: v1alpha1.NodePoolObservation{NodePoolID: testNodePoolID}},
	}
	meta.SetExternalName(basicUpdateNodePool, testNodePoolID)

	tests := []struct {
		name string
		nodepoolMocker
		args          resource.Managed
		want          managed.ExternalUpdate
		wantErr       bool
		wantCondition xpv1.Condition
	}{
		{
			name: "Wrong Type, don't expect any calls",
			nodepoolMocker: nodepoolMocker{
				setupNodePoolService: func(service *k8snodepool.MockClient) {},
			},
			args:    &v1alpha1.Cluster{},
			wantErr: true,
			wantCondition: xpv1.Condition{
				Type:   "Ready",
				Status: "Unknown",
			},
		},
		{
			name: "Cluster not found",
			nodepoolMocker: nodepoolMocker{
				setupClusterService: clusterNotFoundService,
			},
			args:    basicUpdateNodePool.DeepCopy(),
			wantErr: true,
			wantCondition: xpv1.Condition{
				Type:   "Ready",
				Status: "Unknown",
			},
		},
		{
			name: "Get cluster fails",
			nodepoolMocker: nodepoolMocker{
				setupClusterService: networkErrorClusterService,
			},
			args:    basicUpdateNodePool.DeepCopy(),
			wantErr: true,
			wantCondition: xpv1.Condition{
				Type:   "Ready",
				Status: "Unknown",
			},
		},
		{
			name: "Cluster not active",
			nodepoolMocker: nodepoolMocker{
				setupClusterService: clusterNotActiveService,
			},
			args:    basicUpdateNodePool.DeepCopy(),
			wantErr: true,
			wantCondition: xpv1.Condition{
				Type:   "Ready",
				Status: "Unknown",
			},
		},
		{
			name: "Already Updating",
			nodepoolMocker: nodepoolMocker{
				setupClusterService: clusterActiveService,
			},
			args: func() *v1alpha1.NodePool {
				np := basicUpdateNodePool.DeepCopy()
				np.Status.AtProvider.State = k8s.UPDATING
				return np
			}(),
			wantErr: false,
			wantCondition: xpv1.Condition{
				Type:   "Ready",
				Status: "Unknown",
			},
		},
		{
			name: "Can't find IPBlocks",
			nodepoolMocker: nodepoolMocker{
				setupClusterService: clusterActiveService,
				setupIPBlockService: networkErrorIPBlockService,
			},
			args: func() *v1alpha1.NodePool {
				np := basicUpdateNodePool.DeepCopy()
				np.Spec.ForProvider.PublicIPsCfg.IPBlockCfgs = []v1alpha1.IPsBlockConfig{{IPBlockID: testIPBlockID}}
				return np
			}(),
			wantErr: true,
			wantCondition: xpv1.Condition{
				Type:   "Ready",
				Status: "Unknown",
			},
		},
		{
			name: "Put NodePool Fails",
			nodepoolMocker: nodepoolMocker{
				setupNodePoolService: func(service *k8snodepool.MockClient) {
					service.EXPECT().
						UpdateK8sNodePool(context.Background(),
							testClusterID,
							testNodePoolID,
							gomock.AssignableToTypeOf(ionoscloud.KubernetesNodePoolForPut{}),
						).
						Return(ionoscloud.KubernetesNodePool{}, nil, errors.New("put failed"))
				},
				setupClusterService: clusterActiveService,
			},
			args:    basicUpdateNodePool.DeepCopy(),
			wantErr: true,
			wantCondition: xpv1.Condition{
				Type:   "Ready",
				Status: "Unknown",
			},
		},
		{
			name: "API success",
			nodepoolMocker: nodepoolMocker{
				setupNodePoolService: func(service *k8snodepool.MockClient) {
					service.EXPECT().UpdateK8sNodePool(context.Background(), testClusterID, testNodePoolID,
						gomock.GotFormatterAdapter(nodePoolGotFormatter{}, matchesNodePoolPut(ionoscloud.KubernetesNodePoolForPut{
							Properties: &ionoscloud.KubernetesNodePoolPropertiesForPut{
								NodeCount:  ionoscloud.PtrInt32(0),
								K8sVersion: ionoscloud.PtrString(""),
								MaintenanceWindow: &ionoscloud.KubernetesMaintenanceWindow{
									DayOfTheWeek: ionoscloud.PtrString("Mon"),
									Time:         ionoscloud.PtrString("15:24:30Z"),
								},
								AutoScaling: nil,
								Lans:        &[]ionoscloud.KubernetesNodePoolLan{},
								Labels:      &map[string]string{},
								Annotations: &map[string]string{},
								PublicIps:   &[]string{},
							},
						})),
					).
						Return(ionoscloud.KubernetesNodePool{}, nil, nil)
				},
				setupClusterService: clusterActiveService,
			},
			args: func() *v1alpha1.NodePool {
				np := &v1alpha1.NodePool{
					Spec: v1alpha1.NodePoolSpec{
						ForProvider: v1alpha1.NodePoolParameters{
							Name:          "nodepool",
							DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: "12345"},
							ClusterCfg: v1alpha1.ClusterConfig{
								ClusterID: testClusterID,
							},
							MaintenanceWindow: v1alpha1.MaintenanceWindow{
								Time:         "15:24:30Z",
								DayOfTheWeek: "Mon",
							},
						},
					},
					Status: v1alpha1.NodePoolStatus{AtProvider: v1alpha1.NodePoolObservation{NodePoolID: testNodePoolID}},
				}
				meta.SetExternalName(np, np.Status.AtProvider.NodePoolID)
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
			c := tt.NewExternalNodepool(ctrl)
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

type nodePoolGotFormatter struct {
}

func (n nodePoolGotFormatter) Got(got interface{}) string {
	return mustMarshal(got)
}

type nodePoolPostMatcher struct {
	expected ionoscloud.KubernetesNodePoolForPost
}

func matchesNodePoolPost(expected ionoscloud.KubernetesNodePoolForPost) nodePoolPostMatcher {
	return nodePoolPostMatcher{
		expected: expected,
	}
}

func (n nodePoolPostMatcher) Matches(x interface{}) bool {
	np, ok := x.(ionoscloud.KubernetesNodePoolForPost)
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

func (n nodePoolPostMatcher) String() string {
	return mustMarshal(n.expected)
}

type nodePoolPutMatcher struct {
	expected ionoscloud.KubernetesNodePoolForPut
}

func matchesNodePoolPut(expected ionoscloud.KubernetesNodePoolForPut) nodePoolPutMatcher {
	return nodePoolPutMatcher{
		expected: expected,
	}
}

func (n nodePoolPutMatcher) Matches(x interface{}) bool {
	np, ok := x.(ionoscloud.KubernetesNodePoolForPut)
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

func (n nodePoolPutMatcher) String() string {
	return mustMarshal(n.expected)
}

func mustMarshal(data interface{}) string {
	bytes, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}

type nodepoolMocker struct {
	setupNodePoolService   func(service *k8snodepool.MockClient)
	setupDatacenterService func(service *datacenter.MockClient)
	setupIPBlockService    func(service *ipblock.MockClient)
	setupClusterService    func(service *k8scluster.MockClient)
}

func (nm *nodepoolMocker) NewExternalNodepool(ctrl *gomock.Controller) *externalNodePool {
	nodepoolService := k8snodepool.NewMockClient(ctrl)
	if nm.setupNodePoolService != nil {
		nm.setupNodePoolService(nodepoolService)
	}

	clusterService := k8scluster.NewMockClient(ctrl)
	if nm.setupClusterService != nil {
		nm.setupClusterService(clusterService)
	}

	datacenterService := datacenter.NewMockClient(ctrl)
	if nm.setupDatacenterService != nil {
		nm.setupDatacenterService(datacenterService)
	}

	ipBlockService := ipblock.NewMockClient(ctrl)
	if nm.setupIPBlockService != nil {
		nm.setupIPBlockService(ipBlockService)
	}

	return &externalNodePool{
		service:           nodepoolService,
		clusterService:    clusterService,
		datacenterService: datacenterService,
		ipBlockService:    ipBlockService,
		log:               utils.NewTestLogger(),
	}
}

func clusterActiveService(service *k8scluster.MockClient) {
	service.EXPECT().GetK8sCluster(context.Background(), testClusterID).
		Return(
			ionoscloud.KubernetesCluster{Metadata: &ionoscloud.DatacenterElementMetadata{State: ionoscloud.PtrString(k8s.ACTIVE)}},
			nil,
			nil,
		)
}

func clusterNotActiveService(service *k8scluster.MockClient) {
	service.EXPECT().GetK8sCluster(context.Background(), testClusterID).
		Return(
			ionoscloud.KubernetesCluster{Metadata: &ionoscloud.DatacenterElementMetadata{State: ionoscloud.PtrString(k8s.DEPLOYING)}},
			nil,
			nil,
		)
}

func clusterNotFoundService(service *k8scluster.MockClient) {
	service.EXPECT().GetK8sCluster(context.Background(), testClusterID).
		Return(
			ionoscloud.KubernetesCluster{},
			&ionoscloud.APIResponse{Response: &http.Response{StatusCode: http.StatusNotFound}},
			errors.New("cluster not found"),
		)
}

func networkErrorClusterService(service *k8scluster.MockClient) {
	service.EXPECT().GetK8sCluster(context.Background(), testClusterID).
		Return(
			ionoscloud.KubernetesCluster{},
			nil,
			errors.New("cluster not found"),
		)
}

func networkErrorIPBlockService(service *ipblock.MockClient) {
	service.EXPECT().GetIPs(context.Background(), testIPBlockID).Return(nil, errors.New("no IPBlocks found"))
}
