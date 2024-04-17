APP_NAME = provisioner
APP_PATH = components/provisioner
ENTRYPOINT = cmd/main.go
BUILDPACK = europe-docker.pkg.dev/kyma-project/prod/testimages/buildpack-go:v20240404-de49794e
SCRIPTS_DIR = $(realpath $(shell pwd)/../..)/scripts
DOCKER_SOCKET = /var/run/docker.sock
TESTING_DB_NETWORK = test_network
PIPELINE_BUILD = 1
export PIPELINE_BUILD

include $(SCRIPTS_DIR)/generic_make_go.mk

.PHONY: gqlgen check-gqlgen testing-with-database-network clean-up mod-verify go-mod-check

verify:: gqlgen check-gqlgen test-really-local mod-verify go-mod-check

resolve-local:
	GO111MODULE=on go mod vendor -v

ensure-local:
	@echo "Go modules present in component - omitting."

dep-status:
	@echo "Go modules present in component - omitting."

dep-status-local:
	@echo "Go modules present in component - omitting."

mod-verify-local:
	GO111MODULE=on go mod verify

go-mod-check-local:
	@echo make go-mod-check
	go mod tidy
	@if [ -z "$$(git status -s go.*)" ]; then \
		echo -e "${RED}✗ go mod tidy modified go.mod or go.sum files${NC}"; \
		git status -s git status -s go.*; \
		exit 1; \
	fi;


errcheck-local: ;
# errcheck-local:
# 	@docker run $(DOCKER_INTERACTIVE) \
# 		-v $(COMPONENT_DIR):$(WORKSPACE_COMPONENT_DIR):delegated \
# 		$(DOCKER_CREATE_OPTS) errcheck -blank -asserts -ignorepkg '$$($(DIRS_TO_CHECK) | tr '\n' ',')' -ignoregenerated ./...

LOCALBIN ?= $(shell pwd)/bin
ENVTEST ?= $(LOCALBIN)/setup-envtest
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

test-local: ;

test-really-local: envtest
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./... -timeout 15m
