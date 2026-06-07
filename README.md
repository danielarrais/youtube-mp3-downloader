# YouTube MP3 Downloader

Aplicativo desktop em Go, Wails e React para baixar vídeos e playlists do
YouTube como MP3. A fila, a pasta de saída, a qualidade e o idioma permanecem
salvos entre reinicializações.

O mesmo frontend também pode rodar como aplicação web. No desktop, ele usa a
ponte nativa do Wails; no servidor web, usa uma API HTTP do backend Go.

## Desenvolvimento

Requisitos:

- Go 1.26.4+
- Node.js 20.19+
- Dependências nativas do Wails para o sistema operacional

Na primeira conversão, o aplicativo procura o FFmpeg empacotado ou instalado no
sistema. Caso não encontre, baixa e valida automaticamente uma build para
Linux ou Windows (`amd64` e `arm64`), armazenada em
`~/.youtube-mp3-downloader-bin/`. Portanto, o primeiro uso requer acesso à internet,
mas o binário não precisa ser versionado no repositório.

No Linux com WebKitGTK 4.1:

```bash
go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 dev -tags webkit2_41
```

O frontend também fica disponível em `http://localhost:5173`.

Para usar no navegador com acesso aos métodos Go durante o desenvolvimento,
abra a URL informada pelo Wails como `To develop in the browser`, e não a porta
direta do Vite.

## Web com Docker

A imagem contém o backend, o frontend compilado, FFmpeg e certificados TLS. A
pipeline a publicará para `linux/amd64` e `linux/arm64` em:

```text
docker.io/danielarrais/youtube-mp3-downloader
```

### Docker Compose

Crie a pasta que receberá os MP3 e suba a imagem pública:

```bash
cp .env.example .env
mkdir -p downloads
docker compose pull
docker compose up -d
```

Abra `http://localhost:8080`. Os MP3 ficam disponíveis pelo botão de download
do navegador e também na pasta local `downloads/`.

O arquivo `.env` permite trocar imagem, tag, porta publicada e pasta local:

| Variável do Compose | Padrão | Descrição |
| --- | --- | --- |
| `DOCKER_IMAGE` | `danielarrais/youtube-mp3-downloader` | Repositório no Docker Hub. |
| `DOCKER_TAG` | `latest` | Tag da imagem; por exemplo `1.0.16`. |
| `HOST_PORT` | `8080` | Porta exposta no host. |
| `DOWNLOADS_PATH` | `./downloads` | Pasta do host que recebe os MP3. |

Para construir a imagem a partir do código local:

```bash
docker compose up --build -d
```

Para encerrar:

```bash
docker compose down
```

Para atualizar a imagem publicada:

```bash
docker compose pull
docker compose up -d
```

### Docker Run

O mesmo serviço pode ser iniciado sem Compose:

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

### Variáveis de ambiente

| Variável | Padrão | Descrição |
| --- | --- | --- |
| `WEB_ADDR` | `:8080` | Endereço e porta em que o servidor HTTP escuta. |
| `DATA_DIR` | `/data` | Diretório da configuração, fila persistida e cache temporário. |
| `DOWNLOAD_DIR` | `/downloads` | Diretório em que os MP3 concluídos são gravados. |
| `HEALTHCHECK_URL` | `http://127.0.0.1:8080/healthz` | URL interna usada pelo healthcheck da imagem. |

Ao alterar `DATA_DIR` ou `DOWNLOAD_DIR`, ajuste também o destino do volume
correspondente. Ao alterar a porta interna em `WEB_ADDR`, atualize também
`HEALTHCHECK_URL`. Para publicar outra porta somente no host, mantenha as
variáveis padrão e use, por exemplo, `-p 9090:8080`.

### Volumes

| Caminho no container | Conteúdo | Recomendação |
| --- | --- | --- |
| `/data` | `config.json`, `queue.json` e cache de trabalho. | Volume Docker persistente. |
| `/downloads` | Arquivos MP3 concluídos. | Bind mount para uma pasta do host. |

Remover o container não apaga esses dados. `docker compose down -v` também
remove o volume de dados e deve ser usado somente quando a fila e as
configurações puderem ser descartadas.

Essa configuração não possui autenticação e foi projetada para uso local ou em
rede privada. Não exponha a porta diretamente à internet.

### Publicação da imagem

O workflow `Docker`:

- valida o build em pull requests sem publicar;
- publica `latest` quando há push na branch `main`;
- publica `1.2.3`, `1.2`, `1` e `sha-...` para a tag `v1.2.3`;
- permite execução manual pelo GitHub Actions.

Configure estes secrets em **Settings > Secrets and variables > Actions**:

| Secret | Valor |
| --- | --- |
| `DOCKERHUB_USERNAME` | Nome do usuário ou organização no Docker Hub. |
| `DOCKERHUB_TOKEN` | Access token do Docker Hub com permissão de escrita. |

Crie previamente no Docker Hub o repositório público
`youtube-mp3-downloader`. A pipeline fará login e publicará nele; executar os
arquivos localmente não envia nenhuma imagem ao registry.

## Validação

```bash
npm ci --prefix frontend
npm run build --prefix frontend
go test -race ./...
go test -race -tags web ./...
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
