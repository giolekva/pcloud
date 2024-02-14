clean:
	rm -f server server_*

build: clean
	go build -o server *.go

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

push_arm64: clean build_arm64
	docker build --platform linux/arm64 --tag=dtabidze/auth-ui:arm64 .
	docker push dtabidze/auth-ui:arm64

push_amd64: clean build_amd64
	docker build --platform linux/amd64 --tag=dtabidze/auth-ui:amd64 .
	docker push dtabidze/auth-ui:amd64


push: push_arm64 push_amd64
	docker manifest create dtabidze/auth-ui:latest dtabidze/auth-ui:arm64 dtabidze/auth-ui:amd64
	docker manifest push dtabidze/auth-ui:latest
	docker manifest rm dtabidze/auth-ui:latest
