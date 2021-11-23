package command

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	mothership "github.com/kyma-project/control-plane/components/reconciler/pkg"
	msmock "github.com/kyma-project/control-plane/components/reconciler/pkg/automock"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	"github.com/stretchr/testify/require"
)

func TestReconciliationOperationInfoCommand_Run(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type fields struct {
		ctx                context.Context
		log                logger.Logger
		output             string
		schedulingID       string
		provideMshipClient mothershipClientProvider
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "reconciliation info: reconciliation not found",
			fields: fields{
				ctx:          testCtx,
				output:       outputJSON,
				schedulingID: "",
				provideMshipClient: func(url string, _ *http.Client) (mothership.ClientInterface, error) {
					m := msmock.NewMockClientInterface(ctrl)
					m.EXPECT().
						GetReconciliationsSchedulingIDInfo(gomock.Any(), gomock.Any()).
						Return(&http.Response{
							StatusCode: 404,
							Body:       io.NopCloser(strings.NewReader("{}")),
						}, nil).
						Times(1)
					return m, nil
				},
			},
			wantErr: true,
		},
		{
			name: "reconciliation info: mothership provider error",
			fields: fields{
				ctx:    testCtx,
				output: outputJSON,
				provideMshipClient: func(url string, _ *http.Client) (mothership.ClientInterface, error) {
					m := msmock.NewMockClientInterface(ctrl)
					return m, errTest
				},
			},
			wantErr: true,
		},
		{
			name: "reconciliation info: mothership error",
			fields: fields{
				ctx:    testCtx,
				output: outputJSON,
				provideMshipClient: func(url string, _ *http.Client) (mothership.ClientInterface, error) {
					m := msmock.NewMockClientInterface(ctrl)
					m.EXPECT().
						GetReconciliationsSchedulingIDInfo(gomock.Any(), gomock.Any()).
						Return(&http.Response{}, errTest).
						Times(1)
					return m, nil
				},
			},
			wantErr: true,
		},
		{
			name: "reconciliation info: mothership error response",
			fields: fields{
				ctx:    testCtx,
				output: outputJSON,
				provideMshipClient: func(url string, _ *http.Client) (mothership.ClientInterface, error) {
					m := msmock.NewMockClientInterface(ctrl)
					m.EXPECT().
						GetReconciliationsSchedulingIDInfo(gomock.Any(), gomock.Any()).
						Return(&http500TestResponse, nil).
						Times(1)
					return m, nil
				},
			},
			wantErr: true,
		},
		{
			name: "reconciliation info: happy path - reconciliation found",
			fields: fields{
				ctx:    testCtx,
				output: outputJSON,
				provideMshipClient: func(url string, _ *http.Client) (mothership.ClientInterface, error) {
					m := msmock.NewMockClientInterface(ctrl)
					m.EXPECT().
						GetReconciliationsSchedulingIDInfo(gomock.Any(), gomock.Any()).
						Return(&http.Response{
							StatusCode: 200,
							Body:       io.NopCloser(strings.NewReader(readTestResponseFromFile(t, "testdata/reconciliationInfoResponse.json"))),
						}, nil).
						Times(1)
					m.EXPECT().
						GetClustersRuntimeIDConfigVersion(gomock.Any(), gomock.Any(), "1").
						Return(&http.Response{
							StatusCode: 200,
							Body:       io.NopCloser(strings.NewReader(readTestResponseFromFile(t, "testdata/configVersionResponse.json"))),
						}, nil).
						Times(1)
					return m, nil
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ReconciliationOperationInfoCommand{
				ctx:                tt.fields.ctx,
				log:                tt.fields.log,
				output:             tt.fields.output,
				schedulingID:       tt.fields.schedulingID,
				provideMshipClient: tt.fields.provideMshipClient,
			}
			if err := cmd.Run(); (err != nil) != tt.wantErr {
				t.Errorf("ReconciliationOperationInfoCommand.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func readTestResponseFromFile(t *testing.T, testFilePath string) string {
	r, err := ioutil.ReadFile("testdata/reconciliationInfoResponse.json")
	require.NoError(t, err)
	return string(r)
}
