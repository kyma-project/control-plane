package broker

import (
	"context"
	"net/http"

	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/pivotal-cf/brokerapi/v8/handlers"
	"github.com/pivotal-cf/brokerapi/v8/middlewares"
)

// copied from github.com/pivotal-cf/brokerapi/api.go
func AttachRoutes(router *mux.Router, serviceBroker domain.ServiceBroker, logger lager.Logger) *mux.Router {
	apiHandler := handlers.NewApiHandler(serviceBroker, logger)
	deprovision := func(w http.ResponseWriter, req *http.Request) {
		req2 := req.WithContext(context.WithValue(req.Context(), "User-Agent", req.Header.Get("User-Agent")))
		apiHandler.Deprovision(w, req2)
	}
	router.HandleFunc("/v2/catalog", apiHandler.Catalog).Methods("GET")

	router.HandleFunc("/v2/service_instances/{instance_id}", apiHandler.GetInstance).Methods("GET")
	router.HandleFunc("/v2/service_instances/{instance_id}", apiHandler.Provision).Methods("PUT")
	router.HandleFunc("/v2/service_instances/{instance_id}", deprovision).Methods("DELETE")
	router.HandleFunc("/v2/service_instances/{instance_id}/last_operation", apiHandler.LastOperation).Methods("GET")
	router.HandleFunc("/v2/service_instances/{instance_id}", apiHandler.Update).Methods("PATCH")

	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", apiHandler.GetBinding).Methods("GET")
	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", apiHandler.Bind).Methods("PUT")
	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", apiHandler.Unbind).Methods("DELETE")

	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}/last_operation", apiHandler.LastBindingOperation).Methods("GET")

	router.Use(middlewares.AddCorrelationIDToContext)
	apiVersionMiddleware := middlewares.APIVersionMiddleware{LoggerFactory: logger}

	router.Use(middlewares.AddOriginatingIdentityToContext)
	router.Use(apiVersionMiddleware.ValidateAPIVersionHdr)
	router.Use(middlewares.AddInfoLocationToContext)

	return router
}
