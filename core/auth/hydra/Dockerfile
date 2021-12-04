FROM alpine:3.14.2

ARG TARGETARCH

RUN addgroup -S ory; \
    adduser -S ory -G ory -D -u 10000 -h /home/ory -s /bin/nologin; \
    chown -R ory:ory /home/ory

RUN apk add -U --no-cache ca-certificates

WORKDIR /downloads
RUN if [[ "${TARGETARCH}" == "amd64" ]]; \
    then \
      wget https://github.com/ory/hydra/releases/download/v1.10.6/hydra_1.10.6_linux_64bit.tar.gz -O hydra.tar.gz ; \
    else \
      wget https://github.com/ory/hydra/releases/download/v1.10.6/hydra_1.10.6_linux_${TARGETARCH}.tar.gz -O hydra.tar.gz ; \
    fi

RUN tar -xvf hydra.tar.gz
RUN mv hydra /usr/bin

VOLUME /home/ory
WORKDIR /home/ory
RUN rm -r /downloads

# Declare the standard ports used by Hydra (4433 for public service endpoint, 4434 for admin service endpoint)
EXPOSE 4433 4434

USER 10000

ENTRYPOINT ["hydra"]
CMD ["serve"]
