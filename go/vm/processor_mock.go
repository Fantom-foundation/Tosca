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

// MockProcessor is a mock of Processor interface.
type MockProcessor struct {
	ctrl     *gomock.Controller
	recorder *MockProcessorMockRecorder
}

// MockProcessorMockRecorder is the mock recorder for MockProcessor.
type MockProcessorMockRecorder struct {
	mock *MockProcessor
}

// NewMockProcessor creates a new mock instance.
func NewMockProcessor(ctrl *gomock.Controller) *MockProcessor {
	mock := &MockProcessor{ctrl: ctrl}
	mock.recorder = &MockProcessorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockProcessor) EXPECT() *MockProcessorMockRecorder {
	return m.recorder
}

// Run mocks base method.
func (m *MockProcessor) Run(arg0 BlockParameters, arg1 Transaction, arg2 TransactionContext) (Receipt, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Run", arg0, arg1, arg2)
	ret0, _ := ret[0].(Receipt)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Run indicates an expected call of Run.
func (mr *MockProcessorMockRecorder) Run(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Run", reflect.TypeOf((*MockProcessor)(nil).Run), arg0, arg1, arg2)
}
