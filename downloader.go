package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/kkdai/youtube/v2"
)

type androidVRTransport struct {
	base           http.RoundTripper
	mu             sync.RWMutex
	playerResponse []byte
}

func (t *androidVRTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method == http.MethodGet && req.URL.Host == "www.youtube.com" && req.URL.Path == "/watch" {
		t.mu.RLock()
		playerResponse := append([]byte(nil), t.playerResponse...)
		t.mu.RUnlock()
		if len(playerResponse) > 0 {
			var compact bytes.Buffer
			if err := json.Compact(&compact, playerResponse); err != nil {
				return nil, err
			}
			body := append([]byte("var ytInitialPlayerResponse = "), compact.Bytes()...)
			body = append(body, ';')
			return &http.Response{
				StatusCode:    http.StatusOK,
				Status:        "200 OK",
				Header:        http.Header{"Content-Type": []string{"text/html; charset=utf-8"}},
				Body:          io.NopCloser(bytes.NewReader(body)),
				ContentLength: int64(len(body)),
				Request:       req,
			}, nil
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

func init() {
	youtube.AndroidVRClient.Version = "1.60.19"
	youtube.AndroidVRClient.Key = "AIzaSyAO_FJ2SlqU8Q4STEHLGCilw_Y9_11qcW8"
	youtube.AndroidVRClient.UserAgent = "com.google.android.apps.youtube.vr.oculus/1.60.19 (Linux; U; Android 12L; eureka-user Build/SQ3A.220605.009.A1) gzip"
	youtube.AndroidVRClient.AndroidVersion = 32
	youtube.AndroidVRClient.DeviceModel = "Quest 3"
	youtube.DefaultClient = youtube.AndroidVRClient
}

func SanitizeFilename(filename string) string {
	re := regexp.MustCompile(`[\\/*?:"<>|]`)
	return re.ReplaceAllString(filename, "_")
}

func getClient() *youtube.Client {
	return &youtube.Client{
		HTTPClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: &androidVRTransport{base: http.DefaultTransport},
		},
	}
}

func GetVideoInfo(url string) (*youtube.Video, error) {
	return GetVideoInfoContext(context.Background(), url)
}

func GetVideoInfoContext(ctx context.Context, url string) (*youtube.Video, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		video, err := getClient().GetVideoContext(ctx, url)
		if err == nil {
			return video, nil
		}
		lastErr = err

		var playbackError *youtube.ErrPlayabiltyStatus
		if !errors.As(err, &playbackError) ||
			!strings.EqualFold(playbackError.Reason, "This video is unavailable") ||
			attempt == 1 {
			break
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return nil, lastErr
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

func DownloadAudio(ctx context.Context, video *youtube.Video, destPath string, onProgress func(percent float64, downloaded int64, total int64)) (string, error) {
	client := getClient()

	formats := video.Formats.Type("audio")
	if len(formats) == 0 {
		return "", fmt.Errorf("no audio formats found")
	}

	format := &formats[0]
	stream, totalSize, err := client.GetStreamContext(ctx, video, format)
	if err != nil {
		return "", err
	}
	defer stream.Close()

	file, err := os.Create(destPath)
	if err != nil {
		return "", err
	}

	var downloaded int64
	buffer := make([]byte, 64*1024)
	for {
		if err := ctx.Err(); err != nil {
			file.Close()
			return "", err
		}
		n, err := stream.Read(buffer)
		if n > 0 {
			_, writeErr := file.Write(buffer[:n])
			if writeErr != nil {
				file.Close()
				return "", writeErr
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
			file.Close()
			return "", err
		}
	}

	file.Sync()
	file.Close()
	return destPath, nil
}

func ConvertToMp3(ctx context.Context, inputPath string, outputPath string, quality string) error {
	bitrate := "192k"
	if quality != "" {
		bitrate = strings.TrimSuffix(quality, "k") + "k"
	}
	ffmpegPath, err := CheckAndDownloadFFmpeg()
	if err != nil {
		return err
	}

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, ffmpegPath, "-y", "-i", inputPath, "-b:a", bitrate, outputPath)
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("ffmpeg error: %v, detail: %s", err, stderr.String())
	}
	return nil
}
