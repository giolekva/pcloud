FROM golang:1-buster as build

ENV GOPATH /go
ENV GO111MODULE on

WORKDIR /app
RUN wget https://github.com/dgraph-io/dgraph/archive/v20.03.3.tar.gz
RUN tar -zxvf v20.03.3.tar.gz
RUN mkdir -p $GOPATH/src/github.com/dgraph-io
RUN mv dgraph-20.03.3 $GOPATH/src/github.com/dgraph-io/dgraph
WORKDIR $GOPATH/src/github.com/dgraph-io/dgraph/dgraph
RUN go get -v -d ./...

ENV CGO_ENABLED 1
# ENV GOOS linux
# ENV GOARCH arm64
ENV GOFLAGS '-ldflags=-s -ldflags=-w -trimpath'

RUN mkdir -p /app/build
RUN go build -o /app/build/dgraph

FROM debian:stable-slim
COPY --from=build /app/build/dgraph /usr/bin/
RUN chmod +x /usr/bin/dgraph

EXPOSE 8080
EXPOSE 9080

CMD ["dgraph"]