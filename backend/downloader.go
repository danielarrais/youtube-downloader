package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kkdai/youtube/v2"
)

var invalidFilenameCharacters = regexp.MustCompile(`[\\/*?:"<>|]`)

type MusicMetadata struct {
	Song     string
	Artist   string
	Album    string
	CoverURL string
}

var (
	ytInitialDataPattern   = regexp.MustCompile(`var ytInitialData\s*=\s*(\{.+?\});`)
	ytInitialPlayerPattern = regexp.MustCompile(`var ytInitialPlayerResponse\s*=\s*(\{.+?\});`)
)

var watchPageClient = &http.Client{
	Timeout: 30 * time.Second,
}

func fetchWatchPage(ctx context.Context, videoID string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			delay := time.Duration(1<<uint(attempt)) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.youtube.com/watch?v="+videoID, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
		req.Header.Set("Origin", "https://www.youtube.com")
		req.Header.Set("Sec-Fetch-Mode", "navigate")
		req.AddCookie(&http.Cookie{
			Name:  "CONSENT",
			Value: "YES+cb.20210328-17-p0.en+FX+111",
		})

		resp, err := watchPageClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			return body, readErr
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("watch page returned HTTP %d", resp.StatusCode)
			continue
		}
		return nil, fmt.Errorf("watch page returned HTTP %d", resp.StatusCode)
	}
	return nil, lastErr
}

func ExtractMusicMetadata(ctx context.Context, videoID string) (*MusicMetadata, error) {
	html, err := fetchWatchPage(ctx, videoID)
	if err != nil {
		return nil, err
	}

	return parseMusicMetadataFromHTML(html)
}

func parseMusicMetadataFromHTML(html []byte) (*MusicMetadata, error) {
	match := ytInitialDataPattern.FindSubmatch(html)
	if len(match) < 2 {
		return nil, nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal(match[1], &data); err != nil {
		return nil, err
	}

	panels, ok := data["engagementPanels"].([]interface{})
	if !ok {
		return nil, nil
	}

	for _, p := range panels {
		panel, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		section, ok := panel["engagementPanelSectionListRenderer"].(map[string]interface{})
		if !ok {
			continue
		}
		content, ok := section["content"].(map[string]interface{})
		if !ok {
			continue
		}
		desc, ok := content["structuredDescriptionContentRenderer"].(map[string]interface{})
		if !ok {
			continue
		}
		items, ok := desc["items"].([]interface{})
		if !ok {
			continue
		}
		for _, item := range items {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			cards, ok := itemMap["horizontalCardListRenderer"].(map[string]interface{})
			if !ok {
				continue
			}
			if meta := parseMusicCards(cards); meta != nil {
				return meta, nil
			}
		}
	}

	return nil, nil
}

func parseMusicCards(cards map[string]interface{}) *MusicMetadata {
	cardList, _ := cards["cards"].([]interface{})
	if len(cardList) == 0 {
		return nil
	}

	card, ok := cardList[0].(map[string]interface{})
	if !ok {
		return nil
	}
	vm, ok := card["videoAttributeViewModel"].(map[string]interface{})
	if !ok {
		return nil
	}

	meta := &MusicMetadata{}

	if v, ok := vm["title"].(string); ok {
		meta.Song = v
	}
	if v, ok := vm["subtitle"].(string); ok {
		meta.Artist = v
	}
	if sec, ok := vm["secondarySubtitle"].(map[string]interface{}); ok {
		if v, ok := sec["content"].(string); ok {
			meta.Album = v
		}
	}

	if img, ok := vm["image"].(map[string]interface{}); ok {
		sources, _ := img["sources"].([]interface{})
		if len(sources) > 0 {
			if src, ok := sources[0].(map[string]interface{}); ok {
				if url, ok := src["url"].(string); ok {
					meta.CoverURL = url
				}
			}
		}
	}

	if meta.Song == "" && meta.Artist == "" {
		return nil
	}
	return meta
}

func DownloadCoverArt(ctx context.Context, url, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := watchPageClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("cover art download returned HTTP %d", resp.StatusCode)
	}

	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

type androidVRTransport struct {
	base           http.RoundTripper
	mu             sync.RWMutex
	playerResponse []byte
	watchPageHTML  []byte
}

type retryTransport struct {
	base        http.RoundTripper
	maxAttempts int
}

type YouTubeSession struct {
	client    *youtube.Client
	transport *androidVRTransport
}

func (t *androidVRTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method == http.MethodGet && req.URL.Host == "www.youtube.com" && req.URL.Path == "/watch" {
		t.mu.RLock()
		playerResponse := append([]byte(nil), t.playerResponse...)
		t.mu.RUnlock()

		if len(playerResponse) > 0 {
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
			resp, err := t.base.RoundTrip(req)
			if err != nil {
				return nil, err
			}
			if resp.StatusCode == http.StatusOK {
				body, readErr := io.ReadAll(resp.Body)
				resp.Body.Close()
				if readErr == nil {
					t.mu.Lock()
					t.watchPageHTML = append(t.watchPageHTML[:0], body...)
					t.mu.Unlock()
					resp.Body = io.NopCloser(bytes.NewReader(body))
					return resp, nil
				}
				return nil, readErr
			}
			return resp, nil
		}
	}

	if req.Method == http.MethodPost && req.URL.Path == "/youtubei/v1/player" {
		req.Header.Set("X-Youtube-Client-Name", "28")
	}

	response, err := t.base.RoundTrip(req)
	if err != nil || response.StatusCode != http.StatusOK ||
		req.Method != http.MethodPost || req.URL.Path != "/youtubei/v1/player" {
		return response, err
	}

	body, readErr := io.ReadAll(response.Body)
	response.Body.Close()
	if readErr != nil {
		return nil, readErr
	}
	response.Body = io.NopCloser(bytes.NewReader(body))

	t.mu.Lock()
	t.playerResponse = append(t.playerResponse[:0], body...)
	t.mu.Unlock()

	return response, nil
}

func (t *androidVRTransport) WatchPageHTML() []byte {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return append([]byte(nil), t.watchPageHTML...)
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for attempt := 0; attempt < t.maxAttempts; attempt++ {
		currentRequest := req.Clone(req.Context())
		if attempt > 0 && req.Body != nil {
			if req.GetBody == nil {
				return t.base.RoundTrip(req)
			}
			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			currentRequest.Body = body
		}

		response, err := t.base.RoundTrip(currentRequest)
		if err != nil {
			return nil, err
		}
		if response.StatusCode != http.StatusTooManyRequests &&
			response.StatusCode < http.StatusInternalServerError {
			return response, nil
		}
		if attempt == t.maxAttempts-1 {
			return response, nil
		}

		delay := retryDelay(response.Header.Get("Retry-After"), attempt)
		response.Body.Close()
		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(delay):
		}
	}
	return nil, errors.New("HTTP retry loop ended unexpectedly")
}

func retryDelay(value string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	if date, err := http.ParseTime(value); err == nil {
		if delay := time.Until(date); delay > 0 {
			return delay
		}
	}
	return time.Duration(attempt+1) * time.Second
}

func init() {
	youtube.AndroidVRClient.Version = "1.60.19"
	youtube.AndroidVRClient.Key = "AIzaSyAO_FJ2SlqU8Q4STEHLGCilw_Y9_11qcW8"
	youtube.AndroidVRClient.UserAgent = "com.google.android.apps.youtube.vr.oculus/1.60.19 (Linux; U; Android 12L; eureka-user Build/SQ3A.220605.009.A1) gzip"
	youtube.AndroidVRClient.AndroidVersion = 32
	youtube.AndroidVRClient.DeviceModel = "Quest 3"
	youtube.DefaultClient = youtube.AndroidVRClient
}

func SanitizeFilename(filename string) string {
	filename = invalidFilenameCharacters.ReplaceAllString(filename, "_")
	filename = strings.Trim(filename, " .")
	if filename == "" {
		return "audio"
	}
	return filename
}

func newMetadataHTTPClient() *http.Client {
	baseTransport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ForceAttemptHTTP2:     true,
		DialContext:           (&net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 20 * time.Second,
		IdleConnTimeout:       60 * time.Second,
	}
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &androidVRTransport{base: &retryTransport{
			base:        baseTransport,
			maxAttempts: 3,
		}},
	}
}

func newStreamingHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			ForceAttemptHTTP2:     true,
			DialContext:           (&net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
			IdleConnTimeout:       90 * time.Second,
		},
	}
}

func NewYouTubeSession() *YouTubeSession {
	httpClient := newMetadataHTTPClient()
	transport := httpClient.Transport.(*androidVRTransport)
	return &YouTubeSession{
		client:    &youtube.Client{HTTPClient: httpClient},
		transport: transport,
	}
}

func getClient() *youtube.Client {
	return NewYouTubeSession().client
}

func GetVideoInfo(url string) (*youtube.Video, error) {
	return GetVideoInfoContext(context.Background(), url)
}

func GetVideoInfoContext(ctx context.Context, url string) (*youtube.Video, error) {
	return NewYouTubeSession().GetVideo(ctx, url)
}

func (s *YouTubeSession) GetVideo(ctx context.Context, url string) (*youtube.Video, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		video, err := s.client.GetVideoContext(ctx, url)
		if err == nil {
			return video, nil
		}
		lastErr = err
		if !isTransientYouTubeError(err) || attempt == 2 {
			break
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(attempt+1) * time.Second):
		}
	}
	return nil, lastErr
}

func (s *YouTubeSession) ExtractMusicMetadata() *MusicMetadata {
	html := s.transport.WatchPageHTML()
	if len(html) == 0 {
		return nil
	}
	meta, _ := parseMusicMetadataFromHTML(html)
	return meta
}

func isTransientYouTubeError(err error) bool {
	var networkError net.Error
	return errors.As(err, &networkError)
}

func FormatYouTubeError(err error, language string) string {
	var status youtube.ErrUnexpectedStatusCode
	if errors.As(err, &status) && status == http.StatusTooManyRequests {
		if language == "en-US" {
			return "YouTube temporarily limited this IP address (HTTP 429). Wait before trying again."
		}
		return "YouTube limitou temporariamente este IP (HTTP 429). Aguarde antes de tentar novamente."
	}

	if errors.Is(err, youtube.ErrNotPlayableInEmbed) {
		if language == "en-US" {
			return "This video has an age or playback restriction that prevents downloading without signing in."
		}
		return "Este vídeo possui uma restrição de idade ou reprodução que impede o download sem login."
	}

	if errors.Is(err, youtube.ErrVideoPrivate) {
		if language == "en-US" {
			return "This video is private and cannot be downloaded."
		}
		return "Este vídeo é privado e não pode ser baixado."
	}

	var playbackError *youtube.ErrPlayabiltyStatus
	if errors.As(err, &playbackError) &&
		strings.EqualFold(playbackError.Reason, "This video is unavailable") {
		if language == "en-US" {
			return "YouTube reported that this video is unavailable. It may have been removed, made private, or blocked in this region."
		}
		return "O YouTube informou que este vídeo está indisponível. Ele pode ter sido removido, tornado privado ou bloqueado para esta região."
	}

	if language == "en-US" {
		return "Error while querying YouTube: " + err.Error()
	}
	return "Erro ao consultar o YouTube: " + err.Error()
}

func TranslateStoredYouTubeError(message, language string) string {
	lowerMessage := strings.ToLower(message)
	switch {
	case strings.Contains(lowerMessage, "can't bypass age restriction"),
		strings.Contains(lowerMessage, "restrição de idade"),
		strings.Contains(lowerMessage, "age or playback restriction"):
		if language == "en-US" {
			return "This video has an age or playback restriction that prevents downloading without signing in."
		}
		return "Este vídeo possui uma restrição de idade ou reprodução que impede o download sem login."
	case strings.Contains(lowerMessage, "this video is unavailable"),
		strings.Contains(lowerMessage, "vídeo está indisponível"),
		strings.Contains(lowerMessage, "video is unavailable"):
		if language == "en-US" {
			return "YouTube reported that this video is unavailable. It may have been removed, made private, or blocked in this region."
		}
		return "O YouTube informou que este vídeo está indisponível. Ele pode ter sido removido, tornado privado ou bloqueado para esta região."
	default:
		return message
	}
}

func FormatOperationError(operation string, err error, language string) string {
	if language == "en-US" {
		switch operation {
		case "conversion":
			return "Conversion error: " + err.Error()
		case "finalize":
			return "Error while finalizing the file: " + err.Error()
		default:
			return "Download error: " + err.Error()
		}
	}
	switch operation {
	case "conversion":
		return "Erro na conversão: " + err.Error()
	case "finalize":
		return "Erro ao finalizar o arquivo: " + err.Error()
	default:
		return "Erro no download: " + err.Error()
	}
}

func (s *YouTubeSession) DownloadAudio(ctx context.Context, video *youtube.Video, destPath string, onProgress func(percent float64, downloaded int64, total int64)) (string, error) {
	formats := video.Formats.Type("audio")
	if len(formats) == 0 {
		return "", fmt.Errorf("no audio formats found")
	}

	sort.SliceStable(formats, func(i, j int) bool {
		return formats[i].Bitrate > formats[j].Bitrate
	})

	format := &formats[0]
	s.client.HTTPClient = newStreamingHTTPClient()
	stream, totalSize, err := s.client.GetStreamContext(ctx, video, format)
	if err != nil {
		return "", err
	}
	defer stream.Close()

	file, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	closeWithError := func(operationErr error) error {
		if closeErr := file.Close(); operationErr == nil {
			return closeErr
		}
		return operationErr
	}

	var downloaded int64
	buffer := make([]byte, 64*1024)
	for {
		if err := ctx.Err(); err != nil {
			return "", closeWithError(err)
		}
		n, err := stream.Read(buffer)
		if n > 0 {
			_, writeErr := file.Write(buffer[:n])
			if writeErr != nil {
				return "", closeWithError(writeErr)
			}
			downloaded += int64(n)
			if onProgress != nil && totalSize > 0 {
				percent := float64(downloaded) / float64(totalSize) * 100
				onProgress(percent, downloaded, totalSize)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", closeWithError(err)
		}
	}

	if err := file.Sync(); err != nil {
		return "", closeWithError(err)
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return destPath, nil
}

func ConvertToMp3(ctx context.Context, inputPath string, outputPath string, quality string, metadata *MusicMetadata, coverPath string) error {
	bitrate := "192k"
	if quality != "" {
		bitrate = strings.TrimSuffix(quality, "k") + "k"
	}
	ffmpegPath, err := CheckAndDownloadFFmpeg()
	if err != nil {
		return err
	}

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, ffmpegPath, ffmpegMP3Args(inputPath, outputPath, bitrate, metadata, coverPath)...)
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("ffmpeg error: %v, detail: %s", err, stderr.String())
	}
	return nil
}

func ffmpegMP3Args(inputPath, outputPath, bitrate string, metadata *MusicMetadata, coverPath string) []string {
	args := []string{"-y", "-i", inputPath}

	if coverPath != "" {
		args = append(args, "-i", coverPath)
	}
	args = append(args,
		"-c:a", "libmp3lame",
		"-b:a", bitrate,
		"-ar", "44100",
		"-ac", "2",
		"-joint_stereo", "1",
	)

	if metadata != nil {
		if metadata.Song != "" {
			args = append(args, "-metadata", "title="+metadata.Song)
		}
		if metadata.Artist != "" {
			args = append(args, "-metadata", "artist="+metadata.Artist)
		}
		if metadata.Album != "" {
			args = append(args, "-metadata", "album="+metadata.Album)
		}
	}

	args = append(args, "-map_metadata", "0")

	if coverPath != "" {
		args = append(args,
			"-map", "0:a:0",
			"-map", "1:v:0",
			"-c:v", "copy",
			"-disposition:v", "attached_pic",
		)
	} else {
		args = append(args, "-vn")
	}

	args = append(args, "-id3v2_version", "3", "-f", "mp3", outputPath)
	return args
}
