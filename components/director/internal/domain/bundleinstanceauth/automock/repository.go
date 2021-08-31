// Code generated by mockery (devel). DO NOT EDIT.

package automock

import (
	context "context"

	model "github.com/kyma-incubator/compass/components/director/internal/model"
	mock "github.com/stretchr/testify/mock"
)

// Repository is an autogenerated mock type for the Repository type
type Repository struct {
	mock.Mock
}

// Create provides a mock function with given fields: ctx, item
func (_m *Repository) Create(ctx context.Context, item *model.BundleInstanceAuth) error {
	ret := _m.Called(ctx, item)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *model.BundleInstanceAuth) error); ok {
		r0 = rf(ctx, item)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Delete provides a mock function with given fields: ctx, tenantID, id
func (_m *Repository) Delete(ctx context.Context, tenantID string, id string) error {
	ret := _m.Called(ctx, tenantID, id)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) error); ok {
		r0 = rf(ctx, tenantID, id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetByID provides a mock function with given fields: ctx, tenantID, id
func (_m *Repository) GetByID(ctx context.Context, tenantID string, id string) (*model.BundleInstanceAuth, error) {
	ret := _m.Called(ctx, tenantID, id)

	var r0 *model.BundleInstanceAuth
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *model.BundleInstanceAuth); ok {
		r0 = rf(ctx, tenantID, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.BundleInstanceAuth)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, tenantID, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetForBundle provides a mock function with given fields: ctx, tenant, id, bundleID
func (_m *Repository) GetForBundle(ctx context.Context, tenant string, id string, bundleID string) (*model.BundleInstanceAuth, error) {
	ret := _m.Called(ctx, tenant, id, bundleID)

	var r0 *model.BundleInstanceAuth
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string) *model.BundleInstanceAuth); ok {
		r0 = rf(ctx, tenant, id, bundleID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.BundleInstanceAuth)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string, string) error); ok {
		r1 = rf(ctx, tenant, id, bundleID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListByBundleID provides a mock function with given fields: ctx, tenantID, bundleID
func (_m *Repository) ListByBundleID(ctx context.Context, tenantID string, bundleID string) ([]*model.BundleInstanceAuth, error) {
	ret := _m.Called(ctx, tenantID, bundleID)

	var r0 []*model.BundleInstanceAuth
	if rf, ok := ret.Get(0).(func(context.Context, string, string) []*model.BundleInstanceAuth); ok {
		r0 = rf(ctx, tenantID, bundleID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*model.BundleInstanceAuth)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, tenantID, bundleID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListByRuntimeID provides a mock function with given fields: ctx, tenantID, runtimeID
func (_m *Repository) ListByRuntimeID(ctx context.Context, tenantID string, runtimeID string) ([]*model.BundleInstanceAuth, error) {
	ret := _m.Called(ctx, tenantID, runtimeID)

	var r0 []*model.BundleInstanceAuth
	if rf, ok := ret.Get(0).(func(context.Context, string, string) []*model.BundleInstanceAuth); ok {
		r0 = rf(ctx, tenantID, runtimeID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*model.BundleInstanceAuth)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, tenantID, runtimeID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Update provides a mock function with given fields: ctx, item
func (_m *Repository) Update(ctx context.Context, item *model.BundleInstanceAuth) error {
	ret := _m.Called(ctx, item)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *model.BundleInstanceAuth) error); ok {
		r0 = rf(ctx, item)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
