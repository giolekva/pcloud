FROM golang:1-alpine AS build

RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh

ENV GOOS linux
ENV GOARCH $BUILDPLATFORM
ENV CGO_ENABLED 0
ENV GO111MODULE off

RUN go get github.com/golang/glog
RUN go get github.com/itaysk/regogo

RUN mkdir -p $GOPATH/src/github.com/giolekva/pcloud/apps/minio/importer
COPY . $GOPATH/src/github.com/giolekva/pcloud/apps/minio/importer
WORKDIR $GOPATH/src/github.com/giolekva/pcloud/apps/minio/importer/cmd

RUN mkdir -p /app/build
RUN ls -la
RUN go build -o /app/build/minio-importer -trimpath -ldflags="-s -w" main.go

FROM alpine:latest
WORKDIR /
COPY --from=build /app/build/minio-importer /usr/bin
RUN chmod a+x /usr/bin/minio-importer

ENV PORT 80
ENV API_ADDR ""
EXPOSE ${PORT}
CMD minio-importer --logtostderr --port=${PORT} --api_addr=${API_ADDR}
