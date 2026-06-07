package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"

	"github.com/kkdai/youtube/v2"
)

func TestGetClientReturnsIsolatedClients(t *testing.T) {
	if getClient() == getClient() {
		t.Fatal("getClient() reused a client with mutable YouTube state")
	}
}

type temporaryNetworkError struct{}

func (temporaryNetworkError) Error() string   { return "temporary network error" }
func (temporaryNetworkError) Timeout() bool   { return true }
func (temporaryNetworkError) Temporary() bool { return true }

func TestTransientYouTubeErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "rate limit handled by transport", err: youtube.ErrUnexpectedStatusCode(http.StatusTooManyRequests), want: false},
		{name: "server error handled by transport", err: youtube.ErrUnexpectedStatusCode(http.StatusBadGateway), want: false},
		{name: "bad request", err: youtube.ErrUnexpectedStatusCode(http.StatusBadRequest), want: false},
		{name: "network", err: net.Error(temporaryNetworkError{}), want: true},
		{name: "permanent playback", err: youtube.ErrNotPlayableInEmbed, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isTransientYouTubeError(test.err); got != test.want {
				t.Fatalf("isTransientYouTubeError(%v) = %v, want %v", test.err, got, test.want)
			}
		})
	}
}

type statusSequenceTransport struct {
	statuses []int
	calls    int
}

func (t *statusSequenceTransport) RoundTrip(*http.Request) (*http.Response, error) {
	status := t.statuses[t.calls]
	t.calls++
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Retry-After": []string{"0"}},
		Body:       io.NopCloser(bytes.NewReader(nil)),
	}, nil
}

func TestRetryTransportRetriesRateLimit(t *testing.T) {
	base := &statusSequenceTransport{statuses: []int{
		http.StatusTooManyRequests,
		http.StatusBadGateway,
		http.StatusOK,
	}}
	transport := &retryTransport{base: base, maxAttempts: 3}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatal(err)
	}

	response, err := transport.RoundTrip(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK || base.calls != 3 {
		t.Fatalf("status=%d calls=%d, want status=200 calls=3", response.StatusCode, base.calls)
	}
}

func TestStreamingClientHasNoOverallTimeout(t *testing.T) {
	if timeout := newStreamingHTTPClient().Timeout; timeout != 0 {
		t.Fatalf("streaming client timeout = %s, want no overall timeout", timeout)
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := map[string]string{
		`invalid:/title*?`: "invalid__title__",
		" ... ":            "audio",
		"valid title":      "valid title",
	}
	for input, want := range tests {
		if got := SanitizeFilename(input); got != want {
			t.Fatalf("SanitizeFilename(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestFormatYouTubeUnavailableError(t *testing.T) {
	err := &youtube.ErrPlayabiltyStatus{
		Status: "ERROR",
		Reason: "This video is unavailable",
	}

	message := FormatYouTubeError(err, "pt-BR")
	if message != "O YouTube informou que este vídeo está indisponível. Ele pode ter sido removido, tornado privado ou bloqueado para esta região." {
		t.Fatalf("unexpected message: %q", message)
	}
}

func TestFormatYouTubeOtherPlaybackErrorKeepsDetails(t *testing.T) {
	err := errors.New("playback failed")
	message := FormatYouTubeError(err, "pt-BR")
	if message != "Erro ao consultar o YouTube: playback failed" {
		t.Fatalf("unexpected message: %q", message)
	}
}

func TestFormatAgeRestrictionError(t *testing.T) {
	err := fmt.Errorf("can't bypass age restriction: %w", youtube.ErrNotPlayableInEmbed)

	portuguese := FormatYouTubeError(err, "pt-BR")
	if portuguese != "Este vídeo possui uma restrição de idade ou reprodução que impede o download sem login." {
		t.Fatalf("unexpected Portuguese message: %q", portuguese)
	}

	english := FormatYouTubeError(err, "en-US")
	if english != "This video has an age or playback restriction that prevents downloading without signing in." {
		t.Fatalf("unexpected English message: %q", english)
	}
}

func TestTranslateStoredAgeRestrictionError(t *testing.T) {
	oldMessage := "Erro ao consultar o YouTube: can't bypass age restriction: embedding of this video has been disabled"

	message := TranslateStoredYouTubeError(oldMessage, "pt-BR")
	if message != "Este vídeo possui uma restrição de idade ou reprodução que impede o download sem login." {
		t.Fatalf("unexpected translated message: %q", message)
	}
}

func TestFormatOperationErrorUsesLanguage(t *testing.T) {
	err := errors.New("disk full")
	if got := FormatOperationError("conversion", err, "pt-BR"); got != "Erro na conversão: disk full" {
		t.Fatalf("unexpected Portuguese error: %q", got)
	}
	if got := FormatOperationError("download", err, "en-US"); got != "Download error: disk full" {
		t.Fatalf("unexpected English error: %q", got)
	}
}

func TestFFmpegMP3ArgsDeclaresFormatForPartFile(t *testing.T) {
	args := ffmpegMP3Args("input.webm", "song.mp3.part", "192k")
	want := []string{"-y", "-i", "input.webm", "-b:a", "192k", "-f", "mp3", "song.mp3.part"}

	if fmt.Sprint(args) != fmt.Sprint(want) {
		t.Fatalf("ffmpegMP3Args() = %v, want %v", args, want)
	}
}
