// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

// Code generated by MockGen. DO NOT EDIT.
// Source: specification.go
//
// Generated by this command:
//
//	mockgen -source specification.go -destination specification_mock.go -package spc
//

// Package spc is a generated GoMock package.
package spc

import (
	reflect "reflect"

	. "github.com/Fantom-foundation/Tosca/go/ct/rlz"
	st "github.com/Fantom-foundation/Tosca/go/ct/st"
	gomock "go.uber.org/mock/gomock"
)

// MockSpecification is a mock of Specification interface.
type MockSpecification struct {
	ctrl     *gomock.Controller
	recorder *MockSpecificationMockRecorder
}

// MockSpecificationMockRecorder is the mock recorder for MockSpecification.
type MockSpecificationMockRecorder struct {
	mock *MockSpecification
}

// NewMockSpecification creates a new mock instance.
func NewMockSpecification(ctrl *gomock.Controller) *MockSpecification {
	mock := &MockSpecification{ctrl: ctrl}
	mock.recorder = &MockSpecificationMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSpecification) EXPECT() *MockSpecificationMockRecorder {
	return m.recorder
}

// GetRules mocks base method.
func (m *MockSpecification) GetRules() []Rule {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRules")
	ret0, _ := ret[0].([]Rule)
	return ret0
}

// GetRules indicates an expected call of GetRules.
func (mr *MockSpecificationMockRecorder) GetRules() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRules", reflect.TypeOf((*MockSpecification)(nil).GetRules))
}

// GetRulesFor mocks base method.
func (m *MockSpecification) GetRulesFor(arg0 *st.State) []Rule {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRulesFor", arg0)
	ret0, _ := ret[0].([]Rule)
	return ret0
}

// GetRulesFor indicates an expected call of GetRulesFor.
func (mr *MockSpecificationMockRecorder) GetRulesFor(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRulesFor", reflect.TypeOf((*MockSpecification)(nil).GetRulesFor), arg0)
}
