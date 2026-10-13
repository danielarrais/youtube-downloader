package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type App struct {
	ctx                 context.Context
	config              Config
	configPath          string
	items               map[string]*DownloadItem
	queueOrder          []string
	mu                  sync.Mutex
	configMu            sync.Mutex
	persistMu           sync.Mutex
	cacheDir            string
	queuePath           string
	paused              bool
	active              map[string]context.CancelFunc
	parallelOverride    int
	wakeWorker          chan struct{}
	onItem              func(DownloadItem)
	onStats             func(QueueStats)
	fixedDir            string
	processDownloadFunc func(context.Context, *DownloadItem)
}

func NewApp() *App {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = filepath.Join(home, ".config")
	}
	return &App{
		config:     defaultConfig(home),
		configPath: filepath.Join(home, ".youtube-mp3-downloader-config.json"),
		items:      make(map[string]*DownloadItem),
		active:     make(map[string]context.CancelFunc),
		queueOrder: make([]string, 0),
		cacheDir:   filepath.Join(home, ".youtube-mp3-downloader-cache"),
		queuePath:  filepath.Join(configDir, "youtube-mp3-downloader", "queue.json"),
		wakeWorker: make(chan struct{}, 1),
		onItem:     func(DownloadItem) {},
		onStats:    func(QueueStats) {},
	}
}

func NewAppWithPaths(dataDir, downloadDir string) *App {
	app := NewApp()
	app.config = defaultConfig(dataDir)
	app.config.DownloadDir = downloadDir
	app.configPath = filepath.Join(dataDir, "config.json")
	app.cacheDir = filepath.Join(dataDir, "cache")
	app.queuePath = filepath.Join(dataDir, "queue.json")
	app.fixedDir = downloadDir
	return app
}

func (a *App) start() {
	config, err := loadConfigFile(a.configPath, a.config)
	if err != nil {
		fmt.Printf("Erro ao carregar configuração: %v\n", err)
	} else {
		a.config = config
	}
	if a.fixedDir != "" {
		a.config.DownloadDir = a.fixedDir
	}
	if err := os.MkdirAll(a.config.DownloadDir, 0755); err != nil {
		fmt.Printf("Erro ao criar pasta de download: %v\n", err)
	}
	if err := os.MkdirAll(a.cacheDir, 0755); err != nil {
		fmt.Printf("Erro ao criar cache: %v\n", err)
	}
	if err := cleanupPartialFiles(a.config.DownloadDir); err != nil {
		fmt.Printf("Erro ao limpar arquivos parciais: %v\n", err)
	}
	if err := a.loadQueue(); err != nil {
		fmt.Printf("Erro ao carregar fila: %v\n", err)
	}
	go a.worker()
}

func (a *App) stop() {
	a.mu.Lock()
	stops := make([]context.CancelFunc, 0, len(a.active))
	for _, stop := range a.active {
		stops = append(stops, stop)
	}
	a.mu.Unlock()
	for _, stop := range stops {
		stop()
	}
	a.persistQueue()
}

func (a *App) setEventHandlers(onItem func(DownloadItem), onStats func(QueueStats)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if onItem != nil {
		a.onItem = onItem
	}
	if onStats != nil {
		a.onStats = onStats
	}
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
			MediaType: MediaTypeAudio,
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

func (a *App) AddAudioDownloads(requests []AudioDownloadRequest, quality string) []DownloadItem {
	a.mu.Lock()
	newItems := make([]DownloadItem, 0, len(requests))
	for _, request := range requests {
		url := cleanYouTubeURL(request.URL)
		format := request.Format
		status := StatusPending
		if request.Error != "" {
			status = StatusFailed
		}
		item := &DownloadItem{
			ID:          uuid.New().String(),
			URL:         url,
			Title:       url,
			Quality:     quality,
			MediaType:   MediaTypeAudio,
			AudioFormat: &format,
			Status:      status,
			Error:       request.Error,
			CreatedAt:   time.Now().Format(time.RFC3339),
			Progress:    DownloadProgress{Percent: 0, Speed: "---", ETA: "---"},
		}
		if request.Error == "" {
			item.Title = ""
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

func (a *App) AddVideoDownloads(requests []VideoDownloadRequest) []DownloadItem {
	a.mu.Lock()
	newItems := make([]DownloadItem, 0, len(requests))
	for _, request := range requests {
		url := cleanYouTubeURL(request.URL)
		format := request.Format
		status := StatusPending
		if request.Error != "" {
			status = StatusFailed
		}
		item := &DownloadItem{
			ID:          uuid.New().String(),
			URL:         url,
			Title:       url,
			MediaType:   MediaTypeVideo,
			VideoFormat: &format,
			Status:      status,
			Error:       request.Error,
			CreatedAt:   time.Now().Format(time.RFC3339),
			Progress:    DownloadProgress{Percent: 0, Speed: "---", ETA: "---"},
		}
		if request.Error == "" {
			item.Title = ""
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
	if !isYouTubeURL(url) {
		return url
	}
	if index := strings.IndexByte(url, '&'); index >= 0 {
		return url[:index]
	}
	return url
}

func (a *App) GetVideoFormats(url string) (VideoInfo, error) {
	url = cleanYouTubeURL(url)
	if !isYouTubeURL(url) {
		return getGenericVideoInfo(context.Background(), url)
	}
	return NewYouTubeSession().GetVideoFormats(context.Background(), url)
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
		switch {
		case item.Status == StatusPending:
			stats.Pending++
		case isRunningStatus(item.Status):
			stats.Downloading++
		case isCompletedStatus(item.Status):
			stats.Completed++
		case item.Status == StatusFailed:
			stats.Failed++
		}
	}
	return stats
}

func (a *App) worker() {
	for {
		a.startAvailableDownloads()
		<-a.wakeWorker
	}
}

func (a *App) startAvailableDownloads() {
	for {
		a.mu.Lock()
		var targetItem *DownloadItem
		var workCtx context.Context
		if !a.paused && len(a.active) < a.maxParallelDownloadsLocked() {
			for _, id := range a.queueOrder {
				if item, ok := a.items[id]; ok && item.Status == StatusPending {
					targetItem = item
					var stop context.CancelFunc
					workCtx, stop = context.WithCancel(context.Background())
					a.ensureActiveMapLocked()
					a.active[id] = stop
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
			return
		}
		a.persistQueue()
		a.emitItemUpdate(targetItem.ID)
		a.emitStats()
		go a.runDownload(workCtx, targetItem)
	}
}

func (a *App) runDownload(ctx context.Context, item *DownloadItem) {
	if a.processDownloadFunc != nil {
		a.processDownloadFunc(ctx, item)
	} else {
		a.processDownload(ctx, item)
	}
	a.mu.Lock()
	delete(a.active, item.ID)
	a.mu.Unlock()
	a.signalWorker()
}

func (a *App) ensureActiveMapLocked() {
	if a.active == nil {
		a.active = make(map[string]context.CancelFunc)
	}
}

func (a *App) maxParallelDownloadsLocked() int {
	if a.parallelOverride > 0 {
		return normalizeParallelDownloads(a.parallelOverride, a.parallelOverride)
	}
	return normalizeParallelDownloads(a.config.ParallelDownloads, defaultParallelDownloads())
}

func (a *App) effectiveParallelDownloads() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.maxParallelDownloadsLocked()
}

func (a *App) signalWorker() {
	select {
	case a.wakeWorker <- struct{}{}:
	default:
	}
}

func (a *App) isActiveItemLocked(id string) bool {
	item, ok := a.items[id]
	_, active := a.active[id]
	return ok && active && canUpdateActiveStatus(item.Status)
}

func (a *App) setActiveItemStatus(id string, status DownloadStatus) bool {
	a.mu.Lock()
	item, ok := a.items[id]
	if _, active := a.active[id]; !ok || !active || !canUpdateActiveStatus(item.Status) {
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
	a.mu.Lock()
	item, ok := a.items[id]
	if !ok {
		a.mu.Unlock()
		return
	}
	val := *item
	onItem := a.onItem
	a.mu.Unlock()
	if onItem != nil {
		onItem(val)
	}
}

func (a *App) emitStats() {
	s := a.GetStats()
	a.mu.Lock()
	onStats := a.onStats
	a.mu.Unlock()
	if onStats != nil {
		onStats(s)
	}
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
	if activeStop, ok := a.active[id]; ok {
		stop = activeStop
	}
	a.mu.Unlock()
	if stop != nil {
		stop()
	}
	a.persistQueue()
	a.emitItemUpdate(id)
	a.emitStats()
}

func (a *App) downloadByID(id string) (DownloadItem, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	item, ok := a.items[id]
	if !ok {
		return DownloadItem{}, false
	}
	return *item, true
}

func (a *App) RetryDownload(id string) (DownloadItem, error) {
	a.mu.Lock()
	item, ok := a.items[id]
	if !ok {
		a.mu.Unlock()
		return DownloadItem{}, os.ErrNotExist
	}
	resetItemForRetry(item)
	result := *item
	a.mu.Unlock()
	a.persistQueue()
	a.emitItemUpdate(id)
	a.emitStats()
	a.signalWorker()
	return result, nil
}

func (a *App) RetryFailed() {
	a.mu.Lock()
	for _, item := range a.items {
		if item.Status != StatusFailed {
			continue
		}
		resetItemForRetry(item)
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
	stops := make([]context.CancelFunc, 0, len(a.active))
	activeIDs := make([]string, 0, len(a.active))
	for id, stop := range a.active {
		if item, ok := a.items[id]; ok {
			resetItemForRetry(item)
			activeIDs = append(activeIDs, id)
		}
		stops = append(stops, stop)
	}
	a.mu.Unlock()
	for _, stop := range stops {
		stop()
	}
	a.persistQueue()
	for _, id := range activeIDs {
		a.emitItemUpdate(id)
	}
	a.emitStats()
}

func resetItemForRetry(item *DownloadItem) {
	item.Status = StatusPending
	item.Error = ""
	item.StartedAt = ""
	item.CompletedAt = ""
	item.Progress = DownloadProgress{Speed: "---", ETA: "---"}
}

func (a *App) currentLanguage() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.config.Language
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
func (a *App) GetPlaylistInfo(url string) (PlaylistInfo, error) {
	return FetchPlaylistInfo(url)
}
func (a *App) RemoveDownload(id string, deleteFile bool) error {
	a.mu.Lock()
	item, ok := a.items[id]
	if !ok {
		a.mu.Unlock()
		return os.ErrNotExist
	}
	itemCopy := *item
	downloadDir := a.config.DownloadDir
	stop := context.CancelFunc(nil)
	if activeStop, ok := a.active[id]; ok {
		item.Status = StatusCancelled
		stop = activeStop
	}
	a.mu.Unlock()

	if stop != nil {
		stop()
	}
	if deleteFile && isCompletedStatus(itemCopy.Status) {
		if err := removeDownloadFile(downloadDir, itemCopy.FilePath); err != nil {
			return err
		}
	}
	for _, suffix := range []string{".tmp", ".video.tmp", ".audio.tmp"} {
		_ = os.Remove(filepath.Join(a.cacheDir, id+suffix))
	}

	a.mu.Lock()
	if _, ok := a.items[id]; !ok {
		a.mu.Unlock()
		return os.ErrNotExist
	}
	delete(a.items, id)
	for index, queuedID := range a.queueOrder {
		if queuedID == id {
			a.queueOrder = append(a.queueOrder[:index], a.queueOrder[index+1:]...)
			break
		}
	}
	a.mu.Unlock()
	a.persistQueue()
	a.emitStats()
	return nil
}

func (a *App) ClearCompleted(deleteFiles bool) error {
	items := a.GetDownloads()
	for _, item := range items {
		if !isCompletedStatus(item.Status) {
			continue
		}
		if err := a.RemoveDownload(item.ID, deleteFiles); err != nil {
			return err
		}
	}
	return nil
}
func (a *App) CancelAll() {
	a.mu.Lock()
	activeIDs := make([]string, 0, len(a.active))
	stops := make([]context.CancelFunc, 0, len(a.active))
	for _, item := range a.items {
		if isCancellableStatus(item.Status) {
			item.Status = StatusCancelled
		}
	}
	for id, stop := range a.active {
		activeIDs = append(activeIDs, id)
		stops = append(stops, stop)
	}
	a.mu.Unlock()
	for _, stop := range stops {
		stop()
	}
	a.persistQueue()
	for _, id := range activeIDs {
		a.emitItemUpdate(id)
	}
	a.emitStats()
}
func (a *App) ClearAll(deleteFiles bool) error {
	items := a.GetDownloads()
	for _, item := range items {
		if err := a.RemoveDownload(item.ID, deleteFiles); err != nil {
			return err
		}
	}
	return nil
}
