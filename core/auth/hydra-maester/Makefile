clone:
	git clone --depth 1 --branch v0.0.20 https://github.com/ory/hydra-maester.git

push_arm64: export GOOS=linux
push_arm64: export GOARCH=arm64
push_arm64: export CGO_ENABLED=0
push_arm64: export GO111MODULE=on
build:
	cd hydra-maester && go build -o manager main.go

image_arm64: build
	docker build --tag=giolekva/ory-hydra-maester:latest . --platform=linux/arm64

push_arm64: image_arm64
	docker push giolekva/ory-hydra-maester:latest
