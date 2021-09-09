package command

import (
	"testing"

	mothership "github.com/kyma-project/control-plane/components/reconciler/pkg"
)

func Test_validateReconciliationStates(t *testing.T) {
	type args struct {
		rawStates []string
		params    mothership.GetReconciliationsParams
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
				params:    mothership.GetReconciliationsParams{},
			},
			wantErr: false,
		},
		{
			name: "err",
			args: args{
				rawStates: []string{"reconcile_pending", "unknown"},
				params:    mothership.GetReconciliationsParams{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := toReconciliationStatuses(tt.args.rawStates); (err != nil) != tt.wantErr {
				t.Errorf("validateReconciliationStates() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReconciliationCommand_Validate(t *testing.T) {
	type fields struct {
		output      string
		rawStatuses []string
		runtimeIds  []string
		shoots      []string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "happy path",
			fields: fields{
				output:      "json",
				runtimeIds:  []string{"id1", "id2", "id3"},
				rawStatuses: []string{"reconcile_pending", "ready"},
				shoots:      []string{"shoot1"},
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
				output:      "table",
				rawStatuses: []string{"invalid-state"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ReconciliationCommand{
				output:      tt.fields.output,
				rawStatuses: tt.fields.rawStatuses,
				runtimeIds:  tt.fields.runtimeIds,
			}
			if err := cmd.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("ReconciliationCommand.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
