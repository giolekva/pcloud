clean:
	rm -f capture-config

build: clean
	go build -o capture-config *.go

image: build
	docker build --tag=giolekva/capture-config:latest . --platform=linux/arm64

push: image
	docker push giolekva/capture-config:latest

push_arm64: export GOOS=linux
push_arm64: export GOARCH=arm64
push_arm64: export CGO_ENABLED=0
push_arm64: export GO111MODULE=on
push_arm64: push
