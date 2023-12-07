clean:
	rm -f dns-challenge-solver*

# TODO(gio): fix go path
build:
	/usr/local/go/bin/go build -o dns-challenge-solver *.go

build_arm64: export CGO_ENABLED=0
build_arm64: export GO111MODULE=on
build_arm64: export GOOS=linux
build_arm64: export GOARCH=arm64
build_arm64:
	/usr/local/go/bin/go build -o dns-challenge-solver_arm64 *.go

build_amd64: export CGO_ENABLED=0
build_amd64: export GO111MODULE=on
build_amd64: export GOOS=linux
build_amd64: export GOARCH=amd64
build_amd64:
	/usr/local/go/bin/go build -o dns-challenge-solver_amd64 *.go

push_arm64: clean build_arm64
	podman build --platform linux/arm64 --tag=giolekva/dns-challenge-solver:arm64 .
	podman push giolekva/dns-challenge-solver:arm64

push_amd64: clean build_amd64
	podman build --platform linux/amd64 --tag=giolekva/dns-challenge-solver:amd64 .
	podman push giolekva/dns-challenge-solver:amd64

push: push_arm64 push_amd64
	podman manifest create giolekva/dns-challenge-solver:latest giolekva/dns-challenge-solver:arm64 giolekva/dns-challenge-solver:amd64
	podman manifest push giolekva/dns-challenge-solver:latest docker://docker.io/giolekva/dns-challenge-solver:latest
	podman manifest rm giolekva/dns-challenge-solver:latest
