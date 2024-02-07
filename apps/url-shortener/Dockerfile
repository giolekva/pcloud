FROM alpine:latest

ARG TARGETARCH

COPY url-shortener_${TARGETARCH} /usr/bin/url-shortener
RUN chmod +x /usr/bin/url-shortener
