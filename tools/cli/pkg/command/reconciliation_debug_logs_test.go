package command

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReconciliationDebugLogsCmd(t *testing.T) {
	testCtx = context.Background()

	schedulingID := fmt.Sprintf("%s--%s", uuid.NewString(), uuid.NewString())
	expectedPath := fmt.Sprintf("/reconciliations/%s/debug", schedulingID)

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
		}, "Reconciliation Not Found": {
			ctx:            testCtx,
			wantErr:        true,
			expectedErrMsg: "Reconciliation not found",
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

			cmd := &reconciliationDebugLogsCmd{
				reconcilerURL: mshipSvrMock.URL,
				ctx:           testCase.ctx,
				opts: reconciliationDebugLogsOpts{
					schedulingID: schedulingID,
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
