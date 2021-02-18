package event_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/event"
	"github.com/sirupsen/logrus"
	logrusTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"
)

func TestPubSub(t *testing.T) {
	// given
	var gotEventAList1 []eventA
	var gotEventAList2 []eventA
	var mu sync.Mutex
	handlerA1 := func(ctx context.Context, ev interface{}) error {
		mu.Lock()
		defer mu.Unlock()
		gotEventAList1 = append(gotEventAList1, ev.(eventA))
		return nil
	}
	handlerA2 := func(ctx context.Context, ev interface{}) error {
		mu.Lock()
		defer mu.Unlock()
		gotEventAList2 = append(gotEventAList2, ev.(eventA))
		return nil
	}
	var gotEventBList []eventB
	handlerB := func(ctx context.Context, ev interface{}) error {
		mu.Lock()
		defer mu.Unlock()
		gotEventBList = append(gotEventBList, ev.(eventB))
		return nil
	}
	svc := event.NewPubSub(logrus.New())
	svc.Subscribe(eventA{}, handlerA1)
	svc.Subscribe(eventB{}, handlerB)
	svc.Subscribe(eventA{}, handlerA2)

	// when
	svc.Publish(context.TODO(), eventA{msg: "first event"})
	svc.Publish(context.TODO(), eventB{msg: "second event"})
	svc.Publish(context.TODO(), eventA{msg: "third event"})

	time.Sleep(1 * time.Millisecond)

	// then
	assert.NoError(t, wait.PollImmediate(20*time.Millisecond, 2*time.Second, func() (bool, error) {
		return containsA(gotEventAList1, eventA{msg: "first event"}) &&
			containsA(gotEventAList1, eventA{msg: "third event"}) &&
			containsA(gotEventAList2, eventA{msg: "first event"}) &&
			containsA(gotEventAList2, eventA{msg: "third event"}) &&
			containsB(gotEventBList, eventB{msg: "second event"}), nil
	}))
}

func TestPubSub_WhenHandlerReturnsError(t *testing.T) {
	// given
	logger, hook := logrusTest.NewNullLogger()
	var mu sync.Mutex
	handlerA1 := func(ctx context.Context, ev interface{}) error {
		mu.Lock()
		defer mu.Unlock()
		return errors.New("some error")
	}
	svc := event.NewPubSub(logger)
	svc.Subscribe(eventA{}, handlerA1)

	// when
	svc.Publish(context.TODO(), eventA{msg: "first event"})

	time.Sleep(1 * time.Millisecond)

	// then
	require.Equal(t, 1, len(hook.Entries))
	require.Equal(t, hook.LastEntry().Message, "error while calling pubsub event handler: some error")
}

func containsA(slice []eventA, item eventA) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsB(slice []eventB, item eventB) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

type eventA struct {
	msg string
}

type eventB struct {
	msg string
}
