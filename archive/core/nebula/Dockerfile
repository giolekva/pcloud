FROM alpine:latest

ARG TARGETARCH

RUN wget https://github.com/slackhq/nebula/releases/download/v1.4.0/nebula-linux-${TARGETARCH}.tar.gz -O nebula.tar.gz
RUN tar -xvf nebula.tar.gz
RUN mv nebula /usr/bin
RUN mv nebula-cert /usr/bin
RUN chmod +x /usr/bin/nebula
RUN chmod +x /usr/bin/nebula-cert
