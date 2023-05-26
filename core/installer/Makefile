image_arm64:
	docker build --file=Dockerfile.flux --tag=giolekva/flux:latest . --platform=linux/arm64

push_arm64: image_arm64
	docker push giolekva/flux:latest

build:
	go build -o pcloud cmd/*.go

bootstrap:
	./pcloud bootstrap --kubeconfig=../../priv/kubeconfig --charts-dir=../../charts --admin-pub-key=/Users/lekva/.ssh/id_rsa.pub --admin-priv-key=/Users/lekva/.ssh/id_rsa --soft-serve-ip=192.168.0.211

create_env:
	./pcloud create-env --admin-priv-key=/Users/lekva/.ssh/id_rsa --name=foo

rpuppy:
	./pcloud install --ssh-key=/Users/lekva/.ssh/id_rsa --config=/Users/lekva/dev/src/pcloud/priv/lekva.yaml --app=rpuppy --repo-addr=ssh://localhost:2222/lekva
