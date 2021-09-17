package mothership

import (
	"net/url"
	"testing"
)

func Test_urlProvider_Provide(t *testing.T) {
	type fields struct {
		URL url.URL
	}
	type args struct {
		endpoint    string
		queryParams map[string]string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "happy path",
			fields: fields{
				URL: url.URL{
					Scheme: "https",
					Host:   "test-me.pl",
				},
			},
			args: args{
				endpoint: EndpointReconcile,
				queryParams: map[string]string{
					"instance-id": "123",
				},
			},
			want: "https://test-me.pl/reconcile?instance-id=123",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := urlProvider{
				tt.fields.URL,
			}

			u := p.Provide(tt.args.endpoint, tt.args.queryParams)
			if got := u.String(); got != tt.want {
				t.Errorf("urlProvider.Provide() = %v, want %v", got, tt.want)
			}
		})
	}
}
