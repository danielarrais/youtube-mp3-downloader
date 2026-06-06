package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/ulikunitz/xz"
)

func TestFFmpegArtifactFor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		goos     string
		goarch   string
		filename string
		archive  string
		wantErr  bool
	}{
		{
			name:     "linux amd64",
			goos:     "linux",
			goarch:   "amd64",
			filename: "ffmpeg-master-latest-linux64-gpl.tar.xz",
			archive:  "tar.xz",
		},
		{
			name:     "windows amd64",
			goos:     "windows",
			goarch:   "amd64",
			filename: "ffmpeg-master-latest-win64-gpl.zip",
			archive:  "zip",
		},
		{
			name:     "linux arm64",
			goos:     "linux",
			goarch:   "arm64",
			filename: "ffmpeg-master-latest-linuxarm64-gpl.tar.xz",
			archive:  "tar.xz",
		},
		{
			name:     "windows arm64",
			goos:     "windows",
			goarch:   "arm64",
			filename: "ffmpeg-master-latest-winarm64-gpl.zip",
			archive:  "zip",
		},
		{name: "unsupported", goos: "darwin", goarch: "amd64", wantErr: true},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			artifact, err := ffmpegArtifactFor(test.goos, test.goarch)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ffmpegArtifactFor() error = %v", err)
			}
			if artifact.filename != test.filename || artifact.archive != test.archive {
				t.Fatalf("ffmpegArtifactFor() = %#v", artifact)
			}
		})
	}
}

func TestParseFFmpegChecksum(t *testing.T) {
	t.Parallel()

	filename := "ffmpeg.zip"
	checksum := fmt.Sprintf("%x", sha256.Sum256([]byte("archive")))
	contents := "invalid line\n" + checksum + " *" + filename + "\n"

	got, err := parseFFmpegChecksum(contents, filename)
	if err != nil {
		t.Fatalf("parseFFmpegChecksum() error = %v", err)
	}
	if got != checksum {
		t.Fatalf("parseFFmpegChecksum() = %q, want %q", got, checksum)
	}
}

func TestExtractFFmpegZip(t *testing.T) {
	t.Parallel()

	archivePath := filepath.Join(t.TempDir(), "ffmpeg.zip")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	entry, err := writer.Create("ffmpeg-build/bin/ffmpeg.exe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(entry, "windows ffmpeg"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	var extracted bytes.Buffer
	if err := extractFFmpegZip(archivePath, "ffmpeg.exe", &extracted); err != nil {
		t.Fatalf("extractFFmpegZip() error = %v", err)
	}
	if extracted.String() != "windows ffmpeg" {
		t.Fatalf("extracted content = %q", extracted.String())
	}
}

func TestExtractFFmpegTarXZ(t *testing.T) {
	t.Parallel()

	archivePath := filepath.Join(t.TempDir(), "ffmpeg.tar.xz")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	xzWriter, err := xz.NewWriter(file)
	if err != nil {
		t.Fatal(err)
	}
	tarWriter := tar.NewWriter(xzWriter)
	content := []byte("linux ffmpeg")
	if err := tarWriter.WriteHeader(&tar.Header{
		Name: "ffmpeg-build/bin/ffmpeg",
		Mode: 0755,
		Size: int64(len(content)),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := tarWriter.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := xzWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	var extracted bytes.Buffer
	if err := extractFFmpegTarXZ(archivePath, "ffmpeg", &extracted); err != nil {
		t.Fatalf("extractFFmpegTarXZ() error = %v", err)
	}
	if extracted.String() != string(content) {
		t.Fatalf("extracted content = %q", extracted.String())
	}
}
