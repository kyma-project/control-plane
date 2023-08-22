// Code generated by mockery v2.30.1. DO NOT EDIT.

package mocks

import (
	apperrors "github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	mock "github.com/stretchr/testify/mock"

	model "github.com/kyma-project/control-plane/components/provisioner/internal/model"
)

// Provisioner is an autogenerated mock type for the Provisioner type
type Provisioner struct {
	mock.Mock
}

// DeprovisionCluster provides a mock function with given fields: cluster, operationId
func (_m *Provisioner) DeprovisionCluster(cluster model.Cluster, operationId string) (model.Operation, apperrors.AppError) {
	ret := _m.Called(cluster, operationId)

	var r0 model.Operation
	var r1 apperrors.AppError
	if rf, ok := ret.Get(0).(func(model.Cluster, string) (model.Operation, apperrors.AppError)); ok {
		return rf(cluster, operationId)
	}
	if rf, ok := ret.Get(0).(func(model.Cluster, string) model.Operation); ok {
		r0 = rf(cluster, operationId)
	} else {
		r0 = ret.Get(0).(model.Operation)
	}

	if rf, ok := ret.Get(1).(func(model.Cluster, string) apperrors.AppError); ok {
		r1 = rf(cluster, operationId)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(apperrors.AppError)
		}
	}

	return r0, r1
}

// GetHibernationStatus provides a mock function with given fields: clusterID, gardenerConfig
func (_m *Provisioner) GetHibernationStatus(clusterID string, gardenerConfig model.GardenerConfig) (model.HibernationStatus, apperrors.AppError) {
	ret := _m.Called(clusterID, gardenerConfig)

	var r0 model.HibernationStatus
	var r1 apperrors.AppError
	if rf, ok := ret.Get(0).(func(string, model.GardenerConfig) (model.HibernationStatus, apperrors.AppError)); ok {
		return rf(clusterID, gardenerConfig)
	}
	if rf, ok := ret.Get(0).(func(string, model.GardenerConfig) model.HibernationStatus); ok {
		r0 = rf(clusterID, gardenerConfig)
	} else {
		r0 = ret.Get(0).(model.HibernationStatus)
	}

	if rf, ok := ret.Get(1).(func(string, model.GardenerConfig) apperrors.AppError); ok {
		r1 = rf(clusterID, gardenerConfig)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(apperrors.AppError)
		}
	}

	return r0, r1
}

// HibernateCluster provides a mock function with given fields: clusterID, upgradeConfig
func (_m *Provisioner) HibernateCluster(clusterID string, upgradeConfig model.GardenerConfig) apperrors.AppError {
	ret := _m.Called(clusterID, upgradeConfig)

	var r0 apperrors.AppError
	if rf, ok := ret.Get(0).(func(string, model.GardenerConfig) apperrors.AppError); ok {
		r0 = rf(clusterID, upgradeConfig)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(apperrors.AppError)
		}
	}

	return r0
}

// ProvisionCluster provides a mock function with given fields: cluster, operationId
func (_m *Provisioner) ProvisionCluster(cluster model.Cluster, operationId string) apperrors.AppError {
	ret := _m.Called(cluster, operationId)

	var r0 apperrors.AppError
	if rf, ok := ret.Get(0).(func(model.Cluster, string) apperrors.AppError); ok {
		r0 = rf(cluster, operationId)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(apperrors.AppError)
		}
	}

	return r0
}

// UpgradeCluster provides a mock function with given fields: clusterID, upgradeConfig
func (_m *Provisioner) UpgradeCluster(clusterID string, upgradeConfig model.GardenerConfig) apperrors.AppError {
	ret := _m.Called(clusterID, upgradeConfig)

	var r0 apperrors.AppError
	if rf, ok := ret.Get(0).(func(string, model.GardenerConfig) apperrors.AppError); ok {
		r0 = rf(clusterID, upgradeConfig)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(apperrors.AppError)
		}
	}

	return r0
}

// NewProvisioner creates a new instance of Provisioner. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewProvisioner(t interface {
	mock.TestingT
	Cleanup(func())
}) *Provisioner {
	mock := &Provisioner{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
