package mothership

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/kyma-project/control-plane/components/mothership/pkg/automock"
)

var (
	errTest        = fmt.Errorf("test error")
	okResponseBody = `[{"id": "test"}]`
)

func Test_client_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type args struct {
		fileters map[string]string
		timeout  time.Duration
	}
	tests := []struct {
		name    string
		c       Client
		args    args
		want    []Reconciliation
		wantErr bool
	}{
		{
			name: "context deadline exceeded",
			args: args{
				timeout: time.Second,
			},
			c: func() Client {
				// mock URL provider
				urlProvider := automock.NewMockURLProvider(ctrl)
				urlProvider.EXPECT().Provide(EndpointReconcile, gomock.Any()).
					DoAndReturn(func(_ string, _ map[string]string) url.URL {
						return url.URL{}
					})

				// mock http client
				httpClient := automock.NewMockHttpClient(ctrl)
				httpClient.EXPECT().Do(gomock.Any()).
					DoAndReturn(func(req *http.Request) (*http.Response, error) {
						select {
						case <-req.Context().Done():
							return nil, req.Context().Err()
						}
					}).MaxTimes(1)

				return &client{
					URLProvider: urlProvider,
					HttpClient:  httpClient,
				}
			}(),
			wantErr: true,
		},
		{
			name: "url error",
			c: func() Client {
				// mock URL provider
				urlProvider := automock.NewMockURLProvider(ctrl)
				urlProvider.EXPECT().Provide(EndpointReconcile, gomock.Any()).
					DoAndReturn(func(_ string, _ map[string]string) url.URL {
						return url.URL{
							Scheme: "ąę",
							Path:   "ś",
						}
					})

				return &client{
					URLProvider: urlProvider,
				}
			}(),
			wantErr: true,
		},
		{
			name: "http client error",
			c: func() Client {
				// mock http client
				httpClient := automock.NewMockHttpClient(ctrl)
				httpClient.EXPECT().Do(gomock.Any()).Return(nil, errTest).MaxTimes(1)

				// mock URL provider
				urlProvider := automock.NewMockURLProvider(ctrl)
				urlProvider.EXPECT().Provide(EndpointReconcile, gomock.Any()).
					DoAndReturn(func(endpoint string, _ map[string]string) url.URL {
						if endpoint != EndpointReconcile {
							t.Errorf("invalid endpoint = %s want: %s, ", endpoint, EndpointReconcile)
						}
						return url.URL{
							Scheme: "https",
						}
					})

				return &client{
					HttpClient:  httpClient,
					URLProvider: urlProvider,
				}
			}(),
			wantErr: true,
		},
		{
			name: "happy path",
			c: func() Client {
				// mock URL provider
				urlProvider := automock.NewMockURLProvider(ctrl)
				urlProvider.EXPECT().Provide(EndpointReconcile, gomock.Any()).
					DoAndReturn(func(endpoint string, filters map[string]string) url.URL {
						u := url.URL{
							Scheme: "https",
							Path:   endpoint,
						}
						for k, v := range filters {
							u.Query().Add(k, v)
						}
						return u
					})

				// mock http client
				httpClient := automock.NewMockHttpClient(ctrl)
				httpClient.EXPECT().Do(gomock.Any()).
					DoAndReturn(func(_ *http.Request) (*http.Response, error) {
						reader := strings.NewReader(okResponseBody)
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(reader),
						}, nil
					}).MaxTimes(1)

				return &client{
					URLProvider: urlProvider,
					HttpClient:  httpClient,
				}
			}(),
			want: []Reconciliation{
				{
					ID: "test",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.args.timeout)
			defer cancel()

			got, err := tt.c.List(ctx, tt.args.fileters)
			if (err != nil) != tt.wantErr {
				t.Errorf("client.List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.List() = %v, want %v", got, tt.want)
			}
		})
	}
}
