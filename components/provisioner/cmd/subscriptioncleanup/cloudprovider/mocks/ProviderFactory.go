// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	cloudprovider "github.com/kyma-project/control-plane/components/provisioner/cmd/subscriptioncleanup/cloudprovider"
	mock "github.com/stretchr/testify/mock"

	model "github.com/kyma-project/control-plane/components/provisioner/cmd/subscriptioncleanup/model"
)

// ProviderFactory is an autogenerated mock type for the ProviderFactory type
type ProviderFactory struct {
	mock.Mock
}

// New provides a mock function with given fields: hyperscalerType, secretData
func (_m *ProviderFactory) New(hyperscalerType model.HyperscalerType, secretData map[string][]byte) (cloudprovider.ResourceCleaner, error) {
	ret := _m.Called(hyperscalerType, secretData)

	var r0 cloudprovider.ResourceCleaner
	if rf, ok := ret.Get(0).(func(model.HyperscalerType, map[string][]byte) cloudprovider.ResourceCleaner); ok {
		r0 = rf(hyperscalerType, secretData)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(cloudprovider.ResourceCleaner)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(model.HyperscalerType, map[string][]byte) error); ok {
		r1 = rf(hyperscalerType, secretData)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewProviderFactory interface {
	mock.TestingT
	Cleanup(func())
}

// NewProviderFactory creates a new instance of ProviderFactory. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewProviderFactory(t mockConstructorTestingTNewProviderFactory) *ProviderFactory {
	mock := &ProviderFactory{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
