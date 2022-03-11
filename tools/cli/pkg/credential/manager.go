package credential

import (
	"context"
	"sync"
	"time"

	credentialpluginwriter "github.com/int128/kubelogin/pkg/credentialplugin"
	"github.com/int128/kubelogin/pkg/infrastructure/browser"
	"github.com/int128/kubelogin/pkg/infrastructure/clock"
	"github.com/int128/kubelogin/pkg/infrastructure/mutex"
	"github.com/int128/kubelogin/pkg/infrastructure/reader"
	"github.com/int128/kubelogin/pkg/oidc"
	oidcclient "github.com/int128/kubelogin/pkg/oidc/client"
	tokencache "github.com/int128/kubelogin/pkg/tokencache/repository"
	"github.com/int128/kubelogin/pkg/usecases/authentication"
	"github.com/int128/kubelogin/pkg/usecases/authentication/authcode"
	"github.com/int128/kubelogin/pkg/usecases/authentication/ropc"
	"github.com/int128/kubelogin/pkg/usecases/credentialplugin"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	"golang.org/x/oauth2"
	"k8s.io/client-go/util/homedir"
)

var defaultTokenCacheDir = homedir.HomeDir() + "/.kube/cache/oidc-login"
var defaultListenAddress = []string{"127.0.0.1:8000", "127.0.0.1:18000"}

const defaultAuthenticationTimeout = 180 * time.Second

// Manager is a client for an OIDC provider capable of authenticating users and retrieving ID tokens through
//   - Authorization code grant flow using browser for interactive use
//   - Resource owner password credentials flow for non-interactive use
// Manager implements the oauth2.TokenSource interface to interact with client libraries depending on the oauth2 package for obtaining auth token.
type Manager interface {
	GetTokenByAuthCode(ctx context.Context) (string, error)
	GetTokenByROPC(ctx context.Context, username, password string) (string, error)
	TokenExpiry() time.Time
	Token() (*oauth2.Token, error)
}

type manager struct {
	getter   *credentialplugin.GetToken
	input    credentialplugin.Input
	token    string
	expiry   time.Time
	mux      sync.Mutex
	username string
}

type tokenWriter struct {
	mgr *manager
}

func (w *tokenWriter) Write(out credentialpluginwriter.Output) error {
	w.mgr.cacheToken(out.Token, out.Expiry)
	return nil
}

// NewManager Constructs a new credential.Manager using the given OIDC provider and client credentials
func NewManager(oidcIssuerURL, oidcClientID, oidcClientSecret, username string, log logger.Logger) Manager {
	clock := &clock.Real{}
	reader := &reader.Reader{}
	auth := &authentication.Authentication{
		Clock:  clock,
		Logger: log,
		ClientFactory: &oidcclient.Factory{
			Clock:  clock,
			Logger: log,
		},
		AuthCodeBrowser: &authcode.Browser{
			Logger:  log,
			Browser: &browser.Browser{},
		},
		AuthCodeKeyboard: &authcode.Keyboard{
			Logger: log,
			Reader: reader,
		},
		ROPC: &ropc.ROPC{
			Logger: log,
			Reader: reader,
		},
	}

	mgr := &manager{
		username: username,
		input: credentialplugin.Input{
			Provider: oidc.Provider{
				IssuerURL:    oidcIssuerURL,
				ClientID:     oidcClientID,
				ClientSecret: oidcClientSecret,
				UsePKCE:      oidcClientSecret == "",
				ExtraScopes:  []string{"email", "openid"},
			},
			TokenCacheDir: defaultTokenCacheDir,
		},
	}
	writer := &tokenWriter{mgr: mgr}
	getToken := &credentialplugin.GetToken{
		Logger:               log,
		Authentication:       auth,
		TokenCacheRepository: &tokencache.Repository{},
		Writer:               writer,
		Mutex: &mutex.Mutex{
			Logger: log,
		},
	}
	mgr.getter = getToken

	return mgr
}

// GetTokenByAuthCode fetches an ID token from local cache if a valid token is found, or else initiates interactive authorization code grant flow with browser to request a new ID token
func (mgr *manager) GetTokenByAuthCode(ctx context.Context) (string, error) {
	in := mgr.input
	in.GrantOptionSet.AuthCodeBrowserOption = &authcode.BrowserOption{
		BindAddress:           defaultListenAddress,
		SkipOpenBrowser:       false,
		AuthenticationTimeout: defaultAuthenticationTimeout,
		RedirectURLHostname:   "localhost",
	}
	mgr.mux.Lock()
	defer mgr.mux.Unlock()
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
	mgr.mux.Lock()
	defer mgr.mux.Unlock()
	err := mgr.getter.Do(ctx, in)
	if err != nil {
		return "", err
	}
	return mgr.token, nil
}

// Token uses auth code grant flow to obtain an ID token in oauth2.Token format. This method implements the oauth2.TokenSource interface
func (mgr *manager) Token() (*oauth2.Token, error) {
	in := mgr.input
	if mgr.username != "" {
		in.GrantOptionSet.ROPCOption = &ropc.Option{
			Username: mgr.username,
		}
	} else {
		in.GrantOptionSet.AuthCodeBrowserOption = &authcode.BrowserOption{
			BindAddress:           defaultListenAddress,
			SkipOpenBrowser:       false,
			AuthenticationTimeout: defaultAuthenticationTimeout,
			RedirectURLHostname:   "localhost",
		}
	}

	mgr.mux.Lock()
	defer mgr.mux.Unlock()
	err := mgr.getter.Do(context.TODO(), in)
	if err != nil {
		return nil, err
	}
	return &oauth2.Token{AccessToken: mgr.token, Expiry: mgr.expiry}, nil
}

func (mgr *manager) TokenExpiry() time.Time {
	mgr.mux.Lock()
	defer mgr.mux.Unlock()
	return mgr.expiry
}

func (mgr *manager) cacheToken(token string, expiry time.Time) {
	mgr.token = token
	mgr.expiry = expiry
}
