# yt-mp3-go

Aplicativo desktop em Go, Wails e React para baixar vídeos e playlists do
YouTube como MP3. A fila, a pasta de saída, a qualidade e o idioma permanecem
salvos entre reinicializações.

## Desenvolvimento

Requisitos:

- Go 1.26
- Node.js 20+
- FFmpeg
- Dependências nativas do Wails para o sistema operacional

No Linux com WebKitGTK 4.1:

```bash
go run github.com/wailsapp/wails/v2/cmd/wails@v2.11.0 dev -tags webkit2_41
```

O frontend também fica disponível em `http://localhost:5173`.

## Validação

```bash
npm ci --prefix frontend
npm run build --prefix frontend
go test -race ./...
go vet ./...
```

## Build

Build nativo:

```bash
go run github.com/wailsapp/wails/v2/cmd/wails@v2.11.0 build
```

### Instalador Windows no Linux

O build via Docker baixa as ferramentas de compilação, dependências Go e Node,
FFmpeg e WebView2:

```bash
./scripts/build-windows.sh
```

O instalador será gravado em `build/windows/dist/`.

### GitHub Actions

O workflow `Build Windows` pode ser executado manualmente ou por uma tag `v*`.
O artefato contém o aplicativo, `ffmpeg.exe` e o bootstrapper do WebView2.
