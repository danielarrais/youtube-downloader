FROM --platform=$BUILDPLATFORM node:24.22.1-bookworm-slim AS frontend

WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm install --global npm@12.2.1
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.27.3-bookworm AS backend

ARG TARGETARCH
ARG TARGETOS

WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend /src/backend/frontend/dist ./frontend/dist
RUN CGO_ENABLED=0 GOARCH="$TARGETARCH" GOOS="$TARGETOS" go build \
    -tags web \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/youtube-mp3-downloader .

FROM --platform=$BUILDPLATFORM debian:bookworm-slim AS ffmpeg

ARG TARGETARCH
ARG FFMPEG_BASE_URL=https://github.com/BtbN/FFmpeg-Builds/releases/download/latest

RUN apt-get update \
    && apt-get install --yes --no-install-recommends ca-certificates curl xz-utils \
    && rm -rf /var/lib/apt/lists/*
RUN case "$TARGETARCH" in \
        amd64) filename="ffmpeg-master-latest-linux64-lgpl.tar.xz" ;; \
        arm64) filename="ffmpeg-master-latest-linuxarm64-lgpl.tar.xz" ;; \
        *) echo "unsupported architecture: $TARGETARCH" >&2; exit 1 ;; \
    esac \
    && curl --fail --location --show-error --retry 8 --retry-all-errors --retry-delay 5 \
        "$FFMPEG_BASE_URL/checksums.sha256" \
        --output /tmp/checksums.sha256 \
    && curl --fail --location --show-error --retry 8 --retry-all-errors --retry-delay 5 \
        "$FFMPEG_BASE_URL/$filename" \
        --output "/tmp/$filename" \
    && expected_checksum="$(grep "  $filename$" /tmp/checksums.sha256 | cut -d ' ' -f 1)" \
    && actual_checksum="$(sha256sum "/tmp/$filename" | cut -d ' ' -f 1)" \
    && test "$actual_checksum" = "$expected_checksum" \
    && mkdir -p /out \
    && tar --extract --xz --file "/tmp/$filename" --directory /tmp \
    && ffmpeg_path="$(find /tmp -type f -name ffmpeg -print -quit)" \
    && test -n "$ffmpeg_path" \
    && install -m 0755 "$ffmpeg_path" /out/ffmpeg

FROM --platform=$BUILDPLATFORM debian:bookworm-slim AS ytdlp

ARG TARGETARCH
ARG YTDLP_VERSION=2026.10.21
ARG YTDLP_BASE_URL=https://github.com/yt-dlp/yt-dlp/releases/download/${YTDLP_VERSION}

RUN apt-get update \
    && apt-get install --yes --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/*
RUN case "$TARGETARCH" in \
        amd64) filename="yt-dlp_linux" ;; \
        arm64) filename="yt-dlp_linux_aarch64" ;; \
        *) echo "unsupported architecture: $TARGETARCH" >&2; exit 1 ;; \
    esac \
    && curl --fail --location --show-error --retry 8 --retry-all-errors --retry-delay 5 \
        "$YTDLP_BASE_URL/SHA2-256SUMS" \
        --output /tmp/ytdlp-checksums.sha256 \
    && curl --fail --location --show-error --retry 8 --retry-all-errors --retry-delay 5 \
        "$YTDLP_BASE_URL/$filename" \
        --output "/tmp/$filename" \
    && grep "  $filename$" /tmp/ytdlp-checksums.sha256 \
        > /tmp/ytdlp-checksum.sha256 \
    && (cd /tmp && sha256sum --check ytdlp-checksum.sha256) \
    && mkdir -p /out \
    && install -m 0755 "/tmp/$filename" /out/yt-dlp

FROM node:24.22.1-bookworm-slim

RUN apt-get update \
	&& apt-get install --yes --no-install-recommends \
		ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=ffmpeg /out/ffmpeg /usr/local/bin/ffmpeg
COPY --from=ytdlp /out/yt-dlp /usr/local/bin/yt-dlp
COPY --from=backend /out/youtube-mp3-downloader /usr/local/bin/youtube-mp3-downloader

ENV WEB_ADDR=:8080 \
    DATA_DIR=/data \
    DOWNLOAD_DIR=/downloads \
    HEALTHCHECK_URL=http://127.0.0.1:8080/healthz \
    HOME=/data

VOLUME ["/data", "/downloads"]
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/usr/local/bin/youtube-mp3-downloader", "healthcheck"]

ENTRYPOINT ["youtube-mp3-downloader"]
