repo_name ?= dtabidze
podman ?= docker
ifeq ($(podman), podman)
manifest_dest=docker://docker.io/$(repo_name)/pcloud-installer:latest
endif

clean:
	rm -f dns-api*

# TODO(gio): fix go path
build:
	/usr/local/go/bin/go build -o dns-api *.go

build_arm64: export CGO_ENABLED=0
build_arm64: export GO111MODULE=on
build_arm64: export GOOS=linux
build_arm64: export GOARCH=arm64
build_arm64:
	/usr/local/go/bin/go build -o dns-api_arm64 *.go

build_amd64: export CGO_ENABLED=0
build_amd64: export GO111MODULE=on
build_amd64: export GOOS=linux
build_amd64: export GOARCH=amd64
build_amd64:
	/usr/local/go/bin/go build -o dns-api_amd64 *.go

push_arm64: clean build_arm64
	$(podman) build --platform linux/arm64 --tag=giolekva/dns-api:arm64 .
	$(podman) push giolekva/dns-api:arm64

push_amd64: clean build_amd64
	$(podman) build --platform linux/amd64 --tag=giolekva/dns-api:amd64 .
	$(podman) push giolekva/dns-api:amd64

push: push_arm64 push_amd64
	$(podman) manifest create giolekva/dns-api:latest giolekva/dns-api:arm64 giolekva/dns-api:amd64
	$(podman) manifest push giolekva/dns-api:latest $(manifest_dest)
	$(podman) manifest rm giolekva/dns-api:latest
