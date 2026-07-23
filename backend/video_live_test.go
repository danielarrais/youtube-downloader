package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLiveVideoDownloadAndMux(t *testing.T) {
	url := os.Getenv("YOUTUBE_VIDEO_TEST_URL")
	if url == "" {
		t.Skip("YOUTUBE_VIDEO_TEST_URL is not set")
	}

	session := NewYouTubeSession()
	video, err := session.GetVideo(context.Background(), url)
	if err != nil {
		t.Fatalf("GetVideo error: %v", err)
	}
	format, ok := liveMuxFormat(AvailableVideoFormats(video))
	if !ok {
		t.Fatal("no adaptive video format with a compatible audio stream")
	}
	videoStream, ok := videoFormatByItag(video, format.VideoItag)
	if !ok {
		t.Fatalf("video itag %d was not found", format.VideoItag)
	}
	audioStream, ok := videoFormatByItag(video, format.AudioItag)
	if !ok {
		t.Fatalf("audio itag %d was not found", format.AudioItag)
	}

	dir := t.TempDir()
	videoPath := filepath.Join(dir, "video.tmp")
	audioPath := filepath.Join(dir, "audio.tmp")
	outputPath := filepath.Join(dir, "output."+format.Extension)
	if _, err := session.DownloadFormat(context.Background(), video, videoStream, videoPath, nil); err != nil {
		t.Fatalf("video stream download failed: %v", err)
	}
	if _, err := session.DownloadFormat(context.Background(), video, audioStream, audioPath, nil); err != nil {
		t.Fatalf("audio stream download failed: %v", err)
	}
	if err := ConvertToVideo(context.Background(), videoPath, audioPath, outputPath, format.Container); err != nil {
		t.Fatalf("mux failed: %v", err)
	}
	if info, err := os.Stat(outputPath); err != nil || info.Size() == 0 {
		t.Fatalf("output was not created: info=%v err=%v", info, err)
	}
	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		t.Skip("ffprobe is not available")
	}
	output, err := exec.Command(ffprobe, "-v", "error", "-show_entries", "stream=codec_type", "-of", "csv=p=0", outputPath).Output()
	if err != nil {
		t.Fatalf("ffprobe failed: %v", err)
	}
	streams := string(output)
	if !strings.Contains(streams, "video") || !strings.Contains(streams, "audio") {
		t.Fatalf("muxed output streams = %q, want video and audio", streams)
	}
}

func liveMuxFormat(formats []VideoFormat) (VideoFormat, bool) {
	for _, format := range formats {
		if format.AudioItag != 0 && videoHeight(format.Resolution) <= 720 {
			return format, true
		}
	}
	for _, format := range formats {
		if format.AudioItag != 0 {
			return format, true
		}
	}
	return VideoFormat{}, false
}
