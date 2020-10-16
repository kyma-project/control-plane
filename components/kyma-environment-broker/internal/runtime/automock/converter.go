// Code generated by mockery v1.0.0. DO NOT EDIT.

package automock

import (
	internal "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"

	mock "github.com/stretchr/testify/mock"

	runtime "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
)

// Converter is an autogenerated mock type for the Converter type
type Converter struct {
	mock.Mock
}

// InstancesAndOperationsToDTO provides a mock function with given fields: _a0, _a1, _a2, _a3
func (_m *Converter) InstancesAndOperationsToDTO(_a0 internal.Instance, _a1 *internal.ProvisioningOperation, _a2 *internal.DeprovisioningOperation, _a3 *internal.UpgradeKymaOperation) (runtime.RuntimeDTO, error) {
	ret := _m.Called(_a0, _a1, _a2, _a3)

	var r0 runtime.RuntimeDTO
	if rf, ok := ret.Get(0).(func(internal.Instance, *internal.ProvisioningOperation, *internal.DeprovisioningOperation, *internal.UpgradeKymaOperation) runtime.RuntimeDTO); ok {
		r0 = rf(_a0, _a1, _a2, _a3)
	} else {
		r0 = ret.Get(0).(runtime.RuntimeDTO)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(internal.Instance, *internal.ProvisioningOperation, *internal.DeprovisioningOperation, *internal.UpgradeKymaOperation) error); ok {
		r1 = rf(_a0, _a1, _a2, _a3)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
