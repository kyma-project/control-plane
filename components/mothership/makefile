export OAPI_GENERATOR=oapi-codegen

export OAPI_FILE=./docs/external_api.yaml

docs/external_api.yaml:
	# TODO make sure to fetch oapi from main when it will be available!
	curl https://raw.githubusercontent.com/kyma-incubator/reconciler/5311fd7eefdf62523549b29dbc5658c17a44899f/openapi/external_api.yaml > $(OAPI_FILE)

export CLIENT_FILE=./pkg/client_gen.go
export TYPES_FILE=./pkg/types_gen.go

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