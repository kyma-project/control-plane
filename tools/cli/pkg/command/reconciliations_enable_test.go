package command

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	reconciler "github.com/kyma-project/control-plane/components/reconciler/pkg"
	"github.com/stretchr/testify/require"
)

func Test_reconciliationEnableCmd_Run(t *testing.T) {
	type fields struct {
		ctx  context.Context
		opts reconciliationEnableOpts
	}
	type httpFields struct {
		mothershipExpectedStatus string
		mothershipStatusCode     int
		kebData                  runtime.RuntimesPage
	}
	tests := []struct {
		name       string
		fields     fields
		httpFields httpFields
		wantErr    bool
	}{
		{
			name: "ok with status 201 and runtime-id",
			fields: fields{
				ctx: context.Background(),
				opts: reconciliationEnableOpts{
					runtimeID: "test-runtime-id",
				},
			},
			httpFields: httpFields{
				mothershipExpectedStatus: string(reconciler.StatusReady),
				mothershipStatusCode:     201,
			},
			wantErr: false,
		},
		{
			name: "ok with status 201 and shoot",
			fields: fields{
				ctx: context.Background(),
				opts: reconciliationEnableOpts{
					shootName: "test-shoot",
				},
			},
			httpFields: httpFields{
				mothershipExpectedStatus: string(reconciler.StatusReady),
				mothershipStatusCode:     201,
				kebData: runtime.RuntimesPage{
					Count: 1,
					Data: []runtime.RuntimeDTO{
						{
							RuntimeID: "test-id",
						},
					},
				},
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
				mothershipExpectedStatus: string(reconciler.StatusReconcilePending),
				mothershipStatusCode:     201,
			},
			wantErr: false,
		},
		{
			name: "err with status 500 from the mothership",
			fields: fields{
				ctx:  context.Background(),
				opts: reconciliationEnableOpts{},
			},
			httpFields: httpFields{
				mothershipExpectedStatus: string(reconciler.StatusReady),
				mothershipStatusCode:     500,
			},
			wantErr: true,
		},
		{
			name: "err with no content from the keb",
			fields: fields{
				ctx: context.Background(),
				opts: reconciliationEnableOpts{
					shootName: "test-shoot",
				},
			},
			httpFields: httpFields{
				mothershipExpectedStatus: string(reconciler.StatusReady),
				mothershipStatusCode:     201,
				kebData: runtime.RuntimesPage{
					Count: 0,
				},
			},
			wantErr: true,
		},
		{
			name: "err with no unique content from the keb",
			fields: fields{
				ctx: context.Background(),
				opts: reconciliationEnableOpts{
					shootName: "test-shoot",
				},
			},
			httpFields: httpFields{
				mothershipExpectedStatus: string(reconciler.StatusReady),
				mothershipStatusCode:     201,
				kebData: runtime.RuntimesPage{
					Count: 21,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mothershipSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request reconciler.PutClustersRuntimeIDStatusJSONRequestBody
				err := json.NewDecoder(r.Body).Decode(&request)

				require.NoError(t, err)
				require.Equal(t, tt.httpFields.mothershipExpectedStatus, string(request.Status))

				w.WriteHeader(tt.httpFields.mothershipStatusCode)
			}))
			defer mothershipSvr.Close()

			kebSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				err := json.NewEncoder(w).Encode(tt.httpFields.kebData)
				require.NoError(t, err)
			}))
			defer kebSvr.Close()

			cmd := reconciliationEnableCmd{
				reconcilerURL: mothershipSvr.URL,
				kebURL:        kebSvr.URL,
				auth:          nil,
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
