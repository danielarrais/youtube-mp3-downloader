package main

import "testing"

func TestParsePlaylistPageKeepsUnavailableVideos(t *testing.T) {
	data := []byte(`{
		"header": {
			"playlistHeaderRenderer": {
				"title": {"runs": [{"text": "Minha playlist"}]},
				"ownerText": {"runs": [{"text": "Autor"}]}
			}
		},
		"contents": [
			{"playlistVideoRenderer": {
				"videoId": "available123",
				"title": {"runs": [{"text": "Vídeo disponível"}]},
				"shortBylineText": {"runs": [{"text": "Canal"}]},
				"lengthSeconds": "125",
				"thumbnail": {"thumbnails": [
					{"url": "small.jpg"},
					{"url": "large.jpg"}
				]}
			}},
			{"playlistVideoRenderer": {
				"videoId": "private12345",
				"title": {"runs": [{"text": "[Vídeo privado]"}]},
				"thumbnail": {"thumbnails": [{"url": "none.jpg"}]}
			}},
			{"continuations": [{
				"nextContinuationData": {"continuation": "next-page"}
			}]}
		]
	}`)

	playlist, token, err := parsePlaylistPage(data)
	if err != nil {
		t.Fatalf("parsePlaylistPage returned error: %v", err)
	}
	if playlist.Title != "Minha playlist" || playlist.Author != "Autor" {
		t.Fatalf("unexpected metadata: %#v", playlist)
	}
	if token != "next-page" {
		t.Fatalf("unexpected continuation token: %q", token)
	}
	if len(playlist.Videos) != 2 {
		t.Fatalf("expected 2 videos, got %d", len(playlist.Videos))
	}

	available := playlist.Videos[0]
	if !available.Available || available.DurationSeconds != 125 {
		t.Fatalf("available video parsed incorrectly: %#v", available)
	}
	if available.ThumbnailURL != "large.jpg" {
		t.Fatalf("expected largest thumbnail, got %q", available.ThumbnailURL)
	}

	private := playlist.Videos[1]
	if private.Available {
		t.Fatalf("private video should be unavailable: %#v", private)
	}
	if private.UnavailableReason != "Vídeo privado" {
		t.Fatalf("unexpected unavailable reason: %q", private.UnavailableReason)
	}
}

func TestParsePlaylistPageReadsContinuationItem(t *testing.T) {
	data := []byte(`{
		"continuationContents": {
			"playlistVideoListContinuation": {
				"contents": [{
					"continuationItemRenderer": {
						"continuationEndpoint": {
							"continuationCommand": {"token": "third-page"}
						}
					}
				}]
			}
		}
	}`)

	_, token, err := parsePlaylistPage(data)
	if err != nil {
		t.Fatalf("parsePlaylistPage returned error: %v", err)
	}
	if token != "third-page" {
		t.Fatalf("unexpected continuation token: %q", token)
	}
}

func TestParsePlaylistPageReturnsYouTubeError(t *testing.T) {
	data := []byte(`{
		"alerts": [{
			"alertRenderer": {
				"type": "ERROR",
				"text": {"runs": [{"text": "Playlist privada"}]}
			}
		}]
	}`)

	_, _, err := parsePlaylistPage(data)
	if err == nil || err.Error() != "YouTube: Playlist privada" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExtractPlaylistID(t *testing.T) {
	id, err := extractPlaylistID("https://www.youtube.com/watch?v=abc&list=PLrzuA2--JVdSPr5FdYtBixGMw9uCWfsSi")
	if err != nil {
		t.Fatalf("extractPlaylistID returned error: %v", err)
	}
	if id != "PLrzuA2--JVdSPr5FdYtBixGMw9uCWfsSi" {
		t.Fatalf("unexpected playlist ID: %q", id)
	}

	if _, err := extractPlaylistID("https://www.youtube.com/watch?v=abc"); err == nil {
		t.Fatal("expected an error for a non-playlist URL")
	}
}
