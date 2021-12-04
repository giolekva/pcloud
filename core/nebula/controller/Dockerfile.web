FROM alpine:latest

ARG TARGETARCH

COPY web_${TARGETARCH} /usr/bin/nebula-web
RUN chmod +x /usr/bin/nebula-web
