package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
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
		{
			name: "keeps parameters from other platforms",
			url:  " https://www.tiktok.com/@user/video/123?is_from_webapp=1&sender_device=pc ",
			want: "https://www.tiktok.com/@user/video/123?is_from_webapp=1&sender_device=pc",
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

func TestStartAvailableDownloadsHonorsParallelLimit(t *testing.T) {
	app := newParallelSchedulerTestApp(t, 2, []string{"first", "second", "third"})
	release := make(chan struct{})
	started := make(chan string, 3)
	app.processDownloadFunc = func(ctx context.Context, item *DownloadItem) {
		started <- item.ID
		select {
		case <-ctx.Done():
		case <-release:
		}
	}
	t.Cleanup(func() { close(release) })

	app.startAvailableDownloads()
	waitForCondition(t, func() bool { return len(started) == 2 })

	app.mu.Lock()
	defer app.mu.Unlock()
	if len(app.active) != 2 {
		t.Fatalf("active downloads = %d, want 2", len(app.active))
	}
	if app.items["first"].Status != StatusFetching || app.items["second"].Status != StatusFetching {
		t.Fatalf("first two items should be fetching: %#v", app.items)
	}
	if app.items["third"].Status != StatusPending {
		t.Fatalf("third item status = %s, want pending", app.items["third"].Status)
	}
}

func TestStartAvailableDownloadsFillsFreedSlot(t *testing.T) {
	app := newParallelSchedulerTestApp(t, 2, []string{"first", "second", "third"})
	releases := map[string]*testRelease{
		"first":  newTestRelease(),
		"second": newTestRelease(),
		"third":  newTestRelease(),
	}
	var mu sync.Mutex
	started := make([]string, 0, 3)
	app.processDownloadFunc = func(ctx context.Context, item *DownloadItem) {
		mu.Lock()
		started = append(started, item.ID)
		mu.Unlock()
		select {
		case <-ctx.Done():
		case <-releases[item.ID].ch:
		}
	}
	t.Cleanup(func() {
		for _, release := range releases {
			release.close()
		}
	})

	app.startAvailableDownloads()
	waitForCondition(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(started) == 2
	})
	releases["first"].close()
	waitForCondition(t, func() bool {
		app.mu.Lock()
		defer app.mu.Unlock()
		_, active := app.active["first"]
		return !active
	})

	app.startAvailableDownloads()
	waitForCondition(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(started) == 3
	})
	app.mu.Lock()
	defer app.mu.Unlock()
	if _, active := app.active["third"]; !active {
		t.Fatalf("third item was not started after freeing a slot: active=%#v", app.active)
	}
}

type testRelease struct {
	ch   chan struct{}
	once sync.Once
}

func newTestRelease() *testRelease {
	return &testRelease{ch: make(chan struct{})}
}

func (r *testRelease) close() {
	r.once.Do(func() { close(r.ch) })
}

func newParallelSchedulerTestApp(t *testing.T, parallelDownloads int, ids []string) *App {
	t.Helper()
	app := newPersistenceTestApp(t)
	app.config.ParallelDownloads = parallelDownloads
	for _, id := range ids {
		app.items[id] = &DownloadItem{ID: id, Status: StatusPending}
		app.queueOrder = append(app.queueOrder, id)
	}
	return app
}

func waitForCondition(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition was not met before timeout")
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

func TestAddFormatSelectionErrorsAsFailedItems(t *testing.T) {
	app := newPersistenceTestApp(t)
	audio := app.AddAudioDownloads([]AudioDownloadRequest{{URL: "https://youtu.be/audio", Error: "audio error"}}, "192k")
	video := app.AddVideoDownloads([]VideoDownloadRequest{{URL: "https://youtu.be/video", Error: "video error"}})

	if len(audio) != 1 || audio[0].Status != StatusFailed || audio[0].Error != "audio error" {
		t.Fatalf("audio failed item = %#v", audio)
	}
	if len(video) != 1 || video[0].Status != StatusFailed || video[0].Error != "video error" {
		t.Fatalf("video failed item = %#v", video)
	}
	if stats := app.GetStats(); stats.Failed != 2 || stats.Pending != 0 {
		t.Fatalf("stats = %#v", stats)
	}
}
