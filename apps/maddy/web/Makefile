clean:
	rm -f maddy-web

build: clean
	go build -o maddy-web *.go

image: build
	docker build --tag=giolekva/maddy-web:latest . --platform=linux/arm64

push: image
	docker push giolekva/maddy-web:latest

push_arm64: export GOOS=linux
push_arm64: export GOARCH=arm64
push_arm64: export CGO_ENABLED=0
push_arm64: export GO111MODULE=on
push_arm64: push
