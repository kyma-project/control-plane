package avs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

type Client struct {
	httpClient *http.Client
	avsConfig  Config
}

func NewClient(ctx context.Context, avsConfig Config) (*Client, error) {
	config, initialToken, err := createInitialToken(avsConfig)
	if err != nil {
		return nil, errors.Wrap(err, "while creating oauth config and token")
	}

	return &Client{
		httpClient: config.Client(ctx, initialToken),
		avsConfig:  avsConfig,
	}, nil
}

func createInitialToken(cfg Config) (*oauth2.Config, *oauth2.Token, error) {
	config := &oauth2.Config{
		ClientID: cfg.OauthClientId,
		Endpoint: oauth2.Endpoint{
			TokenURL:  cfg.OauthTokenEndpoint,
			AuthStyle: oauth2.AuthStyleInHeader,
		},
	}

	initialToken, err := config.PasswordCredentialsToken(context.TODO(), cfg.OauthUsername, cfg.OauthPassword)
	if err != nil {
		return nil, nil, errors.Wrap(err, "while fetching initial token")
	}

	return config, initialToken, nil
}

func (c *Client) resetHTTPClient() error {
	config, initialToken, err := createInitialToken(c.avsConfig)
	if err != nil {
		return errors.Wrap(err, "while resetting initial token")
	}
	c.httpClient = config.Client(context.TODO(), initialToken)

	return nil
}

func (c *Client) CreateEvaluation(evaluationRequest *BasicEvaluationCreateRequest) (*BasicEvaluationCreateResponse, error) {
	var responseObject BasicEvaluationCreateResponse

	objAsBytes, err := json.Marshal(evaluationRequest)
	if err != nil {
		return &responseObject, errors.Wrap(err, "while marshaling evaluation request")
	}

	request, err := http.NewRequest(http.MethodPost, c.avsConfig.ApiEndpoint, bytes.NewReader(objAsBytes))
	if err != nil {
		return &responseObject, errors.Wrap(err, "while creating request")
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.execute(request, false, true)
	if err != nil {
		return &responseObject, errors.Wrap(err, "while executing CreateEvaluation request")
	}

	err = json.NewDecoder(response.Body).Decode(&responseObject)
	if err != nil {
		return nil, errors.Wrap(err, "while decode create evaluation response")
	}

	if err := response.Body.Close(); err != nil {
		return &responseObject, errors.Wrap(err, "while closing CreateEvaluation response")
	}

	return &responseObject, nil
}

func (c *Client) RemoveReferenceFromParentEval(evaluationId int64) error {
	absoluteURL := fmt.Sprintf("%s/child/%d", appendId(c.avsConfig.ApiEndpoint, c.avsConfig.ParentId), evaluationId)
	response, err := c.deleteRequest(absoluteURL)
	if err == nil {
		return nil
	}

	if response != nil {
		defer response.Body.Close()
		var responseObject avsNonSuccessResp
		err := json.NewDecoder(response.Body).Decode(&responseObject)
		if err != nil {
			return errors.Wrapf(err, "while decoding avs non success response body for ID: %d", evaluationId)
		}

		if strings.Contains(strings.ToLower(responseObject.Message), "does not contain subevaluation") {
			return nil
		}
	}

	return fmt.Errorf("unexpected response for evaluationId: %d while deleting reference from parent evaluation, error: %s", evaluationId, err)
}

func (c *Client) DeleteEvaluation(evaluationId int64) error {
	absoluteURL := appendId(c.avsConfig.ApiEndpoint, evaluationId)
	response, err := c.deleteRequest(absoluteURL)
	if err != nil {
		return errors.Wrap(err, "while deleting evaluation")
	}

	if err := response.Body.Close(); err != nil {
		return errors.Wrap(err, "while closing DeleteEvaluation response body")
	}

	return nil
}

func appendId(baseUrl string, id int64) string {
	if strings.HasSuffix(baseUrl, "/") {
		return baseUrl + strconv.FormatInt(id, 10)
	} else {
		return baseUrl + "/" + strconv.FormatInt(id, 10)
	}
}

func (c *Client) deleteRequest(absoluteURL string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodDelete, absoluteURL, nil)
	if err != nil {
		return &http.Response{}, errors.Wrap(err, "while creating delete request")
	}

	response, err := c.execute(req, true, true)
	if err != nil {
		return &http.Response{}, errors.Wrapf(err, "while executing delete request for path: %s", absoluteURL)
	}

	return response, nil
}

func (c *Client) execute(request *http.Request, allowNotFound bool, allowResetToken bool) (*http.Response, error) {
	response, err := c.httpClient.Do(request)
	if err != nil {
		return &http.Response{}, errors.Wrap(err, "while executing request by http client")
	}

	switch response.StatusCode {
	case http.StatusOK, http.StatusCreated:
		return response, nil
	case http.StatusNotFound:
		if allowNotFound {
			return response, nil
		}
		return response, fmt.Errorf("response status code: %d for %s", http.StatusNotFound, request.URL.String())
	case http.StatusUnauthorized:
		if allowResetToken {
			if err := c.resetHTTPClient(); err != nil {
				return response, errors.Wrap(err, "while resetting http auth client")
			}
			return c.execute(request, allowNotFound, false)
		}
		return response, fmt.Errorf("avs server returned %d status code twice for %s (response body: %s)", http.StatusUnauthorized, request.URL.String(), responseBody(response))
	default:
		return response, fmt.Errorf("unsupported status code: %d for %s (response body: %s)", response.StatusCode, request.URL.String(), responseBody(response))
	}
}

func responseBody(resp *http.Response) string {
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	return string(bodyBytes)
}
