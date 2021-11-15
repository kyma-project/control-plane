package command

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	mothership "github.com/kyma-project/control-plane/components/reconciler/pkg"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestName(t *testing.T) {
	testCtx = context.Background()

	schedulingID := "258bd75d-58ba-4db9-8592-f6ea6489082d--dd0391d4-eab8-49a3-8bfb-d5c8dadd2cba"
	correlationID := "13a8754c-edb2-4aaa-9c39-15bce8e1d8cd--f2c44235-9440-44a4-adfe-2f7455bd784c"
	expectedPath := fmt.Sprintf("/operations/%s/%s/stop", schedulingID, correlationID)

	expectedErrMsg := "Test error message"

	testCases := map[string]struct {
		ctx          context.Context
		wantErr      bool
		mockResponse func(t *testing.T) func(http.ResponseWriter, *http.Request)
	}{
		"Success": {
			ctx:     testCtx,
			wantErr: false,
			mockResponse: func(t *testing.T) func(http.ResponseWriter, *http.Request) {
				return func(writer http.ResponseWriter, r *http.Request) {
					assertRequest(t, r, expectedPath)
				}
			},
		}, "Request Failed": {
			ctx:     testCtx,
			wantErr: true,
			mockResponse: func(t *testing.T) func(http.ResponseWriter, *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					assertRequest(t, r, expectedPath)
					writeErrResponse(t, w, expectedErrMsg)
				}
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			//GIVEN
			mshipSvrMock := httptest.NewServer(http.HandlerFunc(testCase.mockResponse(t)))
			defer mshipSvrMock.Close()

			cmd := &operationStopCmd{
				reconcilerURL: mshipSvrMock.URL,
				ctx:           testCase.ctx,
				opts: operationDisableOpts{
					correlationID: correlationID,
					schedulingID:  schedulingID,
				},
			}

			//WHEN
			err := cmd.Run()

			//THEN
			if testCase.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), expectedErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func assertRequest(t *testing.T, r *http.Request, expectedPath string) {
	require.Equal(t, expectedPath, r.URL.Path)

	out := unmarshallRequest(t, r)
	require.Equal(t, "Operation set to DONE manually via KCP CLI", out.Reason)
}

func unmarshallRequest(t *testing.T, r *http.Request) mothership.PostOperationsSchedulingIDCorrelationIDStopJSONRequestBody {
	out, err := io.ReadAll(r.Body)
	require.NoError(t, err)

	var reqBody mothership.PostOperationsSchedulingIDCorrelationIDStopJSONRequestBody
	require.NoError(t, json.Unmarshal(out, &reqBody))

	return reqBody
}

func writeErrResponse(t *testing.T, w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusForbidden)

	resp := mothership.HTTPErrorResponse{Error: msg}
	out, err := json.Marshal(&resp)
	require.NoError(t, err)

	_, err = w.Write(out)
	require.NoError(t, err)
}
