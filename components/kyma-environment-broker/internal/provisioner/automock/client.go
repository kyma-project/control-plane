// Code generated by mockery v2.14.0. DO NOT EDIT.

package automock

import (
	gqlschema "github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	mock "github.com/stretchr/testify/mock"
)

// Client is an autogenerated mock type for the Client type
type Client struct {
	mock.Mock
}

// DeprovisionRuntime provides a mock function with given fields: accountID, runtimeID
func (_m *Client) DeprovisionRuntime(accountID string, runtimeID string) (string, error) {
	ret := _m.Called(accountID, runtimeID)

	var r0 string
	if rf, ok := ret.Get(0).(func(string, string) string); ok {
		r0 = rf(accountID, runtimeID)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(accountID, runtimeID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ProvisionRuntime provides a mock function with given fields: accountID, subAccountID, config
func (_m *Client) ProvisionRuntime(accountID string, subAccountID string, config gqlschema.ProvisionRuntimeInput) (gqlschema.OperationStatus, error) {
	ret := _m.Called(accountID, subAccountID, config)

	var r0 gqlschema.OperationStatus
	if rf, ok := ret.Get(0).(func(string, string, gqlschema.ProvisionRuntimeInput) gqlschema.OperationStatus); ok {
		r0 = rf(accountID, subAccountID, config)
	} else {
		r0 = ret.Get(0).(gqlschema.OperationStatus)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string, gqlschema.ProvisionRuntimeInput) error); ok {
		r1 = rf(accountID, subAccountID, config)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ReconnectRuntimeAgent provides a mock function with given fields: accountID, runtimeID
func (_m *Client) ReconnectRuntimeAgent(accountID string, runtimeID string) (string, error) {
	ret := _m.Called(accountID, runtimeID)

	var r0 string
	if rf, ok := ret.Get(0).(func(string, string) string); ok {
		r0 = rf(accountID, runtimeID)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(accountID, runtimeID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RuntimeOperationStatus provides a mock function with given fields: accountID, operationID
func (_m *Client) RuntimeOperationStatus(accountID string, operationID string) (gqlschema.OperationStatus, error) {
	ret := _m.Called(accountID, operationID)

	var r0 gqlschema.OperationStatus
	if rf, ok := ret.Get(0).(func(string, string) gqlschema.OperationStatus); ok {
		r0 = rf(accountID, operationID)
	} else {
		r0 = ret.Get(0).(gqlschema.OperationStatus)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(accountID, operationID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RuntimeStatus provides a mock function with given fields: accountID, runtimeID
func (_m *Client) RuntimeStatus(accountID string, runtimeID string) (gqlschema.RuntimeStatus, error) {
	ret := _m.Called(accountID, runtimeID)

	var r0 gqlschema.RuntimeStatus
	if rf, ok := ret.Get(0).(func(string, string) gqlschema.RuntimeStatus); ok {
		r0 = rf(accountID, runtimeID)
	} else {
		r0 = ret.Get(0).(gqlschema.RuntimeStatus)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(accountID, runtimeID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpgradeRuntime provides a mock function with given fields: accountID, runtimeID, config
func (_m *Client) UpgradeRuntime(accountID string, runtimeID string, config gqlschema.UpgradeRuntimeInput) (gqlschema.OperationStatus, error) {
	ret := _m.Called(accountID, runtimeID, config)

	var r0 gqlschema.OperationStatus
	if rf, ok := ret.Get(0).(func(string, string, gqlschema.UpgradeRuntimeInput) gqlschema.OperationStatus); ok {
		r0 = rf(accountID, runtimeID, config)
	} else {
		r0 = ret.Get(0).(gqlschema.OperationStatus)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string, gqlschema.UpgradeRuntimeInput) error); ok {
		r1 = rf(accountID, runtimeID, config)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpgradeShoot provides a mock function with given fields: accountID, runtimeID, config
func (_m *Client) UpgradeShoot(accountID string, runtimeID string, config gqlschema.UpgradeShootInput) (gqlschema.OperationStatus, error) {
	ret := _m.Called(accountID, runtimeID, config)

	var r0 gqlschema.OperationStatus
	if rf, ok := ret.Get(0).(func(string, string, gqlschema.UpgradeShootInput) gqlschema.OperationStatus); ok {
		r0 = rf(accountID, runtimeID, config)
	} else {
		r0 = ret.Get(0).(gqlschema.OperationStatus)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string, gqlschema.UpgradeShootInput) error); ok {
		r1 = rf(accountID, runtimeID, config)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewClient interface {
	mock.TestingT
	Cleanup(func())
}

// NewClient creates a new instance of Client. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewClient(t mockConstructorTestingTNewClient) *Client {
	mock := &Client{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
