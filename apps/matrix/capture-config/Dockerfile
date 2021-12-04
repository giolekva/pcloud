FROM alpine:latest

ARG TARGETARCH

COPY capture-config_${TARGETARCH} /usr/bin/capture-config
RUN chmod +x /usr/bin/capture-config
