// Code generated by mockery (devel). DO NOT EDIT.

package automock

import (
	context "context"

	model "github.com/kyma-incubator/compass/components/director/internal/model"
	mock "github.com/stretchr/testify/mock"
)

// PackageService is an autogenerated mock type for the PackageService type
type PackageService struct {
	mock.Mock
}

// Create provides a mock function with given fields: ctx, applicationID, in, pkgHash
func (_m *PackageService) Create(ctx context.Context, applicationID string, in model.PackageInput, pkgHash uint64) (string, error) {
	ret := _m.Called(ctx, applicationID, in, pkgHash)

	var r0 string
	if rf, ok := ret.Get(0).(func(context.Context, string, model.PackageInput, uint64) string); ok {
		r0 = rf(ctx, applicationID, in, pkgHash)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, model.PackageInput, uint64) error); ok {
		r1 = rf(ctx, applicationID, in, pkgHash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Delete provides a mock function with given fields: ctx, id
func (_m *PackageService) Delete(ctx context.Context, id string) error {
	ret := _m.Called(ctx, id)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ListByApplicationID provides a mock function with given fields: ctx, appID
func (_m *PackageService) ListByApplicationID(ctx context.Context, appID string) ([]*model.Package, error) {
	ret := _m.Called(ctx, appID)

	var r0 []*model.Package
	if rf, ok := ret.Get(0).(func(context.Context, string) []*model.Package); ok {
		r0 = rf(ctx, appID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*model.Package)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, appID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Update provides a mock function with given fields: ctx, id, in, pkgHash
func (_m *PackageService) Update(ctx context.Context, id string, in model.PackageInput, pkgHash uint64) error {
	ret := _m.Called(ctx, id, in, pkgHash)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, model.PackageInput, uint64) error); ok {
		r0 = rf(ctx, id, in, pkgHash)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
