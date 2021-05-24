// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import dbconnection "github.com/kyma-project/control-plane/components/provisioners-model-migrating-job/internal/persistence/dbconnection"
import dberrors "github.com/kyma-project/control-plane/components/provisioners-model-migrating-job/internal/persistence/dberrors"
import mock "github.com/stretchr/testify/mock"

// Factory is an autogenerated mock type for the Factory type
type Factory struct {
	mock.Mock
}

// NewReadWriteSession provides a mock function with given fields:
func (_m *Factory) NewReadWriteSession() dbconnection.ReadWriteSession {
	ret := _m.Called()

	var r0 dbconnection.ReadWriteSession
	if rf, ok := ret.Get(0).(func() dbconnection.ReadWriteSession); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(dbconnection.ReadWriteSession)
		}
	}

	return r0
}

// NewSessionWithinTransaction provides a mock function with given fields:
func (_m *Factory) NewSessionWithinTransaction() (dbconnection.WriteSessionWithinTransaction, dberrors.Error) {
	ret := _m.Called()

	var r0 dbconnection.WriteSessionWithinTransaction
	if rf, ok := ret.Get(0).(func() dbconnection.WriteSessionWithinTransaction); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(dbconnection.WriteSessionWithinTransaction)
		}
	}

	var r1 dberrors.Error
	if rf, ok := ret.Get(1).(func() dberrors.Error); ok {
		r1 = rf()
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(dberrors.Error)
		}
	}

	return r0, r1
}
