package azure

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/stretchr/testify/assert"

	"github.com/kyma-project/control-plane/components/metris/internal/edp"
	"github.com/kyma-project/control-plane/components/metris/internal/gardener"
	"github.com/kyma-project/control-plane/components/metris/internal/log"
	"github.com/kyma-project/control-plane/components/metris/internal/provider"
	"github.com/kyma-project/control-plane/components/metris/internal/storage"
)

var (
	noopLogger     = log.NewNoopLogger()
	providerConfig = &provider.Config{
		PollInterval:   time.Minute,
		Workers:        1,
		Buffer:         1,
		ClusterChannel: make(chan *gardener.Cluster, 1),
		EventsChannel:  make(chan *edp.Event, 1),
		Logger:         noopLogger,
	}

	testCluster = &gardener.Cluster{
		TechnicalID:  "test-technicalid",
		ProviderType: "az",
		CredentialData: map[string][]byte{
			"clientID":       []byte("test-clientid"),
			"clientSecret":   []byte("test-clientsecret"),
			"subscriptionID": []byte("test-subscriptionid"),
			"tenantID":       []byte("test-tenantid"),
		},
		AccountID:    "test-accountid",
		SubAccountID: "test-subaccountid",
	}
)

func TestNewAzureProvider(t *testing.T) {
	p := NewAzureProvider(providerConfig)
	assert.Implements(t, (*provider.Provider)(nil), p, "")
}

func Test_processError(t *testing.T) {
	var testInstance = &Instance{
		lastEvent:     nil,
		retryAttempts: 0,
		cluster:       testCluster,
	}
	type want struct {
		eventData       *EventData
		throttled       bool
		instance        *Instance
		instanceDeleted bool
	}
	type args struct {
		workerlogger    log.Logger
		instance        *Instance
		eventData       *EventData
		instanceStorage storage.Storage
		err             error
		maxRetries      int
	}
	type test struct {
		name string
		args args
		want want
	}
	tests := []test{
		func() test {
			instance := &Instance{
				lastEvent:     &EventData{},
				retryAttempts: 0,
				cluster:       testCluster,
			}

			instanceStorage := storage.NewMemoryStorage("clusters")
			instanceStorage.Put(testInstance.cluster.TechnicalID, instance)

			return test{
				name: "Throttled with last eventData",
				args: args{
					workerlogger:    noopLogger,
					instanceStorage: instanceStorage,
					instance:        instance,
					err: autorest.DetailedError{
						StatusCode: 429,
					},
				},
				want: want{
					eventData: instance.lastEvent,
					throttled: true,
					instance:  instance,
				},
			}
		}(),
		func() test {
			instanceStorage := storage.NewMemoryStorage("clusters")
			instanceStorage.Put(testInstance.cluster.TechnicalID, testInstance)
			return test{
				name: "Throttled without eventData",
				args: args{
					workerlogger:    noopLogger,
					instanceStorage: instanceStorage,
					instance:        testInstance,
					err: autorest.DetailedError{
						StatusCode: 429,
					},
				},
				want: want{
					eventData: nil,
					throttled: true,
					instance:  testInstance,
				},
			}
		}(),
		func() test {
			instanceStorage := storage.NewMemoryStorage("clusters")
			instanceStorage.Put(testInstance.cluster.TechnicalID, testInstance)

			return test{
				name: "Not Found",
				args: args{
					workerlogger:    noopLogger,
					instanceStorage: instanceStorage,
					err: autorest.DetailedError{
						StatusCode: 404,
						Original:   fmt.Errorf("BAD THINGS HAPPENED"),
					},
					instance: testInstance,
				},
				want: want{
					eventData: nil,
					throttled: false,
					instance: &Instance{
						lastEvent:     nil,
						retryAttempts: 0,
						cluster:       testCluster,
					},
				},
			}
		}(),
		func() test {
			instanceStorage := storage.NewMemoryStorage("clusters")
			instanceStorage.Put(testInstance.cluster.TechnicalID, testInstance)

			return test{
				name: "ResourceGroup NotFound",
				args: args{
					workerlogger:    noopLogger,
					instanceStorage: instanceStorage,
					err: autorest.DetailedError{
						StatusCode: 404,
						Original:   fmt.Errorf(ResponseErrCodeResourceGroupNotFound),
					},
					instance:   testInstance,
					maxRetries: 5,
				},
				want: want{
					eventData: nil,
					throttled: false,
					instance: &Instance{
						lastEvent:     nil,
						retryAttempts: 1,
						cluster:       testCluster,
					},
				},
			}
		}(),
		func() test {
			instanceStorage := storage.NewMemoryStorage("clusters")
			instance := &Instance{
				lastEvent:     &EventData{},
				retryAttempts: 1,
				cluster:       testCluster,
			}
			instanceStorage.Put(testInstance.cluster.TechnicalID, instance)

			return test{
				name: "instance deleted",
				args: args{
					workerlogger:    noopLogger,
					instanceStorage: instanceStorage,
					err: autorest.DetailedError{
						StatusCode: 404,
						Original:   fmt.Errorf(ResponseErrCodeResourceGroupNotFound),
					},
					instance:   instance,
					maxRetries: 1,
				},
				want: want{
					instanceDeleted: true,
					eventData:       &EventData{},
					throttled:       false,
					instance: &Instance{
						lastEvent:     nil,
						retryAttempts: 1,
						cluster:       testCluster,
					},
				},
			}
		}(),
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventData, throttled, instanceDeleted := processError(context.Background(), tt.args.workerlogger, tt.args.instance, tt.args.eventData, tt.args.err, tt.args.maxRetries, tt.args.instanceStorage)
			if !reflect.DeepEqual(eventData, tt.want.eventData) {
				t.Errorf("eventData got = %#v, want %#v", eventData, tt.want.eventData)
			}
			if throttled != tt.want.throttled {
				t.Errorf("throttled got = %v, want %v", throttled, tt.want.throttled)
			}

			if instanceDeleted != tt.want.instanceDeleted {
				t.Errorf("instanceDeleted got = %#v, want %#v", instanceDeleted, tt.want.instanceDeleted)
			}

			obj, ok := tt.args.instanceStorage.Get(tt.args.instance.cluster.TechnicalID)
			instanceFound := !tt.want.instanceDeleted
			if ok != instanceFound {
				t.Errorf("instanceFound got = %v, want %#v", ok, instanceFound)
			}
			if instanceFound && ok && !reflect.DeepEqual(obj.(*Instance), tt.want.instance) {
				t.Errorf("instance got = %v, want %#v", obj.(*Instance), tt.want.instance)
			}
		})
	}
}
