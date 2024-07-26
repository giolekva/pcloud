repo_name ?= giolekva
podman ?= docker
ifeq ($(podman), podman)
manifest_dest_golang_1_22_0=docker://docker.io/$(repo_name)/app-runner:golang-1.22.0
manifest_dest_golang_1_20_0=docker://docker.io/$(repo_name)/app-runner:golang-1.20.0
manifest_dest_hugo_latest=docker://docker.io/$(repo_name)/app-runner:hugo-latest
endif

clean:
	rm -f app-runner

build_arm64: export CGO_ENABLED=0
build_arm64: export GO111MODULE=on
build_arm64: export GOOS=linux
build_arm64: export GOARCH=arm64
build_arm64:
	/usr/local/go/bin/go build -o app-runner_arm64 *.go

build_amd64: export CGO_ENABLED=0
build_amd64: export GO111MODULE=on
build_amd64: export GOOS=linux
build_amd64: export GOARCH=amd64
build_amd64:
	/usr/local/go/bin/go build -o app-runner_amd64 *.go

# Golang 1.22.0

push_golang_1_22_0_arm64: clean build_arm64
	$(podman) build --platform linux/arm64 --tag=$(repo_name)/app-runner:golang-1.22.0-arm64 -f Dockerfile.golang.1.22.0 .
	$(podman) push $(repo_name)/app-runner:golang-1.22.0-arm64

push_golang_1_22_0_amd64: clean build_amd64
	$(podman) build --platform linux/amd64 --tag=$(repo_name)/app-runner:golang-1.22.0-amd64 -f Dockerfile.golang.1.22.0 .
	$(podman) push $(repo_name)/app-runner:golang-1.22.0-amd64

push_golang_1_22_0: push_golang_1_22_0_arm64 push_golang_1_22_0_amd64
	$(podman) manifest create $(repo_name)/app-runner:golang-1.22.0 $(repo_name)/app-runner:golang-1.22.0-arm64 $(repo_name)/app-runner:golang-1.22.0-amd64
	$(podman) manifest push $(repo_name)/app-runner:golang-1.22.0 $(manifest_dest_golang_1_22_0)
	$(podman) manifest rm $(repo_name)/app-runner:golang-1.22.0

# Golang 1.20.0

push_golang_1_20_0_arm64: clean build_arm64
	$(podman) build --platform linux/arm64 --tag=$(repo_name)/app-runner:golang-1.20.0-arm64 -f Dockerfile.golang.1.20.0 .
	$(podman) push $(repo_name)/app-runner:golang-1.20.0-arm64

push_golang_1_20_0_amd64: clean build_amd64
	$(podman) build --platform linux/amd64 --tag=$(repo_name)/app-runner:golang-1.20.0-amd64 -f Dockerfile.golang.1.20.0 .
	$(podman) push $(repo_name)/app-runner:golang-1.20.0-amd64

push_golang_1_20_0: push_golang_1_20_0_arm64 push_golang_1_20_0_amd64
	$(podman) manifest create $(repo_name)/app-runner:golang-1.20.0 $(repo_name)/app-runner:golang-1.20.0-arm64 $(repo_name)/app-runner:golang-1.20.0-amd64
	$(podman) manifest push $(repo_name)/app-runner:golang-1.20.0 $(manifest_dest_golang_1_20_0)
	$(podman) manifest rm $(repo_name)/app-runner:golang-1.20.0

# Hugo

push_hugo_arm64: clean build_arm64
	$(podman) build --platform linux/arm64 --tag=$(repo_name)/app-runner:hugo-latest-arm64 -f Dockerfile.hugo .
	$(podman) push $(repo_name)/app-runner:hugo-latest-arm64

push_hugo_amd64: clean build_amd64
	$(podman) build --platform linux/amd64 --tag=$(repo_name)/app-runner:hugo-latest-amd64 -f Dockerfile.hugo .
	$(podman) push $(repo_name)/app-runner:hugo-latest-amd64

push_hugo: push_hugo_arm64 push_hugo_amd64
	$(podman) manifest create $(repo_name)/app-runner:hugo-latest $(repo_name)/app-runner:hugo-latest-arm64 $(repo_name)/app-runner:hugo-latest-amd64
	$(podman) manifest push $(repo_name)/app-runner:hugo-latest $(manifest_dest_hugo_latest)
	$(podman) manifest rm $(repo_name)/app-runner:hugo-latest

# all
push: push_golang_1_22_0 push_golang_1_20_0 push_hugo
