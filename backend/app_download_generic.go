package main

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxGenericTitleLength = 100

var genericBestAudioFormat = AudioFormat{
	FormatID: "bestaudio/best",
	Label:    "Melhor áudio disponível",
}

var genericBestVideoFormat = VideoFormat{
	VideoItag: 1,
	Container: "mp4",
	Extension: "mp4",
	Label:     "Melhor qualidade (MP4)",
}

// genericInfoArgs limits posts with several videos (X, Instagram carousels)
// to the first one so a single queue item always produces a single file.
func genericInfoArgs() []string {
	return []string{"--playlist-items", "1"}
}

func genericDownloadArgs() []string {
	args := genericInfoArgs()
	if ffmpegPath, err := CheckAndDownloadFFmpeg(); err == nil {
		args = append(args, "--ffmpeg-location", ffmpegPath)
	}
	return args
}

func fetchGenericInfo(ctx context.Context, url string) (ytDLPInfo, error) {
	info, err := fetchYTDLPInfo(ctx, url, genericInfoArgs()...)
	if err != nil {
		return ytDLPInfo{}, err
	}
	if len(info.AudioFormats) == 0 {
		info.AudioFormats = []AudioFormat{genericBestAudioFormat}
	}
	if len(info.VideoFormats) == 0 {
		info.VideoFormats = []VideoFormat{genericBestVideoFormat}
	}
	info.Title = genericTitle(info.Title, url, info.ID)
	return info, nil
}

func getGenericVideoInfo(ctx context.Context, url string) (VideoInfo, error) {
	info, err := fetchGenericInfo(ctx, url)
	if err != nil {
		return VideoInfo{}, err
	}
	return VideoInfo{
		Title:        info.Title,
		ThumbnailURL: info.ThumbnailURL,
		Formats:      info.VideoFormats,
		AudioFormats: info.AudioFormats,
	}, nil
}

// genericTitle collapses captions into a single line short enough to be used
// as a filename.
func genericTitle(title, url, id string) string {
	title = strings.Join(strings.Fields(title), " ")
	if runes := []rune(title); len(runes) > maxGenericTitleLength {
		title = strings.TrimSpace(string(runes[:maxGenericTitleLength]))
	}
	if title != "" {
		return title
	}
	if id == "" {
		return string(detectPlatform(url))
	}
	return string(detectPlatform(url)) + " " + id
}

func formatGenericError(err error, language string) string {
	return localizedPrefix(language, "Erro ao consultar o link: ", "Error while querying the link: ") + err.Error()
}

func (a *App) processGenericDownload(ctx context.Context, item *DownloadItem) {
	info, err := fetchGenericInfo(ctx, item.URL)
	if err != nil {
		if ctx.Err() != nil {
			a.cleanupInterruptedDownload(item.ID, "")
			return
		}
		a.updateError(item.ID, formatGenericError(err, a.currentLanguage()))
		return
	}

	a.mu.Lock()
	if !a.isActiveItemLocked(item.ID) {
		a.mu.Unlock()
		return
	}
	item.Title = info.Title
	item.ThumbnailURL = info.ThumbnailURL
	a.mu.Unlock()
	a.persistQueue()
	a.emitItemUpdate(item.ID)

	if item.MediaType == MediaTypeVideo {
		a.processGenericVideoDownload(ctx, item, info)
		return
	}
	a.processGenericAudioDownload(ctx, item, info)
}

func (a *App) processGenericAudioDownload(ctx context.Context, item *DownloadItem, info ytDLPInfo) {
	a.mu.Lock()
	downloadDir := a.config.DownloadDir
	audioBitrateTarget := a.config.AudioBitrateTarget
	a.mu.Unlock()

	if item.AudioFormat == nil || item.AudioFormat.FormatID == "" {
		if format, ok := preferredAudioFormat(info.AudioFormats, audioBitrateTarget); ok {
			a.mu.Lock()
			if a.isActiveItemLocked(item.ID) {
				item.AudioFormat = &format
			}
			a.mu.Unlock()
			a.persistQueue()
			a.emitItemUpdate(item.ID)
		}
	}

	outputPath := a.resolveOutputPath(item, info.ID, filepath.Join(downloadDir, SanitizeFilename(info.Title)+".mp3"))
	if !a.prepareGenericOutput(item, outputPath) {
		return
	}
	if !a.setActiveItemStatus(item.ID, StatusDownloading) {
		return
	}

	selector := "bestaudio/best"
	if item.AudioFormat != nil && item.AudioFormat.FormatID != "" {
		selector = item.AudioFormat.FormatID + "/bestaudio/best"
	}
	sourcePath, err := runYTDLPDownload(ctx, ytDLPDownloadRequest{
		URL:            item.URL,
		FormatSelector: selector,
		OutputTemplate: ytDLPOutputTemplate(a.cacheDir, item.ID+"-audio"),
		ExtraArgs:      genericDownloadArgs(),
	}, a.genericProgressHandler(item))
	if err != nil {
		a.handleVideoDownloadError(ctx, item, err)
		return
	}
	defer os.Remove(sourcePath)

	if !a.setActiveItemStatus(item.ID, StatusConverting) {
		return
	}
	if err := convertAndPublish(ctx, sourcePath, outputPath, item.Quality, nil, "", func(percent float64) {
		a.mu.Lock()
		if a.isActiveItemLocked(item.ID) {
			item.Progress.Percent = percent
			item.Progress.Speed = "---"
			item.Progress.ETA = "---"
		}
		a.mu.Unlock()
		a.emitItemUpdate(item.ID)
	}); err != nil {
		if ctx.Err() != nil {
			a.cleanupInterruptedDownload(item.ID, "")
			return
		}
		operation := "conversion"
		if errors.Is(err, errPublishConvertedFile) {
			operation = "finalize"
		}
		a.updateError(item.ID, FormatOperationError(operation, err, a.currentLanguage()))
		return
	}
	a.completeGenericDownload(item, outputPath)
}

func (a *App) processGenericVideoDownload(ctx context.Context, item *DownloadItem, info ytDLPInfo) {
	if item.VideoFormat == nil || item.VideoFormat.Extension == "" {
		a.mu.Lock()
		videoContainer := a.config.VideoContainer
		videoQuality := a.config.VideoQuality
		a.mu.Unlock()
		format, ok := preferredVideoFormat(info.VideoFormats, videoContainer, videoQuality)
		if !ok {
			format = genericBestVideoFormat
		}
		a.mu.Lock()
		if !a.isActiveItemLocked(item.ID) {
			a.mu.Unlock()
			return
		}
		item.VideoFormat = &format
		a.mu.Unlock()
		a.persistQueue()
		a.emitItemUpdate(item.ID)
	}
	format := *item.VideoFormat
	if format.Extension != "mp4" && format.Extension != "webm" && format.Extension != "mkv" {
		a.updateError(item.ID, FormatOperationError("download", errors.New("formato de vídeo inválido"), a.currentLanguage()))
		return
	}

	a.mu.Lock()
	downloadDir := a.config.DownloadDir
	a.mu.Unlock()
	name := SanitizeFilename(info.Title)
	if format.Resolution != "" {
		name += " [" + SanitizeFilename(format.Resolution) + "]"
	}
	outputPath := a.resolveOutputPath(item, info.ID, filepath.Join(downloadDir, name+"."+format.Extension))
	if !a.prepareGenericOutput(item, outputPath) {
		return
	}
	if !a.setActiveItemStatus(item.ID, StatusDownloading) {
		return
	}
	log.Printf(
		"[generic-video] selected representative_format_id=%s resolution=%s dimensions=%dx%d container=%s url=%s",
		format.FormatID,
		format.Resolution,
		format.Width,
		format.Height,
		format.Extension,
		item.URL,
	)

	sourcePath, err := runYTDLPDownload(ctx, ytDLPDownloadRequest{
		URL:               item.URL,
		FormatSelector:    ytDLPGenericVideoSelector(format),
		OutputTemplate:    ytDLPOutputTemplate(a.cacheDir, item.ID+"-video"),
		MergeOutputFormat: format.Extension,
		ExtraArgs:         genericDownloadArgs(),
	}, a.genericProgressHandler(item))
	if err != nil {
		a.handleVideoDownloadError(ctx, item, err)
		return
	}
	defer os.Remove(sourcePath)

	partPath := outputPath + ".part"
	defer os.Remove(partPath)
	if err := copyFile(sourcePath, partPath); err != nil {
		a.updateError(item.ID, FormatOperationError("finalize", err, a.currentLanguage()))
		return
	}
	if err := publishConvertedFile(partPath, outputPath); err != nil {
		a.updateError(item.ID, FormatOperationError("finalize", err, a.currentLanguage()))
		return
	}
	a.completeGenericDownload(item, outputPath)
}

// prepareGenericOutput handles an already existing output file and reports
// whether the download should continue.
func (a *App) prepareGenericOutput(item *DownloadItem, outputPath string) bool {
	fileInfo, err := os.Stat(outputPath)
	if err != nil {
		if os.IsNotExist(err) {
			return true
		}
		a.updateError(item.ID, FormatOperationError("finalize", err, a.currentLanguage()))
		return false
	}
	if !a.shouldSkipExistingOutput(item, outputPath) {
		if err := os.Remove(outputPath); err != nil {
			a.updateError(item.ID, FormatOperationError("finalize", err, a.currentLanguage()))
			return false
		}
		return true
	}
	if !a.setActiveItemStatus(item.ID, StatusSkipped) {
		return false
	}
	a.mu.Lock()
	if a.isActiveItemLocked(item.ID) {
		item.FilePath = outputPath
		item.FileSize = fileInfo.Size()
	}
	a.mu.Unlock()
	a.persistQueue()
	a.emitItemUpdate(item.ID)
	return false
}

func (a *App) genericProgressHandler(item *DownloadItem) func(mediaDownloadProgress) {
	lastProgressEvent := time.Time{}
	return func(progress mediaDownloadProgress) {
		a.mu.Lock()
		if a.isActiveItemLocked(item.ID) {
			item.Progress.Percent = progress.Percent
			item.Progress.DownloadedBytes = progress.DownloadedBytes
			item.Progress.TotalBytes = progress.TotalBytes
			if progress.Speed != "" {
				item.Progress.Speed = progress.Speed
			}
			if progress.ETA != "" {
				item.Progress.ETA = progress.ETA
			}
		}
		a.mu.Unlock()
		now := time.Now()
		if progress.Percent >= 100 || lastProgressEvent.IsZero() || now.Sub(lastProgressEvent) >= 200*time.Millisecond {
			lastProgressEvent = now
			a.emitItemUpdate(item.ID)
		}
	}
}

func (a *App) completeGenericDownload(item *DownloadItem, outputPath string) {
	fileInfo, err := os.Stat(outputPath)
	if err != nil {
		a.updateError(item.ID, FormatOperationError("finalize", err, a.currentLanguage()))
		return
	}
	a.mu.Lock()
	if !a.isActiveItemLocked(item.ID) {
		a.mu.Unlock()
		os.Remove(outputPath)
		return
	}
	item.Status = StatusCompleted
	item.FilePath = outputPath
	item.FileSize = fileInfo.Size()
	item.CompletedAt = time.Now().Format(time.RFC3339)
	item.Progress.Percent = 100
	a.mu.Unlock()
	a.persistQueue()
	a.emitItemUpdate(item.ID)
	a.emitStats()
}
