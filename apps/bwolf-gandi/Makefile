IMAGE_NAME := "giolekva/cert-manager-webhook-gandi"
IMAGE_TAG := "v0.2.0"

checkout:
	git clone --depth 1 --branch $(IMAGE_TAG) https://github.com/bwolf/cert-manager-webhook-gandi.git

clean:
	rm -f webhook_*

build_arm64: export CGO_ENABLED=0
build_arm64: export GO111MODULE=on
build_arm64: export GOOS=linux
build_arm64: export GOARCH=arm64
build_arm64: clean
	cd cert-manager-webhook-gandi && go build -o ../webhook_arm64 -ldflags '-w -extldflags "-static"' .

build_amd64: export CGO_ENABLED=0
build_amd64: export GO111MODULE=on
build_amd64: export GOOS=linux
build_amd64: export GOARCH=amd64
build_amd64: clean
	cd cert-manager-webhook-gandi && go build -o ../webhook_amd64 -ldflags '-w -extldflags "-static"' .

build: clean build_arm64 build_amd64

push: build
	docker buildx build --tag=$(IMAGE_NAME):$(IMAGE_TAG) . --platform=linux/arm64,linux/amd64 --push
