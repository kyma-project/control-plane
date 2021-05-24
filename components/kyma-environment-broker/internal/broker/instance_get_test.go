package broker

import (
	"context"
	"net/http"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v7/domain/apiresponses"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestGetEndpoint_GetNonExistingInstance(t *testing.T) {
	// given
	st := storage.NewMemoryStorage()
	svc := NewGetInstance(st.Instances(), logrus.New())

	// when
	_, err := svc.GetInstance(context.Background(), instanceID)

	// then
	assert.IsType(t, err, &apiresponses.FailureResponse{}, "Get returned error of unexpected type")
	apierr := err.(*apiresponses.FailureResponse)
	assert.Equal(t, apierr.ValidatedStatusCode(nil), http.StatusNotFound, "Get status code not matching")
}
