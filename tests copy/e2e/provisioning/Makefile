APP_NAME = e2e-provisioning-test
APP_PATH = tests/e2e/provisioning
BUILDPACK = eu.gcr.io/kyma-project/test-infra/buildpack-golang:v20221215-c20ffd65
SCRIPTS_DIR = $(realpath $(shell pwd)/../../..)/scripts

include $(SCRIPTS_DIR)/generic_make_go.mk

VERIFY_IGNORE := /vendor\|/automock\|/testdata

verify:: vet

test-local: ;

resolve-local:
	GO111MODULE=on go mod vendor -v

ensure-local:
	@echo "Go modules present in component - omitting."

dep-status:
	@echo "Go modules present in component - omitting."

dep-status-local:
	@echo "Go modules present in component - omitting."