clean:
	rm -f controller web

generate:
	rm -rf generated
	./hack/generate.sh

controller: clean
	go mod tidy
	go mod vendor
	go build -o controller main.go

web: clean
	go build -o web web.go


# image: clean build
# 	docker build --tag=giolekva/rpuppy-arm .

# push: image
# 	docker push giolekva/rpuppy-arm:latest


# push_arm64: export GOOS=linux
# push_arm64: export GOARCH=arm64
# push_arm64: export CGO_ENABLED=0
# push_arm64: export GO111MODULE=on
# push_arm64: push
