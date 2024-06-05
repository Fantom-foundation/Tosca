// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

// Package vm_test is a generated GoMock package.
package vm_test

import (
	reflect "reflect"

	vm "github.com/Fantom-foundation/Tosca/go/vm"
	gomock "go.uber.org/mock/gomock"
)

// MockStateDB is a mock of StateDB interface.
type MockStateDB struct {
	ctrl     *gomock.Controller
	recorder *MockStateDBMockRecorder
}

// MockStateDBMockRecorder is the mock recorder for MockStateDB.
type MockStateDBMockRecorder struct {
	mock *MockStateDB
}

// NewMockStateDB creates a new mock instance.
func NewMockStateDB(ctrl *gomock.Controller) *MockStateDB {
	mock := &MockStateDB{ctrl: ctrl}
	mock.recorder = &MockStateDBMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStateDB) EXPECT() *MockStateDBMockRecorder {
	return m.recorder
}

// AccessAccount mocks base method.
func (m *MockStateDB) AccessAccount(arg0 vm.Address) vm.AccessStatus {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AccessAccount", arg0)
	ret0, _ := ret[0].(vm.AccessStatus)
	return ret0
}

// AccessAccount indicates an expected call of AccessAccount.
func (mr *MockStateDBMockRecorder) AccessAccount(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AccessAccount", reflect.TypeOf((*MockStateDB)(nil).AccessAccount), arg0)
}

// AccessStorage mocks base method.
func (m *MockStateDB) AccessStorage(arg0 vm.Address, arg1 vm.Key) vm.AccessStatus {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AccessStorage", arg0, arg1)
	ret0, _ := ret[0].(vm.AccessStatus)
	return ret0
}

// AccessStorage indicates an expected call of AccessStorage.
func (mr *MockStateDBMockRecorder) AccessStorage(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AccessStorage", reflect.TypeOf((*MockStateDB)(nil).AccessStorage), arg0, arg1)
}

// AccountExists mocks base method.
func (m *MockStateDB) AccountExists(arg0 vm.Address) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AccountExists", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// AccountExists indicates an expected call of AccountExists.
func (mr *MockStateDBMockRecorder) AccountExists(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AccountExists", reflect.TypeOf((*MockStateDB)(nil).AccountExists), arg0)
}

// EmitLog mocks base method.
func (m *MockStateDB) EmitLog(arg0 vm.Log) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "EmitLog", arg0)
}

// EmitLog indicates an expected call of EmitLog.
func (mr *MockStateDBMockRecorder) EmitLog(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EmitLog", reflect.TypeOf((*MockStateDB)(nil).EmitLog), arg0)
}

// GetBalance mocks base method.
func (m *MockStateDB) GetBalance(arg0 vm.Address) vm.Value {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBalance", arg0)
	ret0, _ := ret[0].(vm.Value)
	return ret0
}

// GetBalance indicates an expected call of GetBalance.
func (mr *MockStateDBMockRecorder) GetBalance(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBalance", reflect.TypeOf((*MockStateDB)(nil).GetBalance), arg0)
}

// GetBlockHash mocks base method.
func (m *MockStateDB) GetBlockHash(arg0 int64) vm.Hash {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBlockHash", arg0)
	ret0, _ := ret[0].(vm.Hash)
	return ret0
}

// GetBlockHash indicates an expected call of GetBlockHash.
func (mr *MockStateDBMockRecorder) GetBlockHash(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBlockHash", reflect.TypeOf((*MockStateDB)(nil).GetBlockHash), arg0)
}

// GetCode mocks base method.
func (m *MockStateDB) GetCode(arg0 vm.Address) vm.Code {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCode", arg0)
	ret0, _ := ret[0].(vm.Code)
	return ret0
}

// GetCode indicates an expected call of GetCode.
func (mr *MockStateDBMockRecorder) GetCode(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCode", reflect.TypeOf((*MockStateDB)(nil).GetCode), arg0)
}

// GetCodeHash mocks base method.
func (m *MockStateDB) GetCodeHash(arg0 vm.Address) vm.Hash {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCodeHash", arg0)
	ret0, _ := ret[0].(vm.Hash)
	return ret0
}

// GetCodeHash indicates an expected call of GetCodeHash.
func (mr *MockStateDBMockRecorder) GetCodeHash(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCodeHash", reflect.TypeOf((*MockStateDB)(nil).GetCodeHash), arg0)
}

// GetCodeSize mocks base method.
func (m *MockStateDB) GetCodeSize(arg0 vm.Address) int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCodeSize", arg0)
	ret0, _ := ret[0].(int)
	return ret0
}

// GetCodeSize indicates an expected call of GetCodeSize.
func (mr *MockStateDBMockRecorder) GetCodeSize(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCodeSize", reflect.TypeOf((*MockStateDB)(nil).GetCodeSize), arg0)
}

// GetCommittedStorage mocks base method.
func (m *MockStateDB) GetCommittedStorage(arg0 vm.Address, arg1 vm.Key) vm.Word {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCommittedStorage", arg0, arg1)
	ret0, _ := ret[0].(vm.Word)
	return ret0
}

// GetCommittedStorage indicates an expected call of GetCommittedStorage.
func (mr *MockStateDBMockRecorder) GetCommittedStorage(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCommittedStorage", reflect.TypeOf((*MockStateDB)(nil).GetCommittedStorage), arg0, arg1)
}

// GetNonce mocks base method.
func (m *MockStateDB) GetNonce(arg0 vm.Address) uint64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNonce", arg0)
	ret0, _ := ret[0].(uint64)
	return ret0
}

// GetNonce indicates an expected call of GetNonce.
func (mr *MockStateDBMockRecorder) GetNonce(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNonce", reflect.TypeOf((*MockStateDB)(nil).GetNonce), arg0)
}

// GetStorage mocks base method.
func (m *MockStateDB) GetStorage(arg0 vm.Address, arg1 vm.Key) vm.Word {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStorage", arg0, arg1)
	ret0, _ := ret[0].(vm.Word)
	return ret0
}

// GetStorage indicates an expected call of GetStorage.
func (mr *MockStateDBMockRecorder) GetStorage(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStorage", reflect.TypeOf((*MockStateDB)(nil).GetStorage), arg0, arg1)
}

// GetTransientStorage mocks base method.
func (m *MockStateDB) GetTransientStorage(arg0 vm.Address, arg1 vm.Key) vm.Word {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTransientStorage", arg0, arg1)
	ret0, _ := ret[0].(vm.Word)
	return ret0
}

// GetTransientStorage indicates an expected call of GetTransientStorage.
func (mr *MockStateDBMockRecorder) GetTransientStorage(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTransientStorage", reflect.TypeOf((*MockStateDB)(nil).GetTransientStorage), arg0, arg1)
}

// HasSelfDestructed mocks base method.
func (m *MockStateDB) HasSelfDestructed(arg0 vm.Address) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HasSelfDestructed", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// HasSelfDestructed indicates an expected call of HasSelfDestructed.
func (mr *MockStateDBMockRecorder) HasSelfDestructed(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HasSelfDestructed", reflect.TypeOf((*MockStateDB)(nil).HasSelfDestructed), arg0)
}

// IsAddressInAccessList mocks base method.
func (m *MockStateDB) IsAddressInAccessList(arg0 vm.Address) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsAddressInAccessList", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsAddressInAccessList indicates an expected call of IsAddressInAccessList.
func (mr *MockStateDBMockRecorder) IsAddressInAccessList(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsAddressInAccessList", reflect.TypeOf((*MockStateDB)(nil).IsAddressInAccessList), arg0)
}

// IsSlotInAccessList mocks base method.
func (m *MockStateDB) IsSlotInAccessList(arg0 vm.Address, arg1 vm.Key) (bool, bool) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsSlotInAccessList", arg0, arg1)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(bool)
	return ret0, ret1
}

// IsSlotInAccessList indicates an expected call of IsSlotInAccessList.
func (mr *MockStateDBMockRecorder) IsSlotInAccessList(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsSlotInAccessList", reflect.TypeOf((*MockStateDB)(nil).IsSlotInAccessList), arg0, arg1)
}

// SetBalance mocks base method.
func (m *MockStateDB) SetBalance(arg0 vm.Address, arg1 vm.Value) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetBalance", arg0, arg1)
}

// SetBalance indicates an expected call of SetBalance.
func (mr *MockStateDBMockRecorder) SetBalance(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetBalance", reflect.TypeOf((*MockStateDB)(nil).SetBalance), arg0, arg1)
}

// SetCode mocks base method.
func (m *MockStateDB) SetCode(arg0 vm.Address, arg1 vm.Code) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetCode", arg0, arg1)
}

// SetCode indicates an expected call of SetCode.
func (mr *MockStateDBMockRecorder) SetCode(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetCode", reflect.TypeOf((*MockStateDB)(nil).SetCode), arg0, arg1)
}

// SetNonce mocks base method.
func (m *MockStateDB) SetNonce(arg0 vm.Address, arg1 uint64) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetNonce", arg0, arg1)
}

// SetNonce indicates an expected call of SetNonce.
func (mr *MockStateDBMockRecorder) SetNonce(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetNonce", reflect.TypeOf((*MockStateDB)(nil).SetNonce), arg0, arg1)
}

// SetStorage mocks base method.
func (m *MockStateDB) SetStorage(arg0 vm.Address, arg1 vm.Key, arg2 vm.Word) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetStorage", arg0, arg1, arg2)
}

// SetStorage indicates an expected call of SetStorage.
func (mr *MockStateDBMockRecorder) SetStorage(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetStorage", reflect.TypeOf((*MockStateDB)(nil).SetStorage), arg0, arg1, arg2)
}

// SetTransientStorage mocks base method.
func (m *MockStateDB) SetTransientStorage(arg0 vm.Address, arg1 vm.Key, arg2 vm.Word) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetTransientStorage", arg0, arg1, arg2)
}

// SetTransientStorage indicates an expected call of SetTransientStorage.
func (mr *MockStateDBMockRecorder) SetTransientStorage(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetTransientStorage", reflect.TypeOf((*MockStateDB)(nil).SetTransientStorage), arg0, arg1, arg2)
}
