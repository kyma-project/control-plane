// Code generated by mockery v1.1.2. DO NOT EDIT.

package mocks

import (
	apperrors "github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	gqlschema "github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

	mock "github.com/stretchr/testify/mock"
)

// ResourceDownloader is an autogenerated mock type for the ResourceDownloader type
type ResourceDownloader struct {
	mock.Mock
}

// Download provides a mock function with given fields: _a0, _a1
func (_m *ResourceDownloader) Download(_a0 string, _a1 []*gqlschema.ComponentConfigurationInput) apperrors.AppError {
	ret := _m.Called(_a0, _a1)

	var r0 apperrors.AppError
	if rf, ok := ret.Get(0).(func(string, []*gqlschema.ComponentConfigurationInput) apperrors.AppError); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(apperrors.AppError)
		}
	}

	return r0
}
