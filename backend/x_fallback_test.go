package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestXStatusID(t *testing.T) {
	tests := map[string]string{
		"https://x.com/i/status/2124199165802537367":               "2124199165802537367",
		"https://x.com/paralherme/status/2124199165802537367?s=20": "2124199165802537367",
		"https://mobile.twitter.com/user/status/123":               "123",
	}
	for rawURL, want := range tests {
		got, err := xStatusID(rawURL)
		if err != nil {
			t.Errorf("xStatusID(%q) error = %v", rawURL, err)
			continue
		}
		if got != want {
			t.Errorf("xStatusID(%q) = %q, want %q", rawURL, got, want)
		}
	}
	if _, err := xStatusID("https://x.com/user"); err == nil {
		t.Fatal("xStatusID accepted a link without a status ID")
	}
}

func TestFetchXFallbackInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/status/2124199165802537367" {
			t.Errorf("request path = %q", request.URL.Path)
		}
		_, _ = writer.Write([]byte(`{
			"code": 200,
			"tweet": {
				"id": "2124199165802537367",
				"text": "Post with a video",
				"media": {
					"videos": [{
						"url": "https://video.twimg.com/amplify_video/video.mp4",
						"thumbnail_url": "https://pbs.twimg.com/video.jpg",
						"width": 720,
						"height": 926
					}]
				}
			}
		}`))
	}))
	defer server.Close()

	originalBaseURL, originalClient := xFallbackBaseURL, xFallbackClient
	xFallbackBaseURL, xFallbackClient = server.URL, server.Client()
	t.Cleanup(func() {
		xFallbackBaseURL, xFallbackClient = originalBaseURL, originalClient
	})

	info, err := fetchXFallbackInfo(context.Background(), "https://x.com/i/status/2124199165802537367")
	if err != nil {
		t.Fatalf("fetchXFallbackInfo error: %v", err)
	}
	if info.ID != "2124199165802537367" || info.Title != "Post with a video" {
		t.Fatalf("unexpected info: %+v", info)
	}
	if info.SourceURL != "https://video.twimg.com/amplify_video/video.mp4" {
		t.Fatalf("source URL = %q", info.SourceURL)
	}
	if len(info.VideoFormats) != 1 {
		t.Fatalf("formats = %+v", info.VideoFormats)
	}
	format := info.VideoFormats[0]
	if format.Resolution != "720p" || format.Width != 720 || format.Height != 926 {
		t.Fatalf("format = %+v", format)
	}
}

func TestShouldUseXFallback(t *testing.T) {
	if !shouldUseXFallback(&testStringError{"yt-dlp error: No video could be found in this tweet"}) {
		t.Fatal("expected the missing X video error to use the fallback")
	}
	if shouldUseXFallback(&testStringError{"yt-dlp error: HTTP Error 403: Forbidden"}) {
		t.Fatal("unexpected X fallback for a generic 403 error")
	}
}

func TestIsXVideoURL(t *testing.T) {
	if !isXVideoURL("https://video.twimg.com/amplify_video/video.mp4") {
		t.Fatal("expected a video.twimg.com URL to be accepted")
	}
	for _, rawURL := range []string{
		"http://video.twimg.com/video.mp4",
		"https://video.twimg.com.evil.example/video.mp4",
		"https://example.com/video.mp4",
	} {
		if isXVideoURL(rawURL) {
			t.Errorf("isXVideoURL(%q) = true", rawURL)
		}
	}
}
