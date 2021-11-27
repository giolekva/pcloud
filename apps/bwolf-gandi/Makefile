IMAGE_NAME := "giolekva/cert-manager-webhook-gandi"
IMAGE_TAG := "v0.2.0"

checkout:
	git clone --depth 1 --branch v0.2.0 https://github.com/bwolf/cert-manager-webhook-gandi.git

clean:
	rm -f webhook

build: export CGO_ENABLED=0
build: export GO111MODULE=on
build: clean
	cd cert-manager-webhook-gandi && go build -o ../webhook -ldflags '-w -extldflags "-static"' .

image: build
	docker build --tag=$(IMAGE_NAME):$(IMAGE_TAG) . --platform=linux/arm64

push: image
	docker push $(IMAGE_NAME):$(IMAGE_TAG)


push_arm64: export GOOS=linux
push_arm64: export GOARCH=arm64
push_arm64: push
