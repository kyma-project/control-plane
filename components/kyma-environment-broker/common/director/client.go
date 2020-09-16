package director

import (
	"context"
	"fmt"
	"net/url"
	"sync"

	"github.com/kyma-incubator/compass/components/director/pkg/graphql"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/director/oauth"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	gcli "github.com/machinebox/graphql"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	// accountIDKey is a header key name for request send by graphQL client
	accountIDKey = "tenant"

	// amount of request attempt to director service
	reqAttempt = 3

	authorizationKey = "Authorization"
)

//go:generate mockery -name=GraphQLClient -output=automock
type GraphQLClient interface {
	Run(ctx context.Context, req *gcli.Request, resp interface{}) error
}

//go:generate mockery -name=OauthClient -output=automock
type OauthClient interface {
	GetAuthorizationToken() (oauth.Token, error)
}

type Client struct {
	graphQLClient GraphQLClient
	oauthClient   OauthClient
	queryProvider queryProvider
	token         oauth.Token
	log           logrus.FieldLogger
}

type (
	getURLResponse struct {
		Result graphql.RuntimeExt `json:"result"`
	}

	runtimeLabelResponse struct {
		Result *graphql.Label `json:"result"`
	}

	getRuntimeIdResponse struct {
		Result graphql.RuntimePageExt `json:"result"`
	}
)

var lock sync.Mutex

// NewDirectorClient returns new director client struct pointer
func NewDirectorClient(oauthClient OauthClient, gqlClient GraphQLClient, log logrus.FieldLogger) *Client {
	return &Client{
		graphQLClient: gqlClient,
		oauthClient:   oauthClient,
		queryProvider: queryProvider{},
		token:         oauth.Token{},
		log:           log,
	}
}

// GetConsoleURL fetches, validates and returns console URL from director component based on runtime ID
func (dc *Client) GetConsoleURL(accountID, runtimeID string) (string, error) {
	query := dc.queryProvider.Runtime(runtimeID)
	req := gcli.NewRequest(query)
	req.Header.Add(accountIDKey, accountID)

	dc.log.Info("Send request to director")
	response, err := dc.fetchURLFromDirector(req)
	if err != nil {
		return "", errors.Wrap(err, "while making call to director")
	}

	dc.log.Info("Extract the URL from the response")
	return dc.getURLFromRuntime(&response.Result)
}

// SetLabel adds key-value label to a Runtime
func (dc *Client) SetLabel(accountID, runtimeID, key, value string) error {
	query := dc.queryProvider.SetRuntimeLabel(runtimeID, key, value)
	req := gcli.NewRequest(query)
	req.Header.Add(accountIDKey, accountID)

	dc.log.Info("Setup label in director")
	response, err := dc.setLabelsInDirector(req)
	if err != nil {
		return errors.Wrapf(err, "while setting %s Runtime label to value %s", key, value)
	}

	if response.Result == nil {
		return errors.Errorf("failed to set %s Runtime label to value %s. Received nil response.", key, value)
	}

	dc.log.Infof("Label %s:%s set correctly", response.Result.Key, response.Result.Value)
	return nil
}

// GetRuntimeID fetches runtime ID with given label name from director component
func (dc *Client) GetRuntimeID(accountID, instanceID string) (string, error) {
	query := dc.queryProvider.RuntimeForInstanceId(instanceID)
	req := gcli.NewRequest(query)
	req.Header.Add(accountIDKey, accountID)

	dc.log.Info("Send request to director")
	response, err := dc.getRuntimeIdFromDirector(req)
	if err != nil {
		return "", err
	}

	dc.log.Info("Extract the RuntimeID from the response")
	return dc.getIDFromRuntime(&response.Result)
}

func (dc *Client) fetchURLFromDirector(req *gcli.Request) (*getURLResponse, error) {
	var response getURLResponse
	var lastError error
	var success bool

	for i := 0; i < reqAttempt; i++ {
		err := dc.setToken()
		if err != nil {
			lastError = err
			dc.log.Errorf("cannot set token to director client (attempt %d): %s", i, err)
			continue
		}
		req.Header.Add(authorizationKey, fmt.Sprintf("Bearer %s", dc.token.AccessToken))
		err = dc.graphQLClient.Run(context.Background(), req, &response)
		if err != nil {
			lastError = kebError.AsTemporaryError(err, "while requesting to director client")
			dc.token.AccessToken = ""
			req.Header.Del(authorizationKey)
			dc.log.Errorf("call to director failed (attempt %d): %s", i, err)
			continue
		}
		success = true
		break
	}

	if !success {
		return &getURLResponse{}, lastError
	}

	return &response, nil
}

func (dc *Client) setLabelsInDirector(req *gcli.Request) (*runtimeLabelResponse, error) {
	var response runtimeLabelResponse
	var lastError error
	var success bool

	for i := 0; i < reqAttempt; i++ {
		err := dc.setToken()
		if err != nil {
			lastError = err
			dc.log.Errorf("cannot set token to director client (attempt %d): %s", i, err)
			continue
		}
		req.Header.Add(authorizationKey, fmt.Sprintf("Bearer %s", dc.token.AccessToken))
		err = dc.graphQLClient.Run(context.Background(), req, &response)
		if err != nil {
			lastError = kebError.AsTemporaryError(err, "while requesting to director client")
			dc.token.AccessToken = ""
			req.Header.Del(authorizationKey)
			dc.log.Errorf("call to director failed (attempt %d): %s", i, err)
			continue
		}
		success = true
		break
	}

	if !success {
		return &runtimeLabelResponse{}, lastError
	}

	return &response, nil
}

func (dc *Client) getRuntimeIdFromDirector(req *gcli.Request) (*getRuntimeIdResponse, error) {
	var response getRuntimeIdResponse
	var lastError error
	var success bool

	for i := 0; i < reqAttempt; i++ {
		err := dc.setToken()
		if err != nil {
			lastError = err
			dc.log.Errorf("cannot set token to director client (attempt %d): %s", i, err)
			continue
		}
		req.Header.Add(authorizationKey, fmt.Sprintf("Bearer %s", dc.token.AccessToken))
		err = dc.graphQLClient.Run(context.Background(), req, &response)
		if err != nil {
			lastError = kebError.AsTemporaryError(err, "while requesting to director client")
			dc.token.AccessToken = ""
			req.Header.Del(authorizationKey)
			dc.log.Errorf("call to director failed (attempt %d): %s", i, err)
			continue
		}
		success = true
		break
	}

	if !success {
		return &getRuntimeIdResponse{}, lastError
	}

	return &response, nil
}

func (dc *Client) setToken() error {
	lock.Lock()
	defer lock.Unlock()
	if !dc.token.EmptyOrExpired() {
		return nil
	}

	token, err := dc.oauthClient.GetAuthorizationToken()
	if err != nil {
		return errors.Wrap(err, "Error while obtaining token")
	}
	dc.token = token

	return nil
}

func (dc *Client) getURLFromRuntime(response *graphql.RuntimeExt) (string, error) {
	if response.Status == nil {
		return "", kebError.NewTemporaryError("response status from director is nil")
	}
	if response.Status.Condition == graphql.RuntimeStatusConditionFailed {
		return "", fmt.Errorf("response status condition from director is %s", graphql.RuntimeStatusConditionFailed)
	}

	value, ok := response.Labels[consoleURLLabelKey]
	if !ok {
		return "", kebError.NewTemporaryError("response label key is not equal to %q", consoleURLLabelKey)
	}

	var URL string
	switch value.(type) {
	case string:
		URL = value.(string)
	default:
		return "", errors.New("response label value is not string")
	}

	_, err := url.ParseRequestURI(URL)
	if err != nil {
		return "", errors.Wrap(err, "while parsing raw URL")
	}

	return URL, nil
}

func (dc *Client) getIDFromRuntime(response *graphql.RuntimePageExt) (string, error) {
	if response.Data == nil || len(response.Data) == 0 || response.Data[0] == nil {
		return "", errors.New("got empty data from director response")
	}
	if len(response.Data) > 1 {
		return "", errors.Errorf("expected single runtime, got: %v", response.Data)
	}
	return response.Data[0].ID, nil
}
