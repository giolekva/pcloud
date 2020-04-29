FROM arm64v8/alpine

RUN apk add --no-cache curl

RUN curl --silent --show-error --fail --location \
    --header "Accept: application/tar+gzip, application/x-gzip, application/octet-stream" -o /usr/bin/minio \
    "https://dl.minio.io/server/minio/release/linux-arm64/minio" \
    && chmod 0755 /usr/bin/minio
RUN curl --silent --show-error --fail --location \
    --header "Accept: application/tar+gzip, application/x-gzip, application/octet-stream" -o /usr/bin/docker-entrypoint.sh \
    "https://raw.githubusercontent.com/minio/minio/master/dockerscripts/docker-entrypoint.sh" \
    && chmod 0755 /usr/bin/docker-entrypoint.sh

EXPOSE 9000
ENTRYPOINT ["/usr/bin/docker-entrypoint.sh"]
VOLUME ["/data"]
CMD ["minio"]
