package edp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/metris/internal/log"

	"github.com/stretchr/testify/assert"
)

var (
	defaultLogger = log.NewNoopLogger()

	defaultconfig = &Config{
		URL:               "http://127.0.0.1:9999",
		Token:             "E6B99A13-783F-4A3B-8605-C5EA32CA44B5",
		Timeout:           30 * time.Second,
		Namespace:         "kyma-dev",
		DataStream:        "consumption-metrics",
		DataStreamVersion: "1",
		DataStreamEnv:     "dev",
		Buffer:            100,
		Workers:           1,
		EventRetry:        1,
	}
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func fakeTestClient(t *testing.T, status int, err error) *http.Client {
	t.Helper()

	var fn roundTripFunc = func(req *http.Request) (*http.Response, error) {
		if err != nil {
			return nil, err
		}

		return &http.Response{StatusCode: status, Body: ioutil.NopCloser(bytes.NewBufferString("")), Header: make(http.Header)}, nil
	}

	return &http.Client{
		Transport: fn,
	}
}

func TestClient_NewClientNil(t *testing.T) {
	client := NewClient(defaultconfig, nil, nil, defaultLogger)

	assert.NotEmpty(t, client, "client object should not be empty")
}

func TestClient_Run(t *testing.T) {
	fakehttpclient := fakeTestClient(t, http.StatusCreated, nil)
	client := NewClient(defaultconfig, fakehttpclient, nil, defaultLogger)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	client.Start(ctx)

	assert.EqualErrorf(t, ctx.Err(), context.DeadlineExceeded.Error(), "should got error %s but got %s", context.DeadlineExceeded.Error(), ctx.Err().Error())
}

func TestClient_CreateEventSucess(t *testing.T) {
	fakehttpclient := fakeTestClient(t, http.StatusCreated, nil)
	client := NewClient(defaultconfig, fakehttpclient, nil, defaultLogger)

	data := json.RawMessage(`{"event":[{"data":"test"}]}`)
	event := &Event{Datatenant: "bob", Data: &data}

	err := client.Write(context.TODO(), event, defaultLogger)
	assert.NoError(t, err)
}

func TestClient_CreateEventInvalid(t *testing.T) {
	fakehttpclient := fakeTestClient(t, http.StatusBadRequest, nil)
	client := NewClient(defaultconfig, fakehttpclient, nil, defaultLogger)

	data := json.RawMessage(`{"event":[{"data":"test2"}]}`)
	event := &Event{Datatenant: "bob", Data: &data}

	err := client.Write(context.TODO(), event, defaultLogger)
	assert.Conditionf(t, func() bool {
		return errors.Is(err, ErrEventInvalidRequest)
	}, "invalid error, got %s, should be: %s", err, ErrEventInvalidRequest)
}

func TestClient_CreateEventMissingParam(t *testing.T) {
	fakehttpclient := fakeTestClient(t, http.StatusNotFound, nil)
	client := NewClient(defaultconfig, fakehttpclient, nil, defaultLogger)

	data := json.RawMessage(`{"test3":[{"error":""}]}`)
	event := &Event{Datatenant: "bob", Data: &data}

	err := client.Write(context.TODO(), event, defaultLogger)
	assert.Conditionf(t, func() bool {
		return errors.Is(err, ErrEventMissingParameters)
	}, "invalid error, got %s, should be: %s", err, ErrEventMissingParameters)
}

func TestClient_CreateEventUnknownError(t *testing.T) {
	fakehttpclient := fakeTestClient(t, http.StatusUnauthorized, nil)
	client := NewClient(defaultconfig, fakehttpclient, nil, defaultLogger)

	data := json.RawMessage(`{"test4":[{"error":""},{"error":""}]}`)
	event := &Event{Datatenant: "bob", Data: &data}

	err := client.Write(context.TODO(), event, defaultLogger)
	assert.Conditionf(t, func() bool {
		return errors.Is(err, ErrEventUnknown)
	}, "invalid error, got %s, should be: %s", err, ErrEventUnknown)
}

func TestClient_CreateEventJSONError(t *testing.T) {
	fakehttpclient := fakeTestClient(t, http.StatusCreated, nil)
	client := NewClient(defaultconfig, fakehttpclient, nil, defaultLogger)

	data := json.RawMessage(`{"test5":[{"error":""}]`)
	event := &Event{Datatenant: "bob", Data: &data}

	err := client.Write(context.TODO(), event, defaultLogger)
	assert.Conditionf(t, func() bool {
		return errors.Is(err, ErrEventMarshal)
	}, "invalid error, got %s, should be: %s", err, ErrEventMarshal)
}

func TestClient_CreateEventHTTPError(t *testing.T) {
	fakehttpclient := fakeTestClient(t, 0, fmt.Errorf("network error"))
	client := NewClient(defaultconfig, fakehttpclient, nil, defaultLogger)

	data := json.RawMessage(`{"test6":[{"error":""}]}`)
	event := &Event{Datatenant: "bob", Data: &data}

	err := client.Write(context.TODO(), event, defaultLogger)
	assert.Conditionf(t, func() bool {
		return errors.Is(err, ErrEventHTTPRequest)
	}, "invalid error, got %s, should be: %s", err, ErrEventHTTPRequest)
}

func TestClient_handleErr(t *testing.T) {
	fakehttpclient := fakeTestClient(t, http.StatusBadRequest, nil)
	client := NewClient(defaultconfig, fakehttpclient, nil, defaultLogger)

	tests := []struct {
		name     string
		err      error
		requeues []int
		len      []int
	}{
		{
			name:     "success",
			err:      nil,
			requeues: []int{0},
			len:      []int{0},
		},
		{
			name:     "marshal",
			err:      ErrEventMarshal,
			requeues: []int{0},
			len:      []int{0},
		},
		{
			name:     "requeue",
			err:      statusError(0),
			requeues: []int{1, 0},
			len:      []int{0, 0},
		},
	}
	for _, tt := range tests {
		tt := tt // pin!

		t.Run(tt.name, func(t *testing.T) {
			asserts := assert.New(t)

			data := json.RawMessage(fmt.Sprintf(`{"test":[{"error":"%s"}]}`, tt.name))
			fakeevent := &Event{Datatenant: "bob", Data: &data}
			client.queue.Add(fakeevent)

			for i, requeues := range tt.requeues {
				obj, _ := client.queue.Get()
				event, ok := obj.(*Event)
				asserts.Truef(ok, "object from queue should be of type *[]byte but got %T", obj)

				client.handleErr(tt.err, event, defaultLogger)
				client.queue.Done(event)
				asserts.Equalf(requeues, client.queue.NumRequeues(event), "number of requeue should be %d but got %d", requeues, client.queue.NumRequeues(event))
				asserts.Equalf(tt.len[i], client.queue.Len(), "queue len should be %d but got %d", tt.len[i], client.queue.Len())
			}
		})
	}
}
