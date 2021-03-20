.PHONY: help

.DEFAULT_GOAL := help

help:
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

generate: ## generates the gRPC server interfaces from `*.proto` service definition
	@protoc --go_out=. --go-grpc_out=. user.proto 