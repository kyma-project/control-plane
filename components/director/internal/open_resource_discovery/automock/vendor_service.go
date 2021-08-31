// Code generated by mockery (devel). DO NOT EDIT.

package automock

import (
	context "context"

	model "github.com/kyma-incubator/compass/components/director/internal/model"
	mock "github.com/stretchr/testify/mock"
)

// VendorService is an autogenerated mock type for the VendorService type
type VendorService struct {
	mock.Mock
}

// Create provides a mock function with given fields: ctx, applicationID, in
func (_m *VendorService) Create(ctx context.Context, applicationID string, in model.VendorInput) (string, error) {
	ret := _m.Called(ctx, applicationID, in)

	var r0 string
	if rf, ok := ret.Get(0).(func(context.Context, string, model.VendorInput) string); ok {
		r0 = rf(ctx, applicationID, in)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, model.VendorInput) error); ok {
		r1 = rf(ctx, applicationID, in)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Delete provides a mock function with given fields: ctx, id
func (_m *VendorService) Delete(ctx context.Context, id string) error {
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
func (_m *VendorService) ListByApplicationID(ctx context.Context, appID string) ([]*model.Vendor, error) {
	ret := _m.Called(ctx, appID)

	var r0 []*model.Vendor
	if rf, ok := ret.Get(0).(func(context.Context, string) []*model.Vendor); ok {
		r0 = rf(ctx, appID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*model.Vendor)
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

// Update provides a mock function with given fields: ctx, id, in
func (_m *VendorService) Update(ctx context.Context, id string, in model.VendorInput) error {
	ret := _m.Called(ctx, id, in)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, model.VendorInput) error); ok {
		r0 = rf(ctx, id, in)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
