FROM alpine:latest

ARG TARGETARCH

COPY tmp/charts /charts

COPY server_${TARGETARCH} /usr/bin/pcloud-installer
RUN chmod +x /usr/bin/pcloud-installer
