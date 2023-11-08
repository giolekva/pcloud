FROM alpine:latest

ARG TARGETARCH

COPY apps /pcloud/apps

COPY server_${TARGETARCH} /usr/bin/app-repository
RUN chmod +x /usr/bin/app-repository
