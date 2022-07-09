image_arm64:
	docker build --file=Dockerfile.flux --tag=giolekva/flux:latest . --platform=linux/arm64

push_arm64: image_arm64
	docker push giolekva/flux:latest

build:
	go1.18 build -o pcloud cmd/*.go

bootstrap:
	./pcloud bootstrap --kubeconfig=../../priv/kubeconfig --charts-dir=../../charts --admin-pub-key=/Users/lekva/.ssh/id_rsa.pub --admin-priv-key=/Users/lekva/.ssh/id_rsa

create_env:
	./pcloud create-env --admin-priv-key=/Users/lekva/.ssh/id_rsa --name=foo

