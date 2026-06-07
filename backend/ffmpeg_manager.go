package main

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/ulikunitz/xz"
)

const (
	ffmpegReleaseBaseURL = "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest"
	maxFFmpegBinarySize  = 500 << 20
)

var (
	ffmpegMu         sync.Mutex
	cachedFFmpegPath string
)

type ffmpegArtifact struct {
	filename   string
	archive    string
	binaryName string
}

// CheckAndDownloadFFmpeg returns a bundled, cached or system FFmpeg. When none
// is available, it downloads a verified build into the user's private cache.
func CheckAndDownloadFFmpeg() (string, error) {
	ffmpegMu.Lock()
	defer ffmpegMu.Unlock()

	if cachedFFmpegPath != "" {
		return cachedFFmpegPath, nil
	}

	artifact, artifactErr := ffmpegArtifactFor(runtime.GOOS, runtime.GOARCH)
	binaryName := "ffmpeg"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	if executablePath, err := os.Executable(); err == nil {
		bundledPath := filepath.Join(filepath.Dir(executablePath), binaryName)
		if validateFFmpeg(bundledPath) == nil {
			return cacheFFmpegPath(bundledPath), nil
		}
	}

	binDir, err := ffmpegCacheDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", fmt.Errorf("não foi possível criar a pasta de binários: %w", err)
	}

	ffmpegPath := filepath.Join(binDir, binaryName)
	if err := validateFFmpeg(ffmpegPath); err == nil {
		return cacheFFmpegPath(ffmpegPath), nil
	}

	if systemPath, err := exec.LookPath(binaryName); err == nil {
		if validateFFmpeg(systemPath) == nil {
			return cacheFFmpegPath(systemPath), nil
		}
	}

	if artifactErr != nil {
		return "", artifactErr
	}

	fmt.Printf(">>> FFmpeg não encontrado. Baixando %s...\n", artifact.filename)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	if err := downloadAndInstallFFmpeg(ctx, newFFmpegHTTPClient(), ffmpegReleaseBaseURL, artifact, ffmpegPath); err != nil {
		return "", fmt.Errorf("não foi possível baixar o FFmpeg: %w", err)
	}

	fmt.Printf(">>> FFmpeg instalado em %s\n", ffmpegPath)
	return cacheFFmpegPath(ffmpegPath), nil
}

func cacheFFmpegPath(path string) string {
	cachedFFmpegPath = path
	return path
}

func ffmpegCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("não foi possível localizar a pasta do usuário: %w", err)
	}
	return filepath.Join(home, ".youtube-mp3-downloader-bin"), nil
}

func ffmpegArtifactFor(goos, goarch string) (ffmpegArtifact, error) {
	switch {
	case goos == "linux" && goarch == "amd64":
		return ffmpegArtifact{
			filename:   "ffmpeg-master-latest-linux64-gpl.tar.xz",
			archive:    "tar.xz",
			binaryName: "ffmpeg",
		}, nil
	case goos == "windows" && goarch == "amd64":
		return ffmpegArtifact{
			filename:   "ffmpeg-master-latest-win64-gpl.zip",
			archive:    "zip",
			binaryName: "ffmpeg.exe",
		}, nil
	case goos == "linux" && goarch == "arm64":
		return ffmpegArtifact{
			filename:   "ffmpeg-master-latest-linuxarm64-gpl.tar.xz",
			archive:    "tar.xz",
			binaryName: "ffmpeg",
		}, nil
	case goos == "windows" && goarch == "arm64":
		return ffmpegArtifact{
			filename:   "ffmpeg-master-latest-winarm64-gpl.zip",
			archive:    "zip",
			binaryName: "ffmpeg.exe",
		}, nil
	default:
		return ffmpegArtifact{}, fmt.Errorf(
			"download automático do FFmpeg não suportado em %s/%s; instale o FFmpeg no sistema",
			goos,
			goarch,
		)
	}
}

func newFFmpegHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
			TLSHandshakeTimeout:   30 * time.Second,
			ResponseHeaderTimeout: 60 * time.Second,
			IdleConnTimeout:       90 * time.Second,
		},
	}
}

func downloadAndInstallFFmpeg(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	artifact ffmpegArtifact,
	targetPath string,
) error {
	expectedChecksum, err := fetchFFmpegChecksum(ctx, client, baseURL, artifact.filename)
	if err != nil {
		return err
	}

	archiveFile, err := os.CreateTemp(filepath.Dir(targetPath), "ffmpeg-archive-*")
	if err != nil {
		return fmt.Errorf("não foi possível criar o arquivo temporário: %w", err)
	}
	archivePath := archiveFile.Name()
	defer os.Remove(archivePath)

	checksum := sha256.New()
	if err := downloadURL(ctx, client, baseURL+"/"+artifact.filename, io.MultiWriter(archiveFile, checksum)); err != nil {
		archiveFile.Close()
		return err
	}
	if err := archiveFile.Close(); err != nil {
		return fmt.Errorf("não foi possível fechar o download: %w", err)
	}

	actualChecksum := hex.EncodeToString(checksum.Sum(nil))
	if !strings.EqualFold(actualChecksum, expectedChecksum) {
		return fmt.Errorf("checksum inválido para %s", artifact.filename)
	}

	binaryPattern := "ffmpeg-binary-*"
	if extension := filepath.Ext(artifact.binaryName); extension != "" {
		binaryPattern += extension
	}
	binaryFile, err := os.CreateTemp(filepath.Dir(targetPath), binaryPattern)
	if err != nil {
		return fmt.Errorf("não foi possível criar o binário temporário: %w", err)
	}
	binaryPath := binaryFile.Name()
	defer os.Remove(binaryPath)

	if err := extractFFmpeg(archivePath, artifact, binaryFile); err != nil {
		binaryFile.Close()
		return err
	}
	if err := binaryFile.Sync(); err != nil {
		binaryFile.Close()
		return fmt.Errorf("não foi possível sincronizar o binário: %w", err)
	}
	if err := binaryFile.Close(); err != nil {
		return fmt.Errorf("não foi possível fechar o binário: %w", err)
	}
	if err := os.Chmod(binaryPath, 0755); err != nil {
		return fmt.Errorf("não foi possível tornar o FFmpeg executável: %w", err)
	}
	if err := validateFFmpeg(binaryPath); err != nil {
		return fmt.Errorf("o FFmpeg baixado é inválido: %w", err)
	}

	if err := os.Remove(targetPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("não foi possível substituir o FFmpeg existente: %w", err)
	}
	if err := os.Rename(binaryPath, targetPath); err != nil {
		return fmt.Errorf("não foi possível instalar o FFmpeg: %w", err)
	}
	return nil
}

func fetchFFmpegChecksum(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	filename string,
) (string, error) {
	var checksums bytes.Buffer
	if err := downloadURL(ctx, client, baseURL+"/checksums.sha256", &checksums); err != nil {
		return "", fmt.Errorf("não foi possível baixar os checksums: %w", err)
	}

	checksum, err := parseFFmpegChecksum(checksums.String(), filename)
	if err != nil {
		return "", err
	}
	return checksum, nil
}

func downloadURL(ctx context.Context, client *http.Client, url string, destination io.Writer) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("não foi possível criar a requisição: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("falha ao baixar %s: %w", url, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("falha ao baixar %s: status HTTP %d", url, response.StatusCode)
	}
	if _, err := io.Copy(destination, response.Body); err != nil {
		return fmt.Errorf("falha ao salvar %s: %w", url, err)
	}
	return nil
}

func parseFFmpegChecksum(contents, filename string) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(contents))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		entryName := strings.TrimPrefix(fields[len(fields)-1], "*")
		if entryName == filename {
			if len(fields[0]) != sha256.Size*2 {
				return "", fmt.Errorf("checksum inválido para %s", filename)
			}
			if _, err := hex.DecodeString(fields[0]); err != nil {
				return "", fmt.Errorf("checksum inválido para %s", filename)
			}
			return strings.ToLower(fields[0]), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("não foi possível ler os checksums: %w", err)
	}
	return "", fmt.Errorf("checksum não encontrado para %s", filename)
}

func extractFFmpeg(archivePath string, artifact ffmpegArtifact, destination io.Writer) error {
	switch artifact.archive {
	case "zip":
		return extractFFmpegZip(archivePath, artifact.binaryName, destination)
	case "tar.xz":
		return extractFFmpegTarXZ(archivePath, artifact.binaryName, destination)
	default:
		return fmt.Errorf("formato de arquivo desconhecido: %s", artifact.archive)
	}
}

func extractFFmpegZip(archivePath, binaryName string, destination io.Writer) error {
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("não foi possível abrir o ZIP do FFmpeg: %w", err)
	}
	defer archive.Close()

	for _, entry := range archive.File {
		if entry.FileInfo().IsDir() || filepath.Base(entry.Name) != binaryName {
			continue
		}
		source, err := entry.Open()
		if err != nil {
			return fmt.Errorf("não foi possível abrir %s no ZIP: %w", binaryName, err)
		}
		err = copyFFmpegBinary(destination, source)
		source.Close()
		return err
	}
	return fmt.Errorf("%s não encontrado no ZIP", binaryName)
}

func extractFFmpegTarXZ(archivePath, binaryName string, destination io.Writer) error {
	archive, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("não foi possível abrir o TAR.XZ do FFmpeg: %w", err)
	}
	defer archive.Close()

	xzReader, err := xz.NewReader(archive)
	if err != nil {
		return fmt.Errorf("não foi possível descompactar o XZ do FFmpeg: %w", err)
	}
	tarReader := tar.NewReader(xzReader)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("não foi possível ler o TAR do FFmpeg: %w", err)
		}
		if header.Typeflag != tar.TypeReg || filepath.Base(header.Name) != binaryName {
			continue
		}
		return copyFFmpegBinary(destination, tarReader)
	}
	return fmt.Errorf("%s não encontrado no TAR.XZ", binaryName)
}

func copyFFmpegBinary(destination io.Writer, source io.Reader) error {
	written, err := io.Copy(destination, io.LimitReader(source, maxFFmpegBinarySize+1))
	if err != nil {
		return fmt.Errorf("não foi possível extrair o FFmpeg: %w", err)
	}
	if written > maxFFmpegBinarySize {
		return fmt.Errorf("binário do FFmpeg excede o tamanho máximo permitido")
	}
	return nil
}

func validateFFmpeg(path string) error {
	if path == "" {
		return errors.New("caminho vazio")
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return errors.New("o caminho do FFmpeg é um diretório")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, path, "-version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("falha ao executar FFmpeg: %w", err)
	}
	if !bytes.Contains(bytes.ToLower(output), []byte("ffmpeg version")) {
		return errors.New("saída de versão do FFmpeg não reconhecida")
	}
	return nil
}
