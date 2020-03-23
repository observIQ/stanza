// Code generated by mockery v1.0.0. DO NOT EDIT.

package testutil

import (
	entry "github.com/bluemedora/bplogagent/entry"
	mock "github.com/stretchr/testify/mock"

	plugin "github.com/bluemedora/bplogagent/plugin"
)

// Inputter is an autogenerated mock type for the Inputter type
type Inputter struct {
	mock.Mock
}

// ID provides a mock function with given fields:
func (_m *Inputter) ID() plugin.PluginID {
	ret := _m.Called()

	var r0 plugin.PluginID
	if rf, ok := ret.Get(0).(func() plugin.PluginID); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(plugin.PluginID)
	}

	return r0
}

// Input provides a mock function with given fields: _a0
func (_m *Inputter) Input(_a0 *entry.Entry) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(*entry.Entry) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Start provides a mock function with given fields:
func (_m *Inputter) Start() error {
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
func (_m *Inputter) Stop() error {
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
func (_m *Inputter) Type() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}
