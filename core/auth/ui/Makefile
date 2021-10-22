build:
	go build -o server *.go

clean:
	rm -f server

image: clean build
	docker build --tag=giolekva/auth-ui . --platform=linux/arm64

push: image
	docker push giolekva/auth-ui:latest


push_arm64: export GOOS=linux
push_arm64: export GOARCH=arm64
push_arm64: export CGO_ENABLED=0
push_arm64: export GO111MODULE=on
push_arm64: push
