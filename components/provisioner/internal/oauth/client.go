package oauth

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"

	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	log "github.com/sirupsen/logrus"
)

//go:generate mockery --name=Client
type Client interface {
	GetAuthorizationToken() (Token, apperrors.AppError)
}

type oauthClient struct {
	httpClient *http.Client
	creds      credentials
}

func NewOauthClient(client *http.Client, clientID, clientSecret, tokensEndpoint string) Client {
	return &oauthClient{
		httpClient: client,
		creds: credentials{
			clientID:       clientID,
			clientSecret:   clientSecret,
			tokensEndpoint: tokensEndpoint,
		},
	}
}

func (c *oauthClient) GetAuthorizationToken() (Token, apperrors.AppError) {
	return c.getAuthorizationToken(c.creds)
}

func (c *oauthClient) getAuthorizationToken(credentials credentials) (Token, apperrors.AppError) {
	log.Infof("Getting authorisation token for credentials to access Director from endpoint: %s", credentials.tokensEndpoint)

	form := url.Values{}
	form.Add(grantTypeFieldName, credentialsGrantType)
	form.Add(scopeFieldName, scopes)

	request, err := http.NewRequest(http.MethodPost, credentials.tokensEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		log.Errorf("Failed to create authorisation token request")
		return Token{}, apperrors.Internal("Failed to create authorisation token request: %s", err.Error())
	}

	now := time.Now().Unix()

	request.SetBasicAuth(credentials.clientID, credentials.clientSecret)
	request.Header.Set(contentTypeHeader, contentTypeApplicationURLEncoded)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return Token{}, apperrors.Internal("Failed to execute http call: %s", err.Error())
	}
	defer util.Close(response.Body)

	if response.StatusCode != http.StatusOK {
		dump, err := httputil.DumpResponse(response, true)
		if err != nil {
			dump = []byte("failed to dump response body")
		}
		return Token{}, apperrors.External("Get token call returned unexpected status: %s. Response dump: %s", response.Status, string(dump)).SetComponent(apperrors.ErrMpsOAuth2)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return Token{}, apperrors.Internal("Failed to read token response body from '%s': %s", credentials.tokensEndpoint, err.Error())
	}

	tokenResponse := Token{}

	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		return Token{}, apperrors.Internal("failed to unmarshal token response body: %s", err.Error())
	}

	log.Infof("Successfully unmarshal response oauth token for accessing Director")

	tokenResponse.Expiration += now

	return tokenResponse, nil
}
