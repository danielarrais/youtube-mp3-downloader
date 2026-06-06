package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/kkdai/youtube/v2"
)

const (
	youtubeBrowseURL = "https://www.youtube.com/youtubei/v1/browse?key=" + youtubeAPIKey
	youtubeAPIKey    = "AIzaSyAO_FJ2SlqU8Q4STEHLGCilw_Y9_11qcW8"
)

type playlistBrowseRequest struct {
	BrowseID     string                 `json:"browseId,omitempty"`
	Continuation string                 `json:"continuation,omitempty"`
	Context      playlistRequestContext `json:"context"`
	ContentCheck bool                   `json:"contentCheckOk"`
	RacyCheck    bool                   `json:"racyCheckOk"`
	Params       string                 `json:"params"`
}

type playlistRequestContext struct {
	Client playlistRequestClient `json:"client"`
}

type playlistRequestClient struct {
	ClientName        string `json:"clientName"`
	ClientVersion     string `json:"clientVersion"`
	AndroidSDKVersion int    `json:"androidSdkVersion"`
	DeviceModel       string `json:"deviceModel"`
	UserAgent         string `json:"userAgent"`
	HL                string `json:"hl"`
	GL                string `json:"gl"`
	TimeZone          string `json:"timeZone"`
}

func FetchPlaylistInfo(rawURL string) (PlaylistInfo, error) {
	return FetchPlaylistInfoContext(context.Background(), rawURL)
}

func FetchPlaylistInfoContext(ctx context.Context, rawURL string) (PlaylistInfo, error) {
	playlistID, err := extractPlaylistID(rawURL)
	if err != nil {
		return PlaylistInfo{}, err
	}

	info := PlaylistInfo{ID: playlistID, Videos: make([]PlaylistVideo, 0)}
	seenTokens := make(map[string]bool)
	continuation := ""

	for {
		body, err := fetchPlaylistPage(ctx, playlistID, continuation)
		if err != nil {
			return PlaylistInfo{}, err
		}

		page, nextToken, err := parsePlaylistPage(body)
		if err != nil {
			return PlaylistInfo{}, err
		}
		if info.Title == "" {
			info.Title = page.Title
			info.Author = page.Author
		}
		for _, video := range page.Videos {
			if video.ID == "" {
				continue
			}
			video.Index = len(info.Videos) + 1
			info.Videos = append(info.Videos, video)
		}

		if nextToken == "" {
			break
		}
		if seenTokens[nextToken] {
			return PlaylistInfo{}, fmt.Errorf("playlist pagination returned a repeated token")
		}
		seenTokens[nextToken] = true
		continuation = nextToken
	}

	if len(info.Videos) == 0 {
		return PlaylistInfo{}, fmt.Errorf("nenhum vídeo encontrado na playlist")
	}
	if info.Title == "" {
		info.Title = "Playlist"
	}
	return info, nil
}

func extractPlaylistID(rawURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", fmt.Errorf("URL de playlist inválida")
	}
	id := parsed.Query().Get("list")
	if id == "" && parsed.Host == "" && parsed.Path != "" {
		id = parsed.Path
	}
	if len(id) < 13 || len(id) > 42 {
		return "", fmt.Errorf("URL de playlist inválida")
	}
	for _, char := range id {
		if !(char == '-' || char == '_' ||
			char >= 'a' && char <= 'z' ||
			char >= 'A' && char <= 'Z' ||
			char >= '0' && char <= '9') {
			return "", fmt.Errorf("URL de playlist inválida")
		}
	}
	return id, nil
}

func fetchPlaylistPage(ctx context.Context, playlistID, continuation string) ([]byte, error) {
	requestBody := playlistBrowseRequest{
		Context: playlistRequestContext{Client: playlistRequestClient{
			ClientName:        "ANDROID_VR",
			ClientVersion:     "1.60.19",
			AndroidSDKVersion: 32,
			DeviceModel:       "Quest 3",
			UserAgent:         youtube.AndroidVRClient.UserAgent,
			HL:                "pt-BR",
			GL:                "BR",
			TimeZone:          "America/Bahia",
		}},
		ContentCheck: true,
		RacyCheck:    true,
		Params:       "CgIQBg==",
	}
	if continuation == "" {
		requestBody.BrowseID = "VL" + playlistID
	} else {
		requestBody.Continuation = continuation
	}

	data, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, youtubeBrowseURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", youtube.AndroidVRClient.UserAgent)
	request.Header.Set("X-Youtube-Client-Name", "28")
	request.Header.Set("X-Youtube-Client-Version", "1.60.19")

	response, err := getClient().HTTPClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("YouTube retornou HTTP %d ao consultar a playlist", response.StatusCode)
	}
	return io.ReadAll(response.Body)
}

func parsePlaylistPage(data []byte) (PlaylistInfo, string, error) {
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return PlaylistInfo{}, "", err
	}

	result := PlaylistInfo{Videos: make([]PlaylistVideo, 0)}
	continuation := ""
	var parseErr error

	walkJSON(root, func(key string, value map[string]any) {
		switch key {
		case "alertRenderer":
			if stringValue(value["type"]) == "ERROR" && parseErr == nil {
				parseErr = fmt.Errorf("YouTube: %s", textValue(value["text"]))
			}
		case "playlistHeaderRenderer":
			if result.Title == "" {
				result.Title = textValue(value["title"])
				result.Author = textValue(value["ownerText"])
			}
		case "playlistVideoRenderer":
			result.Videos = append(result.Videos, parsePlaylistVideo(value))
		case "continuationItemRenderer":
			if continuation == "" {
				continuation = nestedString(value, "continuationEndpoint", "continuationCommand", "token")
			}
		case "nextContinuationData":
			if continuation == "" {
				continuation = stringValue(value["continuation"])
			}
		}
	})

	if parseErr != nil {
		return PlaylistInfo{}, "", parseErr
	}
	return result, continuation, nil
}

func parsePlaylistVideo(renderer map[string]any) PlaylistVideo {
	id := stringValue(renderer["videoId"])
	title := textValue(renderer["title"])
	duration, _ := strconv.Atoi(stringValue(renderer["lengthSeconds"]))
	available := id != "" && duration > 0

	video := PlaylistVideo{
		ID:              id,
		URL:             "https://www.youtube.com/watch?v=" + id,
		Title:           title,
		Author:          textValue(renderer["shortBylineText"]),
		DurationSeconds: duration,
		ThumbnailURL:    largestThumbnail(renderer["thumbnail"]),
		Available:       available,
	}
	if !available {
		video.UnavailableReason = strings.Trim(title, "[]")
		if video.UnavailableReason == "" {
			video.UnavailableReason = "Vídeo indisponível"
		}
	}
	return video
}

func walkJSON(value any, visit func(string, map[string]any)) {
	switch node := value.(type) {
	case map[string]any:
		for key, child := range node {
			if object, ok := child.(map[string]any); ok {
				visit(key, object)
			}
			walkJSON(child, visit)
		}
	case []any:
		for _, child := range node {
			walkJSON(child, visit)
		}
	}
}

func textValue(value any) string {
	object, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	if simple := stringValue(object["simpleText"]); simple != "" {
		return simple
	}
	runs, _ := object["runs"].([]any)
	var text strings.Builder
	for _, run := range runs {
		if item, ok := run.(map[string]any); ok {
			text.WriteString(stringValue(item["text"]))
		}
	}
	return text.String()
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func nestedString(object map[string]any, path ...string) string {
	var current any = object
	for _, key := range path {
		node, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = node[key]
	}
	return stringValue(current)
}

func largestThumbnail(value any) string {
	object, _ := value.(map[string]any)
	thumbnails, _ := object["thumbnails"].([]any)
	for index := len(thumbnails) - 1; index >= 0; index-- {
		if thumbnail, ok := thumbnails[index].(map[string]any); ok {
			if imageURL := stringValue(thumbnail["url"]); imageURL != "" {
				return imageURL
			}
		}
	}
	return ""
}
