clean: rm -rf hydra-maester/manager_*

clone:
	git clone --depth 1 --branch v0.0.20 https://github.com/ory/hydra-maester.git

build_arm64: export CGO_ENABLED=0
build_arm64: export GO111MODULE=on
build_arm64: export GOOS=linux
build_arm64: export GOARCH=arm64
build_arm64:
	cd hydra-maester && go build -o manager_arm64 main.go

build_amd64: export CGO_ENABLED=0
build_amd64: export GO111MODULE=on
build_amd64: export GOOS=linux
build_amd64: export GOARCH=amd64
build_amd64:
	cd hydra-maester && go build -o manager_amd64 main.go

build: clean build_arm64 build_amd64

push: build
	docker buildx build --tag=giolekva/ory-hydra-maester:latest . --platform=linux/arm64,linux/amd64 --push
