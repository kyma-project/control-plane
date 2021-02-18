APP_NAME = metris
APP_PATH = components/metris
ENTRYPOINT = cmd/metris/main.go
BUILDPACK = eu.gcr.io/kyma-project/test-infra/buildpack-golang-toolbox:v20200423-1d9d6590
SCRIPTS_DIR = $(realpath $(shell pwd)/../..)/scripts
COMMITHASH = $(shell git rev-list -1 HEAD)

export GO111MODULE=on
export CGO_ENABLED=0
export SKIP_STEP_MESSAGE = "Do nothing for Go modules project"

include $(SCRIPTS_DIR)/generic_make_go.mk

resolve-local:
	@echo ${SKIP_STEP_MESSAGE}

ensure-local:
	@echo ${SKIP_STEP_MESSAGE}

dep-status-local:
	@echo ${SKIP_STEP_MESSAGE}

# overide build-image to add `build-args`
build-image: 
	@docker build --build-arg VERSION=$(TAG) --build-arg COMMITHASH=$(COMMITHASH) -t $(IMG_NAME) .