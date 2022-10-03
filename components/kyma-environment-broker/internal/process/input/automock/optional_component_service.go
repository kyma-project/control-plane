// Code generated by mockery v2.14.0. DO NOT EDIT.

package automock

import (
	internal "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	mock "github.com/stretchr/testify/mock"

	runtime "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"
)

// OptionalComponentService is an autogenerated mock type for the OptionalComponentService type
type OptionalComponentService struct {
	mock.Mock
}

// AddComponentToDisable provides a mock function with given fields: name, disabler
func (_m *OptionalComponentService) AddComponentToDisable(name string, disabler runtime.ComponentDisabler) {
	_m.Called(name, disabler)
}

// ComputeComponentsToDisable provides a mock function with given fields: optComponentsToKeep
func (_m *OptionalComponentService) ComputeComponentsToDisable(optComponentsToKeep []string) []string {
	ret := _m.Called(optComponentsToKeep)

	var r0 []string
	if rf, ok := ret.Get(0).(func([]string) []string); ok {
		r0 = rf(optComponentsToKeep)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	return r0
}

// ExecuteDisablers provides a mock function with given fields: components, names
func (_m *OptionalComponentService) ExecuteDisablers(components internal.ComponentConfigurationInputList, names ...string) (internal.ComponentConfigurationInputList, error) {
	_va := make([]interface{}, len(names))
	for _i := range names {
		_va[_i] = names[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, components)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 internal.ComponentConfigurationInputList
	if rf, ok := ret.Get(0).(func(internal.ComponentConfigurationInputList, ...string) internal.ComponentConfigurationInputList); ok {
		r0 = rf(components, names...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(internal.ComponentConfigurationInputList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(internal.ComponentConfigurationInputList, ...string) error); ok {
		r1 = rf(components, names...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewOptionalComponentService interface {
	mock.TestingT
	Cleanup(func())
}

// NewOptionalComponentService creates a new instance of OptionalComponentService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewOptionalComponentService(t mockConstructorTestingTNewOptionalComponentService) *OptionalComponentService {
	mock := &OptionalComponentService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
