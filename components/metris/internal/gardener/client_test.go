package gardener

import (
	"path/filepath"
	"testing"

	gfake "github.com/gardener/gardener/pkg/client/core/clientset/versioned/fake"
	"k8s.io/client-go/kubernetes/fake"
)

func newFakeClient(t *testing.T) *Client {
	t.Helper()

	return &Client{
		Namespace:  "test-ns",
		GClientset: gfake.NewSimpleClientset(),
		KClientset: fake.NewSimpleClientset(),
	}
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name       string
		kubeconfig string
		wantErr    bool
	}{
		{
			name:       "good kubeconfig",
			kubeconfig: "good.kubeconfig",
			wantErr:    false,
		},
		{
			name:       "bad kubeconfig",
			kubeconfig: "bad.kubeconfig",
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		tt := tt // pin!

		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join("testdata", tt.kubeconfig)

			_, err := NewClient(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
