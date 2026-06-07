# YouTube MP3 Downloader

Desktop application built with Go, Wails, and React for downloading YouTube
videos and playlists as MP3 files. The queue, output directory, quality, and
language are persisted between restarts.

The same frontend can also run as a web application. The desktop version uses
the native Wails bridge, while the web version communicates with the Go backend
through an HTTP API.

## Project structure

- `backend/` - Go application, Wails desktop host, web API, packaging, and platform build assets
- `frontend/` - React and TypeScript interface shared by desktop and web builds
- `scripts/` - local build helpers
- `.github/workflows/` - validation, release, and Docker image pipelines

## Technologies and libraries

The versions below correspond to the direct dependencies currently in use.
See `backend/go.mod` and `frontend/package-lock.json` for the complete
dependency lists and exact versions.

### Backend and desktop

| Project | Version | Purpose |
| --- | --- | --- |
| [Go](https://go.dev/) | 1.26.4 | Backend, HTTP API, queue management, persistence, and download processing. |
| [Wails](https://wails.io/docs/introduction/) | 2.12.0 | Desktop application and bridge between Go and the React frontend. |
| [kkdai/youtube](https://github.com/kkdai/youtube) | 2.10.6 | YouTube video and playlist metadata and audio stream retrieval. |
| [FFmpeg](https://ffmpeg.org/documentation.html) | Provided by the system or image | Converts downloaded audio into MP3 at the supported quality settings. |
| [google/uuid](https://github.com/google/uuid) | 1.6.0 | Generates unique identifiers for queue items. |
| [ulikunitz/xz](https://github.com/ulikunitz/xz) | 0.5.15 | Extracts Linux FFmpeg builds distributed as `tar.xz` archives. |

### Frontend

| Project | Version | Purpose |
| --- | --- | --- |
| [React](https://react.dev/reference/react) | 18.2 | Shared interface for the desktop and web versions. |
| [TypeScript](https://www.typescriptlang.org/docs/) | 5.3 | Type checking and frontend compilation. |
| [Vite](https://vite.dev/guide/) | 8.0 | Development server and static frontend build. |
| [Tailwind CSS](https://tailwindcss.com/docs/) | 3.4 | Interface styling. |
| [PostCSS](https://postcss.org/) and [Autoprefixer](https://github.com/postcss/autoprefixer) | 8.4 / 10.4 | Generated CSS processing and browser compatibility. |

### Packaging and automation

| Project | Purpose |
| --- | --- |
| [Docker](https://docs.docker.com/) | Multi-stage build and isolated execution of the web version. |
| [Docker Compose](https://docs.docker.com/compose/) | Port, persistent volume, and container update configuration. |
| [GitHub Actions](https://docs.github.com/actions) | Project validation and automated Docker Hub image publishing. |

## Development

Requirements:

- Go 1.26.4+
- Node.js 20.19+
- Native Wails dependencies for the target operating system

During the first conversion, the application looks for a bundled or
system-installed FFmpeg. If none is found, it automatically downloads and
validates a build for Linux or Windows (`amd64` and `arm64`) and stores it in
`~/.youtube-mp3-downloader-bin/`. The first use therefore requires internet
access, but the binary does not need to be committed to the repository.

On Linux with WebKitGTK 4.1:

```bash
cd backend
go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 dev -tags webkit2_41
```

The Vite frontend is also available at `http://localhost:5173`.

To use the browser with access to the Go methods during development, open the
URL shown by Wails after `To develop in the browser`, not the direct Vite port.

## Web with Docker

The image contains the backend, compiled frontend, FFmpeg, and TLS
certificates. The pipeline will publish `linux/amd64` and `linux/arm64` images
to:

```text
docker.io/danielarrais/youtube-mp3-downloader
```

### Docker Compose

Create the directory that will receive the MP3 files and start the public
image:

```bash
cp .env.example .env
mkdir -p downloads
docker compose pull
docker compose up -d
```

Open `http://localhost:8080`. Completed MP3 files can be downloaded through the
browser and are also stored in the local `downloads/` directory.

The `.env` file can override the image, tag, published port, and local download
directory:

| Compose variable | Default | Description |
| --- | --- | --- |
| `DOCKER_IMAGE` | `danielarrais/youtube-mp3-downloader` | Docker Hub repository. |
| `DOCKER_TAG` | `latest` | Image tag, for example `1.0.16`. |
| `HOST_PORT` | `8080` | Port published on the host. |
| `DOWNLOADS_PATH` | `./downloads` | Host directory that receives completed MP3 files. |

To build the image from the local source code:

```bash
docker compose up --build -d
```

To stop the service:

```bash
docker compose down
```

To update to the latest published image:

```bash
docker compose pull
docker compose up -d
```

### Docker Run

The service can also be started without Compose:

```bash
docker volume create youtube-mp3-data
mkdir -p downloads

docker run -d \
  --name youtube-mp3-downloader \
  --restart unless-stopped \
  -p 8080:8080 \
  -e WEB_ADDR=:8080 \
  -e DATA_DIR=/data \
  -e DOWNLOAD_DIR=/downloads \
  -e HEALTHCHECK_URL=http://127.0.0.1:8080/healthz \
  -v youtube-mp3-data:/data \
  -v "$(pwd)/downloads:/downloads" \
  danielarrais/youtube-mp3-downloader:latest
```

### Environment variables

| Variable | Default | Description |
| --- | --- | --- |
| `WEB_ADDR` | `:8080` | Address and port on which the HTTP server listens. |
| `DATA_DIR` | `/data` | Directory for configuration, persisted queue, and temporary cache. |
| `DOWNLOAD_DIR` | `/downloads` | Directory where completed MP3 files are written. |
| `HEALTHCHECK_URL` | `http://127.0.0.1:8080/healthz` | Internal URL used by the image health check. |

When changing `DATA_DIR` or `DOWNLOAD_DIR`, update the corresponding volume
destination as well. When changing the internal port through `WEB_ADDR`, also
update `HEALTHCHECK_URL`. To publish a different host port without changing the
container, keep the default environment variables and use, for example,
`-p 9090:8080`.

### Volumes

| Container path | Contents | Recommendation |
| --- | --- | --- |
| `/data` | `config.json`, `queue.json`, and the working cache. | Persistent Docker volume. |
| `/downloads` | Completed MP3 files. | Bind mount to a host directory. |

Removing the container does not delete this data. `docker compose down -v`
also removes the data volume and should only be used when the queue and
configuration can be discarded.

This configuration does not provide authentication and is intended for local
or private network use. Do not expose the port directly to the internet.

### Image publishing

The `Docker` workflow:

- validates the image build on pull requests without publishing it;
- publishes `latest` on pushes to the `main` branch;
- publishes `1.2.3`, `1.2`, `1`, and `sha-...` for a `v1.2.3` tag;
- supports manual execution through GitHub Actions.

Configure these secrets under
**Settings > Secrets and variables > Actions**:

| Secret | Value |
| --- | --- |
| `DOCKERHUB_USERNAME` | Docker Hub user or organization name. |
| `DOCKERHUB_TOKEN` | Docker Hub access token with write permission. |

Create the public `youtube-mp3-downloader` repository on Docker Hub before the
first publication. The pipeline logs in and publishes to that repository;
running the project locally does not upload any image to the registry.

## Validation

```bash
npm ci --prefix frontend
npm run build --prefix frontend
go -C backend test -race -tags webkit2_41 ./...
go -C backend test -race -tags web ./...
go -C backend vet -tags webkit2_41 ./...
```

## Build

Native build:

```bash
cd backend
go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 build
```

### Windows installer from Linux

The Docker build downloads the compilation tools, Go and Node dependencies,
FFmpeg, and WebView2:

```bash
./scripts/build-windows.sh
```

The installer is written to `backend/build/windows/dist/`.

## Release

Tags in the `vX.Y.Z` format run the `Release` workflow. It validates the
project, creates a Windows `amd64` installer and an Ubuntu 24.04+ `amd64` DEB
package, includes FFmpeg in both, and publishes the artifacts with
`SHA256SUMS` to the GitHub Release.

```bash
git tag -a v1.0.16 -m "Release v1.0.16"
git push origin v1.0.16
```
