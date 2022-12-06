package ias

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
)

const (
	PathServiceProviders  = "/service/sps"
	PathCompanyGlobal     = "/service/company/global"
	PathAccess            = "/service/sps/%s/rba"
	PathIdentityProviders = "/service/idp"
	PathDelete            = "/service/sps/delete"
	PathDeleteSecret      = "/service/sps/clientSecret"
)

type (
	ClientConfig struct {
		URL    string
		ID     string
		Secret string
	}

	Client struct {
		config         ClientConfig
		httpClient     *http.Client
		closeBodyError error
	}

	Request struct {
		Method  string
		Path    string
		Body    io.Reader
		Headers map[string]string
		Delete  bool
	}
)

func NewClient(cli *http.Client, cfg ClientConfig) *Client {
	return &Client{
		config:     cfg,
		httpClient: cli,
	}
}

func (c *Client) SetOIDCConfiguration(spID string, payload OIDCType) error {
	return c.call(c.serviceProviderPath(spID), payload)
}

func (c *Client) SetSAMLConfiguration(spID string, payload SAMLType) error {
	return c.call(c.serviceProviderPath(spID), payload)
}

func (c *Client) SetAssertionAttribute(spID string, payload PostAssertionAttributes) error {
	return c.call(c.serviceProviderPath(spID), payload)
}

func (c *Client) SetSubjectNameIdentifier(spID string, payload SubjectNameIdentifier) error {
	return c.call(c.serviceProviderPath(spID), payload)
}

func (c *Client) SetAuthenticationAndAccess(spID string, payload AuthenticationAndAccess) error {
	pathAccess := fmt.Sprintf(PathAccess, spID)

	return c.call(pathAccess, payload)
}

func (c *Client) SetDefaultAuthenticatingIDP(payload DefaultAuthIDPConfig) error {
	return c.call(PathServiceProviders, payload)
}

func (c *Client) GetCompany() (_ *Company, err error) {
	company := &Company{}
	request := &Request{Method: http.MethodGet, Path: PathCompanyGlobal}

	response, err := c.do(request)
	defer func() {
		if closeErr := c.closeResponseBody(response); closeErr != nil {
			err = kebError.AsTemporaryError(closeErr, "while closing response body with company data")
		}
	}()
	if err != nil {
		return company, fmt.Errorf("while making request to ias platform about company: %w", err)
	}

	err = json.NewDecoder(response.Body).Decode(company)
	if err != nil {
		return company, fmt.Errorf("while decoding response body with company data: %w", err)
	}

	return company, nil
}

func (c *Client) CreateServiceProvider(serviceName, companyID string) (err error) {
	payload := fmt.Sprintf("sp_name=%s&company_id=%s", serviceName, companyID)
	request := &Request{
		Method:  http.MethodPost,
		Path:    PathServiceProviders,
		Body:    strings.NewReader(payload),
		Headers: map[string]string{"content-type": "application/x-www-form-urlencoded"},
	}

	response, err := c.do(request)
	defer func() {
		if closeErr := c.closeResponseBody(response); closeErr != nil {
			err = kebError.AsTemporaryError(closeErr, "while closing response body for ServiceProvider creation")
		}
	}()
	if err != nil {
		return fmt.Errorf("while making request with ServiceProvider creation: %w", err)
	}

	return nil
}

func (c *Client) DeleteServiceProvider(spID string) (err error) {
	request := &Request{
		Method: http.MethodPut,
		Path:   fmt.Sprintf("%s?sp_id=%s", PathDelete, spID),
		Delete: true,
	}
	response, err := c.do(request)
	defer func() {
		if closeErr := c.closeResponseBody(response); closeErr != nil {
			err = kebError.AsTemporaryError(closeErr, "while closing response body for ServiceProvider deletion")
		}
	}()
	if err != nil {
		return fmt.Errorf("while making request to delete ServiceProvider: %w", err)
	}

	return nil
}

func (c *Client) DeleteSecret(payload SecretsRef) (err error) {
	request, err := c.jsonRequest(PathDeleteSecret, http.MethodDelete, payload)
	if err != nil {
		return fmt.Errorf("while creating json request for path %s: %w", PathDeleteSecret, err)
	}
	request.Delete = true

	response, err := c.do(request)
	defer func() {
		if closeErr := c.closeResponseBody(response); closeErr != nil {
			err = kebError.AsTemporaryError(closeErr, "while closing response body for Secret deletion")
		}
	}()
	if err != nil {
		return fmt.Errorf("while making request to delete ServiceProvider secrets: %w", err)
	}

	return nil
}

func (c *Client) GenerateServiceProviderSecret(secretCfg SecretConfiguration) (_ *ServiceProviderSecret, err error) {
	secretResponse := &ServiceProviderSecret{}
	request, err := c.jsonRequest(PathServiceProviders, http.MethodPut, secretCfg)
	if err != nil {
		return secretResponse, fmt.Errorf("while creating request for secret provider: %w", err)
	}

	response, err := c.do(request)
	defer func() {
		if closeErr := c.closeResponseBody(response); closeErr != nil {
			err = kebError.AsTemporaryError(closeErr, "while closing response body for ServiceProviderSecret generating")
		}
	}()
	if err != nil {
		return secretResponse, fmt.Errorf("while making request to generate ServiceProvider secret: %w", err)
	}

	err = json.NewDecoder(response.Body).Decode(secretResponse)
	if err != nil {
		return secretResponse, fmt.Errorf("while decoding response with secret provider: %w", err)
	}

	return secretResponse, nil
}

func (c Client) AuthenticationURL(id ProviderID) string {
	return fmt.Sprintf("%s%s/%s", c.config.URL, PathIdentityProviders, id)
}

func (c *Client) serviceProviderPath(spID string) string {
	return fmt.Sprintf("%s/%s", PathServiceProviders, spID)
}

func (c *Client) call(path string, payload interface{}) (err error) {
	request, err := c.jsonRequest(path, http.MethodPut, payload)
	if err != nil {
		return fmt.Errorf("while creating json request for path %s: %w", path, err)
	}

	response, err := c.do(request)
	defer func() {
		if closeErr := c.closeResponseBody(response); closeErr != nil {
			err = kebError.AsTemporaryError(closeErr, "while closing response body for call method")
		}
	}()
	if err != nil {
		return fmt.Errorf("while making request for path %s: %w", path, err)
	}

	return nil
}

func (c *Client) jsonRequest(path string, method string, payload interface{}) (*Request, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	err := encoder.Encode(payload)
	if err != nil {
		return &Request{}, err
	}

	return &Request{
		Method:  method,
		Path:    path,
		Body:    buffer,
		Headers: map[string]string{"content-type": "application/json"},
	}, nil
}

func (c *Client) do(sciReq *Request) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", c.config.URL, sciReq.Path)
	req, err := http.NewRequest(sciReq.Method, url, sciReq.Body)
	if err != nil {
		return nil, err
	}

	req.Close = true
	req.SetBasicAuth(c.config.ID, c.config.Secret)
	for h, v := range sciReq.Headers {
		req.Header.Set(h, v)
	}

	response, err := c.httpClient.Do(req)
	if err != nil {
		return &http.Response{}, kebError.AsTemporaryError(err, "while making request")
	}

	switch response.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusNoContent:
		return response, nil
	case http.StatusNotFound:
		if sciReq.Delete {
			return response, nil
		}
	case http.StatusRequestTimeout:
		return response, kebError.NewTemporaryError(c.responseErrorMessage(response))
	}

	if response.StatusCode >= http.StatusInternalServerError {
		return response, kebError.NewTemporaryError(c.responseErrorMessage(response))
	}
	return response, fmt.Errorf("while sending request to IAS: %s", c.responseErrorMessage(response))
}

func (c *Client) closeResponseBody(response *http.Response) error {
	if response == nil {
		return nil
	}
	if response.Body == nil {
		return nil
	}
	return response.Body.Close()
}

func (c *Client) responseErrorMessage(response *http.Response) string {
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Sprintf("unexpected status code %d cannot read body response: %s", response.StatusCode, err)
	}
	return fmt.Sprintf("unexpected status code %d with body: %s", response.StatusCode, string(body))
}
