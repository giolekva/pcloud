clean:
	rm -f server server_*

build: clean
	go build -o server *.go

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
	podman build --platform linux/arm64 --tag=giolekva/auth-ui:arm64 .
	podman push giolekva/auth-ui:arm64

push_amd64: clean build_amd64
	podman build --platform linux/amd64 --tag=giolekva/auth-ui:amd64 .
	podman push giolekva/auth-ui:amd64


push: push_arm64 push_amd64
	podman manifest create giolekva/auth-ui:latest giolekva/auth-ui:arm64 giolekva/auth-ui:amd64
	podman manifest push giolekva/auth-ui:latest docker://docker.io/giolekva/auth-ui:latest
	podman manifest rm giolekva/auth-ui:latest
