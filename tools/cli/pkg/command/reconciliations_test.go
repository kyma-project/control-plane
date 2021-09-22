package command

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	mothership "github.com/kyma-project/control-plane/components/mothership/pkg"
)

func Test_validateReconciliationStates(t *testing.T) {
	type args struct {
		rawStates []string
		params    mothership.GetReconcilesParams
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "happy path",
			args: args{
				rawStates: []string{"error", "reconcile_pending"},
				params:    mothership.GetReconcilesParams{},
			},
			wantErr: false,
		},
		{
			name: "err",
			args: args{
				rawStates: []string{"reconcile_pending", "unknown"},
				params:    mothership.GetReconcilesParams{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateReconciliationStates(tt.args.rawStates, &tt.args.params); (err != nil) != tt.wantErr {
				t.Errorf("validateReconciliationStates() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReconciliationCommand_Validate(t *testing.T) {
	type fields struct {
		output    string
		params    mothership.GetReconcilesParams
		rawStates []string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "happy path",
			fields: fields{
				output: "json",
				params: mothership.GetReconcilesParams{
					RuntimeIDs: &[]string{"id1", "id2", "id3"},
					Shoots:     &[]string{"shoot1", "shoot2"},
				},
				rawStates: []string{"reconcile_pending", "ready"},
			},
		},
		{
			name: "output error",
			fields: fields{
				output: "invalid-output",
			},
			wantErr: true,
		},
		{
			name: "reconciliation params error",
			fields: fields{
				output:    "table",
				params:    mothership.GetReconcilesParams{},
				rawStates: []string{"invalid-state"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ReconciliationCommand{
				output:    tt.fields.output,
				params:    tt.fields.params,
				rawStates: tt.fields.rawStates,
			}
			if err := cmd.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("ReconciliationCommand.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReconcileRun(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respoBody := []mothership.ReconcilerStatus{
			{
				Cluster: "test-cluster1",
				Status:  "reconcile_pending",
				Metadata: mothership.Metadata{
					InstanceID: "123",
				},
			},
		}
		var bodyWriter bytes.Buffer
		if err := json.NewEncoder(&bodyWriter).Encode(respoBody); err != nil {
			t.Error(err)
		}

		if _, err := w.Write(bodyWriter.Bytes()); err != nil {
			t.Error(err)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer svr.Close()

	cmd := NewReconciliationCmd(svr.URL)
	if err := cmd.RunE(nil, nil); err != nil {
		t.Error(err)
	}

}
