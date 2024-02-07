repo_name ?= giolekva
podman ?= podman

clean:
	rm -f url-shortener*

build_arm64: export CGO_ENABLED=0
build_arm64: export GO111MODULE=on
build_arm64: export GOOS=linux
build_arm64: export GOARCH=arm64
build_arm64:
	/usr/local/go/bin/go build -o url-shortener_arm64 *.go

build_amd64: export CGO_ENABLED=0
build_amd64: export GO111MODULE=on
build_amd64: export GOOS=linux
build_amd64: export GOARCH=amd64
build_amd64:
	/usr/local/go/bin/go build -o url-shortener_amd64 *.go

push_arm64: clean build_arm64
	$(podman) build --platform linux/arm64 --tag=$(repo_name)/url-shortener:arm64 .
	$(podman) push $(repo_name)/url-shortener:arm64

push_amd64: clean build_amd64
	$(podman) build --platform linux/amd64 --tag=$(repo_name)/url-shortener:amd64 .
	$(podman) push $(repo_name)/url-shortener:amd64

push: push_arm64 push_amd64
	$(podman) manifest create $(repo_name)/url-shortener:latest $(repo_name)/url-shortener:arm64 $(repo_name)/url-shortener:amd64
	$(podman) manifest push $(repo_name)/url-shortener:latest docker://docker.io/$(repo_name)/url-shortener:latest
	$(podman) manifest rm $(repo_name)/url-shortener:latest
