package kubeconfig

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/kubeconfig/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/gorilla/mux"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/require"
)

const (
	instanceID        = "93241a34-8ab5-4f10-978e-eaa6f8ad551c"
	operationID       = "306f2406-e972-4fae-8edd-50fc50e56817"
	instanceRuntimeID = "e04813ba-244a-4150-8670-506c37959388"
)

func TestHandler_GetKubeconfig(t *testing.T) {
	cases := map[string]struct {
		pass                 bool
		instanceID           string
		runtimeID            string
		operationStatus      domain.LastOperationState
		expectedStatusCode   int
		expectedErrorMessage string
	}{
		"new kubeconfig was returned": {
			pass:               true,
			instanceID:         instanceID,
			runtimeID:          instanceRuntimeID,
			expectedStatusCode: http.StatusOK,
		},
		"instance ID is empty": {
			pass:                 false,
			instanceID:           "",
			expectedStatusCode:   http.StatusNotFound,
			expectedErrorMessage: "instanceID is required",
		},
		"runtimeID not exist": {
			pass:                 false,
			instanceID:           instanceID,
			runtimeID:            "",
			expectedStatusCode:   http.StatusNotFound,
			expectedErrorMessage: fmt.Sprintf("kubeconfig for instance %s does not exist. Provisioning could be in progress, please try again later", instanceID),
		},
		"provisioning operation is not ready": {
			pass:                 false,
			instanceID:           instanceID,
			runtimeID:            instanceRuntimeID,
			operationStatus:      domain.InProgress,
			expectedStatusCode:   http.StatusNotFound,
			expectedErrorMessage: fmt.Sprintf("provisioning operation for instance %s is in progress state, kubeconfig not exist yet, please try again later", instanceID),
		},
		"unsuspension operation is not ready": {
			pass:                 false,
			instanceID:           instanceID,
			runtimeID:            instanceRuntimeID,
			operationStatus:      orchestration.Pending,
			expectedStatusCode:   http.StatusNotFound,
			expectedErrorMessage: fmt.Sprintf("provisioning operation for instance %s is in progress state, kubeconfig not exist yet, please try again later", instanceID),
		},
		"provisioning operation failed": {
			pass:                 false,
			instanceID:           instanceID,
			runtimeID:            instanceRuntimeID,
			operationStatus:      domain.Failed,
			expectedStatusCode:   http.StatusNotFound,
			expectedErrorMessage: fmt.Sprintf("provisioning operation for instance %s failed, kubeconfig does not exist", instanceID),
		},
		"kubeconfig builder failed": {
			pass:                 false,
			instanceID:           instanceID,
			runtimeID:            instanceRuntimeID,
			expectedStatusCode:   http.StatusInternalServerError,
			expectedErrorMessage: "cannot fetch SKR kubeconfig: builder error",
		},
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			// given
			instance := internal.Instance{
				InstanceID: d.instanceID,
				RuntimeID:  d.runtimeID,
			}

			operation := internal.ProvisioningOperation{
				Operation: internal.Operation{
					ID:         operationID,
					InstanceID: instance.InstanceID,
					State:      d.operationStatus,
				},
			}

			db := storage.NewMemoryStorage()
			err := db.Instances().Insert(instance)
			require.NoError(t, err)
			err = db.Operations().InsertProvisioningOperation(operation)
			require.NoError(t, err)

			builder := &automock.KcBuilder{}
			if d.pass {
				builder.On("Build", &instance).Return("--kubeconfig file", nil)
				defer builder.AssertExpectations(t)
			} else {
				builder.On("Build", &instance).Return("", fmt.Errorf("builder error"))
			}

			router := mux.NewRouter()

			handler := NewHandler(db, builder, logger.NewLogDummy())
			handler.AttachRoutes(router)

			server := httptest.NewServer(router)

			// when
			response, err := http.Get(fmt.Sprintf("%s/kubeconfig/%s", server.URL, d.instanceID))
			require.NoError(t, err)

			// then
			require.Equal(t, d.expectedStatusCode, response.StatusCode)

			if d.pass {
				require.Equal(t, "application/x-yaml", response.Header.Get("Content-Type"))
			} else {
				require.Equal(t, "application/json", response.Header.Get("Content-Type"))
			}

			body, err := ioutil.ReadAll(response.Body)
			require.NoError(t, err)

			if d.pass {
				require.Equal(t, "--kubeconfig file", string(body))
			} else {
				var errorResponse ErrorResponse
				err := json.Unmarshal(body, &errorResponse)
				require.NoError(t, err)
				require.Equal(t, d.expectedErrorMessage, errorResponse.Error)
			}
		})
	}
}
