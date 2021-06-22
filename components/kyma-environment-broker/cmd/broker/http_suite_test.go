package main

import (
	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"net/http/httptest"
	"testing"
)

type HttpSuite struct {
	t *testing.T
	httpServer *httptest.Server
	router     *mux.Router
}

func NewHttpSuite(t *testing.T) HttpSuite {
	router := mux.NewRouter()
	return HttpSuite{
		router: router,
		t: t,
	}
}

func (s *HttpSuite) CreateAPI(inputFactory broker.PlanValidator, cfg *Config, db storage.BrokerStorage, provisioningQueue *process.Queue, deprovisionQueue *process.Queue, logs logrus.FieldLogger) {
	servicesConfig := map[string]broker.Service{
		broker.KymaServiceName: {
			Metadata: broker.ServiceMetadata{
				DisplayName: "kyma",
				SupportUrl:  "https://kyma-project.io",
			},
		},
	}
	createAPI(s.router, servicesConfig, inputFactory, cfg, db, provisioningQueue, deprovisionQueue, lager.NewLogger("api"), logs)
	s.httpServer = httptest.NewServer(s.router)
}
