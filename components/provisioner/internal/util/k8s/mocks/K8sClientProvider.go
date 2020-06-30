// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import apperrors "github.com/kyma-incubator/compass/components/provisioner/internal/apperrors"

import kubernetes "k8s.io/client-go/kubernetes"
import mock "github.com/stretchr/testify/mock"

// K8sClientProvider is an autogenerated mock type for the K8sClientProvider type
type K8sClientProvider struct {
	mock.Mock
}

// CreateK8SClient provides a mock function with given fields: kubeconfigRaw
func (_m *K8sClientProvider) CreateK8SClient(kubeconfigRaw string) (kubernetes.Interface, apperrors.AppError) {
	ret := _m.Called(kubeconfigRaw)

	var r0 kubernetes.Interface
	if rf, ok := ret.Get(0).(func(string) kubernetes.Interface); ok {
		r0 = rf(kubeconfigRaw)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(kubernetes.Interface)
		}
	}

	var r1 apperrors.AppError
	if rf, ok := ret.Get(1).(func(string) apperrors.AppError); ok {
		r1 = rf(kubeconfigRaw)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(apperrors.AppError)
		}
	}

	return r0, r1
}
