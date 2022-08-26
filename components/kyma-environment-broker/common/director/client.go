package director

import (
	"context"
	"time"

	"github.com/kyma-incubator/compass/components/director/pkg/graphql"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	machineGraph "github.com/machinebox/graphql"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	// accountIDKey is a header key name for request send by graphQL client
	accountIDKey = "tenant"
)

//go:generate mockery --name=GraphQLClient --output=automock
type GraphQLClient interface {
	Run(ctx context.Context, req *machineGraph.Request, resp interface{}) error
}

type Client struct {
	graphQLClient GraphQLClient
	queryProvider queryProvider
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

// NewDirectorClient returns new director client struct pointer
func NewDirectorClient(ctx context.Context, config Config, log logrus.FieldLogger) *Client {
	cfg := clientcredentials.Config{
		ClientID:     config.OauthClientID,
		ClientSecret: config.OauthClientSecret,
		TokenURL:     config.OauthTokenURL,
		Scopes:       []string{config.OauthScope},
	}
	httpClientOAuth := cfg.Client(ctx)
	httpClientOAuth.Timeout = 30 * time.Second

	graphQLClient := machineGraph.NewClient(config.URL, machineGraph.WithHTTPClient(httpClientOAuth))

	return &Client{
		graphQLClient: graphQLClient,
		queryProvider: queryProvider{},
		log:           log,
	}
}

// SetLabel adds key-value label to a Runtime
func (dc *Client) SetLabel(accountID, runtimeID, key, value string) error {
	query := dc.queryProvider.SetRuntimeLabel(runtimeID, key, value)
	req := machineGraph.NewRequest(query)
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
	req := machineGraph.NewRequest(query)
	req.Header.Add(accountIDKey, accountID)

	dc.log.Info("Send request to director")
	response, err := dc.getRuntimeIdFromDirector(req)
	if err != nil {
		return "", err
	}

	dc.log.Info("Extract the RuntimeID from the response")
	return dc.getIDFromRuntime(&response.Result)
}

func (dc *Client) fetchURLFromDirector(req *machineGraph.Request) (*getURLResponse, error) {
	var response getURLResponse

	err := dc.graphQLClient.Run(context.Background(), req, &response)
	if err != nil {
		dc.log.Errorf("call to director failed: %s", err)
		return &getURLResponse{}, kebError.AsTemporaryError(err, "while requesting to director client")
	}

	return &response, nil
}

func (dc *Client) setLabelsInDirector(req *machineGraph.Request) (*runtimeLabelResponse, error) {
	var response runtimeLabelResponse

	err := dc.graphQLClient.Run(context.Background(), req, &response)
	if err != nil {
		dc.log.Errorf("call to director failed: %s", err)
		return &runtimeLabelResponse{}, kebError.AsTemporaryError(err, "while requesting to director client")
	}

	return &response, nil
}

func (dc *Client) getRuntimeIdFromDirector(req *machineGraph.Request) (*getRuntimeIdResponse, error) {
	var response getRuntimeIdResponse

	err := dc.graphQLClient.Run(context.Background(), req, &response)
	if err != nil {
		dc.log.Errorf("call to director failed: %s", err)
		return &getRuntimeIdResponse{}, kebError.AsTemporaryError(err, "while requesting to director client")
	}

	return &response, nil
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
