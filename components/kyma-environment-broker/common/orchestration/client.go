package orchestration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/pagination"
	"golang.org/x/oauth2"
)

const defaultPageSize = 100

// Client is the interface to interact with the KEB /orchestrations and /upgrade API
// as an HTTP client using OIDC ID token in JWT format.
type Client interface {
	ListOrchestrations(params ListParameters) (StatusResponseList, error)
	GetOrchestration(orchestrationID string) (StatusResponse, error)
	ListOperations(orchestrationID string, params ListParameters) (OperationResponseList, error)
	GetOperation(orchestrationID, operationID string) (OperationDetailResponse, error)
	UpgradeKyma(params Parameters) (UpgradeResponse, error)
	UpgradeCluster(params Parameters) (UpgradeResponse, error)
	CancelOrchestration(orchestrationID string) error
	RetryOrchestration(orchestrationID string, operationIDs []string, now bool) (RetryResponse, error)
}

type client struct {
	url        string
	httpClient *http.Client
}

// NewClient constructs and returns new Client for KEB /runtimes API
// It takes the following arguments:
//   - ctx  : context in which the http request will be executed
//   - url  : base url of all KEB APIs, e.g. https://kyma-env-broker.kyma.local
//   - auth : TokenSource object which provides the ID token for the HTTP request
func NewClient(ctx context.Context, url string, auth oauth2.TokenSource) Client {
	return &client{
		url:        url,
		httpClient: oauth2.NewClient(ctx, auth),
	}
}

// ListOrchestrations fetches the orchestrations from KEB according to the given params.
// If params.Page or params.PageSize is not set (zero), the client will fetch and return all orchestrations.
func (c client) ListOrchestrations(params ListParameters) (StatusResponseList, error) {
	orchestrations := StatusResponseList{}
	getAll := false
	fetchedAll := false
	if params.Page == 0 || params.PageSize == 0 {
		getAll = true
		params.Page = 1
		if params.PageSize == 0 {
			params.PageSize = defaultPageSize
		}
	}

	for !fetchedAll {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/orchestrations", c.url), nil)
		if err != nil {
			return orchestrations, fmt.Errorf("while creating request: %w", err)
		}
		setQuery(req.URL, params)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return orchestrations, fmt.Errorf("while calling %s: %w", req.URL.String(), err)
		}

		// Drain response body and close, return error to context if there isn't any.
		defer func() {
			derr := drainResponseBody(resp.Body)
			if err == nil {
				err = derr
			}
			cerr := resp.Body.Close()
			if err == nil {
				err = cerr
			}
		}()

		if resp.StatusCode != http.StatusOK {
			return orchestrations, fmt.Errorf("calling %s returned %s status", req.URL.String(), resp.Status)
		}

		var srl StatusResponseList
		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(&srl)
		if err != nil {
			return orchestrations, fmt.Errorf("while decoding response body: %w", err)
		}

		orchestrations.TotalCount = srl.TotalCount
		orchestrations.Count += srl.Count
		orchestrations.Data = append(orchestrations.Data, srl.Data...)
		if getAll {
			params.Page++
			fetchedAll = orchestrations.Count >= orchestrations.TotalCount
		} else {
			fetchedAll = true
		}
	}

	return orchestrations, nil
}

// GetOrchestration fetches one orchestration by the given ID.
func (c client) GetOrchestration(orchestrationID string) (StatusResponse, error) {
	orchestration := StatusResponse{}
	url := fmt.Sprintf("%s/orchestrations/%s", c.url, orchestrationID)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return orchestration, fmt.Errorf("while calling %s: %w", url, err)
	}

	// Drain response body and close, return error to context if there isn't any.
	defer func() {
		derr := drainResponseBody(resp.Body)
		if err == nil {
			err = derr
		}
		cerr := resp.Body.Close()
		if err == nil {
			err = cerr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return orchestration, fmt.Errorf("calling %s returned %s status", url, resp.Status)
	}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&orchestration)
	if err != nil {
		return orchestration, fmt.Errorf("while decoding response body: %w", err)
	}

	return orchestration, nil
}

// ListOperations fetches the Runtime operations of a given orchestration from KEB according to the given params.
// If params.Page or params.PageSize is not set (zero), the client will fetch and return all operations.
func (c client) ListOperations(orchestrationID string, params ListParameters) (OperationResponseList, error) {
	operations := OperationResponseList{}
	url := fmt.Sprintf("%s/orchestrations/%s/operations", c.url, orchestrationID)
	getAll := false
	fetchedAll := false
	if params.Page == 0 || params.PageSize == 0 {
		getAll = true
		params.Page = 1
		if params.PageSize == 0 {
			params.PageSize = defaultPageSize
		}
	}

	for !fetchedAll {
		if params.Page > 1 {
			failedFound, failedIndex := c.searchFilter(params.States, "failed")
			if failedFound {
				params.States = c.removeIndex(params.States, failedIndex)
			}
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return operations, fmt.Errorf("while creating request: %w", err)
		}
		setQuery(req.URL, params)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return operations, fmt.Errorf("while calling %s: %w", url, err)
		}

		// Drain response body and close, return error to context if there isn't any.
		defer func() {
			derr := drainResponseBody(resp.Body)
			if err == nil {
				err = derr
			}
			cerr := resp.Body.Close()
			if err == nil {
				err = cerr
			}
		}()

		if resp.StatusCode != http.StatusOK {
			return operations, fmt.Errorf("calling %s returned %s status", url, resp.Status)
		}

		var orl OperationResponseList
		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(&orl)
		if err != nil {
			return operations, fmt.Errorf("while decoding response body: %w", err)
		}

		operations.TotalCount = orl.TotalCount
		operations.Count += orl.Count

		operations.Data = append(operations.Data, orl.Data...)
		if getAll {
			params.Page++
			fetchedAll = operations.Count >= operations.TotalCount
		} else {
			fetchedAll = true
		}
	}

	return operations, nil
}

func (c client) searchFilter(states []string, inputState string) (bool, int) {
	var failedFound bool
	var failedIndex int
	for index, state := range states {
		if strings.Contains(state, inputState) {
			failedFound = true
			failedIndex = index
			break
		}
	}
	return failedFound, failedIndex
}

func (c client) removeIndex(arr []string, index int) []string {
	var temp = make([]string, len(arr)-1)
	j := 0
	for i := range arr {
		if i != index {
			temp[j] = arr[i]
			j = j + 1
		}
	}
	return temp
}

// GetOperation fetches detailed Runtime operation corresponding to the given orchestration and operation ID.
func (c client) GetOperation(orchestrationID, operationID string) (OperationDetailResponse, error) {
	operation := OperationDetailResponse{}
	url := fmt.Sprintf("%s/orchestrations/%s/operations/%s", c.url, orchestrationID, operationID)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return operation, fmt.Errorf("while calling %s: %w", url, err)
	}

	// Drain response body and close, return error to context if there isn't any.
	defer func() {
		derr := drainResponseBody(resp.Body)
		if err == nil {
			err = derr
		}
		cerr := resp.Body.Close()
		if err == nil {
			err = cerr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return operation, fmt.Errorf("calling %s returned %s status", url, resp.Status)
	}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&operation)
	if err != nil {
		return operation, fmt.Errorf("while decoding response body: %w", err)
	}

	return operation, nil
}

// UpgradeKyma creates a new Kyma upgrade orchestration according to the given orchestration parameters.
// If successful, the UpgradeResponse returned contains the ID of the newly created orchestration.
func (c client) UpgradeKyma(params Parameters) (UpgradeResponse, error) {
	uri := "/upgrade/kyma"

	ur, err := c.upgradeOperation(uri, params)
	if err != nil {
		return ur, fmt.Errorf("while calling kyma upgrade operation: %w", err)
	}

	return ur, nil
}

// UpgradeCluster creates a new Cluster upgrade orchestration according to the given orchestration parameters.
// If successful, the UpgradeResponse returned contains the ID of the newly created orchestration.
func (c client) UpgradeCluster(params Parameters) (UpgradeResponse, error) {
	uri := "/upgrade/cluster"

	ur, err := c.upgradeOperation(uri, params)
	if err != nil {
		return ur, fmt.Errorf("while calling cluster upgrade operation: %w", err)
	}

	return ur, nil
}

// common func trigger kyma or cluster upgrade
func (c client) upgradeOperation(uri string, params Parameters) (UpgradeResponse, error) {
	ur := UpgradeResponse{}
	blob, err := json.Marshal(params)
	if err != nil {
		return ur, fmt.Errorf("while converting upgrade parameters to JSON: %w", err)
	}

	u, err := url.Parse(c.url)
	if err != nil {
		return ur, fmt.Errorf("while parsing %s: %w", c.url, err)
	}
	u.Path = path.Join(u.Path, uri)

	resp, err := c.httpClient.Post(u.String(), "application/json", bytes.NewBuffer(blob))
	if err != nil {
		return ur, fmt.Errorf("while calling %s: %w", u, err)
	}

	// Drain response body and close, return error to context if there isn't any.
	defer func() {
		derr := drainResponseBody(resp.Body)
		if err == nil {
			err = derr
		}
		cerr := resp.Body.Close()
		if err == nil {
			err = cerr
		}
	}()

	if resp.StatusCode != http.StatusAccepted {
		return ur, fmt.Errorf("calling %s returned %s status", u, resp.Status)
	}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&ur)
	if err != nil {
		return ur, fmt.Errorf("while decoding response body: %w", err)
	}

	return ur, nil
}

func (c client) RetryOrchestration(orchestrationID string, operationIDs []string, now bool) (RetryResponse, error) {
	rr := RetryResponse{}
	uri := fmt.Sprintf("%s/orchestrations/%s/retry", c.url, orchestrationID)

	for i, id := range operationIDs {
		operationIDs[i] = "operation-id=" + id
	}

	str := strings.Join(operationIDs, "&")
	if now {
		str = str + "&immediate=true"
	}
	body := strings.NewReader(str)

	req, err := http.NewRequest(http.MethodPost, uri, body)
	if err != nil {
		return rr, fmt.Errorf("while creating retry request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return rr, fmt.Errorf("while calling %s: %w", uri, err)
	}

	// Drain response body and close, return error to context if there isn't any.
	defer func() {
		derr := drainResponseBody(resp.Body)
		if err == nil {
			err = derr
		}
		cerr := resp.Body.Close()
		if err == nil {
			err = cerr
		}
	}()

	if resp.StatusCode != http.StatusAccepted {
		return rr, fmt.Errorf("calling %s returned %s status", uri, resp.Status)
	}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&rr)
	if err != nil {
		return rr, fmt.Errorf("while decoding response body: %w", err)
	}

	return rr, nil
}

func (c client) CancelOrchestration(orchestrationID string) error {
	url := fmt.Sprintf("%s/orchestrations/%s/cancel", c.url, orchestrationID)

	req, err := http.NewRequest(http.MethodPut, url, nil)
	if err != nil {
		return fmt.Errorf("while creating cancel request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("while calling %s: %w", url, err)
	}

	// Drain response body and close, return error to context if there isn't any.
	defer func() {
		derr := drainResponseBody(resp.Body)
		if err == nil {
			err = derr
		}
		cerr := resp.Body.Close()
		if err == nil {
			err = cerr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("calling %s returned %s status", url, resp.Status)
	}

	return nil
}

func setQuery(url *url.URL, params ListParameters) {
	query := url.Query()
	query.Add(pagination.PageParam, strconv.Itoa(params.Page))
	query.Add(pagination.PageSizeParam, strconv.Itoa(params.PageSize))
	setParamList(query, StateParam, params.States)
	url.RawQuery = query.Encode()
}

func setParamList(query url.Values, key string, values []string) {
	for _, value := range values {
		query.Add(key, value)
	}
}

func drainResponseBody(body io.Reader) error {
	if body == nil {
		return nil
	}
	_, err := io.Copy(ioutil.Discard, io.LimitReader(body, 4096))
	return err
}
