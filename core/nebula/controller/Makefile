clean:
	rm -f controller_* web_*

generate:
	rm -rf generated
	./hack/generate.sh

controller_arm64: export CGO_ENABLED=0
controller_arm64: export GO111MODULE=on
controller_arm64: export GOOS=linux
controller_arm64: export GOARCH=arm64
controller_arm64:
	go mod tidy
	go mod vendor
	go build -o controller_arm64 main.go

controller_amd64: export CGO_ENABLED=0
controller_amd64: export GO111MODULE=on
controller_amd64: export GOOS=linux
controller_amd64: export GOARCH=amd64
controller_amd64:
	go mod tidy
	go mod vendor
	go build -o controller_amd64 main.go

controller: controller_arm64 controller_amd64

web_arm64: export CGO_ENABLED=0
web_arm64: export GO111MODULE=on
web_arm64: export GOOS=linux
web_arm64: export GOARCH=arm64
web_arm64:
	go build -o web_arm64 web.go

web_amd64: export CGO_ENABLED=0
web_amd64: export GO111MODULE=on
web_amd64: export GOOS=linux
web_amd64: export GOARCH=amd64
web_amd64:
	go build -o web_amd64 web.go

web: web_arm64 web_amd64

push_controller: controller
	docker buildx build -f Dockerfile.controller --tag=giolekva/nebula-controller:latest . --platform=linux/arm64,linux/amd64 --push

push_web: web
	docker buildx build -f Dockerfile.web --tag=giolekva/nebula-web:latest . --platform=linux/arm64,linux/amd64 --push

push: push_controller push_web
