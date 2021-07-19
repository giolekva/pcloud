FROM alpine:3.9

RUN apk add --no-cache ca-certificates

COPY webhook /usr/local/bin/webhook
RUN chmod +x /usr/local/bin/webhook

ENTRYPOINT ["webhook"]
