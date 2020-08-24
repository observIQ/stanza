// Code generated by mockery v1.0.0. DO NOT EDIT.

package testutil

import (
	context "context"

	entry "github.com/observiq/stanza/entry"
	mock "github.com/stretchr/testify/mock"

	operator "github.com/observiq/stanza/operator"

	zap "go.uber.org/zap"
)

// Operator is an autogenerated mock type for the Operator type
type Operator struct {
	mock.Mock
}

// AddOutput provides a mock function with given fields: _a0
func (_m *Operator) AddOutput(_a0 operator.Operator) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(operator.Operator) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CanOutput provides a mock function with given fields:
func (_m *Operator) CanOutput() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// CanProcess provides a mock function with given fields:
func (_m *Operator) CanProcess() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// ID provides a mock function with given fields:
func (_m *Operator) ID() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// Logger provides a mock function with given fields:
func (_m *Operator) Logger() *zap.SugaredLogger {
	ret := _m.Called()

	var r0 *zap.SugaredLogger
	if rf, ok := ret.Get(0).(func() *zap.SugaredLogger); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*zap.SugaredLogger)
		}
	}

	return r0
}

// Outputs provides a mock function with given fields:
func (_m *Operator) Outputs() []operator.Operator {
	ret := _m.Called()

	var r0 []operator.Operator
	if rf, ok := ret.Get(0).(func() []operator.Operator); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]operator.Operator)
		}
	}

	return r0
}

// Process provides a mock function with given fields: _a0, _a1
func (_m *Operator) Process(_a0 context.Context, _a1 *entry.Entry) error {
	ret := _m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *entry.Entry) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetOutputs provides a mock function with given fields: _a0
func (_m *Operator) SetOutputs(_a0 []operator.Operator) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func([]operator.Operator) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Start provides a mock function with given fields:
func (_m *Operator) Start() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Stop provides a mock function with given fields:
func (_m *Operator) Stop() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Type provides a mock function with given fields:
func (_m *Operator) Type() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}
