// Code generated by MockGen. DO NOT EDIT.
// Source: ../../../../clients/k8s/k8scluster/cluster.go

// Package k8scluster is a generated GoMock package.
package k8scluster

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
)

// MockClient is a mock of Client interface.
type MockClient struct {
	ctrl     *gomock.Controller
	recorder *MockClientMockRecorder
}

// MockClientMockRecorder is the mock recorder for MockClient.
type MockClientMockRecorder struct {
	mock *MockClient
}

// NewMockClient creates a new mock instance.
func NewMockClient(ctrl *gomock.Controller) *MockClient {
	mock := &MockClient{ctrl: ctrl}
	mock.recorder = &MockClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockClient) EXPECT() *MockClientMockRecorder {
	return m.recorder
}

// CheckDuplicateK8sCluster mocks base method.
func (m *MockClient) CheckDuplicateK8sCluster(ctx context.Context, clusterName string) (*ionoscloud.KubernetesCluster, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckDuplicate", ctx, clusterName)
	ret0, _ := ret[0].(*ionoscloud.KubernetesCluster)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CheckDuplicateK8sCluster indicates an expected call of CheckDuplicateK8sCluster.
func (mr *MockClientMockRecorder) CheckDuplicateK8sCluster(ctx, clusterName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckDuplicate", reflect.TypeOf((*MockClient)(nil).CheckDuplicateK8sCluster), ctx, clusterName)
}

// CreateK8sCluster mocks base method.
func (m *MockClient) CreateK8sCluster(ctx context.Context, cluster ionoscloud.KubernetesClusterForPost) (ionoscloud.KubernetesCluster, *ionoscloud.APIResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateK8sCluster", ctx, cluster)
	ret0, _ := ret[0].(ionoscloud.KubernetesCluster)
	ret1, _ := ret[1].(*ionoscloud.APIResponse)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// CreateK8sCluster indicates an expected call of CreateK8sCluster.
func (mr *MockClientMockRecorder) CreateK8sCluster(ctx, cluster interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateK8sCluster", reflect.TypeOf((*MockClient)(nil).CreateK8sCluster), ctx, cluster)
}

// DeleteK8sCluster mocks base method.
func (m *MockClient) DeleteK8sCluster(ctx context.Context, clusterID string) (*ionoscloud.APIResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteDataPlatformCluster", ctx, clusterID)
	ret0, _ := ret[0].(*ionoscloud.APIResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteK8sCluster indicates an expected call of DeleteK8sCluster.
func (mr *MockClientMockRecorder) DeleteK8sCluster(ctx, clusterID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteDataPlatformCluster", reflect.TypeOf((*MockClient)(nil).DeleteK8sCluster), ctx, clusterID)
}

// GetAPIClient mocks base method.
func (m *MockClient) GetAPIClient() *ionoscloud.APIClient {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAPIClient")
	ret0, _ := ret[0].(*ionoscloud.APIClient)
	return ret0
}

// GetAPIClient indicates an expected call of GetAPIClient.
func (mr *MockClientMockRecorder) GetAPIClient() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAPIClient", reflect.TypeOf((*MockClient)(nil).GetAPIClient))
}

// GetK8sCluster mocks base method.
func (m *MockClient) GetK8sCluster(ctx context.Context, clusterID string) (ionoscloud.KubernetesCluster, *ionoscloud.APIResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDataplatformClusterByID", ctx, clusterID)
	ret0, _ := ret[0].(ionoscloud.KubernetesCluster)
	ret1, _ := ret[1].(*ionoscloud.APIResponse)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetK8sCluster indicates an expected call of GetK8sCluster.
func (mr *MockClientMockRecorder) GetK8sCluster(ctx, clusterID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDataplatformClusterByID", reflect.TypeOf((*MockClient)(nil).GetK8sCluster), ctx, clusterID)
}

// GetK8sClusterID mocks base method.
func (m *MockClient) GetK8sClusterID(cluster *ionoscloud.KubernetesCluster) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetK8sClusterID", cluster)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetK8sClusterID indicates an expected call of GetK8sClusterID.
func (mr *MockClientMockRecorder) GetK8sClusterID(cluster interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetK8sClusterID", reflect.TypeOf((*MockClient)(nil).GetK8sClusterID), cluster)
}

// GetKubeConfig mocks base method.
func (m *MockClient) GetKubeConfig(ctx context.Context, clusterID string) (string, *ionoscloud.APIResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetKubeConfig", ctx, clusterID)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(*ionoscloud.APIResponse)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetKubeConfig indicates an expected call of GetKubeConfig.
func (mr *MockClientMockRecorder) GetKubeConfig(ctx, clusterID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetKubeConfig", reflect.TypeOf((*MockClient)(nil).GetKubeConfig), ctx, clusterID)
}

// HasActiveK8sNodePools mocks base method.
func (m *MockClient) HasActiveK8sNodePools(ctx context.Context, clusterID string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HasActiveK8sNodePools", ctx, clusterID)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// HasActiveK8sNodePools indicates an expected call of HasActiveK8sNodePools.
func (mr *MockClientMockRecorder) HasActiveK8sNodePools(ctx, clusterID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HasActiveK8sNodePools", reflect.TypeOf((*MockClient)(nil).HasActiveK8sNodePools), ctx, clusterID)
}

// UpdateK8sCluster mocks base method.
func (m *MockClient) UpdateK8sCluster(ctx context.Context, clusterID string, cluster ionoscloud.KubernetesClusterForPut) (ionoscloud.KubernetesCluster, *ionoscloud.APIResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateK8sCluster", ctx, clusterID, cluster)
	ret0, _ := ret[0].(ionoscloud.KubernetesCluster)
	ret1, _ := ret[1].(*ionoscloud.APIResponse)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// UpdateK8sCluster indicates an expected call of UpdateK8sCluster.
func (mr *MockClientMockRecorder) UpdateK8sCluster(ctx, clusterID, cluster interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateK8sCluster", reflect.TypeOf((*MockClient)(nil).UpdateK8sCluster), ctx, clusterID, cluster)
}
