FROM golang:1.13-alpine as builder

WORKDIR /home

ENV GOPATH /go
ENV CGO_ENABLED 0
ENV GO111MODULE on

RUN go get github.com/minio/minio
WORKDIR /go/pkg/mod/github.com/minio/minio@v0.0.0-20200426024807-6ecc98fddbfb
RUN go build -tags kqueue -trimpath -o minio

FROM arm32v7/alpine:3.11
COPY --from=builder /go/pkg/mod/github.com/minio/minio@v0.0.0-20200426024807-6ecc98fddbfb/minio /usr/bin/
COPY --from=builder /go/pkg/mod/github.com/minio/minio@v0.0.0-20200426024807-6ecc98fddbfb/dockerscripts/docker-entrypoint.sh /usr/bin/
RUN chmod +x /usr/bin/minio
RUN chmod +x /usr/bin/docker-entrypoint.sh

EXPOSE 9000

ENTRYPOINT ["/usr/bin/docker-entrypoint.sh"]

VOLUME ["/data"]

CMD ["minio"]