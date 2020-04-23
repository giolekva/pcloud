FROM ubuntu:latest

RUN apt-get update --fix-missing
RUN apt-get -y upgrade
RUN apt-get -y install wget git bash unzip

WORKDIR /tmp
RUN wget https://dl.google.com/go/go1.14.linux-amd64.tar.gz
RUN tar -xvf go1.14.linux-amd64.tar.gz
RUN mv go /usr/local
RUN rm go1.14.linux-amd64.tar.gz

ENV GOROOT=/usr/local/go
ENV GOPATH=/src/go
ENV GOBIN=$GOPATH/bin
ENV PATH=$GOBIN:$GOROOT/bin:$PATH

RUN go get -u google.golang.org/grpc

WORKDIR /src/protoc
RUN wget https://github.com/protocolbuffers/protobuf/releases/download/v3.11.4/protoc-3.11.4-linux-x86_64.zip
RUN unzip protoc-3.11.4-linux-x86_64.zip
RUN rm protoc-3.11.4-linux-x86_64.zip
ENV PATH=/src/protoc/bin:$PATH

RUN go get -u github.com/golang/protobuf/protoc-gen-go
RUN go get -u google.golang.org/protobuf/encoding/prototext
RUN go get -u github.com/google/uuid
RUN go get -u github.com/itaysk/regogo
RUN go get -u github.com/vektah/gqlparser
RUN go get -u github.com/golang/glog

COPY . /tmp/pc
WORKDIR /tmp/pc
RUN go build main.go
RUN rm -r /tmp/pc

# Argo cli
# WORKDIR /tmp
# RUN wget https://github.com/argoproj/argo/releases/download/v2.8.0-rc1/argo-linux-amd64
# RUN chmod 0777 argo-linux-amd64
# RUN mv argo-linux-amd64 /usr/local/bin/argo

WORKDIR /src/go/src/github.com/giolekva/pcloud/controller
