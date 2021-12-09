clean:
	rm -f controller_*

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

controller: clean controller_arm64 controller_amd64


push: controller
	docker buildx build --tag=giolekva/nebula-controller:latest . --platform=linux/arm64,linux/amd64 --push
