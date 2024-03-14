FROM alpine:latest

ARG TARGETARCH

COPY memberships_${TARGETARCH} /usr/bin/memberships
RUN chmod +x /usr/bin/memberships