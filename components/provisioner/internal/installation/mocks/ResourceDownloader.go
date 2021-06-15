// Code generated by mockery v1.1.2. DO NOT EDIT.

package mocks

import (
	model "github.com/kyma-project/control-plane/components/provisioner/internal/model"
	mock "github.com/stretchr/testify/mock"
)

// ResourceDownloader is an autogenerated mock type for the ResourceDownloader type
type ResourceDownloader struct {
	mock.Mock
}

// Download provides a mock function with given fields: _a0, _a1
func (_m *ResourceDownloader) Download(_a0 string, _a1 []model.KymaComponentConfig) (string, string, error) {
	ret := _m.Called(_a0, _a1)

	var r0 string
	if rf, ok := ret.Get(0).(func(string, []model.KymaComponentConfig) string); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 string
	if rf, ok := ret.Get(1).(func(string, []model.KymaComponentConfig) string); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Get(1).(string)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(string, []model.KymaComponentConfig) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
