repo_name ?= dtabidze
podman ?= docker

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

bootstrap:
	./pcloud --kubeconfig=../../priv/kubeconfig-hetzner bootstrap --env-name=dodo --charts-dir=../../charts --admin-pub-key=/Users/lekva/.ssh/id_rsa.pub --from-ip=192.168.100.210 --to-ip=192.168.100.240 --storage-dir=/pcloud-storage/longhorn

create_env:
	./pcloud --kubeconfig=../../priv/kubeconfig create-env --admin-priv-key=/Users/lekva/.ssh/id_rsa --name=lekva --ip=192.168.0.211 --admin-username=gio

rpuppy:
	./pcloud --kubeconfig=../../priv/kubeconfig install --ssh-key=/Users/lekva/.ssh/id_rsa --app=rpuppy --repo-addr=ssh://localhost:2222/lekva

appmanager:
	./pcloud --kubeconfig=../../priv/kubeconfig appmanager --ssh-key=/Users/lekva/.ssh/id_ed25519 --repo-addr=ssh://localhost:2222/config --port=9090 --app-repo-addr=http://localhost:8080

welc:
	./pcloud --kubeconfig=../../priv/kubeconfig welcome --ssh-key=/Users/lekva/.ssh/id_rsa --repo-addr=ssh://192.168.0.210/config --port=9090

env:
	./pcloud --kubeconfig=../../priv/kubeconfig-hetzner envmanager --ssh-key=/Users/lekva/.ssh/id_rsa --repo-addr=192.168.100.210:22 --repo-name=config --port=9090




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
	$(podman) manifest push $(repo_name)/pcloud-installer:latest 
	$(podman) manifest rm $(repo_name)/pcloud-installer:latest
