## Build
FROM golang:1.27.4-bookworm AS build

ARG TARGETARCH=amd64

WORKDIR /app

RUN if [ "$TARGETARCH" != "amd64" ]; then \
      echo "Docker image currently supports linux/amd64 only (got linux/$TARGETARCH)" >&2; \
      exit 1; \
    fi

ARG CHROME_VERSION=158.0.8127.88
ARG CHROME_SHA256=59e3b43f8b3aaa292238fbdcfb36a93127debd53e212510dcdc2cc8812b4b57e

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl unzip \
    && rm -rf /var/lib/apt/lists/*
COPY ./ ./
RUN make build
RUN chmod +x ./get_chrome.sh \
    && CHROME_VERSION="${CHROME_VERSION}" CHROME_SHA256="${CHROME_SHA256}" ./get_chrome.sh


## Deploy
FROM ubuntu:24.04

WORKDIR /

COPY --from=build /app/bin/crawlergo /crawlergo
COPY --from=build /app/latest/ /chrome/
RUN set -eu; \
    install_runtime_dependencies() { \
      apt-get install -y --no-install-recommends \
        ca-certificates fonts-liberation libasound2t64 libatk-bridge2.0-0t64 libatk1.0-0t64 \
        libcairo2 libcups2t64 libdbus-1-3 libdrm2 libexpat1 libfontconfig1 libgbm1 libgcc-s1 \
        libglib2.0-0t64 libgtk-3-0t64 libnspr4 libnss3 libpango-1.0-0 libpangocairo-1.0-0 \
        libstdc++6 libx11-6 libx11-xcb1 libxcb1 libxcomposite1 libxcursor1 libxdamage1 \
        libxext6 libxfixes3 libxi6 libxinerama1 libxkbcommon0 libxrandr2 libxrender1 \
        libxshmfence1 libxss1 libxtst6; \
    }; \
    apt-get update -o Acquire::Retries=3; \
    attempt=1; \
    while [ "$attempt" -le 3 ]; do \
      if install_runtime_dependencies; then break; fi; \
      if [ "$attempt" -eq 3 ]; then exit 1; fi; \
      echo "Runtime package installation failed; refreshing package indexes (attempt $attempt/3)." >&2; \
      apt-get update -o Acquire::Retries=3; \
      sleep "$attempt"; \
      attempt=$((attempt + 1)); \
    done; \
    rm -rf /var/lib/apt/lists/*

ENTRYPOINT ["/crawlergo", "-c", "/chrome/chrome"]
