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

func Test_reconciliationDisableCmd_Run(t *testing.T) {
	type fields struct {
		ctx  context.Context
		opts reconciliationDisableOpts
	}
	type httpFields struct {
		statusCode int
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
				opts: reconciliationDisableOpts{},
			},
			httpFields: httpFields{
				statusCode: 201,
			},
			wantErr: false,
		},
		{
			name: "err with status 500",
			fields: fields{
				ctx:  context.Background(),
				opts: reconciliationDisableOpts{},
			},
			httpFields: httpFields{
				statusCode: 500,
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
				require.Equal(t, mothership.StatusReconcileDisabled, request.Status)

				w.WriteHeader(tt.httpFields.statusCode)
			}))
			defer svr.Close()

			cmd := reconciliationDisableCmd{
				mothershipURL: svr.URL,
				ctx:           tt.fields.ctx,
				opts:          tt.fields.opts,
			}
			if err := cmd.Run(); (err != nil) != tt.wantErr {
				t.Errorf("reconciliationDisableCmd.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_reconciliationDisableCmd_Validate(t *testing.T) {
	tests := []struct {
		name    string
		opts    reconciliationDisableOpts
		wantErr bool
	}{
		{
			name: "ok with runtimeID",
			opts: reconciliationDisableOpts{
				runtimeID: "testID",
				shootName: "",
			},
			wantErr: false,
		},
		{
			name: "ok with shootName",
			opts: reconciliationDisableOpts{
				runtimeID: "",
				shootName: "testName",
			},
			wantErr: false,
		},
		{
			name: "err with shootName and runtimeID provided in the same time",
			opts: reconciliationDisableOpts{
				runtimeID: "testID",
				shootName: "testName",
			},
			wantErr: true,
		},
		{
			name: "err with no params",
			opts: reconciliationDisableOpts{
				runtimeID: "",
				shootName: "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &reconciliationDisableCmd{
				opts: tt.opts,
			}
			if err := cmd.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("reconciliationDisableCmd.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
