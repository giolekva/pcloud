FROM giolekva/maddy:v0.4.4 AS maddy

ARG TARGETARCH

COPY maddy-web_${TARGETARCH} /usr/bin/maddy-web
RUN chmod +x /usr/bin/maddy-web
