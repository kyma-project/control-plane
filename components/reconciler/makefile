OAPI_GENERATOR=oapi-codegen
OAPI_FILE=./docs/external_api.yaml
OAPI_REVISION=main

CLIENT_FILE=./pkg/client_gen.go
TYPES_FILE=./pkg/types_gen.go

docs/external_api.yaml:
	# TODO make sure to fetch oapi from main when it will be available!
	curl -o $(OAPI_FILE) --create-dirs https://raw.githubusercontent.com/kyma-incubator/reconciler/$(OAPI_REVISION)/openapi/external_api.yaml

.PHONY: generate
generate: docs/external_api.yaml
	$(OAPI_GENERATOR) -generate 'types' -o $(TYPES_FILE) -package mothership ./docs/external_api.yaml
	$(OAPI_GENERATOR) -generate 'client' -o $(CLIENT_FILE) -package mothership ./docs/external_api.yaml

.PHONY: clean
clean:
	rm $(OAPI_FILE) $(CLIENT_FILE) $(TYPES_FILE) || true

.PHONY: all
all: clean generate

.DEFAULT_GOAL=all
