package kcp

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/dapr/dapr/pkg/apis/components/v1alpha1"
)

type client struct {
	hc  *http.Client
	URL string
}

func NewClient(url string) *client {
	hc := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	return &client{
		hc:  hc,
		URL: url,
	}
}

func (c *client) Fetch(runtimeID string) (*v1alpha1.Component, error) {
	resp, err := c.hc.Get(c.URL + "/runtimes/" + runtimeID)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	log.Println(string(body))

	var data model
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	return &data.Data, nil
}

type model struct {
	Id   string             `json:id`
	Data v1alpha1.Component `json:data`
}
