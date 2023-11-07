clean:
	rm -f rpuppy

build_arm64: export CGO_ENABLED=0
build_arm64: export GO111MODULE=on
build_arm64: export GOOS=linux
build_arm64: export GOARCH=arm64
build_arm64:
	go build -o rpuppy_arm64 *.go

build_amd64: export CGO_ENABLED=0
build_amd64: export GO111MODULE=on
build_amd64: export GOOS=linux
build_amd64: export GOARCH=amd64
build_amd64:
	go build -o rpuppy_amd64 *.go

push_arm64: clean build_arm64
	podman build --platform linux/arm64 --tag=giolekva/rpuppy:arm64 .
	podman push giolekva/rpuppy:arm64

push_amd64: clean build_amd64
	podman build --platform linux/amd64 --tag=giolekva/rpuppy:amd64 .
	podman push giolekva/rpuppy:amd64

push: push_arm64 push_amd64
	podman manifest create giolekva/rpuppy:latest giolekva/rpuppy:arm64 giolekva/rpuppy:amd64
	podman manifest push giolekva/rpuppy:latest docker://docker.io/giolekva/rpuppy:latest
	podman manifest rm giolekva/rpuppy:latest
