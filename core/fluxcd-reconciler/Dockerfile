FROM gcr.io/distroless/static:nonroot

ARG TARGETARCH

COPY tmp/charts /charts

COPY fluxcd_reconciler_${TARGETARCH} /fluxcd_reconciler
