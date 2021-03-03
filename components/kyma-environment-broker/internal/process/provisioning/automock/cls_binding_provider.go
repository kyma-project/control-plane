// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package automock

import (
	cls "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	mock "github.com/stretchr/testify/mock"

	servicemanager "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
)

// ClsBindingProvider is an autogenerated mock type for the ClsBindingProvider type
type ClsBindingProvider struct {
	mock.Mock
}

// CreateBinding provides a mock function with given fields: smClient, request
func (_m *ClsBindingProvider) CreateBinding(smClient servicemanager.Client, request *cls.BindingRequest) (*cls.ClsOverrideParams, error) {
	ret := _m.Called(smClient, request)

	var r0 *cls.ClsOverrideParams
	if rf, ok := ret.Get(0).(func(servicemanager.Client, *cls.BindingRequest) *cls.ClsOverrideParams); ok {
		r0 = rf(smClient, request)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*cls.ClsOverrideParams)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(servicemanager.Client, *cls.BindingRequest) error); ok {
		r1 = rf(smClient, request)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
