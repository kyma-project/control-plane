// Code generated by mockery v2.3.0. DO NOT EDIT.

package automock

import (
	internal "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	mock "github.com/stretchr/testify/mock"
)

// CreatorForPlan is an autogenerated mock type for the CreatorForPlan type
type CreatorForPlan struct {
	mock.Mock
}

// CreateProvisionInput provides a mock function with given fields: parameters, version
func (_m *CreatorForPlan) CreateProvisionInput(parameters internal.ProvisioningParameters, version internal.RuntimeVersionData) (internal.ProvisionerInputCreator, error) {
	ret := _m.Called(parameters, version)

	var r0 internal.ProvisionerInputCreator
	if rf, ok := ret.Get(0).(func(internal.ProvisioningParameters, internal.RuntimeVersionData) internal.ProvisionerInputCreator); ok {
		r0 = rf(parameters, version)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(internal.ProvisionerInputCreator)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(internal.ProvisioningParameters, internal.RuntimeVersionData) error); ok {
		r1 = rf(parameters, version)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateUpgradeInput provides a mock function with given fields: parameters, version
func (_m *CreatorForPlan) CreateUpgradeInput(parameters internal.ProvisioningParameters, version internal.RuntimeVersionData) (internal.ProvisionerInputCreator, error) {
	ret := _m.Called(parameters, version)

	var r0 internal.ProvisionerInputCreator
	if rf, ok := ret.Get(0).(func(internal.ProvisioningParameters, internal.RuntimeVersionData) internal.ProvisionerInputCreator); ok {
		r0 = rf(parameters, version)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(internal.ProvisionerInputCreator)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(internal.ProvisioningParameters, internal.RuntimeVersionData) error); ok {
		r1 = rf(parameters, version)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateUpgradeShootInput provides a mock function with given fields: parameters
func (_m *CreatorForPlan) CreateUpgradeShootInput(parameters internal.ProvisioningParameters) (internal.ProvisionerInputCreator, error) {
	ret := _m.Called(parameters)

	var r0 internal.ProvisionerInputCreator
	if rf, ok := ret.Get(0).(func(internal.ProvisioningParameters) internal.ProvisionerInputCreator); ok {
		r0 = rf(parameters)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(internal.ProvisionerInputCreator)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(internal.ProvisioningParameters) error); ok {
		r1 = rf(parameters)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// IsPlanSupport provides a mock function with given fields: planID
func (_m *CreatorForPlan) IsPlanSupport(planID string) bool {
	ret := _m.Called(planID)

	var r0 bool
	if rf, ok := ret.Get(0).(func(string) bool); ok {
		r0 = rf(planID)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}
