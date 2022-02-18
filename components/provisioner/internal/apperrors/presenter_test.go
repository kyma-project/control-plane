package apperrors_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/persistence/dberrors"
	"github.com/stretchr/testify/require"

	"github.com/sirupsen/logrus/hooks/test"

	"github.com/stretchr/testify/assert"
)

func TestPresenter_ErrorPresenter(t *testing.T) {
	//given
	errMsg := "testErr"
	log, hook := test.NewNullLogger()
	presenter := apperrors.NewPresenter(log)

	t.Run("Unknown error", func(t *testing.T) {
		//when
		err := presenter.Do(context.TODO(), errors.New(errMsg))

		//then
		entry := hook.LastEntry()
		require.NotNil(t, entry)
		assert.Equal(t, fmt.Sprintf("Unknown error: %s\n", errMsg), entry.Message)
		assert.Contains(t, err.Error(), "testErr")
		assert.Equal(t, apperrors.ErrProvisioner, err.Extensions["error_component"])
		assert.Equal(t, apperrors.ErrProvisionerInternal, err.Extensions["error_reason"])
		hook.Reset()
	})

	t.Run("Internal Error", func(t *testing.T) {
		//given
		customErr := apperrors.Internal(errMsg)

		//when
		err := presenter.Do(context.TODO(), customErr)

		//then
		entry := hook.LastEntry()
		require.NotNil(t, entry)
		assert.Equal(t, fmt.Sprintf("Internal Server Error: %s", errMsg), entry.Message)
		assert.Equal(t, customErr.Code(), err.Extensions["error_code"])
		assert.Contains(t, err.Error(), "testErr")
		hook.Reset()
	})

	t.Run("DB Error", func(t *testing.T) {
		//given
		customErr := dberrors.NotFound("db error")

		//when
		err := presenter.Do(context.TODO(), customErr)

		//then
		entry := hook.LastEntry()
		assert.Nil(t, entry)
		assert.Equal(t, dberrors.CodeNotFound, err.Extensions["error_code"])
		assert.Equal(t, dberrors.ErrDBNotFound, err.Extensions["error_reason"])
		assert.Equal(t, apperrors.ErrDB, err.Extensions["error_component"])
		assert.Contains(t, err.Error(), "db error")
		hook.Reset()
	})
}
