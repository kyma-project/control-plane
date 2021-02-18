APP_NAME = kcp-installer
DOCKER_PATH=$(DOCKER_PUSH_REPOSITORY)$(DOCKER_PUSH_DIRECTORY)
IMG = $(DOCKER_PUSH_REPOSITORY)$(DOCKER_PUSH_DIRECTORY)/$(APP_NAME)
TAG = $(DOCKER_TAG)

.PHONY: build-image
build-image:
	cd ./../../; \
	docker build -t $(IMG):$(TAG) \
		-f ./tools/kcp-installer/kcp.Dockerfile  \
		--build-arg INSTALLER_DIR=$(DOCKER_PATH) .

.PHONY: push-image
push-image:
	docker push $(IMG):$(TAG)

.PHONY: ci-pr
ci-pr: build-image push-image

.PHONY: ci-master
ci-master: build-image push-image

.PHONY: ci-release
ci-release: build-image push-image

.PHONY: release
release: build-image push-image
