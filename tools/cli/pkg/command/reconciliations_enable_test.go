package command

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	mothership "github.com/kyma-project/control-plane/components/mothership/pkg"
	"github.com/stretchr/testify/require"
)

func Test_reconciliationEnableCmd_Run(t *testing.T) {
	type fields struct {
		ctx  context.Context
		opts reconciliationEnableOpts
	}
	type httpFields struct {
		expectedStatus string
		statusCode     int
	}
	tests := []struct {
		name       string
		fields     fields
		httpFields httpFields
		wantErr    bool
	}{
		{
			name: "ok with status 201",
			fields: fields{
				ctx:  context.Background(),
				opts: reconciliationEnableOpts{},
			},
			httpFields: httpFields{
				expectedStatus: string(mothership.StatusReady),
				statusCode:     201,
			},
			wantErr: false,
		},
		{
			name: "ok with force parameter and status 201",
			fields: fields{
				ctx: context.Background(),
				opts: reconciliationEnableOpts{
					force: true,
				},
			},
			httpFields: httpFields{
				expectedStatus: string(mothership.StatusReconcilePending),
				statusCode:     201,
			},
			wantErr: false,
		},
		{
			name: "err with status 500",
			fields: fields{
				ctx:  context.Background(),
				opts: reconciliationEnableOpts{},
			},
			httpFields: httpFields{
				expectedStatus: string(mothership.StatusReady),
				statusCode:     500,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request mothership.PutClustersRuntimeIDStatusJSONRequestBody
				err := json.NewDecoder(r.Body).Decode(&request)

				require.NoError(t, err)
				require.Equal(t, tt.httpFields.expectedStatus, string(request.Status))

				w.WriteHeader(tt.httpFields.statusCode)
			}))
			defer svr.Close()

			cmd := reconciliationEnableCmd{
				mothershipURL: svr.URL,
				ctx:           tt.fields.ctx,
				opts:          tt.fields.opts,
			}
			if err := cmd.Run(); (err != nil) != tt.wantErr {
				t.Errorf("reconciliationEnableCmd.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_reconciliationEnableCmd_Validate(t *testing.T) {
	tests := []struct {
		name    string
		opts    reconciliationEnableOpts
		wantErr bool
	}{
		{
			name: "ok with runtimeID",
			opts: reconciliationEnableOpts{
				runtimeID: "testID",
				shootName: "",
				force:     false,
			},
			wantErr: false,
		},
		{
			name: "ok with shootName",
			opts: reconciliationEnableOpts{
				runtimeID: "",
				shootName: "testName",
				force:     false,
			},
			wantErr: false,
		},
		{
			name: "err with shootName and runtimeID provided in the same time",
			opts: reconciliationEnableOpts{
				runtimeID: "testID",
				shootName: "testName",
				force:     false,
			},
			wantErr: true,
		},
		{
			name: "err with no params",
			opts: reconciliationEnableOpts{
				runtimeID: "",
				shootName: "",
				force:     false,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &reconciliationEnableCmd{
				opts: tt.opts,
			}
			if err := cmd.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("reconciliationEnableCmd.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
