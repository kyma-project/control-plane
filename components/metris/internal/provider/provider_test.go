package provider

import (
	"context"
	"testing"

	"github.com/kyma-project/control-plane/components/metris/internal/log"
	"github.com/stretchr/testify/assert"
)

type fakeTestProvider struct{}

func NewTestProvider(config *Config) Provider {
	return &fakeTestProvider{}
}

func (a *fakeTestProvider) Run(ctx context.Context) {}

func TestNewProvider(t *testing.T) {
	asserts := assert.New(t)

	t.Run("get unregistered provider", func(t *testing.T) {
		_, err := NewProvider("test", &Config{Logger: log.NewNoopLogger()})
		asserts.Error(err, "should return an error")
	})

	t.Run("register provider", func(t *testing.T) {
		err := RegisterProvider("test", NewTestProvider)
		asserts.NoError(err, "should not return an error")
	})

	t.Run("register provider twice", func(t *testing.T) {
		err := RegisterProvider("test", NewTestProvider)
		asserts.Error(err, "should return an error")
	})

	t.Run("get registered provider", func(t *testing.T) {
		p, err := NewProvider("test", &Config{Logger: log.NewNoopLogger()})
		asserts.NoError(err, "should not return an error")
		asserts.Implements((*Provider)(nil), p, "should implement Provider interface")
	})
}
