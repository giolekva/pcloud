FROM gcr.io/distroless/static:latest
ARG TARGETARCH
COPY hydra-maester/manager_${TARGETARCH} /manager
USER 1000
ENTRYPOINT ["/manager"]
