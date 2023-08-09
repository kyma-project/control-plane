package avs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type Client struct {
	httpClient *http.Client
	avsConfig  Config
	log        logrus.FieldLogger
	ctx        context.Context
}

func NewClient(ctx context.Context, avsConfig Config, log logrus.FieldLogger) (*Client, error) {
	return &Client{
		avsConfig: avsConfig,
		log:       log,
		ctx:       ctx,
	}, nil
}

func (c *Client) CreateEvaluation(evaluationRequest *BasicEvaluationCreateRequest) (*BasicEvaluationCreateResponse, error) {
	var responseObject BasicEvaluationCreateResponse

	objAsBytes, err := json.Marshal(evaluationRequest)
	if err != nil {
		return &responseObject, fmt.Errorf("while marshaling evaluation request: %w", err)
	}

	request, err := http.NewRequest(http.MethodPost, c.avsConfig.ApiEndpoint, bytes.NewReader(objAsBytes))
	if err != nil {
		return &responseObject, fmt.Errorf("while creating request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.execute(request, false, true)
	if err != nil {
		return &responseObject, fmt.Errorf("while executing CreateEvaluation request: %w", err)
	}
	defer func() {
		if closeErr := c.closeResponseBody(response); closeErr != nil {
			err = kebError.AsTemporaryError(closeErr, "while closing CreateEvaluation response")
		}
	}()

	err = json.NewDecoder(response.Body).Decode(&responseObject)
	if err != nil {
		return nil, fmt.Errorf("while decode create evaluation response: %w", err)
	}

	return &responseObject, nil
}

func (c *Client) GetEvaluation(evaluationID int64) (*BasicEvaluationCreateResponse, error) {
	var responseObject BasicEvaluationCreateResponse
	absoluteURL := appendId(c.avsConfig.ApiEndpoint, evaluationID)

	request, err := http.NewRequest(http.MethodGet, absoluteURL, nil)
	if err != nil {
		return &responseObject, fmt.Errorf("while creating request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.execute(request, false, true)
	if err != nil {
		return &responseObject, fmt.Errorf("while executing GetEvaluation request: %w", err)
	}
	defer func() {
		if closeErr := c.closeResponseBody(response); closeErr != nil {
			err = kebError.AsTemporaryError(closeErr, "while closing GetEvaluation response")
		}
	}()

	err = json.NewDecoder(response.Body).Decode(&responseObject)
	if err != nil {
		return nil, fmt.Errorf("while decode create evaluation response: %w", err)
	}

	return &responseObject, nil
}

func (c *Client) AddTag(evaluationID int64, tag *Tag) (*BasicEvaluationCreateResponse, error) {
	var responseObject BasicEvaluationCreateResponse

	objAsBytes, err := json.Marshal(tag)
	if err != nil {
		return &responseObject, fmt.Errorf("while marshaling AddTag request: %w", err)
	}
	absoluteURL := appendId(c.avsConfig.ApiEndpoint, evaluationID)

	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/tag", absoluteURL), bytes.NewReader(objAsBytes))
	if err != nil {
		return &responseObject, fmt.Errorf("while creating AddTag request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.execute(request, false, true)
	if err != nil {
		return &responseObject, fmt.Errorf("while executing AddTag request: %w", err)
	}
	defer func() {
		if closeErr := c.closeResponseBody(response); closeErr != nil {
			err = kebError.AsTemporaryError(closeErr, "while closing AddTag response")
		}
	}()

	err = json.NewDecoder(response.Body).Decode(&responseObject)
	if err != nil {
		return nil, fmt.Errorf("while decode AddTag response: %w", err)
	}

	return &responseObject, nil
}

func (c *Client) SetStatus(evaluationID int64, status string) (*BasicEvaluationCreateResponse, error) {
	var responseObject BasicEvaluationCreateResponse

	objAsBytes, err := json.Marshal(status)
	if err != nil {
		return &responseObject, fmt.Errorf("while marshaling SetStatus request: %w", err)
	}
	absoluteURL := appendId(c.avsConfig.ApiEndpoint, evaluationID)

	request, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/lifecycle", absoluteURL), bytes.NewReader(objAsBytes))
	if err != nil {
		return &responseObject, fmt.Errorf("while creating SetStatus request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.execute(request, true, true)
	if err != nil {
		return &responseObject, fmt.Errorf("while executing SetStatus request: %w", err)
	}
	defer func() {
		if closeErr := c.closeResponseBody(response); closeErr != nil {
			err = kebError.AsTemporaryError(closeErr, "while closing SetStatus response")
		}
	}()

	err = json.NewDecoder(response.Body).Decode(&responseObject)
	if err != nil {
		return nil, fmt.Errorf("while decode SetStatus response: %w", err)
	}

	return &responseObject, nil
}

func (c *Client) RemoveReferenceFromParentEval(parentID, evaluationID int64) (err error) {
	absoluteURL := fmt.Sprintf("%s/child/%d", appendId(c.avsConfig.ApiEndpoint, parentID), evaluationID)
	response, err := c.deleteRequest(absoluteURL)
	if err == nil {
		return nil
	}

	if response != nil && response.StatusCode == http.StatusBadRequest {
		defer func() {
			if closeErr := c.closeResponseBody(response); closeErr != nil {
				err = kebError.AsTemporaryError(closeErr, "while closing body")
			}
		}()
		buff, err := io.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("unable to read the response body: %w", err)
		}
		var responseObject avsApiErrorResp
		err = json.NewDecoder(bytes.NewReader(buff)).Decode(&responseObject)
		if err != nil {
			return fmt.Errorf("while decoding AvS non success response body for ID: %d, URL: %s, error: %w",
				evaluationID, absoluteURL, err)
		}
		if strings.Contains(strings.ToLower(responseObject.Message), "does not contain subevaluation") {
			return nil
		}
		return fmt.Errorf("unable to delete subevaluation %d reference from the parent evaluation: %s", evaluationID, responseObject.Message)
	}
	return fmt.Errorf("unexpected response for evaluationId: %d while deleting reference from parent evaluation, error: %w", evaluationID, err)
}

func (c *Client) DeleteEvaluation(evaluationId int64) (err error) {
	absoluteURL := appendId(c.avsConfig.ApiEndpoint, evaluationId)
	response, err := c.deleteRequest(absoluteURL)
	defer func() {
		if closeErr := c.closeResponseBody(response); closeErr != nil {
			err = kebError.AsTemporaryError(closeErr, "while closing DeleteEvaluation response body")
		}
	}()
	if err != nil {
		return fmt.Errorf("while deleting evaluation: %w", err)
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
		return nil, fmt.Errorf("while creating delete request: %w", err)
	}

	response, err := c.execute(req, true, true)
	if err != nil {
		return response, fmt.Errorf("while executing delete request for path: %s: %w", absoluteURL, err)
	}

	return response, nil
}

func (c *Client) execute(request *http.Request, allowNotFound bool, allowResetToken bool) (*http.Response, error) {
	httpClient, err := getHttpClient(c.ctx, c.avsConfig)
	if err != nil {
		return &http.Response{}, fmt.Errorf("while getting http client: %w", err)
	}
	defer httpClient.CloseIdleConnections()
	response, err := httpClient.Do(request)
	if err != nil {
		return &http.Response{}, kebError.AsTemporaryError(err, "while executing request by http client")
	}

	if response.StatusCode >= http.StatusInternalServerError {
		return response, kebError.WrapNewTemporaryError(NewAvsError("avs server returned %d status code", response.StatusCode))
	}

	switch response.StatusCode {
	case http.StatusOK, http.StatusCreated:
		return response, nil
	case http.StatusNotFound:
		if allowNotFound {
			return response, nil
		}
		return response, NewAvsError("response status code: %d for %s", http.StatusNotFound, request.URL.String())
	case http.StatusUnauthorized:
		if allowResetToken {
			return c.execute(request, allowNotFound, false)
		}
		return response, NewAvsError("avs server returned %d status code twice for %s", http.StatusUnauthorized, request.URL.String())
	}

	return response, NewAvsError("unsupported status code: %d for %s.", response.StatusCode, request.URL.String())
}

func (c *Client) closeResponseBody(response *http.Response) error {
	if response == nil {
		return nil
	}
	if response.Body == nil {
		return nil
	}
	// drain the body to let the transport reuse the connection
	io.Copy(io.Discard, response.Body)

	return response.Body.Close()
}

func responseBody(resp *http.Response) string {
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	return string(bodyBytes)
}

func getHttpClient(ctx context.Context, cfg Config) (http.Client, error) {
	config := oauth2.Config{
		ClientID: cfg.OauthClientId,
		Endpoint: oauth2.Endpoint{
			TokenURL:  cfg.OauthTokenEndpoint,
			AuthStyle: oauth2.AuthStyleInHeader,
		},
	}

	initialToken, err := config.PasswordCredentialsToken(ctx, cfg.OauthUsername, cfg.OauthPassword)
	if err != nil {
		return http.Client{}, kebError.AsTemporaryError(err, "while fetching initial token")
	}

	return *config.Client(ctx, initialToken), nil
}
