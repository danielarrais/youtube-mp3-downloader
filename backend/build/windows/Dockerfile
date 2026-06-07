FROM golang:1.26-bookworm AS builder

ARG WAILS_VERSION=v2.12.0
ARG FFMPEG_URL=https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip

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

RUN wails build \
    -platform windows/amd64 \
    -nsis \
    -webview2 embed \
    -clean

FROM scratch AS artifact
COPY --from=builder /src/backend/build/bin/ /
