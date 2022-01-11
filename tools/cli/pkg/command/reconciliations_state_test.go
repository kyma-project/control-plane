package command

import (
	"context"
	"testing"
)

func TestRuntimeStateOptions_Validate(t *testing.T) {
	type fields struct {
		output        string
		runtimeID     string
		shootName     string
		correlationID string
		schedulingID  string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name:    "empty args error",
			wantErr: true,
		},
		{
			name: "runtime-id arg",
			fields: fields{
				output:    "table",
				runtimeID: "test-runtime-id",
			},
			wantErr: false,
		},
		{
			name: "scheduling-id arg",
			fields: fields{
				output:       "table",
				schedulingID: "test-scheduling-id",
			},
			wantErr: false,
		},
		{
			name: "correlation-id arg",
			fields: fields{
				output:        "table",
				correlationID: "test-correlation-id",
			},
			wantErr: false,
		},
		{
			name: "shootName arg",
			fields: fields{
				output:    "json",
				shootName: "test-shootName",
			},
			wantErr: false,
		},
		{
			name: "too many args error",
			fields: fields{
				output:        "json",
				shootName:     "test-shootName",
				correlationID: "test-correlation-id",
				runtimeID:     "test-runtime-id",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &RuntimeStateOptions{
				output:        tt.fields.output,
				runtimeID:     tt.fields.runtimeID,
				shootName:     tt.fields.shootName,
				correlationID: tt.fields.correlationID,
				schedulingID:  tt.fields.schedulingID,
			}
			if err := opts.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("RuntimeStateOptions.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRuntimeStateCommand_Run(t *testing.T) {
	type fields struct {
		opts               RuntimeStateOptions
		ctx                context.Context
		provideMshipClient mothershipClientProvider
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &RuntimeStateCommand{
				opts:               tt.fields.opts,
				ctx:                tt.fields.ctx,
				provideMshipClient: tt.fields.provideMshipClient,
			}
			if err := cmd.Run(); (err != nil) != tt.wantErr {
				t.Errorf("RuntimeStateCommand.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
