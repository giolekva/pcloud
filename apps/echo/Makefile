build:
	go build -o echo *.go

clean:
	rm -f echo

image: clean build
	docker build --tag=giolekva/echo .

push: image
	docker push giolekva/echo:latest


push_arm64: export GOOS=linux
push_arm64: export GOARCH=arm64
push_arm64: export CGO_ENABLED=0
push_arm64: export GO111MODULE=on
push_arm64: push
