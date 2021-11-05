#
# Make targets to build kcp CLI artifacts
#

ifndef CLI_VERSION
	CLI_VERSION = ${shell git describe --tags --always}
endif
ifndef ARTIFACTS
	ARTIFACTS = .
endif

# ENTRYPOINT is the relative path to the main Go file
ENTRYPOINT = cmd/main/main.go
# VERIFY_IGNORE is a grep pattern to exclude files and directories from verification
VERIFY_IGNORE := /vendor\|/automock
# FILES_TO_CHECK is a command used to determine which files should be verified
FILES_TO_CHECK = find . -type f -name "*.go" | grep -v "$(VERIFY_IGNORE)"
# DIRS_TO_CHECK is a command used to determine which directories should be verified
DIRS_TO_CHECK = go list ./... | grep -v "$(VERIFY_IGNORE)"
# CLI_FLAGS is passed to go build when compiling the binaries
CLI_FLAGS = -ldflags '-s -w -X github.com/kyma-project/control-plane/tools/cli/pkg/command.Version=$(CLI_VERSION)'

.PHONY: release resolve verify test go-mod-verify go-mod-check errcheck build build-windows build-linux build-darwin docs

release: resolve verify build

resolve:
	GO111MODULE=on go mod vendor -v

verify: check-imports go-mod-verify go-mod-check

test:
	go test ./...

check-imports:
	@if [ -n "$$(goimports -l $$($(FILES_TO_CHECK)))" ]; then \
		echo "✗ some files are not properly formatted or contain not formatted imports. To repair run make imports"; \
		goimports -l $$($(FILES_TO_CHECK)); \
		exit 1; \
	fi;

imports:
	goimports -w -l $$($(FILES_TO_CHECK))

go-mod-verify:
	GO111MODULE=on go mod verify

go-mod-check:
	@echo make go-mod-check
	go mod tidy
	@if [ -n "$$(git status -s go.*)" ]; then \
		echo -e "${RED}✗ go mod tidy modified go.mod or go.sum files${NC}"; \
		git status -s go.*; \
		exit 1; \
	fi;

errcheck:
	errcheck -blank -asserts -ignorepkg '$$($(DIRS_TO_CHECK) | tr '\n' ',')' -ignoregenerated ./...

build: build-windows build-linux build-darwin

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o $(ARTIFACTS)/kcp.exe $(CLI_FLAGS) $(ENTRYPOINT)

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(ARTIFACTS)/kcp-linux $(CLI_FLAGS) $(ENTRYPOINT)

build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o $(ARTIFACTS)/kcp-darwin $(CLI_FLAGS) $(ENTRYPOINT)

docs:
	go run ./cmd/gendocs/gendocs.go
