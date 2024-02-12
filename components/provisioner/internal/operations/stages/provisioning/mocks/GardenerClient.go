// Code generated by mockery v2.35.2. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
)

// GardenerClient is an autogenerated mock type for the GardenerClient type
type GardenerClient struct {
	mock.Mock
}

// Get provides a mock function with given fields: ctx, name, options
func (_m *GardenerClient) Get(ctx context.Context, name string, options v1.GetOptions) (*v1beta1.Shoot, error) {
	ret := _m.Called(ctx, name, options)

	var r0 *v1beta1.Shoot
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, v1.GetOptions) (*v1beta1.Shoot, error)); ok {
		return rf(ctx, name, options)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, v1.GetOptions) *v1beta1.Shoot); ok {
		r0 = rf(ctx, name, options)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1beta1.Shoot)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, v1.GetOptions) error); ok {
		r1 = rf(ctx, name, options)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewGardenerClient creates a new instance of GardenerClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewGardenerClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *GardenerClient {
	mock := &GardenerClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
