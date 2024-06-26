GO ?= $(shell command -v go 2> /dev/null)
GO_TEST_FLAGS ?= -race

export GO111MODULE=on

MINIMUM_SUPPORTED_GO_MAJOR_VERSION = 1
MINIMUM_SUPPORTED_GO_MINOR_VERSION = 14
GO_MAJOR_VERSION = $(shell $(GO) version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f1)
GO_MINOR_VERSION = $(shell $(GO) version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f2)
GO_VERSION_VALIDATION_ERR_MSG = Golang version is not supported, please update to at least $(MINIMUM_SUPPORTED_GO_MAJOR_VERSION).$(MINIMUM_SUPPORTED_GO_MINOR_VERSION)

BUILDER_GOOS_GOARCH="$(shell $(GO) env GOOS)_$(shell $(GO) env GOARCH)"

## Define the default target (make all)
.PHONY: default
default: all

## Checks the code style, tests, runs the code.
.PHONY: all
all: check-style test run

## Runs eslint and golangci-lint
.PHONY: check-style
check-style:
	@if ! [ -x "$$(command -v golangci-lint)" ]; then \
		echo "golangci-lint is not installed. Please see https://github.com/golangci/golangci-lint#install for installation instructions."; \
		exit 1; \
	fi; \

	@echo Running golangci-lint
	golangci-lint run ./...

## Runs unit tests.
.PHONY: test
test:
	@echo Running unit tests
	$(GO) test -v $(GO_TEST_FLAGS) ./...

## Creates a coverage report
.PHONY: coverage
coverage:
	$(GO) test $(GO_TEST_FLAGS) -coverprofile=coverage.txt ./...
	$(GO) tool cover -html=coverage.txt

## Clean removes all artifacts.
.PHONY: clean
clean:
	rm -fr coverage.txt
	rm -fr server.log

.PHONY: run
run: validate-go-version
	@echo Running Knowledge Graph services

	$(GO) run ./cmd/

.PHONY: validate-go-version
validate-go-version: ## Validates the installed version of go against minimum requirement.
	@if [ $(GO_MAJOR_VERSION) -gt $(MINIMUM_SUPPORTED_GO_MAJOR_VERSION) ]; then \
		exit 0 ;\
	elif [ $(GO_MAJOR_VERSION) -lt $(MINIMUM_SUPPORTED_GO_MAJOR_VERSION) ]; then \
		echo '$(GO_VERSION_VALIDATION_ERR_MSG)';\
		exit 1; \
	elif [ $(GO_MINOR_VERSION) -lt $(MINIMUM_SUPPORTED_GO_MINOR_VERSION) ] ; then \
		echo '$(GO_VERSION_VALIDATION_ERR_MSG)';\
		exit 1; \
	fi