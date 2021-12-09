clean:
	rm -f api_*

api_arm64: export CGO_ENABLED=0
api_arm64: export GO111MODULE=on
api_arm64: export GOOS=linux
api_arm64: export GOARCH=arm64
api_arm64:
	go1.17.1 build -o api_arm64 main.go

api_amd64: export CGO_ENABLED=0
api_amd64: export GO111MODULE=on
api_amd64: export GOOS=linux
api_amd64: export GOARCH=amd64
api_amd64:
	go1.17.1 build -o api_amd64 main.go

api: clean api_arm64 api_amd64

push: api
	docker buildx build --tag=giolekva/nebula-api:latest . --platform=linux/arm64,linux/amd64 --push
