# yt-mp3-go

Aplicativo desktop em Go, Wails e React para baixar vídeos e playlists do
YouTube como MP3. A fila, a pasta de saída, a qualidade e o idioma permanecem
salvos entre reinicializações.

## Desenvolvimento

Requisitos:

- Go 1.26.4+
- Node.js 20.19+
- Dependências nativas do Wails para o sistema operacional

Na primeira conversão, o aplicativo procura o FFmpeg empacotado ou instalado no
sistema. Caso não encontre, baixa e valida automaticamente uma build para
Linux ou Windows (`amd64` e `arm64`), armazenada em
`~/.yt-mp3-downloader-bin/`. Portanto, o primeiro uso requer acesso à internet,
mas o binário não precisa ser versionado no repositório.

No Linux com WebKitGTK 4.1:

```bash
go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 dev -tags webkit2_41
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
go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 build
```

### Instalador Windows no Linux

O build via Docker baixa as ferramentas de compilação, dependências Go e Node,
FFmpeg e WebView2:

```bash
./scripts/build-windows.sh
```

O instalador será gravado em `build/windows/dist/`.

## Release

Tags no formato `vX.Y.Z` executam o workflow `Release`. Ele valida o projeto,
gera um instalador Windows `amd64` e um pacote DEB para Ubuntu 24.04+
`amd64`, inclui o FFmpeg em ambos e publica os arquivos com `SHA256SUMS` na
GitHub Release.

```bash
git tag -a v1.0.16 -m "Release v1.0.16"
git push origin v1.0.16
```
