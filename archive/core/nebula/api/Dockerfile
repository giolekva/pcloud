FROM alpine:latest

ARG TARGETARCH

COPY api_${TARGETARCH} /usr/bin/nebula-api
RUN chmod +x /usr/bin/nebula-api
