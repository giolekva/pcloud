FROM golang:1-alpine AS build

RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh

WORKDIR /app
RUN go get github.com/itaysk/regogo
COPY main.go /app/main.go

ENV CGO_ENABLED 0
ENV GO111MODULE off
RUN go build -o rpuppy -trimpath -ldflags="-s -w" main.go

FROM alpine:latest
WORKDIR /
COPY --from=build /app/rpuppy /usr/bin/rpuppy

ENV PORT 80
EXPOSE ${PORT}
CMD rpuppy --port=${PORT}


