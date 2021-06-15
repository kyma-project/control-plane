// Code generated by mockery v1.1.2. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// OverrideBuilder is an autogenerated mock type for the OverrideBuilder type
type OverrideBuilder struct {
	mock.Mock
}

// AddOverrides provides a mock function with given fields: _a0, _a1
func (_m *OverrideBuilder) AddOverrides(_a0 string, _a1 map[string]interface{}) error {
	ret := _m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, map[string]interface{}) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
