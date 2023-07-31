// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/user/casbin/rbac.go

// Package mock_casbin is a generated GoMock package.
package mock_casbin

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockEnforcer is a mock of Enforcer interface.
type MockEnforcer struct {
	ctrl     *gomock.Controller
	recorder *MockEnforcerMockRecorder
}

// MockEnforcerMockRecorder is the mock recorder for MockEnforcer.
type MockEnforcerMockRecorder struct {
	mock *MockEnforcer
}

// NewMockEnforcer creates a new mock instance.
func NewMockEnforcer(ctrl *gomock.Controller) *MockEnforcer {
	mock := &MockEnforcer{ctrl: ctrl}
	mock.recorder = &MockEnforcerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEnforcer) EXPECT() *MockEnforcerMockRecorder {
	return m.recorder
}

// Enforce mocks base method.
func (m *MockEnforcer) Enforce(emailId, resource, action, resourceItem string) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Enforce", emailId, resource, action, resourceItem)
	ret0, _ := ret[0].(bool)
	return ret0
}

// Enforce indicates an expected call of Enforce.
func (mr *MockEnforcerMockRecorder) Enforce(emailId, resource, action, resourceItem interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Enforce", reflect.TypeOf((*MockEnforcer)(nil).Enforce), emailId, resource, action, resourceItem)
}

// EnforceByEmail mocks base method.
func (m *MockEnforcer) EnforceByEmail(emailId, resource, action, resourceItem string) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnforceByEmail", emailId, resource, action, resourceItem)
	ret0, _ := ret[0].(bool)
	return ret0
}

// EnforceByEmail indicates an expected call of EnforceByEmail.
func (mr *MockEnforcerMockRecorder) EnforceByEmail(emailId, resource, action, resourceItem interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnforceByEmail", reflect.TypeOf((*MockEnforcer)(nil).EnforceByEmail), emailId, resource, action, resourceItem)
}

// EnforceByEmailInBatch mocks base method.
func (m *MockEnforcer) EnforceByEmailInBatch(emailId, resource, action string, vals []string) map[string]bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnforceByEmailInBatch", emailId, resource, action, vals)
	ret0, _ := ret[0].(map[string]bool)
	return ret0
}

// EnforceByEmailInBatch indicates an expected call of EnforceByEmailInBatch.
func (mr *MockEnforcerMockRecorder) EnforceByEmailInBatch(emailId, resource, action, vals interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnforceByEmailInBatch", reflect.TypeOf((*MockEnforcer)(nil).EnforceByEmailInBatch), emailId, resource, action, vals)
}

// EnforceErr mocks base method.
func (m *MockEnforcer) EnforceErr(emailId, resource, action, resourceItem string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnforceErr", emailId, resource, action, resourceItem)
	ret0, _ := ret[0].(error)
	return ret0
}

// EnforceErr indicates an expected call of EnforceErr.
func (mr *MockEnforcerMockRecorder) EnforceErr(emailId, resource, action, resourceItem interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnforceErr", reflect.TypeOf((*MockEnforcer)(nil).EnforceErr), emailId, resource, action, resourceItem)
}

// GetCacheDump mocks base method.
func (m *MockEnforcer) GetCacheDump() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCacheDump")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetCacheDump indicates an expected call of GetCacheDump.
func (mr *MockEnforcerMockRecorder) GetCacheDump() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCacheDump", reflect.TypeOf((*MockEnforcer)(nil).GetCacheDump))
}

// InvalidateCache mocks base method.
func (m *MockEnforcer) InvalidateCache(emailId string) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InvalidateCache", emailId)
	ret0, _ := ret[0].(bool)
	return ret0
}

// InvalidateCache indicates an expected call of InvalidateCache.
func (mr *MockEnforcerMockRecorder) InvalidateCache(emailId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InvalidateCache", reflect.TypeOf((*MockEnforcer)(nil).InvalidateCache), emailId)
}

// InvalidateCompleteCache mocks base method.
func (m *MockEnforcer) InvalidateCompleteCache() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "InvalidateCompleteCache")
}

// InvalidateCompleteCache indicates an expected call of InvalidateCompleteCache.
func (mr *MockEnforcerMockRecorder) InvalidateCompleteCache() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InvalidateCompleteCache", reflect.TypeOf((*MockEnforcer)(nil).InvalidateCompleteCache))
}

// ReloadPolicy mocks base method.
func (m *MockEnforcer) ReloadPolicy() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReloadPolicy")
	ret0, _ := ret[0].(error)
	return ret0
}

// ReloadPolicy indicates an expected call of ReloadPolicy.
func (mr *MockEnforcerMockRecorder) ReloadPolicy() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReloadPolicy", reflect.TypeOf((*MockEnforcer)(nil).ReloadPolicy))
}
