package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const xVideoLabURL = "https://x.com/g1/status/2114640245711061410"

func TestXVideoDownloadLab(t *testing.T) {
	if os.Getenv("X_VIDEO_LAB") == "" {
		t.Skip("set X_VIDEO_LAB=1 to run this lab integration test")
	}

	ctx := context.Background()
	info, err := fetchGenericInfo(ctx, xVideoLabURL)
	if err != nil {
		t.Fatalf("fetchGenericInfo error: %v", err)
	}
	format, ok := preferredVideoFormat(info.VideoFormats, "mp4", "1080p")
	if !ok {
		t.Fatal("no MP4 video format was found")
	}
	if format.Width != 1080 || format.Height != 1920 {
		t.Fatalf("selected format = %+v, want 1080x1920", format)
	}

	output, err := runYTDLPDownload(ctx, ytDLPDownloadRequest{
		URL:               xVideoLabURL,
		FormatSelector:    ytDLPGenericVideoSelector(format),
		OutputTemplate:    ytDLPOutputTemplate(t.TempDir(), "x-video"),
		MergeOutputFormat: "mp4",
		ExtraArgs:         genericDownloadArgs(),
	}, nil)
	if err != nil {
		t.Fatalf("runYTDLPDownload error: %v", err)
	}
	if fileInfo, err := os.Stat(output); err != nil || fileInfo.Size() == 0 {
		t.Fatalf("downloaded file is missing or empty: info=%v err=%v", fileInfo, err)
	}

	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		t.Skip("ffprobe is not available")
	}
	probe, err := exec.Command(ffprobe, "-v", "error", "-show_entries", "stream=codec_type,width,height", "-of", "csv=p=0", filepath.Clean(output)).Output()
	if err != nil {
		t.Fatalf("ffprobe error: %v", err)
	}
	result := string(probe)
	if !strings.Contains(result, "video,1080,1920") || !strings.Contains(result, "audio") {
		t.Fatalf("unexpected streams: %q", result)
	}
}
