FROM golang:1.26.9-bookworm AS builder

ARG WAILS_VERSION=v2.12.0
ARG FFMPEG_URL=https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip
ARG YTDLP_BASE_URL=https://github.com/yt-dlp/yt-dlp/releases/latest/download

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        gcc-mingw-w64-x86-64 \
        nsis \
        npm \
        unzip && \
    rm -rf /var/lib/apt/lists/*

RUN go install github.com/wailsapp/wails/v2/cmd/wails@${WAILS_VERSION}

WORKDIR /src
COPY . .

RUN npm ci --prefix frontend

WORKDIR /src/backend

RUN mkdir -p build/windows/installer/resources /tmp/ffmpeg && \
    curl -L --fail --retry 3 "${FFMPEG_URL}" -o /tmp/ffmpeg.zip && \
    unzip -q /tmp/ffmpeg.zip -d /tmp/ffmpeg && \
    find /tmp/ffmpeg -type f -iname ffmpeg.exe -exec cp {} build/windows/installer/resources/ffmpeg.exe \; && \
    test -f build/windows/installer/resources/ffmpeg.exe

RUN curl -L --fail --retry 3 "${YTDLP_BASE_URL}/SHA2-256SUMS" -o /tmp/ytdlp-checksums.sha256 && \
    curl -L --fail --retry 3 "${YTDLP_BASE_URL}/yt-dlp.exe" -o /tmp/yt-dlp.exe && \
    grep "  yt-dlp.exe$" /tmp/ytdlp-checksums.sha256 > /tmp/ytdlp-checksum.sha256 && \
    (cd /tmp && sha256sum --check ytdlp-checksum.sha256) && \
    cp /tmp/yt-dlp.exe build/windows/installer/resources/yt-dlp.exe && \
    test -f build/windows/installer/resources/yt-dlp.exe

RUN wails build \
    -platform windows/amd64 \
    -nsis \
    -webview2 embed \
    -clean

FROM scratch AS artifact
COPY --from=builder /src/backend/build/bin/ /
