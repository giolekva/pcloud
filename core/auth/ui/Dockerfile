FROM alpine:latest

ARG TARGETARCH

COPY server_${TARGETARCH} /usr/bin/server
RUN chmod +x /usr/bin/server
