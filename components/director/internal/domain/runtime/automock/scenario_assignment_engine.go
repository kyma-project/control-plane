// Code generated by mockery v1.0.0. DO NOT EDIT.

package automock

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// ScenarioAssignmentEngine is an autogenerated mock type for the ScenarioAssignmentEngine type
type ScenarioAssignmentEngine struct {
	mock.Mock
}

// GetScenariosForSelectorLabels provides a mock function with given fields: ctx, inputLabels
func (_m *ScenarioAssignmentEngine) GetScenariosForSelectorLabels(ctx context.Context, inputLabels map[string]string) ([]string, error) {
	ret := _m.Called(ctx, inputLabels)

	var r0 []string
	if rf, ok := ret.Get(0).(func(context.Context, map[string]string) []string); ok {
		r0 = rf(ctx, inputLabels)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, map[string]string) error); ok {
		r1 = rf(ctx, inputLabels)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MergeScenarios provides a mock function with given fields: baseScenarios, scenariosToDelete, scenariosToAdd
func (_m *ScenarioAssignmentEngine) MergeScenarios(baseScenarios []interface{}, scenariosToDelete []interface{}, scenariosToAdd []interface{}) []interface{} {
	ret := _m.Called(baseScenarios, scenariosToDelete, scenariosToAdd)

	var r0 []interface{}
	if rf, ok := ret.Get(0).(func([]interface{}, []interface{}, []interface{}) []interface{}); ok {
		r0 = rf(baseScenarios, scenariosToDelete, scenariosToAdd)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]interface{})
		}
	}

	return r0
}

// MergeScenariosFromInputLabelsAndAssignments provides a mock function with given fields: ctx, inputLabels
func (_m *ScenarioAssignmentEngine) MergeScenariosFromInputLabelsAndAssignments(ctx context.Context, inputLabels map[string]interface{}) ([]interface{}, error) {
	ret := _m.Called(ctx, inputLabels)

	var r0 []interface{}
	if rf, ok := ret.Get(0).(func(context.Context, map[string]interface{}) []interface{}); ok {
		r0 = rf(ctx, inputLabels)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]interface{})
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, map[string]interface{}) error); ok {
		r1 = rf(ctx, inputLabels)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
