FROM golang:1-alpine AS build

RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh wget

# WORKDIR /protoc
# RUN wget https://github.com/protocolbuffers/protobuf/releases/download/v3.11.4/protoc-3.11.4-linux-x86_64.zip
# RUN unzip protoc-3.11.4-linux-x86_64.zip
# RUN rm protoc-3.11.4-linux-x86_64.zip
# ENV PATH=/protoc/bin:$PATH

WORKDIR $GOPATH/src/github.com/giolekva/pcloud/core/api
COPY . .
# RUN go get -v ./...

ENV GO111MODULE on

RUN go build -o $GOPATH/bin/pcloud-api-server  main.go

FROM alpine:latest
WORKDIR /
COPY --from=build /go/bin/pcloud-api-server /usr/bin/
RUN chmod a+x /usr/bin/pcloud-api-server

ENV KUBECONFIG ""
ENV PORT 80
ENV GRAPHQL_ADDRESS ""
ENV DGRAPH_ADMIN_ADDRESS ""

EXPOSE ${PORT}

CMD pcloud-api-server \
    --port=${PORT} \
    --kubeconfig=${KUBECONFIG} \
    --graphql_address=${GRAPHQL_ADDRESS} \
    --dgraph_admin_address=${DGRAPH_ADMIN_ADDRESS}
