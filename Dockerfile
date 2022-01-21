FROM scratch
LABEL org.opencontainers.image.authors="Hsn723" \
      org.opencontainers.image.title="dkim-manager" \
      org.opencontainers.image.source="https://github.com/hsn723/dkim-manager"
WORKDIR /
COPY dkim-manager /
COPY LICENSE /LICENSE
USER 65532:65532

ENTRYPOINT ["/dkim-manager"]
