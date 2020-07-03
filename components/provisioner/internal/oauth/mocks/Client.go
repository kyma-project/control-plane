// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import (
	apperrors "github.com/kyma-incubator/compass/components/provisioner/internal/apperrors"
	mock "github.com/stretchr/testify/mock"

	oauth "github.com/kyma-incubator/compass/components/provisioner/internal/oauth"
)

// Client is an autogenerated mock type for the Client type
type Client struct {
	mock.Mock
}

// GetAuthorizationToken provides a mock function with given fields:
func (_m *Client) GetAuthorizationToken() (oauth.Token, apperrors.AppError) {
	ret := _m.Called()

	var r0 oauth.Token
	if rf, ok := ret.Get(0).(func() oauth.Token); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(oauth.Token)
	}

	var r1 apperrors.AppError
	if rf, ok := ret.Get(1).(func() apperrors.AppError); ok {
		r1 = rf()
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(apperrors.AppError)
		}
	}

	return r0, r1
}
