FROM golang:1-alpine AS build

RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh

ENV GOOS linux
ENV CGO_ENABLED 0
ENV GO111MODULE on

RUN mkdir -p $GOPATH/src/github.com/giolekva/pcloud/core/events
COPY . $GOPATH/src/github.com/giolekva/pcloud/core/events
WORKDIR $GOPATH/src/github.com/giolekva/pcloud/core/events/cmd
RUN go get ./...

RUN mkdir -p /app/build
RUN ls -la
RUN go build -o /app/build/event-processor -trimpath -ldflags="-s -w" main.go

FROM alpine:latest
WORKDIR /
COPY --from=build /app/build/event-processor /usr/bin
RUN chmod a+x /usr/bin/event-processor

ENV API_ADDR ""
ENV OBJECT_STORE_ADDR ""
CMD minio-importer --logtostderr --api_addr=${API_ADDR} --object_store_addr=${OBJECT_STORE_ADDR}
