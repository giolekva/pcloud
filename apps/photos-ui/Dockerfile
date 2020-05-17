FROM golang:1-alpine AS build

RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh

WORKDIR /app
COPY *.go .

ENV CGO_ENABLED 0
ENV GO111MODULE off
RUN go build -o photos-ui -trimpath -ldflags="-s -w" main.go

FROM alpine:latest
WORKDIR /app
COPY --from=build /app/photos-ui .
COPY *.html .
WORKDIR /app/static
COPY static/photos.js .
WORKDIR /app

RUN chmod a+x /app/photos-ui

ENV PATH="/app:${PATH}"

ENV PORT 80
EXPOSE ${PORT}
ENV PCLOUD_API_ADDR ""
CMD photos-ui --port=${PORT} --pcloud_api_addr=${PCLOUD_API_ADDR}


