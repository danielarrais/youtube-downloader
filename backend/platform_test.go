package main

import (
	"strings"
	"testing"
)

func TestDetectPlatform(t *testing.T) {
	tests := map[string]Platform{
		"https://www.youtube.com/watch?v=abc":                 PlatformYouTube,
		"https://music.youtube.com/watch?v=abc":               PlatformYouTube,
		"https://youtu.be/abc":                                PlatformYouTube,
		"https://x.com/user/status/123":                       PlatformX,
		"https://twitter.com/user/status/123":                 PlatformX,
		"https://mobile.twitter.com/user/status/123":          PlatformX,
		"https://www.instagram.com/reel/abc/":                 PlatformInstagram,
		"https://www.tiktok.com/@user/video/123":              PlatformTikTok,
		"https://vm.tiktok.com/abc/":                          PlatformTikTok,
		"www.instagram.com/p/abc":                             PlatformInstagram,
		"https://notx.com/video":                              PlatformYouTube,
		"https://example.com/?u=https://www.tiktok.com/video": PlatformYouTube,
	}
	for url, want := range tests {
		if got := detectPlatform(url); got != want {
			t.Errorf("detectPlatform(%q) = %q, want %q", url, got, want)
		}
	}
}

func TestParseYTDLPInfoGenericFormats(t *testing.T) {
	data := []byte(`{
		"id": "123",
		"title": "Post",
		"thumbnail": "https://example.com/thumb.jpg",
		"formats": [
			{"format_id": "hls-audio", "ext": "mp4", "acodec": "mp4a.40.2", "vcodec": "none", "abr": 128.5},
			{"format_id": "http-832", "ext": "mp4", "height": 480, "tbr": 832},
			{"format_id": "http-2176", "ext": "mp4", "height": 720, "tbr": 2176, "fps": 29.97},
			{"format_id": "hls-2176", "ext": "mp4", "vcodec": "avc1", "height": 720, "tbr": 2000}
		]
	}`)
	info, err := parseYTDLPInfo(data)
	if err != nil {
		t.Fatalf("parseYTDLPInfo error: %v", err)
	}
	if info.ID != "123" || info.Title != "Post" {
		t.Fatalf("unexpected info: %+v", info)
	}
	if len(info.AudioFormats) != 1 || info.AudioFormats[0].Bitrate != 128 {
		t.Fatalf("unexpected audio formats: %+v", info.AudioFormats)
	}
	if len(info.VideoFormats) != 2 {
		t.Fatalf("unexpected video formats: %+v", info.VideoFormats)
	}
	if info.VideoFormats[0].Resolution != "720p" || info.VideoFormats[0].VideoItag != 1 || info.VideoFormats[0].Extension != "mp4" {
		t.Fatalf("unexpected first video format: %+v", info.VideoFormats[0])
	}
	if info.VideoFormats[1].Resolution != "480p" || info.VideoFormats[1].VideoItag != 2 {
		t.Fatalf("unexpected second video format: %+v", info.VideoFormats[1])
	}
}

func TestParseYTDLPInfoUsesFirstPlaylistEntry(t *testing.T) {
	data := []byte(`{"title": "Thread", "entries": [{"id": "1", "formats": [{"format_id": "a", "height": 360}]}]}`)
	info, err := parseYTDLPInfo(data)
	if err != nil {
		t.Fatalf("parseYTDLPInfo error: %v", err)
	}
	if info.ID != "1" || info.Title != "Thread" || len(info.VideoFormats) != 1 {
		t.Fatalf("unexpected info: %+v", info)
	}
}

func TestGenericTitle(t *testing.T) {
	if got := genericTitle("  line one\n\nline   two ", "https://x.com/a", "1"); got != "line one line two" {
		t.Fatalf("genericTitle = %q", got)
	}
	if got := genericTitle("", "https://www.tiktok.com/@a/video/9", "9"); got != "tiktok 9" {
		t.Fatalf("genericTitle = %q", got)
	}
}

func TestYTDLPGenericVideoSelectorRequiresVideo(t *testing.T) {
	selector := ytDLPGenericVideoSelector(VideoFormat{Extension: "mp4", Resolution: "720p"})
	if want := "best[vcodec!=none][acodec!=none][height<=720][ext=mp4]"; !strings.HasPrefix(selector, want) {
		t.Fatalf("selector = %q, want prefix %q", selector, want)
	}
	if !strings.HasSuffix(selector, "best[vcodec!=none]") {
		t.Fatalf("selector has an audio-only fallback: %q", selector)
	}
	if strings.Contains(selector, "/bestaudio/") {
		t.Fatalf("selector contains standalone audio fallback: %q", selector)
	}
}

func TestParseYTDLPInfoUsesShortEdgeForPortraitVideo(t *testing.T) {
	data := []byte(`{
		"id": "2114640210013163520",
		"title": "Vertical X post",
		"formats": [
			{"format_id": "hls-audio", "ext": "mp4", "acodec": "mp4a.40.2", "vcodec": "none", "abr": 128},
			{"format_id": "hls-1260", "ext": "mp4", "width": 720, "height": 1280, "vcodec": "avc1", "acodec": "none", "tbr": 1260},
			{"format_id": "http-10368", "ext": "mp4", "width": 1080, "height": 1920, "tbr": 10368},
			{"format_id": "hls-4953", "ext": "mp4", "width": 1080, "height": 1920, "vcodec": "avc1", "acodec": "none", "tbr": 4953}
		]
	}`)

	info, err := parseYTDLPInfo(data)
	if err != nil {
		t.Fatalf("parseYTDLPInfo error: %v", err)
	}
	if len(info.VideoFormats) != 2 {
		t.Fatalf("video formats = %+v, want two qualities", info.VideoFormats)
	}
	format := info.VideoFormats[0]
	if format.Resolution != "1080p" || format.Width != 1080 || format.Height != 1920 {
		t.Fatalf("portrait format = %+v, want 1080p at 1080x1920", format)
	}
	selector := ytDLPGenericVideoSelector(format)
	if !strings.Contains(selector, "[width<=1080]") {
		t.Fatalf("portrait selector = %q, want width filter", selector)
	}
	if strings.Contains(selector, "[height<=1080]") {
		t.Fatalf("portrait selector = %q, must not limit portrait height", selector)
	}
}

func TestShouldRetryYTDLPStrategyOnForbiddenResponse(t *testing.T) {
	err := &testStringError{"yt-dlp error: HTTP Error 403: Forbidden"}
	if !shouldRetryYTDLPStrategy(err) {
		t.Fatal("403 response should retry without impersonation")
	}
}

type testStringError struct {
	message string
}

func (e *testStringError) Error() string {
	return e.message
}
