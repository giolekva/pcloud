FROM headscale/headscale:0.22.3

ARG TARGETARCH

COPY server_${TARGETARCH} /usr/bin/headscale-api
RUN chmod +x /usr/bin/headscale-api
