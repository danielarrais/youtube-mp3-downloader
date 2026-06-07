package main

import (
	"os"
	"testing"
)

func TestFetchPlaylistInfoLive(t *testing.T) {
	url := os.Getenv("YOUTUBE_PLAYLIST_TEST_URL")
	if url == "" {
		t.Skip("YOUTUBE_PLAYLIST_TEST_URL is not set")
	}

	playlist, err := FetchPlaylistInfo(url)
	if err != nil {
		t.Fatalf("FetchPlaylistInfo returned error: %v", err)
	}
	if len(playlist.Videos) == 0 {
		t.Fatal("playlist returned no videos")
	}

	available := 0
	unavailable := 0
	for _, video := range playlist.Videos {
		if video.Available {
			available++
		} else {
			unavailable++
		}
	}
	t.Logf("playlist=%q videos=%d available=%d unavailable=%d", playlist.Title, len(playlist.Videos), available, unavailable)
}
