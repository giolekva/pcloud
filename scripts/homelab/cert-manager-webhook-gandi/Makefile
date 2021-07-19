IMAGE_NAME := "cert-manager-webhook-gandi"
IMAGE_TAG := "latest"

OUT := $(shell pwd)/_out

$(shell mkdir -p "$(OUT)")

verify:
	go test -v .

clean:
	rm -f webhook
build: export CGO_ENABLED=0
build: export GO111MODULE=on
build:
	go1.16 build -o webhook -ldflags '-w -extldflags "-static"' .

image: clean build
	docker build --tag=giolekva/cert-manager-webhook-gandi .

push: image
	docker push giolekva/cert-manager-webhook-gandi:latest


.PHONY: rendered-manifest.yaml
rendered-manifest.yaml:
#	    --name cert-manager-webhook-gandi $BACKSLASH
	helm template \
        --set image.repository=$(IMAGE_NAME) \
        --set image.tag=$(IMAGE_TAG) \
        deploy/cert-manager-webhook-gandi > "$(OUT)/rendered-manifest.yaml"

push_arm64: export CGO_ENABLED=0
push_arm64: export GOOS=linux
push_arm64: export GOARCH=arm64
push_arm64: push
