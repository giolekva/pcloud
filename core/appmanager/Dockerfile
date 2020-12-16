FROM golang:1-alpine AS build

ARG GOOS=linux
ARG GOARCH=amd64
ARG CGO_ENABLED=0
ARG GO111MODULE=on

RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh

WORKDIR /helm
RUN wget -O helm.tar.gz https://get.helm.sh/helm-v3.2.1-$GOOS-$GOARCH.tar.gz
RUN tar -xvf helm.tar.gz

WORKDIR $GOPATH/src/github.com/giolekva/pcloud/core/appmanager
COPY . .
RUN go build -o $GOPATH/bin/app-manager -trimpath -ldflags="-s -w" cmd/main.go

FROM alpine:latest
COPY --from=build /go/bin/app-manager /usr/bin
RUN chmod a+x /usr/bin/app-manager
COPY --from=build /helm/*/helm /usr/bin/helm
RUN chmod a+x /usr/bin/helm
