FROM --platform=$BUILDPLATFORM node:20.19.4-bookworm-slim AS frontend

WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.26.4-bookworm AS backend

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
    && curl --fail --location --retry 3 \
        "$FFMPEG_BASE_URL/checksums.sha256" \
        --output /tmp/checksums.sha256 \
    && curl --fail --location --retry 3 \
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

FROM gcr.io/distroless/cc-debian12

COPY --from=ffmpeg /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=ffmpeg /out/ffmpeg /usr/local/bin/ffmpeg
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
