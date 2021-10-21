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

func Test_reconciliationDisableCmd_Run(t *testing.T) {
	type fields struct {
		ctx  context.Context
		opts reconciliationDisableOpts
	}
	type httpFields struct {
		mothershipStatus int
		kebData          runtime.RuntimesPage
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
				opts: reconciliationDisableOpts{
					runtimeID: "test-runtime-id",
				},
			},
			httpFields: httpFields{
				mothershipStatus: 201,
			},
			wantErr: false,
		},
		{
			name: "ok with status 201 and shoot",
			fields: fields{
				ctx: context.Background(),
				opts: reconciliationDisableOpts{
					shootName: "test-shoot",
				},
			},
			httpFields: httpFields{
				mothershipStatus: 201,
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
			name: "err with status 500 from the mothership",
			fields: fields{
				ctx:  context.Background(),
				opts: reconciliationDisableOpts{},
			},
			httpFields: httpFields{
				mothershipStatus: 500,
			},
			wantErr: true,
		},

		{
			name: "err with empty keb response",
			fields: fields{
				ctx: context.Background(),
				opts: reconciliationDisableOpts{
					shootName: "test-shoot",
				},
			},
			httpFields: httpFields{
				mothershipStatus: 201,
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
				opts: reconciliationDisableOpts{
					shootName: "test-shoot",
				},
			},
			httpFields: httpFields{
				mothershipStatus: 201,
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
				require.Equal(t, reconciler.StatusReconcileDisabled, request.Status)

				w.WriteHeader(tt.httpFields.mothershipStatus)
			}))
			defer mothershipSvr.Close()

			kebSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				err := json.NewEncoder(w).Encode(tt.httpFields.kebData)
				require.NoError(t, err)
			}))
			defer kebSvr.Close()

			cmd := reconciliationDisableCmd{
				reconcilerURL: mothershipSvr.URL,
				kebURL:        kebSvr.URL,
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
