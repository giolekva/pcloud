FROM alpine:3.14.2 AS build

WORKDIR /download
# RUN wget https://github.com/fluxcd/flux2/releases/download/v0.29.5/flux_0.29.5_linux_arm64.tar.gz
# RUN tar -zxvf flux_0.29.5_linux_arm64.tar.gz
RUN wget https://github.com/fluxcd/flux2/releases/download/v2.0.0/flux_2.0.0_linux_arm64.tar.gz
RUN tar -zxvf flux_2.0.0_linux_arm64.tar.gz
FROM alpine:3.14.2

WORKDIR /
COPY --from=build /download/flux ./flux
