FROM gcr.io/distroless/static:nonroot

ARG TARGETARCH

COPY dns-challenge-solver_${TARGETARCH} /dns-challenge-solver
CMD ["/dns-challenge-solver"]
