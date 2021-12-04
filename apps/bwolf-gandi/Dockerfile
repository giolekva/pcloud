FROM alpine:3.9

ARG TARGETARCH

RUN apk add --no-cache ca-certificates

COPY webhook_${TARGETARCH} /usr/local/bin/webhook
RUN chmod +x /usr/local/bin/webhook

ENTRYPOINT ["webhook"]
