APP_NAME = schema-migrator
APP_PATH = components/schema-migrator
BUILDPACK = eu.gcr.io/kyma-project/test-infra/buildpack-golang:v20221017-733bfd36
SCRIPTS_DIR = $(realpath $(shell pwd)/../..)/scripts
export SKIP_DEPLOY_MESSAGE = "Building minikube image and redeployment of Schema Migrator is not allowed"

include $(SCRIPTS_DIR)/generic_make_go.mk

.PHONY: validate

validate:
	./validate.sh

verify:: validate

resolve-local:
	GO111MODULE=on go mod vendor -v

build-to-minikube:
	@echo ${SKIP_DEPLOY_MESSAGE}

deploy-on-minikube:
	@echo ${SKIP_DEPLOY_MESSAGE}

ensure-local:
	@echo ${SKIP_DEPLOY_MESSAGE}

errcheck-local:
	@echo ${SKIP_DEPLOY_MESSAGE}

resolve-local:
	@echo ${SKIP_DEPLOY_MESSAGE}

dep-status:
	@echo ${SKIP_DEPLOY_MESSAGE}

dep-status-local:
	@echo ${SKIP_DEPLOY_MESSAGE}
