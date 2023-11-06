clean:
	rm -rf tmp
	rm -f server_*
	rm -f pcloud

push_fluxcd_arm64:
	docker build --file=Dockerfile.flux --tag=giolekva/flux:latest . --platform=linux/arm64
	docker push giolekva/flux:latest

build: export CGO_ENABLED=0
build: clean
	go build -o pcloud cmd/*.go

bootstrap:
	./pcloud --kubeconfig=../../priv/kubeconfig-hetzner bootstrap --env-name=dodo --charts-dir=../../charts --admin-pub-key=/Users/lekva/.ssh/id_rsa.pub --from-ip=192.168.100.210 --to-ip=192.168.100.240 --storage-dir=/pcloud-storage/longhorn

create_env:
	./pcloud --kubeconfig=../../priv/kubeconfig create-env --admin-priv-key=/Users/lekva/.ssh/id_rsa --name=lekva --ip=192.168.0.211 --admin-username=gio

rpuppy:
	./pcloud --kubeconfig=../../priv/kubeconfig install --ssh-key=/Users/lekva/.ssh/id_rsa --app=rpuppy --repo-addr=ssh://localhost:2222/lekva

appmanager:
	./pcloud --kubeconfig=../../priv/kubeconfig appmanager --ssh-key=/Users/lekva/.ssh/id_rsa --repo-addr=ssh://192.168.0.211/qwe --port=9090

welc:
	./pcloud --kubeconfig=../../priv/kubeconfig welcome --ssh-key=/Users/lekva/.ssh/id_rsa --repo-addr=ssh://192.168.0.211/lekva --port=9090




## installer image
build_arm64: export CGO_ENABLED=0
build_arm64: export GO111MODULE=on
build_arm64: export GOOS=linux
build_arm64: export GOARCH=arm64
build_arm64:
	go build -o server_arm64 cmd/*.go

build_amd64: export CGO_ENABLED=0
build_amd64: export GO111MODULE=on
build_amd64: export GOOS=linux
build_amd64: export GOARCH=amd64
build_amd64:
	go build -o server_amd64 cmd/*.go

push_arm64: clean build_arm64
	mkdir tmp
	cp -r ../../charts tmp/
	podman build --platform linux/arm64 --tag=giolekva/pcloud-installer:arm64 .
	rm -rf tmp
	podman push giolekva/pcloud-installer:arm64

push_amd64: clean build_amd64
	mkdir tmp
	cp -r ../../charts tmp/
	podman build --platform linux/amd64 --tag=giolekva/pcloud-installer:amd64 .
	rm -rf tmp
	podman push giolekva/pcloud-installer:amd64

push: push_arm64 push_amd64
	podman manifest create giolekva/pcloud-installer:latest giolekva/pcloud-installer:arm64 giolekva/pcloud-installer:amd64
	podman manifest push giolekva/pcloud-installer:latest docker://docker.io/giolekva/pcloud-installer:latest
	podman manifest rm giolekva/pcloud-installer:latest
