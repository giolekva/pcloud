FROM arm64v8/alpine

RUN apk add --no-cache curl

RUN curl --silent --show-error --fail --location \
    --header "Accept: application/tar+gzip, application/x-gzip, application/octet-stream" -o /usr/bin/mc \
    "https://dl.minio.io/client/mc/release/linux-arm64/mc" \
    && chmod 0755 /usr/bin/mc
CMD ["mc"]
