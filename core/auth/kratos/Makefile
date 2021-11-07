image:
	docker build --tag=giolekva/ory-kratos:latest . --platform=linux/arm64

push: image
	docker push giolekva/ory-kratos:latest


push_arm64: export GOOS=linux
push_arm64: export GOARCH=arm64
push_arm64: export CGO_ENABLED=0
push_arm64: export GO111MODULE=on
push_arm64: push
