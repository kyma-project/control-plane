// Code generated by mockery (devel). DO NOT EDIT.

package automock

import (
	context "context"

	model "github.com/kyma-incubator/compass/components/director/internal/model"
	mock "github.com/stretchr/testify/mock"
)

// WebhookRepository is an autogenerated mock type for the WebhookRepository type
type WebhookRepository struct {
	mock.Mock
}

// CreateMany provides a mock function with given fields: ctx, items
func (_m *WebhookRepository) CreateMany(ctx context.Context, items []*model.Webhook) error {
	ret := _m.Called(ctx, items)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, []*model.Webhook) error); ok {
		r0 = rf(ctx, items)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
