// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

// Package vm is a generated GoMock package.
package vm

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockWorldState is a mock of WorldState interface.
type MockWorldState struct {
	ctrl     *gomock.Controller
	recorder *MockWorldStateMockRecorder
}

// MockWorldStateMockRecorder is the mock recorder for MockWorldState.
type MockWorldStateMockRecorder struct {
	mock *MockWorldState
}

// NewMockWorldState creates a new mock instance.
func NewMockWorldState(ctrl *gomock.Controller) *MockWorldState {
	mock := &MockWorldState{ctrl: ctrl}
	mock.recorder = &MockWorldStateMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockWorldState) EXPECT() *MockWorldStateMockRecorder {
	return m.recorder
}

// AccountExists mocks base method.
func (m *MockWorldState) AccountExists(arg0 Address) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AccountExists", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// AccountExists indicates an expected call of AccountExists.
func (mr *MockWorldStateMockRecorder) AccountExists(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AccountExists", reflect.TypeOf((*MockWorldState)(nil).AccountExists), arg0)
}

// CreateAccount mocks base method.
func (m *MockWorldState) CreateAccount(arg0 Address, arg1 Code) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateAccount", arg0, arg1)
	ret0, _ := ret[0].(bool)
	return ret0
}

// CreateAccount indicates an expected call of CreateAccount.
func (mr *MockWorldStateMockRecorder) CreateAccount(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateAccount", reflect.TypeOf((*MockWorldState)(nil).CreateAccount), arg0, arg1)
}

// GetBalance mocks base method.
func (m *MockWorldState) GetBalance(arg0 Address) Value {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBalance", arg0)
	ret0, _ := ret[0].(Value)
	return ret0
}

// GetBalance indicates an expected call of GetBalance.
func (mr *MockWorldStateMockRecorder) GetBalance(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBalance", reflect.TypeOf((*MockWorldState)(nil).GetBalance), arg0)
}

// GetCode mocks base method.
func (m *MockWorldState) GetCode(arg0 Address) Code {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCode", arg0)
	ret0, _ := ret[0].(Code)
	return ret0
}

// GetCode indicates an expected call of GetCode.
func (mr *MockWorldStateMockRecorder) GetCode(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCode", reflect.TypeOf((*MockWorldState)(nil).GetCode), arg0)
}

// GetCodeHash mocks base method.
func (m *MockWorldState) GetCodeHash(arg0 Address) Hash {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCodeHash", arg0)
	ret0, _ := ret[0].(Hash)
	return ret0
}

// GetCodeHash indicates an expected call of GetCodeHash.
func (mr *MockWorldStateMockRecorder) GetCodeHash(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCodeHash", reflect.TypeOf((*MockWorldState)(nil).GetCodeHash), arg0)
}

// GetCodeSize mocks base method.
func (m *MockWorldState) GetCodeSize(arg0 Address) int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCodeSize", arg0)
	ret0, _ := ret[0].(int)
	return ret0
}

// GetCodeSize indicates an expected call of GetCodeSize.
func (mr *MockWorldStateMockRecorder) GetCodeSize(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCodeSize", reflect.TypeOf((*MockWorldState)(nil).GetCodeSize), arg0)
}

// GetNonce mocks base method.
func (m *MockWorldState) GetNonce(arg0 Address) uint64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNonce", arg0)
	ret0, _ := ret[0].(uint64)
	return ret0
}

// GetNonce indicates an expected call of GetNonce.
func (mr *MockWorldStateMockRecorder) GetNonce(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNonce", reflect.TypeOf((*MockWorldState)(nil).GetNonce), arg0)
}

// GetStorage mocks base method.
func (m *MockWorldState) GetStorage(arg0 Address, arg1 Key) Word {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStorage", arg0, arg1)
	ret0, _ := ret[0].(Word)
	return ret0
}

// GetStorage indicates an expected call of GetStorage.
func (mr *MockWorldStateMockRecorder) GetStorage(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStorage", reflect.TypeOf((*MockWorldState)(nil).GetStorage), arg0, arg1)
}

// SelfDestruct mocks base method.
func (m *MockWorldState) SelfDestruct(addr, beneficiary Address) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SelfDestruct", addr, beneficiary)
	ret0, _ := ret[0].(bool)
	return ret0
}

// SelfDestruct indicates an expected call of SelfDestruct.
func (mr *MockWorldStateMockRecorder) SelfDestruct(addr, beneficiary any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SelfDestruct", reflect.TypeOf((*MockWorldState)(nil).SelfDestruct), addr, beneficiary)
}

// SetBalance mocks base method.
func (m *MockWorldState) SetBalance(arg0 Address, arg1 Value) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetBalance", arg0, arg1)
}

// SetBalance indicates an expected call of SetBalance.
func (mr *MockWorldStateMockRecorder) SetBalance(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetBalance", reflect.TypeOf((*MockWorldState)(nil).SetBalance), arg0, arg1)
}

// SetCode mocks base method.
func (m *MockWorldState) SetCode(arg0 Address, arg1 Code) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetCode", arg0, arg1)
}

// SetCode indicates an expected call of SetCode.
func (mr *MockWorldStateMockRecorder) SetCode(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetCode", reflect.TypeOf((*MockWorldState)(nil).SetCode), arg0, arg1)
}

// SetNonce mocks base method.
func (m *MockWorldState) SetNonce(arg0 Address, arg1 uint64) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetNonce", arg0, arg1)
}

// SetNonce indicates an expected call of SetNonce.
func (mr *MockWorldStateMockRecorder) SetNonce(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetNonce", reflect.TypeOf((*MockWorldState)(nil).SetNonce), arg0, arg1)
}

// SetStorage mocks base method.
func (m *MockWorldState) SetStorage(arg0 Address, arg1 Key, arg2 Word) StorageStatus {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetStorage", arg0, arg1, arg2)
	ret0, _ := ret[0].(StorageStatus)
	return ret0
}

// SetStorage indicates an expected call of SetStorage.
func (mr *MockWorldStateMockRecorder) SetStorage(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetStorage", reflect.TypeOf((*MockWorldState)(nil).SetStorage), arg0, arg1, arg2)
}
