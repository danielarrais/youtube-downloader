//go:build web

package main

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestWebHealthAndDownloads(t *testing.T) {
	app := newWebTestApp(t)
	handler := newWebHandler(app, testAssets(), app.config.DownloadDir)

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("health status = %d", response.Code)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/downloads",
		strings.NewReader(`{"urls":["https://youtu.be/video"],"quality":"192k"}`))
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("add download status = %d, body = %s", response.Code, response.Body.String())
	}

	var items []DownloadItem
	if err := json.Unmarshal(response.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Status != StatusPending {
		t.Fatalf("added items = %#v", items)
	}
}

func TestWebAddsSelectedVideoDownloads(t *testing.T) {
	app := newWebTestApp(t)
	handler := newWebHandler(app, testAssets(), app.config.DownloadDir)
	request := httptest.NewRequest(http.MethodPost, "/api/downloads", strings.NewReader(`{
		"media_type":"video",
		"video_requests":[{"url":"https://youtu.be/video","format":{"video_itag":137,"audio_itag":140,"container":"mp4","extension":"mp4","resolution":"1080p","label":"1080p (MP4)"}}]
	}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("add video status = %d, body = %s", response.Code, response.Body.String())
	}
	var items []DownloadItem
	if err := json.Unmarshal(response.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].MediaType != MediaTypeVideo || items[0].VideoFormat == nil || items[0].VideoFormat.VideoItag != 137 {
		t.Fatalf("added items = %#v", items)
	}
}

func TestCheckWebHealth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	if err := checkWebHealth(server.URL); err != nil {
		t.Fatalf("checkWebHealth() error = %v", err)
	}
}

func TestCheckWebHealthRejectsFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	if err := checkWebHealth(server.URL); err == nil {
		t.Fatal("checkWebHealth() expected an error")
	}
}

func TestWebRejectsInvalidJSON(t *testing.T) {
	app := newWebTestApp(t)
	handler := newWebHandler(app, testAssets(), app.config.DownloadDir)
	request := httptest.NewRequest(http.MethodPost, "/api/downloads", strings.NewReader(`{"urls":`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestWebRejectsInvalidQuality(t *testing.T) {
	app := newWebTestApp(t)
	handler := newWebHandler(app, testAssets(), app.config.DownloadDir)
	request := httptest.NewRequest(http.MethodPost, "/api/downloads",
		strings.NewReader(`{"urls":["https://youtu.be/video"],"quality":"invalid"}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestWebServesCompletedDownload(t *testing.T) {
	app := newWebTestApp(t)
	path := filepath.Join(app.config.DownloadDir, "song.mp3")
	if err := os.WriteFile(path, []byte("mp3-data"), 0644); err != nil {
		t.Fatal(err)
	}
	app.items["completed"] = &DownloadItem{
		ID:       "completed",
		Status:   StatusCompleted,
		FilePath: path,
	}
	app.queueOrder = []string{"completed"}
	handler := newWebHandler(app, testAssets(), app.config.DownloadDir)
	request := httptest.NewRequest(http.MethodGet, "/api/downloads/completed/file", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if response.Body.String() != "mp3-data" {
		t.Fatalf("body = %q", response.Body.String())
	}
	if disposition := response.Header().Get("Content-Disposition"); !strings.Contains(disposition, "song.mp3") {
		t.Fatalf("Content-Disposition = %q", disposition)
	}
}

func TestWebRejectsFilesOutsideDownloadDirectory(t *testing.T) {
	app := newWebTestApp(t)
	outsidePath := filepath.Join(t.TempDir(), "secret.mp3")
	if err := os.WriteFile(outsidePath, []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}
	app.items["outside"] = &DownloadItem{
		ID:       "outside",
		Status:   StatusCompleted,
		FilePath: outsidePath,
	}
	handler := newWebHandler(app, testAssets(), app.config.DownloadDir)
	request := httptest.NewRequest(http.MethodGet, "/api/downloads/outside/file", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestWebRejectsIncompleteDownloadFile(t *testing.T) {
	app := newWebTestApp(t)
	app.items["pending"] = &DownloadItem{ID: "pending", Status: StatusPending}
	handler := newWebHandler(app, testAssets(), app.config.DownloadDir)
	request := httptest.NewRequest(http.MethodGet, "/api/downloads/pending/file", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusConflict)
	}
}

func TestWebConfigKeepsServerDownloadDirectory(t *testing.T) {
	app := newWebTestApp(t)
	handler := newWebHandler(app, testAssets(), app.config.DownloadDir)
	request := httptest.NewRequest(http.MethodPut, "/api/config", strings.NewReader(
		`{"download_dir":"/tmp/other","quality":"320k","video_container":"webm","video_quality":"720p","language":"en-US"}`,
	))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var config Config
	if err := json.Unmarshal(response.Body.Bytes(), &config); err != nil {
		t.Fatal(err)
	}
	if config.DownloadDir != app.config.DownloadDir {
		t.Fatalf("download dir = %q, want %q", config.DownloadDir, app.config.DownloadDir)
	}
	if config.Quality != "320k" {
		t.Fatalf("quality = %q", config.Quality)
	}
	if config.VideoContainer != "webm" || config.VideoQuality != "720p" {
		t.Fatalf("video preferences = %#v", config)
	}
}

func newWebTestApp(t *testing.T) *App {
	t.Helper()
	dataDir := t.TempDir()
	downloadDir := filepath.Join(t.TempDir(), "downloads")
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		t.Fatal(err)
	}
	app := NewAppWithPaths(dataDir, downloadDir)
	app.config = Config{
		DownloadDir:    downloadDir,
		Quality:        "192k",
		VideoContainer: defaultVideoContainer,
		VideoQuality:   defaultVideoQuality,
		Language:       "pt-BR",
	}
	return app
}

func testAssets() fs.FS {
	return fstest.MapFS{
		"index.html": {Data: []byte("<html>test</html>")},
	}
}
