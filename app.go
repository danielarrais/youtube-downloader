package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Config struct {
	DownloadDir string `json:"download_dir"`
	Quality     string `json:"quality"`
	Language    string `json:"language"`
}

var globalConfig Config

func loadConfig() {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".yt-mp3-downloader-config.json")
	data, err := os.ReadFile(configPath)
	if err == nil {
		json.Unmarshal(data, &globalConfig)
	}
	if globalConfig.DownloadDir == "" {
		globalConfig.DownloadDir = filepath.Join(home, "Downloads", "YT-MP3")
	}
	if globalConfig.Quality == "" {
		globalConfig.Quality = "192k"
	}
	if globalConfig.Language != "en-US" {
		globalConfig.Language = "pt-BR"
	}
}

func saveConfig(conf Config) {
	globalConfig = conf
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".yt-mp3-downloader-config.json")
	data, _ := json.Marshal(conf)
	os.WriteFile(configPath, data, 0644)
}

type App struct {
	ctx        context.Context
	items      map[string]*DownloadItem
	queueOrder []string
	mu         sync.Mutex
	persistMu  sync.Mutex
	cacheDir   string
	queuePath  string
	paused     bool
	activeID   string
	activeStop context.CancelFunc
	wakeWorker chan struct{}
}

func NewApp() *App {
	home, _ := os.UserHomeDir()
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = filepath.Join(home, ".config")
	}
	return &App{
		items:      make(map[string]*DownloadItem),
		queueOrder: make([]string, 0),
		cacheDir:   filepath.Join(home, ".yt-mp3-downloader-cache"),
		queuePath:  filepath.Join(configDir, "yt-mp3-go", "queue.json"),
		wakeWorker: make(chan struct{}, 1),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	loadConfig()
	os.MkdirAll(globalConfig.DownloadDir, 0755)
	os.MkdirAll(a.cacheDir, 0755)
	if err := a.loadQueue(); err != nil {
		fmt.Printf("Erro ao carregar fila: %v\n", err)
	}
	go a.worker()
}

func (a *App) shutdown(context.Context) {
	a.mu.Lock()
	stop := a.activeStop
	a.mu.Unlock()
	if stop != nil {
		stop()
	}
	a.persistQueue()
}

func (a *App) processDownload(ctx context.Context, item *DownloadItem) {
	video, err := GetVideoInfoContext(ctx, item.URL)

	if err != nil {
		if ctx.Err() != nil {
			a.cleanupInterruptedDownload(item.ID, "")
			return
		}
		a.mu.Lock()
		language := globalConfig.Language
		a.mu.Unlock()
		a.updateError(item.ID, FormatYouTubeError(err, language))
		return
	}

	a.mu.Lock()
	if !a.isActiveItemLocked(item.ID) {
		a.mu.Unlock()
		return
	}
	item.Title = video.Title
	a.mu.Unlock()
	a.persistQueue()
	a.emitItemUpdate(item.ID)

	safeTitle := SanitizeFilename(video.Title)
	a.mu.Lock()
	downloadDir := globalConfig.DownloadDir
	a.mu.Unlock()
	outputPath := filepath.Join(downloadDir, safeTitle+".mp3")

	if _, err := os.Stat(outputPath); err == nil {
		if ctx.Err() != nil {
			a.cleanupInterruptedDownload(item.ID, "")
			return
		}
		if !a.setActiveItemStatus(item.ID, StatusSkipped) {
			return
		}
		a.mu.Lock()
		if !a.isActiveItemLocked(item.ID) {
			a.mu.Unlock()
			return
		}
		item.FilePath = outputPath
		fi, _ := os.Stat(outputPath)
		item.FileSize = fi.Size()
		a.mu.Unlock()
		a.persistQueue()
		a.emitItemUpdate(item.ID)
		return
	}

	if !a.setActiveItemStatus(item.ID, StatusDownloading) {
		return
	}
	tempPath := filepath.Join(a.cacheDir, item.ID+".tmp")

	_, err = DownloadAudio(ctx, video, tempPath, func(percent float64, downloaded int64, total int64) {
		a.mu.Lock()
		if a.isActiveItemLocked(item.ID) {
			item.Progress.Percent = percent
			item.Progress.DownloadedBytes = downloaded
			item.Progress.TotalBytes = total
		}
		a.mu.Unlock()
		a.emitItemUpdate(item.ID)
	})

	if err != nil {
		if ctx.Err() != nil {
			a.cleanupInterruptedDownload(item.ID, tempPath)
			return
		}
		a.updateError(item.ID, "Erro download: "+err.Error())
		return
	}

	if !a.setActiveItemStatus(item.ID, StatusConverting) {
		os.Remove(tempPath)
		return
	}
	err = ConvertToMp3(ctx, tempPath, outputPath, item.Quality)
	os.Remove(tempPath)

	if err != nil {
		if ctx.Err() != nil {
			os.Remove(outputPath)
			a.cleanupInterruptedDownload(item.ID, "")
			return
		}
		a.updateError(item.ID, "Erro conversão: "+err.Error())
		return
	}

	fi, _ := os.Stat(outputPath)
	a.mu.Lock()
	if !a.isActiveItemLocked(item.ID) {
		a.mu.Unlock()
		os.Remove(outputPath)
		return
	}
	item.Status = StatusCompleted
	item.FilePath = outputPath
	item.FileSize = fi.Size()
	item.CompletedAt = time.Now().Format(time.RFC3339)
	item.Progress.Percent = 100
	a.mu.Unlock()

	a.persistQueue()
	a.emitItemUpdate(item.ID)
	a.emitStats()
}

func (a *App) AddDownloads(urls []string, quality string) []DownloadItem {
	a.mu.Lock()
	var newItems []DownloadItem
	for _, url := range urls {
		url = cleanYouTubeURL(url)
		item := &DownloadItem{
			ID:        uuid.New().String(),
			URL:       url,
			Quality:   quality,
			Status:    StatusPending,
			CreatedAt: time.Now().Format(time.RFC3339),
			Progress:  DownloadProgress{Percent: 0, Speed: "---", ETA: "---"},
		}
		a.items[item.ID] = item
		a.queueOrder = append(a.queueOrder, item.ID)
		newItems = append(newItems, *item)
	}
	a.mu.Unlock()
	a.persistQueue()
	a.emitStats()
	a.signalWorker()
	return newItems
}

func cleanYouTubeURL(rawURL string) string {
	url := strings.TrimSpace(rawURL)
	if index := strings.IndexByte(url, '&'); index >= 0 {
		return url[:index]
	}
	return url
}

func (a *App) GetDownloads() []DownloadItem {
	a.mu.Lock()
	defer a.mu.Unlock()
	items := make([]DownloadItem, 0, len(a.queueOrder))
	for _, id := range a.queueOrder {
		if item, ok := a.items[id]; ok {
			items = append(items, *item)
		}
	}
	return items
}

func (a *App) GetStats() QueueStats {
	a.mu.Lock()
	defer a.mu.Unlock()
	stats := QueueStats{}
	stats.Paused = a.paused
	for _, item := range a.items {
		stats.Total++
		switch item.Status {
		case StatusPending:
			stats.Pending++
		case StatusFetching, StatusDownloading, StatusConverting:
			stats.Downloading++
		case StatusCompleted, StatusSkipped:
			stats.Completed++
		case StatusFailed:
			stats.Failed++
		}
	}
	return stats
}

func (a *App) worker() {
	for {
		a.mu.Lock()
		var targetItem *DownloadItem
		var workCtx context.Context
		if !a.paused {
			for _, id := range a.queueOrder {
				if item, ok := a.items[id]; ok && item.Status == StatusPending {
					targetItem = item
					workCtx, a.activeStop = context.WithCancel(context.Background())
					a.activeID = id
					item.Status = StatusFetching
					if item.StartedAt == "" {
						item.StartedAt = time.Now().Format(time.RFC3339)
					}
					break
				}
			}
		}
		a.mu.Unlock()
		if targetItem == nil {
			<-a.wakeWorker
			continue
		}
		a.persistQueue()
		a.emitItemUpdate(targetItem.ID)
		a.emitStats()
		a.processDownload(workCtx, targetItem)
		a.mu.Lock()
		if a.activeID == targetItem.ID {
			a.activeID = ""
			a.activeStop = nil
		}
		a.mu.Unlock()
	}
}

func (a *App) signalWorker() {
	select {
	case a.wakeWorker <- struct{}{}:
	default:
	}
}

func (a *App) isActiveItemLocked(id string) bool {
	item, ok := a.items[id]
	return ok && a.activeID == id &&
		item.Status != StatusCancelled && item.Status != StatusPending
}

func (a *App) setActiveItemStatus(id string, status DownloadStatus) bool {
	a.mu.Lock()
	item, ok := a.items[id]
	if !ok || a.activeID != id || item.Status == StatusCancelled || item.Status == StatusPending {
		a.mu.Unlock()
		return false
	}
	item.Status = status
	a.mu.Unlock()
	a.persistQueue()
	a.emitItemUpdate(id)
	a.emitStats()
	return true
}

func (a *App) cleanupInterruptedDownload(id, tempPath string) {
	if tempPath != "" {
		os.Remove(tempPath)
	}
	a.persistQueue()
	a.emitItemUpdate(id)
	a.emitStats()
}

func (a *App) updateError(id string, errMsg string) {
	a.mu.Lock()
	item, ok := a.items[id]
	if !ok || !a.isActiveItemLocked(id) {
		a.mu.Unlock()
		return
	}
	item.Status = StatusFailed
	item.Error = errMsg
	item.CompletedAt = time.Now().Format(time.RFC3339)
	a.mu.Unlock()
	a.persistQueue()
	a.emitItemUpdate(id)
	a.emitStats()
}

func (a *App) emitItemUpdate(id string) {
	if a.ctx == nil {
		return
	}
	a.mu.Lock()
	item, ok := a.items[id]
	if !ok {
		a.mu.Unlock()
		return
	}
	val := *item
	a.mu.Unlock()
	runtime.EventsEmit(a.ctx, "download:update", val)
}

func (a *App) emitStats() {
	if a.ctx == nil {
		return
	}
	s := a.GetStats()
	runtime.EventsEmit(a.ctx, "queue:stats", s)
}

func (a *App) CancelDownload(id string) {
	a.mu.Lock()
	item, ok := a.items[id]
	if !ok {
		a.mu.Unlock()
		return
	}
	item.Status = StatusCancelled
	stop := context.CancelFunc(nil)
	if a.activeID == id {
		stop = a.activeStop
	}
	a.mu.Unlock()
	if stop != nil {
		stop()
	}
	a.persistQueue()
	a.emitItemUpdate(id)
	a.emitStats()
}

func (a *App) RetryDownload(id string) DownloadItem {
	a.mu.Lock()
	item, ok := a.items[id]
	if !ok {
		a.mu.Unlock()
		return DownloadItem{}
	}
	item.Status = StatusPending
	item.Error = ""
	item.StartedAt = ""
	item.CompletedAt = ""
	item.Progress = DownloadProgress{Speed: "---", ETA: "---"}
	result := *item
	a.mu.Unlock()
	a.persistQueue()
	a.emitItemUpdate(id)
	a.emitStats()
	a.signalWorker()
	return result
}

func (a *App) RetryFailed() {
	a.mu.Lock()
	for _, item := range a.items {
		if item.Status != StatusFailed {
			continue
		}
		item.Status = StatusPending
		item.Error = ""
		item.StartedAt = ""
		item.CompletedAt = ""
		item.Progress = DownloadProgress{Speed: "---", ETA: "---"}
	}
	a.paused = false
	a.mu.Unlock()
	a.persistQueue()
	a.emitStats()
	a.signalWorker()
}

func (a *App) PauseQueue() {
	a.mu.Lock()
	if a.paused {
		a.mu.Unlock()
		return
	}
	a.paused = true
	activeID := a.activeID
	stop := a.activeStop
	if item, ok := a.items[activeID]; ok {
		item.Status = StatusPending
		item.Error = ""
		item.StartedAt = ""
		item.CompletedAt = ""
		item.Progress = DownloadProgress{Speed: "---", ETA: "---"}
	}
	a.mu.Unlock()
	if stop != nil {
		stop()
	}
	a.persistQueue()
	if activeID != "" {
		a.emitItemUpdate(activeID)
	}
	a.emitStats()
}

func (a *App) ResumeQueue() {
	a.mu.Lock()
	if !a.paused {
		a.mu.Unlock()
		return
	}
	a.paused = false
	a.mu.Unlock()
	a.persistQueue()
	a.emitStats()
	a.signalWorker()
}
func (a *App) OpenFolder(path string) error {
	return openDirectory(filepath.Dir(path))
}

func (a *App) OpenDirectory(path string) error {
	return openDirectory(path)
}

func openDirectory(path string) error {
	switch goruntime.GOOS {
	case "windows":
		return exec.Command("explorer.exe", path).Start()
	case "darwin":
		return exec.Command("open", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}
func (a *App) GetConfig() Config {
	a.mu.Lock()
	defer a.mu.Unlock()
	return globalConfig
}
func (a *App) SaveConfig(config Config) Config {
	a.mu.Lock()
	defer a.mu.Unlock()
	if config.DownloadDir == "" {
		config.DownloadDir = globalConfig.DownloadDir
	}
	switch config.Quality {
	case "128k", "192k", "320k":
	default:
		config.Quality = "192k"
	}
	if config.Language != "en-US" {
		config.Language = globalConfig.Language
	}
	saveConfig(config)
	os.MkdirAll(globalConfig.DownloadDir, 0755)
	return globalConfig
}
func (a *App) SetLanguage(language string) {
	if language != "en-US" {
		language = "pt-BR"
	}

	a.mu.Lock()
	globalConfig.Language = language
	updatedIDs := make([]string, 0)
	for id, item := range a.items {
		if item.Status == StatusFailed {
			item.Error = TranslateStoredYouTubeError(item.Error, language)
			updatedIDs = append(updatedIDs, id)
		}
	}
	config := globalConfig
	saveConfig(config)
	a.mu.Unlock()

	a.persistQueue()
	for _, id := range updatedIDs {
		a.emitItemUpdate(id)
	}
}
func (a *App) GetPlaylistInfo(url string) (PlaylistInfo, error) {
	return FetchPlaylistInfo(url)
}
func (a *App) SelectFolder() string {
	a.mu.Lock()
	defaultDirectory := globalConfig.DownloadDir
	a.mu.Unlock()
	folder, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Selecione a pasta de download",
		DefaultDirectory: defaultDirectory,
	})
	if err != nil || folder == "" {
		return ""
	}
	return folder
}
func (a *App) ClearCompleted() {
	a.mu.Lock()
	newOrder := make([]string, 0)
	for _, id := range a.queueOrder {
		if a.items[id].Status != StatusCompleted && a.items[id].Status != StatusSkipped {
			newOrder = append(newOrder, id)
		} else {
			delete(a.items, id)
		}
	}
	a.queueOrder = newOrder
	a.mu.Unlock()
	a.persistQueue()
	a.emitStats()
}
func (a *App) CancelAll() {
	a.mu.Lock()
	for _, item := range a.items {
		if item.Status == StatusPending ||
			item.Status == StatusFetching ||
			item.Status == StatusDownloading ||
			item.Status == StatusConverting {
			item.Status = StatusCancelled
		}
	}
	activeID := a.activeID
	stop := a.activeStop
	a.mu.Unlock()
	if stop != nil {
		stop()
	}
	a.persistQueue()
	if activeID != "" {
		a.emitItemUpdate(activeID)
	}
	a.emitStats()
}
func (a *App) ClearAll() {
	a.mu.Lock()
	stop := a.activeStop
	a.items = make(map[string]*DownloadItem)
	a.queueOrder = make([]string, 0)
	a.mu.Unlock()
	if stop != nil {
		stop()
	}
	a.persistQueue()
	a.emitStats()
}
