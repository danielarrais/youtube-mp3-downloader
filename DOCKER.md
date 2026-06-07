# YouTube MP3 Downloader (Web Version)

Desktop and web application built with Go and React for downloading YouTube videos and playlists as MP3 files. This Docker image provides the **Web Version** of the application.

## Quick Start

### Run with Docker

```bash
docker run -d \
  -p 8080:8080 \
  -v ./downloads:/downloads \
  -v youtube-mp3-data:/data \
  --name youtube-mp3-downloader \
  danielarrais/youtube-mp3-downloader:latest
```

The application will be available at `http://localhost:8080`.

### Run with Docker Compose

Create a `compose.yaml` file:

```yaml
services:
  youtube-mp3-downloader:
    image: danielarrais/youtube-mp3-downloader:latest
    ports:
      - "8080:8080"
    volumes:
      - youtube-mp3-data:/data
      - ./downloads:/downloads
    restart: unless-stopped

volumes:
  youtube-mp3-data:
```

Then run:

```bash
docker compose up -d
```

## Features

- Download individual videos or entire playlists.
- Convert automatically to MP3.
- Persistent queue and settings (via `/data` volume).
- Clean and responsive web interface.

## Volumes

- `/downloads`: This is where your downloaded MP3 files will be stored.
- `/data`: Used to persist application configuration and the download queue.

## Source Code

For more information, visit the [GitHub Repository](https://github.com/danielarrais/best-youtube-mp3-downloader).
