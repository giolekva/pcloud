repo_name ?= dtabidze
podman ?= docker
ifeq ($(podman), podman)
manifest_dest=docker://docker.io/$(repo_name)/port-allocator:latest
endif

clean:
	rm -f server*

build: export CGO_ENABLED=0
build: clean
	/usr/local/go/bin/go build -o server *.go

test: export CGO_ENABLED=0
test:
	/usr/local/go/bin/go test ./...

test: export CGO_ENABLED=0
testv:
	/usr/local/go/bin/go test -v ./...

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
	$(podman) build --platform linux/arm64 --tag=$(repo_name)/port-allocator:arm64 .
	$(podman) push $(repo_name)/port-allocator:arm64

push_amd64: clean build_amd64
	$(podman) build --platform linux/amd64 --tag=$(repo_name)/port-allocator:amd64 .
	$(podman) push $(repo_name)/port-allocator:amd64

push: push_arm64 push_amd64
	$(podman) manifest create $(repo_name)/port-allocator:latest $(repo_name)/port-allocator:arm64 $(repo_name)/port-allocator:amd64
	$(podman) manifest push $(repo_name)/port-allocator:latest $(manifest_dest)
	$(podman) manifest rm $(repo_name)/port-allocator:latest
