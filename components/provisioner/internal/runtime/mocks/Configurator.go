// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import (
	apperrors "github.com/kyma-incubator/compass/components/provisioner/internal/apperrors"
	mock "github.com/stretchr/testify/mock"

	model "github.com/kyma-incubator/compass/components/provisioner/internal/model"
)

// Configurator is an autogenerated mock type for the Configurator type
type Configurator struct {
	mock.Mock
}

// ConfigureRuntime provides a mock function with given fields: cluster, kubeconfigRaw
func (_m *Configurator) ConfigureRuntime(cluster model.Cluster, kubeconfigRaw string) apperrors.AppError {
	ret := _m.Called(cluster, kubeconfigRaw)

	var r0 apperrors.AppError
	if rf, ok := ret.Get(0).(func(model.Cluster, string) apperrors.AppError); ok {
		r0 = rf(cluster, kubeconfigRaw)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(apperrors.AppError)
		}
	}

	return r0
}
