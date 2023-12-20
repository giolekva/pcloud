clean:
	rm -f capture-config_*

build_arm64: export CGO_ENABLED=0
build_arm64: export GO111MODULE=on
build_arm64: export GOOS=linux
build_arm64: export GOARCH=arm64
build_arm64:
	go build -o capture-config_arm64 *.go

build_amd64: export CGO_ENABLED=0
build_amd64: export GO111MODULE=on
build_amd64: export GOOS=linux
build_amd64: export GOARCH=amd64
build_amd64:
	go build -o capture-config_amd64 *.go

push_arm64: clean build_arm64
	podman build --platform linux/arm64 --tag=giolekva/capture-config:arm64 .
	podman push giolekva/capture-config:arm64

push_amd64: clean build_amd64
	podman build --platform linux/amd64 --tag=giolekva/capture-config:amd64 .
	podman push giolekva/capture-config:amd64

push: push_arm64 push_amd64
	podman manifest create giolekva/capture-config:latest giolekva/capture-config:arm64 giolekva/capture-config:amd64
	podman manifest push giolekva/capture-config:latest docker://docker.io/giolekva/capture-config:latest
	podman manifest rm giolekva/capture-config:latest
