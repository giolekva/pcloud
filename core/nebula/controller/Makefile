clean:
	rm -f controller web

generate:
	rm -rf generated
	./hack/generate.sh

controller: export CGO_ENABLED=0
controller: export GO111MODULE=on
controller: export GOOS=linux
controller: export GOARCH=arm64
controller:
	go mod tidy
	go mod vendor
	go build -o controller main.go

web: export CGO_ENABLED=0
web: export GO111MODULE=on
web: export GOOS=linux
web: export GOARCH=arm64
web:
	go build -o web web.go

image_controller: controller
	docker build -f Dockerfile.controller --tag=giolekva/nebula-controller:latest . --platform=linux/arm64

image_web: web
	docker build -f Dockerfile.web --tag=giolekva/nebula-web:latest . --platform=linux/arm64

push_controller: image_controller
	docker push giolekva/nebula-controller:latest

push_web: image_web
	docker push giolekva/nebula-web:latest

push_arm64: push_controller push_web
