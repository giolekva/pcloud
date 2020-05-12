FROM golang:1-alpine AS build

RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh

WORKDIR /helm
RUN wget https://get.helm.sh/helm-v3.2.1-linux-arm64.tar.gz
RUN tar -xvf helm-v3.2.1-linux-arm64.tar.gz

ENV GOOS linux
ENV GOARCH $BUILDPLATFORM
ENV CGO_ENABLED 0
ENV GO111MODULE on

WORKDIR $GOPATH/src/github.com/giolekva/pcloud/appmanager
COPY . .
RUN go build -o $GOPATH/bin/app-manager -trimpath -ldflags="-s -w" cmd/main.go

FROM alpine:latest
COPY --from=build /go/bin/app-manager /usr/bin
RUN chmod a+x /usr/bin/app-manager
COPY --from=build /helm/linux-arm64/helm /usr/bin/helm
RUN chmod a+x /usr/bin/helm
