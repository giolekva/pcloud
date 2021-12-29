FROM alpine:latest

ARG TARGETARCH

COPY controller_${TARGETARCH} /usr/bin/nebula-controller
RUN chmod +x /usr/bin/nebula-controller

# RUN wget https://github.com/slackhq/nebula/releases/download/v1.4.0/nebula-linux-${TARGETARCH}.tar.gz -O nebula.tar.gz
# RUN tar -xvf nebula.tar.gz
# RUN mv nebula-cert /usr/bin
# RUN chmod +x /usr/bin/nebula-cert
