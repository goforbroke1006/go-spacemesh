// Code generated by MockGen. DO NOT EDIT.
// Source: ./interface.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	types "github.com/spacemeshos/go-spacemesh/common/types"
)

// MockconservativeState is a mock of conservativeState interface.
type MockconservativeState struct {
	ctrl     *gomock.Controller
	recorder *MockconservativeStateMockRecorder
}

// MockconservativeStateMockRecorder is the mock recorder for MockconservativeState.
type MockconservativeStateMockRecorder struct {
	mock *MockconservativeState
}

// NewMockconservativeState creates a new mock instance.
func NewMockconservativeState(ctrl *gomock.Controller) *MockconservativeState {
	mock := &MockconservativeState{ctrl: ctrl}
	mock.recorder = &MockconservativeStateMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockconservativeState) EXPECT() *MockconservativeStateMockRecorder {
	return m.recorder
}

// ApplyLayer mocks base method.
func (m *MockconservativeState) ApplyLayer(arg0 types.LayerID, arg1 types.BlockID, arg2 []types.TransactionID, arg3 map[types.Address]uint64) ([]*types.Transaction, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ApplyLayer", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].([]*types.Transaction)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ApplyLayer indicates an expected call of ApplyLayer.
func (mr *MockconservativeStateMockRecorder) ApplyLayer(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ApplyLayer", reflect.TypeOf((*MockconservativeState)(nil).ApplyLayer), arg0, arg1, arg2, arg3)
}

// GetStateRoot mocks base method.
func (m *MockconservativeState) GetStateRoot() types.Hash32 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStateRoot")
	ret0, _ := ret[0].(types.Hash32)
	return ret0
}

// GetStateRoot indicates an expected call of GetStateRoot.
func (mr *MockconservativeStateMockRecorder) GetStateRoot() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStateRoot", reflect.TypeOf((*MockconservativeState)(nil).GetStateRoot))
}

// ReinsertTxsToMemPool mocks base method.
func (m *MockconservativeState) ReinsertTxsToMemPool(arg0 []types.TransactionID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReinsertTxsToMemPool", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// ReinsertTxsToMemPool indicates an expected call of ReinsertTxsToMemPool.
func (mr *MockconservativeStateMockRecorder) ReinsertTxsToMemPool(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReinsertTxsToMemPool", reflect.TypeOf((*MockconservativeState)(nil).ReinsertTxsToMemPool), arg0)
}

// Rewind mocks base method.
func (m *MockconservativeState) Rewind(arg0 types.LayerID) (types.Hash32, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Rewind", arg0)
	ret0, _ := ret[0].(types.Hash32)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Rewind indicates an expected call of Rewind.
func (mr *MockconservativeStateMockRecorder) Rewind(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Rewind", reflect.TypeOf((*MockconservativeState)(nil).Rewind), arg0)
}

// StoreTransactionsFromMemPool mocks base method.
func (m *MockconservativeState) StoreTransactionsFromMemPool(arg0 types.LayerID, arg1 types.BlockID, arg2 []types.TransactionID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StoreTransactionsFromMemPool", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// StoreTransactionsFromMemPool indicates an expected call of StoreTransactionsFromMemPool.
func (mr *MockconservativeStateMockRecorder) StoreTransactionsFromMemPool(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StoreTransactionsFromMemPool", reflect.TypeOf((*MockconservativeState)(nil).StoreTransactionsFromMemPool), arg0, arg1, arg2)
}

// Mocktortoise is a mock of tortoise interface.
type Mocktortoise struct {
	ctrl     *gomock.Controller
	recorder *MocktortoiseMockRecorder
}

// MocktortoiseMockRecorder is the mock recorder for Mocktortoise.
type MocktortoiseMockRecorder struct {
	mock *Mocktortoise
}

// NewMocktortoise creates a new mock instance.
func NewMocktortoise(ctrl *gomock.Controller) *Mocktortoise {
	mock := &Mocktortoise{ctrl: ctrl}
	mock.recorder = &MocktortoiseMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *Mocktortoise) EXPECT() *MocktortoiseMockRecorder {
	return m.recorder
}

// HandleIncomingLayer mocks base method.
func (m *Mocktortoise) HandleIncomingLayer(arg0 context.Context, arg1 types.LayerID) (types.LayerID, types.LayerID, bool) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HandleIncomingLayer", arg0, arg1)
	ret0, _ := ret[0].(types.LayerID)
	ret1, _ := ret[1].(types.LayerID)
	ret2, _ := ret[2].(bool)
	return ret0, ret1, ret2
}

// HandleIncomingLayer indicates an expected call of HandleIncomingLayer.
func (mr *MocktortoiseMockRecorder) HandleIncomingLayer(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HandleIncomingLayer", reflect.TypeOf((*Mocktortoise)(nil).HandleIncomingLayer), arg0, arg1)
}

// OnBallot mocks base method.
func (m *Mocktortoise) OnBallot(arg0 *types.Ballot) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "OnBallot", arg0)
}

// OnBallot indicates an expected call of OnBallot.
func (mr *MocktortoiseMockRecorder) OnBallot(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OnBallot", reflect.TypeOf((*Mocktortoise)(nil).OnBallot), arg0)
}

// OnBlock mocks base method.
func (m *Mocktortoise) OnBlock(arg0 *types.Block) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "OnBlock", arg0)
}

// OnBlock indicates an expected call of OnBlock.
func (mr *MocktortoiseMockRecorder) OnBlock(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OnBlock", reflect.TypeOf((*Mocktortoise)(nil).OnBlock), arg0)
}
