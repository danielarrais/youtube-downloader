package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCleanYouTubeURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "removes video parameters",
			url:  "https://www.youtube.com/watch?v=3CWL9WXYSWU&list=RD3CWL9WXYSWU&start_radio=1",
			want: "https://www.youtube.com/watch?v=3CWL9WXYSWU",
		},
		{
			name: "keeps playlist identifier",
			url:  "https://www.youtube.com/playlist?list=PLrzuA2--JVdSPr5FdYtBixGMw9uCWfsSi&index=2",
			want: "https://www.youtube.com/playlist?list=PLrzuA2--JVdSPr5FdYtBixGMw9uCWfsSi",
		},
		{
			name: "trims spaces",
			url:  "  https://youtu.be/3CWL9WXYSWU  ",
			want: "https://youtu.be/3CWL9WXYSWU",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := cleanYouTubeURL(test.url); got != test.want {
				t.Fatalf("cleanYouTubeURL(%q) = %q, want %q", test.url, got, test.want)
			}
		})
	}
}

func TestConvertAndPublishRemovesPartialFileAfterFailure(t *testing.T) {
	originalConverter := convertAudioToMP3
	t.Cleanup(func() {
		convertAudioToMP3 = originalConverter
	})
	convertAudioToMP3 = func(_ context.Context, _, outputPath, _ string, _ *MusicMetadata, _ string) error {
		if err := os.WriteFile(outputPath, []byte("partial"), 0644); err != nil {
			t.Fatal(err)
		}
		return errors.New("ffmpeg failed")
	}

	outputPath := filepath.Join(t.TempDir(), "song.mp3")
	err := convertAndPublish(context.Background(), "input.tmp", outputPath, "192k", nil, "")
	if err == nil {
		t.Fatal("convertAndPublish() returned nil error")
	}
	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
		t.Fatalf("final output should not exist: %v", err)
	}
	if _, err := os.Stat(outputPath + ".part"); !os.IsNotExist(err) {
		t.Fatalf("partial output should be removed: %v", err)
	}
}

func TestConvertAndPublishPublishesSuccessfulConversion(t *testing.T) {
	originalConverter := convertAudioToMP3
	t.Cleanup(func() {
		convertAudioToMP3 = originalConverter
	})
	convertAudioToMP3 = func(_ context.Context, _, outputPath, _ string, _ *MusicMetadata, _ string) error {
		return os.WriteFile(outputPath, []byte("mp3"), 0644)
	}

	outputPath := filepath.Join(t.TempDir(), "song.mp3")
	if err := convertAndPublish(context.Background(), "input.tmp", outputPath, "192k", nil, ""); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "mp3" {
		t.Fatalf("final output = %q, want mp3", data)
	}
}

func TestPublishConvertedFileDoesNotReplaceExistingFile(t *testing.T) {
	dir := t.TempDir()
	partPath := filepath.Join(dir, "song.mp3.part")
	outputPath := filepath.Join(dir, "song.mp3")
	if err := os.WriteFile(partPath, []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outputPath, []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := publishConvertedFile(partPath, outputPath); !os.IsExist(err) {
		t.Fatalf("publishConvertedFile() error = %v, want os.ErrExist", err)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "existing" {
		t.Fatalf("existing output was replaced: %q", data)
	}
}

func TestResolveOutputPathAddsVideoIDForQueueCollision(t *testing.T) {
	app := newPersistenceTestApp(t)
	basePath := filepath.Join(t.TempDir(), "same title.mp3")
	app.items["completed"] = &DownloadItem{
		ID:       "completed",
		URL:      "https://www.youtube.com/watch?v=first",
		FilePath: basePath,
	}
	current := &DownloadItem{
		ID:  "current",
		URL: "https://www.youtube.com/watch?v=second",
	}
	app.items[current.ID] = current

	got := app.resolveOutputPath(current, "second", basePath)
	want := filepath.Join(filepath.Dir(basePath), "same title [second].mp3")
	if got != want {
		t.Fatalf("resolveOutputPath() = %q, want %q", got, want)
	}
}

func TestShouldSkipExistingOutputForSameAudioQuality(t *testing.T) {
	app := newPersistenceTestApp(t)
	path := filepath.Join(t.TempDir(), "song.mp3")
	app.items["completed"] = &DownloadItem{ID: "completed", URL: "https://youtu.be/video", FilePath: path, MediaType: MediaTypeAudio, Quality: "192k"}
	current := &DownloadItem{ID: "current", URL: "https://youtu.be/video", FilePath: path, MediaType: MediaTypeAudio, Quality: "192k"}

	if !app.shouldSkipExistingOutput(current, path) {
		t.Fatal("expected existing audio file to be skipped")
	}
}

func TestShouldReplaceExistingOutputForDifferentAudioQuality(t *testing.T) {
	app := newPersistenceTestApp(t)
	path := filepath.Join(t.TempDir(), "song.mp3")
	app.items["completed"] = &DownloadItem{ID: "completed", URL: "https://youtu.be/video", FilePath: path, MediaType: MediaTypeAudio, Quality: "192k"}
	current := &DownloadItem{ID: "current", URL: "https://youtu.be/video", FilePath: path, MediaType: MediaTypeAudio, Quality: "320k"}

	if app.shouldSkipExistingOutput(current, path) {
		t.Fatal("expected existing audio file with different quality to be replaced")
	}
}

func TestCleanupPartialFilesRemovesKnownMediaParts(t *testing.T) {
	dir := t.TempDir()
	partPath := filepath.Join(dir, "song.mp3.part")
	videoPartPath := filepath.Join(dir, "movie.mp4.part")
	otherPath := filepath.Join(dir, "keep.tmp")
	for _, path := range []string{partPath, videoPartPath, otherPath} {
		if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := cleanupPartialFiles(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(partPath); !os.IsNotExist(err) {
		t.Fatalf("partial MP3 was not removed: %v", err)
	}
	if _, err := os.Stat(videoPartPath); !os.IsNotExist(err) {
		t.Fatalf("partial video was not removed: %v", err)
	}
	if _, err := os.Stat(otherPath); err != nil {
		t.Fatalf("unrelated file was removed: %v", err)
	}
}

func TestRemoveDownloadDeletesCompletedFile(t *testing.T) {
	app := NewApp()
	downloadDir := t.TempDir()
	app.config = defaultConfig(t.TempDir())
	app.config.DownloadDir = downloadDir
	app.cacheDir = t.TempDir()
	app.queuePath = filepath.Join(t.TempDir(), "queue.json")
	path := filepath.Join(downloadDir, "song.mp3")
	if err := os.WriteFile(path, []byte("audio"), 0644); err != nil {
		t.Fatal(err)
	}
	app.items["completed"] = &DownloadItem{ID: "completed", Status: StatusCompleted, FilePath: path}
	app.queueOrder = []string{"completed"}

	if err := app.RemoveDownload("completed", true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("download file was not removed: %v", err)
	}
	if len(app.GetDownloads()) != 0 {
		t.Fatalf("queue still contains removed item: %#v", app.GetDownloads())
	}
}

func TestRemoveDownloadRejectsFileOutsideDownloadDirectory(t *testing.T) {
	app := NewApp()
	app.config = defaultConfig(t.TempDir())
	app.config.DownloadDir = t.TempDir()
	app.queuePath = filepath.Join(t.TempDir(), "queue.json")
	outsidePath := filepath.Join(t.TempDir(), "outside.mp3")
	if err := os.WriteFile(outsidePath, []byte("audio"), 0644); err != nil {
		t.Fatal(err)
	}
	app.items["completed"] = &DownloadItem{ID: "completed", Status: StatusCompleted, FilePath: outsidePath}
	app.queueOrder = []string{"completed"}

	if err := app.RemoveDownload("completed", true); err == nil {
		t.Fatal("RemoveDownload() returned nil error for external path")
	}
	if _, err := os.Stat(outsidePath); err != nil {
		t.Fatalf("external file was changed: %v", err)
	}
	if len(app.GetDownloads()) != 1 {
		t.Fatal("queue item was removed after failed file deletion")
	}
}

func TestClearCompletedDeletesCompletedFiles(t *testing.T) {
	app := NewApp()
	downloadDir := t.TempDir()
	app.config = defaultConfig(t.TempDir())
	app.config.DownloadDir = downloadDir
	app.cacheDir = t.TempDir()
	app.queuePath = filepath.Join(t.TempDir(), "queue.json")
	completedPath := filepath.Join(downloadDir, "completed.mp3")
	skippedPath := filepath.Join(downloadDir, "skipped.mp4")
	for _, path := range []string{completedPath, skippedPath} {
		if err := os.WriteFile(path, []byte("file"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	app.items["completed"] = &DownloadItem{ID: "completed", Status: StatusCompleted, FilePath: completedPath}
	app.items["skipped"] = &DownloadItem{ID: "skipped", Status: StatusSkipped, FilePath: skippedPath}
	app.items["pending"] = &DownloadItem{ID: "pending", Status: StatusPending}
	app.queueOrder = []string{"completed", "skipped", "pending"}

	if err := app.ClearCompleted(true); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{completedPath, skippedPath} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("completed file was not removed: %v", err)
		}
	}
	items := app.GetDownloads()
	if len(items) != 1 || items[0].ID != "pending" {
		t.Fatalf("remaining queue items = %#v", items)
	}
}

func TestSaveConfigDefaultsInvalidQuality(t *testing.T) {
	app := NewApp()
	app.configPath = filepath.Join(t.TempDir(), "config.json")
	app.config = Config{DownloadDir: t.TempDir(), Quality: "192k", Language: "pt-BR", Theme: defaultTheme}
	config, err := app.SaveConfig(Config{
		DownloadDir: t.TempDir(),
		Quality:     "invalid",
	})
	if err != nil {
		t.Fatal(err)
	}

	if config.Quality != "192k" {
		t.Fatalf("SaveConfig() quality = %q, want %q", config.Quality, "192k")
	}
	if config.Theme != defaultTheme {
		t.Fatalf("SaveConfig() theme = %q, want %q", config.Theme, defaultTheme)
	}
}
