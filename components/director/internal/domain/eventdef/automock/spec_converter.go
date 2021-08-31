// Code generated by mockery (devel). DO NOT EDIT.

package automock

import (
	graphql "github.com/kyma-incubator/compass/components/director/pkg/graphql"
	mock "github.com/stretchr/testify/mock"

	model "github.com/kyma-incubator/compass/components/director/internal/model"
)

// SpecConverter is an autogenerated mock type for the SpecConverter type
type SpecConverter struct {
	mock.Mock
}

// InputFromGraphQLEventSpec provides a mock function with given fields: in
func (_m *SpecConverter) InputFromGraphQLEventSpec(in *graphql.EventSpecInput) (*model.SpecInput, error) {
	ret := _m.Called(in)

	var r0 *model.SpecInput
	if rf, ok := ret.Get(0).(func(*graphql.EventSpecInput) *model.SpecInput); ok {
		r0 = rf(in)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.SpecInput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*graphql.EventSpecInput) error); ok {
		r1 = rf(in)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ToGraphQLEventSpec provides a mock function with given fields: in
func (_m *SpecConverter) ToGraphQLEventSpec(in *model.Spec) (*graphql.EventSpec, error) {
	ret := _m.Called(in)

	var r0 *graphql.EventSpec
	if rf, ok := ret.Get(0).(func(*model.Spec) *graphql.EventSpec); ok {
		r0 = rf(in)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*graphql.EventSpec)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*model.Spec) error); ok {
		r1 = rf(in)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
