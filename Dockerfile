FROM node:20.19.4-bookworm-slim AS frontend

WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.26.4-bookworm AS backend

WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend /src/backend/frontend/dist ./frontend/dist
RUN CGO_ENABLED=0 go build \
    -tags web \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/youtube-mp3-downloader .

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install --yes --no-install-recommends ca-certificates ffmpeg wget \
    && rm -rf /var/lib/apt/lists/*

COPY --from=backend /out/youtube-mp3-downloader /usr/local/bin/youtube-mp3-downloader

RUN mkdir -p /data /downloads

ENV WEB_ADDR=:8080 \
    DATA_DIR=/data \
    DOWNLOAD_DIR=/downloads \
    HEALTHCHECK_URL=http://127.0.0.1:8080/healthz

VOLUME ["/data", "/downloads"]
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --quiet --spider "$HEALTHCHECK_URL" || exit 1

ENTRYPOINT ["youtube-mp3-downloader"]
