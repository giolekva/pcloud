repo_name ?= dtabidze
podman ?= docker
ifeq ($(podman), podman)
manifest_dest=docker://docker.io/$(repo_name)/pcloud-installer:latest
endif

clean:
	rm -rf tmp
	rm -f server_*
	rm -f pcloud

push_fluxcd_arm64:
	$(podman) build --file=Dockerfile.flux --tag=$(repo_name)/flux:latest . --platform=linux/arm64
	docker push $(repo_name)/flux:latest

build: export CGO_ENABLED=0
build: clean
	/usr/local/go/bin/go build -o pcloud cmd/*.go

test: export CGO_ENABLED=0
test:
	/usr/local/go/bin/go test ./...

test: export CGO_ENABLED=0
testv:
	/usr/local/go/bin/go test -v ./...

bootstrap:
	./pcloud --kubeconfig=../../priv/kubeconfig-hetzner bootstrap --env-name=dodo --charts-dir=../../charts --admin-pub-key=/Users/lekva/.ssh/id_rsa.pub --from-ip=192.168.100.210 --to-ip=192.168.100.240 --storage-dir=/pcloud-storage/longhorn --public-ip=135.181.48.180,65.108.39.172

create_env:
	./pcloud --kubeconfig=../../priv/kubeconfig create-env --admin-priv-key=/Users/lekva/.ssh/id_rsa --name=lekva --ip=192.168.0.211 --admin-username=gio

appmanager:
	./pcloud --kubeconfig=../../priv/kubeconfig-hetzner appmanager --ssh-key=/Users/lekva/.ssh/id_ed25519 --repo-addr=ssh://10.43.196.174/config --port=9090 # --app-repo-addr=http://localhost:8080

dodo-app:
	./pcloud --kubeconfig=../../priv/kubeconfig-hetzner dodo-app --ssh-key=/Users/lekva/.ssh/id_ed25519 --repo-addr=ssh://10.43.196.174/test

welc:
	./pcloud --kubeconfig=../../priv/kubeconfig welcome --ssh-key=/Users/lekva/.ssh/id_rsa --repo-addr=ssh://192.168.0.210/config --port=9090

env:
	./pcloud --kubeconfig=../../priv/kubeconfig-hetzner envmanager --ssh-key=/Users/lekva/.ssh/id_rsa --repo-addr=192.168.100.210:22 --repo-name=config --port=9090

rewrite:
	./pcloud rewrite --ssh-key=/Users/lekva/.ssh/id_ed25519 --repo-addr=ssh://10.43.196.174/config

launcher:
	./pcloud launcher --port=9090 --logout-url=http://localhost:8080 --ssh-key=/Users/lekva/.ssh/id_ed25519 --repo-addr=ssh://10.43.196.174/config

## installer image
build_arm64: export CGO_ENABLED=0
build_arm64: export GO111MODULE=on
build_arm64: export GOOS=linux
build_arm64: export GOARCH=arm64
build_arm64:
	/usr/local/go/bin/go build -o server_arm64 cmd/*.go

build_amd64: export CGO_ENABLED=0
build_amd64: export GO111MODULE=on
build_amd64: export GOOS=linux
build_amd64: export GOARCH=amd64
build_amd64:
	/usr/local/go/bin/go build -o server_amd64 cmd/*.go

push_arm64: clean build_arm64
	mkdir tmp
	cp -r ../../charts tmp/
	$(podman) build --platform linux/arm64 --tag=$(repo_name)/pcloud-installer:arm64 .
	rm -rf tmp
	$(podman) push $(repo_name)/pcloud-installer:arm64

push_amd64: clean build_amd64
	mkdir tmp
	cp -r ../../charts tmp/
	$(podman) build --platform linux/amd64 --tag=$(repo_name)/pcloud-installer:amd64 .
	rm -rf tmp
	$(podman) push $(repo_name)/pcloud-installer:amd64

push: push_arm64 push_amd64
	$(podman) manifest create $(repo_name)/pcloud-installer:latest $(repo_name)/pcloud-installer:arm64 $(repo_name)/pcloud-installer:amd64
	$(podman) manifest push $(repo_name)/pcloud-installer:latest $(manifest_dest)
	$(podman) manifest rm $(repo_name)/pcloud-installer:latest
