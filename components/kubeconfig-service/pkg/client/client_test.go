package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

type FakeTokenSource string

var fixToken FakeTokenSource = "fake-token-1234"

const (
	testTenant     = "test-tenant-id-0001"
	testRuntime    = "test-runtime-id-0001"
	testKubeConfig = `---
yaml-like: true
multi-line: |-
  lorem impsum
  donor
special-chars-to-escape: "abcdefgh<0123456789>ijkl*:_mnopqrst"`
)

func (t FakeTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{
		AccessToken: string(t),
		Expiry:      time.Now().Add(time.Duration(12 * time.Hour)),
	}, nil
}
func TestClient_GetKubeConfig(t *testing.T) {
	// given

	called := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, fmt.Sprintf("/kubeconfig/%s/%s", testTenant, testRuntime), r.URL.Path)

		assert.Equal(t, r.Header.Get("Authorization"), fmt.Sprintf("Bearer %s", fixToken))
		called = true

		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(testKubeConfig))
		require.NoError(t, err)
	}))
	defer ts.Close()

	client := NewClient(context.TODO(), ts.URL, fixToken)

	// when
	kc, err := client.GetKubeConfig(testTenant, testRuntime)

	// then
	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, testKubeConfig, kc)
}
