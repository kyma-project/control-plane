package cis

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	eventServicePathVer1 = "%s/public/rest/v2/events"
	eventTypeVer1        = "MASTER_SUBACCOUNT_DELETION"
	defaultPageSizeVer1  = "1000"
)

type ClientVer1 struct {
	httpClient *http.Client
	config     Config
	log        logrus.FieldLogger
}

func NewClientVer1(ctx context.Context, config Config, log logrus.FieldLogger) *ClientVer1 {
	cfg := clientcredentials.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		TokenURL:     config.AuthURL,
	}
	httpClientOAuth := cfg.Client(ctx)

	if config.PageSize == "" {
		config.PageSize = defaultPageSizeVer1
	}

	return &ClientVer1{
		httpClient: httpClientOAuth,
		config:     config,
		log:        log.WithField("client", "CIS-1.0"),
	}
}

// SetHttpClient auxiliary method of testing to get rid of oAuth client wrapper
func (c *ClientVer1) SetHttpClient(httpClient *http.Client) {
	c.httpClient = httpClient
}

type subAccountsVer1 struct {
	total int
	ids   []string
}

func (c *ClientVer1) FetchSubAccountsToDelete() ([]string, error) {
	subaccounts := subAccountsVer1{}

	err := c.fetchSubAccountsFromDeleteEvents(&subaccounts, 1)
	if err != nil {
		return []string{}, errors.Wrap(err, "while fetching subaccounts from delete events")
	}

	c.log.Infof("CIS returned total amount of delete events: %d, client fetched %d subaccounts to delete.",
		subaccounts.total,
		len(subaccounts.ids))

	return subaccounts.ids, nil
}

func (c *ClientVer1) fetchSubAccountsFromDeleteEvents(collection *subAccountsVer1, page int) error {
	request, err := c.buildRequest(page)
	if err != nil {
		return errors.Wrap(err, "while building request for event service")
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return errors.Wrap(err, "while executing request to event service")
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("while processing response: %s", c.handleWrongStatusCode(response))
	}

	var cisResponse CisResponseVer1
	err = json.NewDecoder(response.Body).Decode(&cisResponse)
	if err != nil {
		return errors.Wrap(err, "while decoding CIS response")
	}

	collection.total = cisResponse.Total
	for _, event := range cisResponse.Events {
		if event.Type != eventTypeVer1 {
			c.log.Warnf("event type %s is not equal to %s, skip event", event.Type, eventTypeVer1)
			continue
		}
		collection.ids = append(collection.ids, event.Data.SubAccount)
	}

	page++
	if page <= cisResponse.TotalPages {
		return c.fetchSubAccountsFromDeleteEvents(collection, page)
	}

	return nil
}

func (c *ClientVer1) buildRequest(page int) (*http.Request, error) {
	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf(eventServicePathVer1, c.config.EventServiceURL), nil)
	if err != nil {
		return nil, errors.Wrap(err, "while creating request")
	}

	q := request.URL.Query()
	q.Add("ts", "1")
	q.Add("type", eventTypeVer1)
	q.Add("resultsPerPage", c.config.PageSize)
	q.Add("page", strconv.Itoa(page))

	request.URL.RawQuery = q.Encode()

	return request, nil
}

func (c *ClientVer1) handleWrongStatusCode(response *http.Response) string {
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Sprintf("server returned %d status code, response body is unreadable", response.StatusCode)
	}

	return fmt.Sprintf("server returned %d status code, body: %q", response.StatusCode, string(body))
}
