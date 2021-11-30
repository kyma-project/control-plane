package command

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	mothership "github.com/kyma-project/control-plane/components/reconciler/pkg"
	msmock "github.com/kyma-project/control-plane/components/reconciler/pkg/automock"
	"github.com/kyma-project/control-plane/tools/cli/pkg/command/automock"
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

var (
	testCtx    = context.Background()
	errTest    = errors.New("test error")
	outputJSON = "json"

	buildProvideEmptyKebResponse = func(ctrl *gomock.Controller) kebClientProvider {
		return func(_ string, _ *http.Client) kebClient {
			m := automock.NewMockkebClient(ctrl)
			m.EXPECT().
				ListRuntimes(gomock.Any()).
				Return(&runtime.RuntimesPage{
					Data:       []runtime.RuntimeDTO{},
					Count:      0,
					TotalCount: 0,
				}, nil).
				Times(1)
			return m
		}
	}

	http500TestResponse = http.Response{
		StatusCode: 500,
		Body:       io.NopCloser(strings.NewReader("This is fine")),
	}
)

func TestReconciliationCommand_Run(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type fields struct {
		ctx                context.Context
		output             string
		runtimeIds         []string
		shoots             []string
		rawStatuses        []string
		provideKebClient   kebClientProvider
		provideMshipClient mothershipClientProvider
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "happy path: none",
			fields: fields{
				ctx:              testCtx,
				output:           outputJSON,
				provideKebClient: buildProvideEmptyKebResponse(ctrl),
				provideMshipClient: func(_ string, _ *http.Client) (mothership.ClientInterface, error) {
					m := msmock.NewMockClientInterface(ctrl)
					m.EXPECT().
						GetReconciliations(gomock.Any(), gomock.All()).
						Return(&http.Response{
							StatusCode: 200,
							Body:       io.NopCloser(strings.NewReader("[]")),
						}, nil).
						Times(1)
					return m, nil
				},
			},
			wantErr: false,
		},
		{
			name: "keb err response",
			fields: fields{
				ctx:    testCtx,
				output: outputJSON,
				shoots: []string{"test"},
				provideKebClient: func(_ string, _ *http.Client) kebClient {
					m := automock.NewMockkebClient(ctrl)
					m.EXPECT().
						ListRuntimes(gomock.Any()).
						Return(runtime.RuntimesPage{}, errTest).
						Times(1)
					return m
				},
			},
			wantErr: true,
		},
		{
			name: "mothership err",
			fields: fields{
				ctx:              testCtx,
				output:           outputJSON,
				provideKebClient: buildProvideEmptyKebResponse(ctrl),
				provideMshipClient: func(_ string, _ *http.Client) (mothership.ClientInterface, error) {
					m := msmock.NewMockClientInterface(ctrl)
					m.EXPECT().
						GetReconciliations(gomock.Any(), gomock.Any()).
						Return(nil, errTest).
						Times(1)
					return m, nil
				},
			},
			wantErr: true,
		},
		{
			name: "happy path",
			fields: fields{
				ctx:        testCtx,
				output:     outputJSON,
				shoots:     []string{"c1", "c2"},
				runtimeIds: []string{"r1", "r2"},
				provideKebClient: func(_ string, _ *http.Client) kebClient {
					m := automock.NewMockkebClient(ctrl)
					m.EXPECT().
						ListRuntimes(gomock.AssignableToTypeOf(runtime.ListParameters{})).
						Return(
							runtime.RuntimesPage{
								Data: []runtime.RuntimeDTO{
									{RuntimeID: "r3"},
									{RuntimeID: "r4"},
								},
							}, nil).
						Times(1)
					return m
				},
				provideMshipClient: func(_ string, _ *http.Client) (mothership.ClientInterface, error) {
					m := msmock.NewMockClientInterface(ctrl)
					m.EXPECT().
						GetReconciliations(gomock.Any(), gomock.Any()).
						Return(
							&http.Response{
								StatusCode: 200,
								Body:       io.NopCloser(strings.NewReader("[]")),
							}, nil).
						Times(1)
					return m, nil
				},
			},
			wantErr: false,
		},
		{
			name: "Mothership internal error",
			fields: fields{
				ctx:              testCtx,
				output:           outputJSON,
				runtimeIds:       []string{},
				provideKebClient: buildProvideEmptyKebResponse(ctrl),
				provideMshipClient: func(_ string, _ *http.Client) (mothership.ClientInterface, error) {
					m := msmock.NewMockClientInterface(ctrl)
					m.EXPECT().
						GetReconciliations(gomock.Any(), gomock.Any()).
						Return(&http500TestResponse, nil).
						Times(1)
					return m, nil
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ReconciliationCommand{
				ctx: tt.fields.ctx,
				// log:                tt.fields.log,
				output:             tt.fields.output,
				rawStatuses:        tt.fields.rawStatuses,
				runtimeIds:         tt.fields.runtimeIds,
				shoots:             tt.fields.shoots,
				provideKebClient:   tt.fields.provideKebClient,
				provideMshipClient: tt.fields.provideMshipClient,
			}
			if err := cmd.Run(); (err != nil) != tt.wantErr {
				t.Errorf("ReconciliationCommand.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_sorting(t *testing.T) {
	type args struct {
		s []mothership.HTTPReconciliationInfo
	}
	tests := []struct {
		name       string
		args       args
		wantSorted bool
	}{
		{
			name: "Happy path",
			args: args{
				s: []mothership.HTTPReconciliationInfo{
					{
						Created: time.Now().Add(10 * time.Hour),
					},
					{
						Created: time.Now(),
					},
					{
						Created: time.Now().Add(20 * time.Hour),
					},
					{
						Created: time.Now().Add(-30 * time.Hour),
					},
					{
						Created: time.Now().Add(50 * time.Hour),
					},
				},
			},
			wantSorted: true,
		},
		{
			name: "No data",
			args: args{
				s: []mothership.HTTPReconciliationInfo{},
			},
			wantSorted: true,
		},
		{
			name: "One argument",
			args: args{
				s: []mothership.HTTPReconciliationInfo{
					{
						Created: time.Now(),
					},
				},
			},
			wantSorted: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sort(tt.args.s)
			checkIfSorted := sort.SliceIsSorted(tt.args.s, func(i, j int) bool {
				return tt.args.s[i].Created.Before(tt.args.s[j].Created)
			})
			if checkIfSorted != tt.wantSorted {
				t.Errorf("sorting() got = %v, wanted %v", checkIfSorted, tt.wantSorted)
			}
		})
	}
}
