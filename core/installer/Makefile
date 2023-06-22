clean:
	rm -rf tmp
	rm -f server_*
	rm -f pcloud

push_fluxcd_arm64:
	docker build --file=Dockerfile.flux --tag=giolekva/flux:latest . --platform=linux/arm64
	docker push giolekva/flux:latest

build: clean
	go build -o pcloud cmd/*.go

bootstrap:
	./pcloud bootstrap --kubeconfig=../../priv/kubeconfig --charts-dir=../../charts --admin-pub-key=/Users/lekva/.ssh/id_rsa.pub --soft-serve-ip=192.168.0.211 --storage-dir=/pcloud-storage/longhorn

create_env:
	./pcloud create-env --admin-priv-key=/Users/lekva/.ssh/id_rsa --name=lekva --ip=192.168.0.211 --admin-username=gio

rpuppy:
	./pcloud install --ssh-key=/Users/lekva/.ssh/id_rsa --app=rpuppy --repo-addr=ssh://localhost:2222/lekva

appmanager:
	./pcloud appmanager --ssh-key=/Users/lekva/.ssh/id_rsa --repo-addr=ssh://192.168.0.211/lekva --port=9090

welc:
	./pcloud welcome --ssh-key=/Users/lekva/.ssh/id_rsa --repo-addr=ssh://192.168.0.211/lekva --port=9090




## installer image
build_arm64: export CGO_ENABLED=0
build_arm64: export GO111MODULE=on
build_arm64: export GOOS=linux
build_arm64: export GOARCH=arm64
build_arm64:
	go build -o server_arm64 cmd/*.go

push: clean build_arm64
	mkdir tmp
	cp -r ../../charts tmp/
	podman build --tag=giolekva/pcloud-installer:latest .
	rm -rf tmp
	podman push giolekva/pcloud-installer:latest
