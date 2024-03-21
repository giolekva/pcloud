repo_name ?= giolekva
podman ?= docker
ifeq ($(podman), podman)
manifest_dest=docker://docker.io/$(repo_name)/pcloud-installer:latest
endif

clean:
	rm -f server server_*

build: clean
	/usr/local/go/bin/go build -o server *.go

build_arm64: export CGO_ENABLED=0
build_arm64: export GO111MODULE=on
build_arm64: export GOOS=linux
build_arm64: export GOARCH=arm64
build_arm64:
	/usr/local/go/bin/go build -o server_arm64 *.go

build_amd64: export CGO_ENABLED=0
build_amd64: export GO111MODULE=on
build_amd64: export GOOS=linux
build_amd64: export GOARCH=amd64
build_amd64:
	/usr/local/go/bin/go build -o server_amd64 *.go

push_arm64: clean build_arm64
	$(podman) build --platform linux/arm64 --tag=$(repo_name)/auth-proxy:arm64 .
	$(podman) push $(repo_name)/auth-proxy:arm64

push_amd64: clean build_amd64
	$(podman) build --platform linux/amd64 --tag=$(repo_name)/auth-proxy:amd64 .
	$(podman) push $(repo_name)/auth-proxy:amd64


push: push_arm64 push_amd64
	$(podman) manifest create $(repo_name)/auth-proxy:latest $(repo_name)/auth-proxy:arm64 $(repo_name)/auth-proxy:amd64
	$(podman) manifest push $(repo_name)/auth-proxy:latest $(manifest_dest)
	$(podman) manifest rm $(repo_name)/auth-proxy:latest
