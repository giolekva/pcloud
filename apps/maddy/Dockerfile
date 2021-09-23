FROM giolekva/maddy:v0.4.4 AS maddy

# FROM alpine:latest

# COPY --from=maddy /usr/bin/maddyctl /usr/bin
COPY maddy-web /usr/bin
RUN chmod +x /usr/bin/maddy-web
