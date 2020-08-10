package cis

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	eventServicePath = "%s/events/v1/events/central"
	eventType        = "Subaccount_Deletion"
	defaultPageSize  = "150"
)

type Config struct {
	ClientID        string
	ClientSecret    string
	AuthURL         string
	EventServiceURL string
	PageSize        string `envconfig:"optional"`
}

type Client struct {
	httpClient *http.Client
	config     Config
	log        logrus.FieldLogger
}

func NewClient(ctx context.Context, config Config, log logrus.FieldLogger) *Client {
	cfg := clientcredentials.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		TokenURL:     config.AuthURL,
	}
	httpClientOAuth := cfg.Client(ctx)
	httpClientOAuth.Timeout = 30 * time.Second

	if config.PageSize == "" {
		config.PageSize = defaultPageSize
	}

	return &Client{
		httpClient: httpClientOAuth,
		config:     config,
		log:        log.WithField("client", "CIS-2.0"),
	}
}

// SetHttpClient auxiliary method of testing to get rid of oAuth client wrapper
func (c *Client) SetHttpClient(httpClient *http.Client) {
	c.httpClient = httpClient
}

type subAccounts struct {
	total int
	ids   []string
	from  time.Time
	to    time.Time
}

func (c *Client) FetchSubAccountsToDelete() ([]string, error) {
	subaccounts := subAccounts{}

	err := c.fetchSubAccountsFromDeleteEvents(&subaccounts, 0)
	if err != nil {
		return []string{}, errors.Wrap(err, "while fetching subaccounts from delete events")
	}

	c.log.Infof("CIS returned total amount of delete events: %d, client fetched %d subaccounts to delete. "+
		"The events includes a range of time from %s to %s",
		subaccounts.total,
		len(subaccounts.ids),
		subaccounts.from,
		subaccounts.to)

	return subaccounts.ids, nil
}

func (c *Client) fetchSubAccountsFromDeleteEvents(collection *subAccounts, page int) error {
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

	var cisResponse CisResponse
	err = json.NewDecoder(response.Body).Decode(&cisResponse)
	if err != nil {
		return errors.Wrap(err, "while decoding CIS response")
	}

	collection.total = cisResponse.Total
	for _, event := range cisResponse.Events {
		if event.Type != eventType {
			c.log.Warnf("event type %s is not equal to %s, skip event", event.Type, eventType)
			continue
		}
		collection.ids = append(collection.ids, event.SubAccount)

		if collection.from.IsZero() {
			collection.from = time.Unix(0, event.CreationTime*int64(1000000))
		}
		if collection.total == len(collection.ids) {
			collection.to = time.Unix(0, event.CreationTime*int64(1000000))
		}
	}

	page++
	if page <= cisResponse.TotalPages {
		return c.fetchSubAccountsFromDeleteEvents(collection, page)
	}
	return nil
}

func (c *Client) buildRequest(page int) (*http.Request, error) {
	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf(eventServicePath, c.config.EventServiceURL), nil)
	if err != nil {
		return nil, errors.Wrap(err, "while creating request")
	}

	q := request.URL.Query()
	q.Add("eventType", eventType)
	q.Add("pageSize", c.config.PageSize)
	q.Add("pageNum", strconv.Itoa(page))
	q.Add("sortField", "creationTime")
	q.Add("sortOrder", "ASC")

	request.URL.RawQuery = q.Encode()

	return request, nil
}

func (c *Client) handleWrongStatusCode(response *http.Response) string {
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Sprintf("server returned %d status code, response body is unreadable", response.StatusCode)
	}

	return fmt.Sprintf("server returned %d status code, body: %s", response.StatusCode, string(body))
}
