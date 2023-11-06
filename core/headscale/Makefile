clean:
	rm -f server_*

build_arm64: export CGO_ENABLED=0
build_arm64: export GO111MODULE=on
build_arm64: export GOOS=linux
build_arm64: export GOARCH=arm64
build_arm64:
	go build -o server_arm64 *.go

build_amd64: export CGO_ENABLED=0
build_amd64: export GO111MODULE=on
build_amd64: export GOOS=linux
build_amd64: export GOARCH=amd64
build_amd64:
	go build -o server_amd64 *.go

push_arm64: clean build_arm64
	podman build --platform linux/arm64 --tag=giolekva/headscale-api:arm64 .
	podman push giolekva/headscale-api:arm64

push_amd64: clean build_amd64
	podman build --platform linux/amd64 --tag=giolekva/headscale-api:amd64 .
	podman push giolekva/headscale-api:amd64


push: push_arm64 push_amd64
	podman manifest create giolekva/headscale-api:latest giolekva/headscale-api:arm64 giolekva/headscale-api:amd64
	podman manifest push giolekva/headscale-api:latest docker://docker.io/giolekva/headscale-api:latest
	podman manifest rm giolekva/headscale-api:latest
