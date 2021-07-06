APP_NAME = kyma-metrics-collector
APP_PATH = components/kyma-metrics-collector
ENTRYPOINT = cmd/main.go
BUILDPACK = eu.gcr.io/kyma-project/test-infra/buildpack-golang-toolbox:v20200423-1d9d6590
SCRIPTS_DIR = $(realpath $(shell pwd)/../..)/scripts
PROMETHEUSRULES_PATH = ../../resources/kcp/charts/kyma-metrics-collector/prometheus

export GO111MODULE=on
export CGO_ENABLED=0
export SKIP_STEP_MESSAGE = "Do nothing for Go modules project"

include $(SCRIPTS_DIR)/generic_make_go.mk

resolve-local:
	GO111MODULE=on go mod vendor -v

ensure-local:
	@echo ${SKIP_STEP_MESSAGE}

dep-status-local:
	@echo ${SKIP_STEP_MESSAGE}

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

test-alerts:
	promtool test rules ${PROMETHEUSRULES_PATH}/alerts_test.yaml