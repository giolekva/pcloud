repo_name ?= giolekva
podman ?= docker
ifeq ($($(podman)), $(podman))
manifest_dest=docker://docker.io/$(repo_name)/fluxcd-reconciler:latest
endif

clean:
	rm -rf tmp
	rm -f fluxcd_reconciler fluxcd_reconciler_*

build: export CGO_ENABLED=0
build: clean
	/usr/local/go/bin/go build -o fluxcd_reconciler main.go

run: build
	./fluxcd_reconciler --port=8080 --kubeconfig=/Users/lekva/dev/src/pcloud/priv/kubeconfig-hetzner

## installer image
build_arm64: export CGO_ENABLED=0
build_arm64: export GO111MODULE=on
build_arm64: export GOOS=linux
build_arm64: export GOARCH=arm64
build_arm64:
	/usr/local/go/bin/go build -o fluxcd_reconciler_arm64 main.go

build_amd64: export CGO_ENABLED=0
build_amd64: export GO111MODULE=on
build_amd64: export GOOS=linux
build_amd64: export GOARCH=amd64
build_amd64:
	/usr/local/go/bin/go build -o fluxcd_reconciler_amd64 main.go

push_arm64: clean build_arm64
	mkdir tmp
	cp -r ../../charts tmp/
	$(podman) build --platform linux/arm64 --tag=$(repo_name)/fluxcd-reconciler:arm64 .
	rm -rf tmp
	$(podman) push $(repo_name)/fluxcd-reconciler:arm64

push_amd64: clean build_amd64
	mkdir tmp
	cp -r ../../charts tmp/
	$(podman) build --platform linux/amd64 --tag=$(repo_name)/fluxcd-reconciler:amd64 .
	rm -rf tmp
	$(podman) push $(repo_name)/fluxcd-reconciler:amd64

push: push_arm64 push_amd64
	$(podman) manifest create $(repo_name)/fluxcd-reconciler:latest $(repo_name)/fluxcd-reconciler:arm64 $(repo_name)/fluxcd-reconciler:amd64
	$(podman) manifest push $(repo_name)/fluxcd-reconciler:latest $(manifest_dest)
	$(podman) manifest rm $(repo_name)/fluxcd-reconciler:latest
