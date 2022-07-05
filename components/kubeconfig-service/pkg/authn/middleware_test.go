package authn

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pkg/errors"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/user"
)

const token_not_base64 = "Bearer token"
const token_empty = "Bearer"
const token_not_containAdmin = "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
const token_containAdmin = "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyLCJncm91cHMiOlsicnVudGltZUFkbWluIiwicnVudGltZU9wZXJhdG9yIl19.yf3oqqMifPZIZ9lo5Hnkj3dVNSyJjwNBMuQ7ticvrE8"
const token_decode_error = "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyLCJncm91cHMiOlsicnVudGltZUFkbWluIiwicnVudGltZU9wZXJhdG9yIl19"

func TestAuthMiddleware(t *testing.T) {
	userInfo := user.DefaultInfo{Name: "Test User", UID: "deadbeef", Groups: []string{"admins", "testers"}}
	t.Run("When HTTP request token is empty", func(t *testing.T) {
		reject := &mockAuthenticator{Authorised: false}
		middleware := AuthMiddleware(reject)
		next := &mockHandler{}
		response := httptest.NewRecorder()
		middleware(next).ServeHTTP(response, newHttpRequest(token_empty))
		t.Run("Then authorizer is called with token", func(t *testing.T) {
			assert.False(t, reject.Called)
		})
		t.Run("Then next handler is not called", func(t *testing.T) {
			assert.False(t, next.Called)
		})
		t.Run("Then request is rejected with status code unauthorised", func(t *testing.T) {
			assert.Equal(t, http.StatusBadRequest, response.Code)
		})
		t.Run("Then response body is "+MALFORMED_TOKEN, func(t *testing.T) {
			assert.Equal(t, PARSING_OIDC_TOKEN+MALFORMED_TOKEN, parseResponseBody(response.Body))
		})
	})

	t.Run("When HTTP request token is not base64", func(t *testing.T) {
		authenticated := &mockAuthenticator{Authorised: true, UserInfo: &userInfo}
		middleware := AuthMiddleware(authenticated)
		next := &mockHandler{}
		response := httptest.NewRecorder()
		response.Code = 0
		middleware(next).ServeHTTP(response, newHttpRequest(token_not_base64))

		t.Run("Then authorizer is called with token", func(t *testing.T) {
			assert.False(t, authenticated.Called)
		})
		t.Run("Then next handler is called", func(t *testing.T) {
			assert.False(t, next.Called)
		})
		t.Run("Then status code is not set", func(t *testing.T) {
			assert.Equal(t, http.StatusBadRequest, response.Code)
		})
		t.Run("Then response body is "+DECODE_TOKEN_FAILD, func(t *testing.T) {
			assert.Equal(t, PARSING_OIDC_TOKEN+DECODE_TOKEN_FAILD, parseResponseBody(response.Body))
		})
	})

	t.Run("When HTTP request token decoded error", func(t *testing.T) {
		authenticated := &mockAuthenticator{Authorised: true, UserInfo: &userInfo}
		middleware := AuthMiddleware(authenticated)
		next := &mockHandler{}
		response := httptest.NewRecorder()
		response.Code = 0
		middleware(next).ServeHTTP(response, newHttpRequest(token_decode_error))

		t.Run("Then authorizer is called with token", func(t *testing.T) {
			assert.False(t, authenticated.Called)
		})
		t.Run("Then next handler is called", func(t *testing.T) {
			assert.False(t, next.Called)
		})
		t.Run("Then status code is not set", func(t *testing.T) {
			assert.Equal(t, http.StatusBadRequest, response.Code)
		})
		t.Run("Then repsonse body is "+DECODE_TOKEN_FAILD, func(t *testing.T) {
			assert.Equal(t, PARSING_OIDC_TOKEN+DECODE_TOKEN_FAILD, parseResponseBody(response.Body))
		})
	})

	t.Run("When HTTP request is forbidden", func(t *testing.T) {
		authenticated := &mockAuthenticator{Authorised: true, UserInfo: &userInfo}
		middleware := AuthMiddleware(authenticated)
		next := &mockHandler{}
		response := httptest.NewRecorder()
		response.Code = 0
		middleware(next).ServeHTTP(response, newHttpRequest(token_not_containAdmin))

		t.Run("Then authorizer is called with token", func(t *testing.T) {
			assert.False(t, authenticated.Called)
		})
		t.Run("Then next handler is called", func(t *testing.T) {
			assert.False(t, next.Called)
		})
		t.Run("Then status code is not set", func(t *testing.T) {
			assert.Equal(t, http.StatusForbidden, response.Code)
		})
		t.Run("Then response body is "+NO_GROUPS_IN_TOKEN, func(t *testing.T) {
			assert.Equal(t, PARSING_OIDC_TOKEN+NO_GROUPS_IN_TOKEN, parseResponseBody(response.Body))
		})
	})

	t.Run("When authentication error occurs on HTTP request", func(t *testing.T) {
		erroneous := &mockAuthenticator{Err: errors.New("failure")}
		middleware := AuthMiddleware(erroneous)
		next := &mockHandler{}
		response := httptest.NewRecorder()
		middleware(next).ServeHTTP(response, newHttpRequest(token_containAdmin))

		t.Run("Then authorizer is called with token", func(t *testing.T) {
			assert.True(t, erroneous.Called)
			assert.Equal(t, token_containAdmin, erroneous.LastReq.Header.Get("Authorization"))
		})
		t.Run("Then next handler is not called", func(t *testing.T) {
			assert.False(t, next.Called)
		})
		t.Run("Then request is rejected with status code unauthorised", func(t *testing.T) {
			assert.Equal(t, http.StatusUnauthorized, response.Code)
		})
	})

	t.Run("When HTTP request is authenticated", func(t *testing.T) {
		authenticated := &mockAuthenticator{Authorised: true, UserInfo: &userInfo}
		middleware := AuthMiddleware(authenticated)
		next := &mockHandler{}
		response := httptest.NewRecorder()
		response.Code = 0
		middleware(next).ServeHTTP(response, newHttpRequest(token_containAdmin))

		t.Run("Then authorizer is called with token", func(t *testing.T) {
			assert.True(t, authenticated.Called)
			assert.Equal(t, token_containAdmin, authenticated.AuthHeader)
		})
		t.Run("Then next handler is called", func(t *testing.T) {
			assert.True(t, next.Called)
		})
		t.Run("Then status code is not set", func(t *testing.T) {
			assert.Equal(t, 0, response.Code)
		})
	})
}

type mockAuthenticator struct {
	UserInfo   user.Info
	Authorised bool
	AuthHeader string
	Err        error
	LastReq    *http.Request
	Called     bool
}

func (a *mockAuthenticator) AuthenticateRequest(req *http.Request) (*authenticator.Response, bool, error) {
	a.Called = true
	a.LastReq = req

	if a.Authorised {
		a.AuthHeader = req.Header.Get("Authorization")
		//Mimic behaviour of k8s.io/apiserver/pkg/authentication/request/bearertoken.Authenticator.AuthenticateRequest
		req.Header.Del("Authorization")
	}

	return &authenticator.Response{User: a.UserInfo}, a.Authorised, a.Err
}

type mockHandler struct {
	Called bool
	w      http.ResponseWriter
	r      *http.Request
}

func (h *mockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.w = w
	h.r = r
	h.Called = true
}

func newHttpRequest(token string) *http.Request {
	req := httptest.NewRequest("POST", "/kube-config", strings.NewReader(""))
	req.Header.Set("Authorization", token)
	return req
}

func parseResponseBody(body io.Reader) string {
	b, err := io.ReadAll(body)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}
