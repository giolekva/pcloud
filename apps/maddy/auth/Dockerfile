FROM giolekva/maddy:v0.4.4

ARG TARGETARCH

COPY auth-smtp_${TARGETARCH} /usr/bin/auth-smtp
RUN chmod +x /usr/bin/auth-smtp
