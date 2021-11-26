clean:
	rm -f auth-smtp

build: clean
	go build -o auth-smtp *.go

image: build
	docker build --tag=giolekva/maddy-auth-smtp:v0.4.4 . --platform=linux/arm64

push: image
	docker push giolekva/maddy-auth-smtp:v0.4.4

push_arm64: export GOOS=linux
push_arm64: export GOARCH=arm64
push_arm64: export CGO_ENABLED=0
push_arm64: export GO111MODULE=on
push_arm64: push
