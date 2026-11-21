package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const xFallbackAPIBaseURL = "https://api.fxtwitter.com"

var (
	xFallbackBaseURL = xFallbackAPIBaseURL
	xFallbackClient  = &http.Client{Timeout: 30 * time.Second}
)

type xFallbackResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Tweet   struct {
		ID    string `json:"id"`
		Text  string `json:"text"`
		Media struct {
			Videos []xFallbackVideo `json:"videos"`
		} `json:"media"`
	} `json:"tweet"`
}

type xFallbackVideo struct {
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnail_url"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
}

// fetchXFallbackInfo uses FxTwitter only when X does not expose a post's media
// to yt-dlp, such as age-restricted posts. It returns the direct MP4 URL, which
// is then still downloaded by yt-dlp.
func fetchXFallbackInfo(ctx context.Context, rawURL string) (ytDLPInfo, error) {
	statusID, err := xStatusID(rawURL)
	if err != nil {
		return ytDLPInfo{}, err
	}

	requestURL := strings.TrimRight(xFallbackBaseURL, "/") + "/status/" + url.PathEscape(statusID)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return ytDLPInfo{}, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "youtube-mp3-downloader")

	response, err := xFallbackClient.Do(request)
	if err != nil {
		return ytDLPInfo{}, fmt.Errorf("não foi possível consultar a mídia do X: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return ytDLPInfo{}, fmt.Errorf("a consulta da mídia do X retornou HTTP %d", response.StatusCode)
	}

	var payload xFallbackResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return ytDLPInfo{}, fmt.Errorf("resposta inválida da consulta de mídia do X: %w", err)
	}
	if payload.Code != 0 && payload.Code != http.StatusOK {
		message := strings.TrimSpace(payload.Message)
		if message == "" {
			message = "a consulta não retornou mídia"
		}
		return ytDLPInfo{}, errors.New(message)
	}
	if len(payload.Tweet.Media.Videos) == 0 {
		return ytDLPInfo{}, errors.New("nenhum vídeo foi encontrado nesta publicação do X")
	}

	video := payload.Tweet.Media.Videos[0]
	if video.URL == "" {
		return ytDLPInfo{}, errors.New("a publicação do X não forneceu uma URL de vídeo")
	}
	if !isXVideoURL(video.URL) {
		return ytDLPInfo{}, errors.New("a publicação do X forneceu uma URL de vídeo inválida")
	}
	info := ytDLPInfo{
		ID:           payload.Tweet.ID,
		Title:        payload.Tweet.Text,
		ThumbnailURL: video.ThumbnailURL,
		SourceURL:    video.URL,
	}
	if info.ID == "" {
		info.ID = statusID
	}
	quality := video.Height
	if video.Width > 0 && (quality == 0 || video.Width < quality) {
		quality = video.Width
	}
	format := VideoFormat{
		VideoItag:  1,
		FormatID:   "direct-mp4",
		Container:  "mp4",
		Extension:  "mp4",
		Width:      video.Width,
		Height:     video.Height,
		VideoCodec: "h264",
	}
	if quality > 0 {
		format.Resolution = strconv.Itoa(quality) + "p"
	}
	format.Label = videoFormatLabel(format)
	info.VideoFormats = []VideoFormat{format}
	return info, nil
}

func xStatusID(rawURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", errors.New("link do X inválido")
	}
	parts := strings.Split(strings.Trim(parsed.EscapedPath(), "/"), "/")
	for index := 0; index+1 < len(parts); index++ {
		if parts[index] != "status" {
			continue
		}
		statusID, err := url.PathUnescape(parts[index+1])
		if err == nil && isXStatusID(statusID) {
			return statusID, nil
		}
	}
	return "", errors.New("ID da publicação do X não encontrado")
}

func isXStatusID(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func isXVideoURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "video.twimg.com" || strings.HasSuffix(host, ".video.twimg.com")
}

func shouldUseXFallback(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "no video could be found") ||
		strings.Contains(message, "age-restricted") ||
		strings.Contains(message, "sensitive content")
}
