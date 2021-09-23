FROM alpine:latest

COPY nebula/build/linux-arm64/nebula /usr/bin
COPY nebula/build/linux-arm64/nebula-cert /usr/bin
RUN chmod +x /usr/bin/nebula
RUN chmod +x /usr/bin/nebula-cert
