clean:
	rm -f server_*

build_arm64: export CGO_ENABLED=0
build_arm64: export GO111MODULE=on
build_arm64: export GOOS=linux
build_arm64: export GOARCH=arm64
build_arm64:
	go build -o server_arm64 *.go

build_amd64: export CGO_ENABLED=0
build_amd64: export GO111MODULE=on
build_amd64: export GOOS=linux
build_amd64: export GOARCH=amd64
build_amd64:
	go build -o server_amd64 *.go

push: clean build_arm64 build_amd64
	docker buildx build --tag=giolekva/auth-ui:latest . --platform=linux/arm64,linux/amd64 --push
