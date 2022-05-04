package command

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	mothership "github.com/kyma-project/control-plane/components/reconciler/pkg"
	msmock "github.com/kyma-project/control-plane/components/reconciler/pkg/automock"
	"github.com/kyma-project/control-plane/tools/cli/pkg/command/automock"
	"github.com/pkg/errors"
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

var (
	buildProvideEmptyMothershipResponse = func(ctrl *gomock.Controller) mothershipClientProvider {
		return func(_ string, _ *http.Client) (mothership.ClientInterface, error) {
			m := msmock.NewMockClientInterface(ctrl)
			m.EXPECT().
				GetClustersState(gomock.Any(), gomock.All()).
				Return(&http.Response{
					StatusCode: 200,
					Body: io.NopCloser(strings.NewReader(`{"cluster": {
						"runtimeID": "testID"
					}, "configuration": {
						"kymaVersion": "testVersion",
						"kymaProfile": "testProfile"
					}, "status": {
						"status":  "testStatus",
						"deleted": false,
						"created": "2022-01-11T13:59:21.933508Z"
					}}`)),
				}, nil).Times(1)
			return m, nil
		}
	}
)

func TestRuntimeStateCommand_Run(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	type fields struct {
		opts               RuntimeStateOptions
		ctx                context.Context
		provideKebClient   kebClientProvider
		provideMshipClient mothershipClientProvider
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "happy path",
			fields: fields{
				opts: RuntimeStateOptions{
					output:    "json",
					runtimeID: "test-runtime-id",
				},
				ctx:                context.Background(),
				provideMshipClient: buildProvideEmptyMothershipResponse(ctrl),
			},
			wantErr: false,
		},
		{
			name: "happy path with shootName",
			fields: fields{
				opts: RuntimeStateOptions{
					output:    "table",
					shootName: "test-shootName",
				},
				ctx:                context.Background(),
				provideKebClient:   buildProvideKebResponse(ctrl, "test-runtime-id"),
				provideMshipClient: buildProvideEmptyMothershipResponse(ctrl),
			},
			wantErr: false,
		},
		{
			name: "mothership resp status 404",
			fields: fields{
				opts: RuntimeStateOptions{
					output:    "json",
					runtimeID: "test-runtime-id",
				},
				ctx: context.Background(),
				provideMshipClient: func(_ string, _ *http.Client) (mothership.ClientInterface, error) {
					m := msmock.NewMockClientInterface(ctrl)
					m.EXPECT().
						GetClustersState(gomock.Any(), gomock.All()).
						Return(&http.Response{
							StatusCode: 404,
							Body:       io.NopCloser(strings.NewReader("{}")),
						}, nil).Times(1)
					return m, nil
				},
			},
			wantErr: true,
		},
		{
			name: "mothership call error",
			fields: fields{
				opts: RuntimeStateOptions{
					output:    "json",
					runtimeID: "test-runtime-id",
				},
				ctx: context.Background(),
				provideMshipClient: func(_ string, _ *http.Client) (mothership.ClientInterface, error) {
					m := msmock.NewMockClientInterface(ctrl)
					m.EXPECT().
						GetClustersState(gomock.Any(), gomock.All()).
						Return(&http.Response{}, errors.New("example error")).Times(1)
					return m, nil
				},
			},
			wantErr: true,
		},
		{
			name: "mothership client error",
			fields: fields{
				opts: RuntimeStateOptions{
					output:    "json",
					runtimeID: "test-runtime-id",
				},
				ctx: context.Background(),
				provideMshipClient: func(_ string, _ *http.Client) (mothership.ClientInterface, error) {
					return nil, errors.New("example error")
				},
			},
			wantErr: true,
		},
		{
			name: "empty keb response",
			fields: fields{
				opts: RuntimeStateOptions{
					output:    "json",
					shootName: "test-shootName",
				},
				ctx:              context.Background(),
				provideKebClient: buildProvideEmptyKebResponse(ctrl),
			},
			wantErr: true,
		},
		{
			name: "keb call error",
			fields: fields{
				opts: RuntimeStateOptions{
					output:    "json",
					shootName: "test-shootName",
				},
				ctx: context.Background(),
				provideKebClient: func(_ string, _ *http.Client) kebClient {
					m := automock.NewMockkebClient(ctrl)
					m.EXPECT().
						ListRuntimes(gomock.Any()).
						Return(runtime.RuntimesPage{}, errors.New("example error")).
						Times(1)
					return m
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &RuntimeStateCommand{
				opts:               tt.fields.opts,
				ctx:                tt.fields.ctx,
				provideKebClient:   tt.fields.provideKebClient,
				provideMshipClient: tt.fields.provideMshipClient,
			}
			if err := cmd.Run(); (err != nil) != tt.wantErr {
				t.Errorf("RuntimeStateCommand.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
