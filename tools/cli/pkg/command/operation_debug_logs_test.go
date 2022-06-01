package command

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	mothership "github.com/kyma-project/control-plane/components/reconciler/pkg"
	"github.com/stretchr/testify/require"
)

func TestOperationDebugLogsCmd(t *testing.T) {
	testCtx = context.Background()

	schedulingID := fmt.Sprintf("%s--%s", uuid.NewString(), uuid.NewString())
	correlationID := fmt.Sprintf("%s--%s", uuid.NewString(), uuid.NewString())
	expectedPath := fmt.Sprintf("/operations/%s/%s/debug", schedulingID, correlationID)

	errMsg := "Test error message"

	testCases := map[string]struct {
		ctx            context.Context
		wantErr        bool
		expectedErrMsg string
		mockResponse   func(t *testing.T) func(http.ResponseWriter, *http.Request)
	}{
		"Success": {
			ctx:     testCtx,
			wantErr: false,
			mockResponse: func(t *testing.T) func(http.ResponseWriter, *http.Request) {
				return func(writer http.ResponseWriter, r *http.Request) {
					assertHttpRequest(t, r, expectedPath)
				}
			},
		}, "Operation Not Found": {
			ctx:            testCtx,
			wantErr:        true,
			expectedErrMsg: "Operation not found",
			mockResponse: func(t *testing.T) func(http.ResponseWriter, *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					assertHttpRequest(t, r, expectedPath)
					w.WriteHeader(http.StatusNotFound)
				}
			},
		}, "Request Failed": {
			ctx:            testCtx,
			wantErr:        true,
			expectedErrMsg: errMsg,
			mockResponse: func(t *testing.T) func(http.ResponseWriter, *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					assertHttpRequest(t, r, expectedPath)
					w.WriteHeader(http.StatusInternalServerError)
					writeErrorResponse(t, w, errMsg)
				}
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {

			//GIVEN
			mshipSvrMock := httptest.NewServer(http.HandlerFunc(testCase.mockResponse(t)))
			defer mshipSvrMock.Close()

			cmd := &operationDebugLogsCmd{
				reconcilerURL: mshipSvrMock.URL,
				ctx:           testCase.ctx,
				opts: operationDebugLogsOpts{
					correlationID: correlationID,
					schedulingID:  schedulingID,
				},
			}

			//WHEN
			err := cmd.Run()

			//THEN
			if testCase.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), testCase.expectedErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func assertHttpRequest(t *testing.T, r *http.Request, expectedPath string) {
	require.Equal(t, expectedPath, r.URL.Path)
	require.Equal(t, http.MethodPut, r.Method)
}

func writeErrorResponse(t *testing.T, w http.ResponseWriter, msg string) {
	resp := mothership.HTTPErrorResponse{Error: msg}
	out, err := json.Marshal(&resp)
	require.NoError(t, err)

	_, err = w.Write(out)
	require.NoError(t, err)
}
