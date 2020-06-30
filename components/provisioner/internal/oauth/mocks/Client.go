// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import (
	oauth "github.com/kyma-project/control-plane/components/provisioner/internal/oauth"
	mock "github.com/stretchr/testify/mock"
)

// Client is an autogenerated mock type for the Client type
type Client struct {
	mock.Mock
}

// GetAuthorizationToken provides a mock function with given fields:
func (_m *Client) GetAuthorizationToken() (oauth.Token, error) {
	ret := _m.Called()

	var r0 oauth.Token
	if rf, ok := ret.Get(0).(func() oauth.Token); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(oauth.Token)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
