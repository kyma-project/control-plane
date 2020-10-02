package credential

import (
	"context"
	"time"

	"github.com/int128/kubelogin/pkg/adaptors/browser"
	"github.com/int128/kubelogin/pkg/adaptors/certpool"
	"github.com/int128/kubelogin/pkg/adaptors/clock"
	"github.com/int128/kubelogin/pkg/adaptors/credentialpluginwriter"
	"github.com/int128/kubelogin/pkg/adaptors/oidcclient"
	"github.com/int128/kubelogin/pkg/adaptors/reader"
	"github.com/int128/kubelogin/pkg/adaptors/tokencache"
	"github.com/int128/kubelogin/pkg/usecases/authentication"
	"github.com/int128/kubelogin/pkg/usecases/authentication/authcode"
	"github.com/int128/kubelogin/pkg/usecases/authentication/ropc"
	"github.com/int128/kubelogin/pkg/usecases/credentialplugin"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/cmd/cli/logger"
	"k8s.io/client-go/util/homedir"
)

var defaultTokenCacheDir = homedir.HomeDir() + "/.kube/cache/oidc-login"
var defaultListenAddress = []string{"127.0.0.1:8000", "127.0.0.1:18000"}

const defaultAuthenticationTimeoutSec = 180

// Manager is a client for an OIDC provider capable of authenticating users and retrieving ID tokens through
//   - Authorization code grant flow using browser for interactive use
//   - Resource owner password credentials flow for non-interactive use
type Manager interface {
	GetToken(ctx context.Context) (string, error)
	GetTokenByROPC(ctx context.Context, username, password string) (string, error)
	TokenExpiry() time.Time
}

type manager struct {
	getter *credentialplugin.GetToken
	input  credentialplugin.Input
	token  string
	expiry time.Time
}

type tokenWriter struct {
	mgr *manager
}

func (w *tokenWriter) Write(out credentialpluginwriter.Output) error {
	w.mgr.cacheToken(out.Token, out.Expiry)
	return nil
}

// NewManager Constructs a new credential.Manager using the given OIDC provider and client credentials
func NewManager(oidcIssuerURL, oidcClientID, oidcClientSecret string, logger logger.Logger) Manager {
	clock := &clock.Real{}
	reader := &reader.Reader{}
	auth := &authentication.Authentication{
		Clock:  clock,
		Logger: logger,
		OIDCClient: &oidcclient.Factory{
			Clock:  clock,
			Logger: logger,
		},
		AuthCodeBrowser: &authcode.Browser{
			Logger:  logger,
			Browser: &browser.Browser{},
		},
		AuthCodeKeyboard: &authcode.Keyboard{
			Logger: logger,
			Reader: reader,
		},
		ROPC: &ropc.ROPC{
			Logger: logger,
			Reader: reader,
		},
	}

	mgr := &manager{
		input: credentialplugin.Input{
			IssuerURL:     oidcIssuerURL,
			ClientID:      oidcClientID,
			ClientSecret:  oidcClientSecret,
			TokenCacheDir: defaultTokenCacheDir,
		},
	}
	writer := &tokenWriter{mgr: mgr}
	getToken := &credentialplugin.GetToken{
		Logger:               logger,
		Authentication:       auth,
		TokenCacheRepository: &tokencache.Repository{},
		NewCertPool:          certpool.New,
		Writer:               writer,
	}
	mgr.getter = getToken

	return mgr
}

// GetToken fetches an ID token from local cache if a valid token is found, or else initiates interactive authorization code grant flow with browser to request a new ID token
func (mgr *manager) GetToken(ctx context.Context) (string, error) {
	in := mgr.input
	in.GrantOptionSet.AuthCodeBrowserOption = &authcode.BrowserOption{
		BindAddress:           defaultListenAddress,
		SkipOpenBrowser:       false,
		AuthenticationTimeout: time.Duration(defaultAuthenticationTimeoutSec) * time.Second,
		RedirectURLHostname:   "localhost",
	}
	err := mgr.getter.Do(ctx, in)
	if err != nil {
		return "", err
	}
	return mgr.token, nil
}

// GetTokenByROPC fetches an ID token from local cache if a valid token is found, or else initiates resource owner password credentials flow to request a new ID token
func (mgr *manager) GetTokenByROPC(ctx context.Context, username, password string) (string, error) {
	in := mgr.input
	in.GrantOptionSet.ROPCOption = &ropc.Option{
		Username: username,
		Password: password,
	}
	err := mgr.getter.Do(ctx, in)
	if err != nil {
		return "", err
	}
	return mgr.token, nil
}

func (mgr *manager) TokenExpiry() time.Time {
	return mgr.expiry
}

func (mgr *manager) cacheToken(token string, expiry time.Time) {
	mgr.token = token
	mgr.expiry = expiry
}
