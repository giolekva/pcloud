FROM gcr.io/distroless/static:nonroot

ARG TARGETARCH

COPY tmp/charts /charts

COPY server_${TARGETARCH} /usr/bin/pcloud-installer
