FROM alpine:3.14.2

RUN addgroup -S ory; \
    adduser -S ory -G ory -D -u 10000 -h /home/ory -s /bin/nologin; \
    chown -R ory:ory /home/ory

RUN apk add -U --no-cache ca-certificates

RUN wget https://github.com/ory/kratos/releases/download/v0.7.6-alpha.1/kratos_0.7.6-alpha.1_linux_arm64.tar.gz -O kratos.tar.gz
RUN tar -xvf kratos.tar.gz
RUN mv kratos /usr/bin

VOLUME /home/ory

# Declare the standard ports used by Kratos (4433 for public service endpoint, 4434 for admin service endpoint)
EXPOSE 4433 4434

USER 10000

ENTRYPOINT ["kratos"]
CMD ["serve"]
