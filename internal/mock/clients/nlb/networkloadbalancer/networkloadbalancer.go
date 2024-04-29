// Code generated by MockGen. DO NOT EDIT.
// Source: ../../clients/nlb/networkloadbalancer/networkloadbalancer.go

// Package networkloadbalancer is a generated GoMock package.
package networkloadbalancer

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

// CheckDuplicateNetworkLoadBalancer mocks base method.
func (m *MockClient) CheckDuplicateNetworkLoadBalancer(ctx context.Context, datacenterID, nlbName string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckDuplicateNetworkLoadBalancer", ctx, datacenterID, nlbName)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CheckDuplicateNetworkLoadBalancer indicates an expected call of CheckDuplicateNetworkLoadBalancer.
func (mr *MockClientMockRecorder) CheckDuplicateNetworkLoadBalancer(ctx, datacenterID, nlbName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckDuplicateNetworkLoadBalancer", reflect.TypeOf((*MockClient)(nil).CheckDuplicateNetworkLoadBalancer), ctx, datacenterID, nlbName)
}

// CreateNetworkLoadBalancer mocks base method.
func (m *MockClient) CreateNetworkLoadBalancer(ctx context.Context, datacenterID string, nlb ionoscloud.NetworkLoadBalancer) (ionoscloud.NetworkLoadBalancer, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateNetworkLoadBalancer", ctx, datacenterID, nlb)
	ret0, _ := ret[0].(ionoscloud.NetworkLoadBalancer)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateNetworkLoadBalancer indicates an expected call of CreateNetworkLoadBalancer.
func (mr *MockClientMockRecorder) CreateNetworkLoadBalancer(ctx, datacenterID, nlb interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateNetworkLoadBalancer", reflect.TypeOf((*MockClient)(nil).CreateNetworkLoadBalancer), ctx, datacenterID, nlb)
}

// DeleteNetworkLoadBalancer mocks base method.
func (m *MockClient) DeleteNetworkLoadBalancer(ctx context.Context, datacenterID, nlbID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteNetworkLoadBalancer", ctx, datacenterID, nlbID)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteNetworkLoadBalancer indicates an expected call of DeleteNetworkLoadBalancer.
func (mr *MockClientMockRecorder) DeleteNetworkLoadBalancer(ctx, datacenterID, nlbID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteNetworkLoadBalancer", reflect.TypeOf((*MockClient)(nil).DeleteNetworkLoadBalancer), ctx, datacenterID, nlbID)
}

// GetNetworkLoadBalancerByID mocks base method.
func (m *MockClient) GetNetworkLoadBalancerByID(ctx context.Context, datacenterID, NetworkLoadBalancerID string) (ionoscloud.NetworkLoadBalancer, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNetworkLoadBalancerByID", ctx, datacenterID, NetworkLoadBalancerID)
	ret0, _ := ret[0].(ionoscloud.NetworkLoadBalancer)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetNetworkLoadBalancerByID indicates an expected call of GetNetworkLoadBalancerByID.
func (mr *MockClientMockRecorder) GetNetworkLoadBalancerByID(ctx, datacenterID, NetworkLoadBalancerID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNetworkLoadBalancerByID", reflect.TypeOf((*MockClient)(nil).GetNetworkLoadBalancerByID), ctx, datacenterID, NetworkLoadBalancerID)
}

// UpdateNetworkLoadBalancer mocks base method.
func (m *MockClient) UpdateNetworkLoadBalancer(ctx context.Context, datacenterID, nlbID string, nlbProperties ionoscloud.NetworkLoadBalancerProperties) (ionoscloud.NetworkLoadBalancer, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateNetworkLoadBalancer", ctx, datacenterID, nlbID, nlbProperties)
	ret0, _ := ret[0].(ionoscloud.NetworkLoadBalancer)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateNetworkLoadBalancer indicates an expected call of UpdateNetworkLoadBalancer.
func (mr *MockClientMockRecorder) UpdateNetworkLoadBalancer(ctx, datacenterID, nlbID, nlbProperties interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateNetworkLoadBalancer", reflect.TypeOf((*MockClient)(nil).UpdateNetworkLoadBalancer), ctx, datacenterID, nlbID, nlbProperties)
}