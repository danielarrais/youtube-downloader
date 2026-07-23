package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kkdai/youtube/v2"
)

var convertAudioToMP3 = ConvertToMp3
var errPublishConvertedFile = errors.New("publish converted file")

func (a *App) processDownload(ctx context.Context, item *DownloadItem) {
	session := NewYouTubeSession()
	video, err := session.GetVideo(ctx, item.URL)
	if err != nil {
		if ctx.Err() != nil {
			a.cleanupInterruptedDownload(item.ID, "")
			return
		}
		a.updateError(item.ID, FormatYouTubeError(err, a.currentLanguage()))
		return
	}

	a.mu.Lock()
	if !a.isActiveItemLocked(item.ID) {
		a.mu.Unlock()
		return
	}
	item.Title = video.Title
	item.ThumbnailURL = largestVideoThumbnail(video.Thumbnails)
	a.mu.Unlock()
	a.persistQueue()
	a.emitItemUpdate(item.ID)
	if item.MediaType == MediaTypeVideo {
		a.processVideoDownload(ctx, item, session, video)
		return
	}

	musicMeta := session.ExtractMusicMetadata()
	var coverPath string
	if musicMeta != nil && musicMeta.Song != "" {
		a.mu.Lock()
		if !a.isActiveItemLocked(item.ID) {
			a.mu.Unlock()
			return
		}
		item.ThumbnailURL = musicMeta.CoverURL
		a.mu.Unlock()
		a.persistQueue()
		a.emitItemUpdate(item.ID)
		if musicMeta.CoverURL != "" {
			coverPath = filepath.Join(a.cacheDir, "cover-"+item.ID+".jpg")
			if err := DownloadCoverArt(ctx, musicMeta.CoverURL, coverPath); err != nil {
				coverPath = ""
			}
		}
	}

	a.mu.Lock()
	downloadDir := a.config.DownloadDir
	a.mu.Unlock()
	outputPath := filepath.Join(downloadDir, SanitizeFilename(video.Title)+".mp3")
	if musicMeta != nil && musicMeta.Song != "" {
		filename := musicMeta.Song
		if musicMeta.Artist != "" {
			filename = musicMeta.Artist + " - " + musicMeta.Song
		}
		outputPath = filepath.Join(downloadDir, SanitizeFilename(filename)+".mp3")
	}
	outputPath = a.resolveOutputPath(item, video.ID, outputPath)

	if fileInfo, err := os.Stat(outputPath); err == nil {
		if ctx.Err() != nil {
			a.cleanupInterruptedDownload(item.ID, "")
			return
		}
		if !a.shouldSkipExistingOutput(item, outputPath) {
			if err := os.Remove(outputPath); err != nil {
				a.updateError(item.ID, FormatOperationError("finalize", err, a.currentLanguage()))
				return
			}
		} else {
			if !a.setActiveItemStatus(item.ID, StatusSkipped) {
				return
			}
			a.mu.Lock()
			if !a.isActiveItemLocked(item.ID) {
				a.mu.Unlock()
				return
			}
			item.FilePath = outputPath
			item.FileSize = fileInfo.Size()
			a.mu.Unlock()
			a.persistQueue()
			a.emitItemUpdate(item.ID)
			return
		}
	} else if !os.IsNotExist(err) {
		if coverPath != "" {
			os.Remove(coverPath)
		}
		a.updateError(item.ID, FormatOperationError("finalize", err, a.currentLanguage()))
		return
	}

	if !a.setActiveItemStatus(item.ID, StatusDownloading) {
		if coverPath != "" {
			os.Remove(coverPath)
		}
		return
	}
	tempPath := filepath.Join(a.cacheDir, item.ID+".tmp")
	defer os.Remove(tempPath)
	if coverPath != "" {
		defer os.Remove(coverPath)
	}

	lastProgressEvent := time.Time{}
	_, err = session.DownloadAudio(ctx, video, tempPath, func(percent float64, downloaded, total int64) {
		a.mu.Lock()
		if a.isActiveItemLocked(item.ID) {
			item.Progress.Percent = percent
			item.Progress.DownloadedBytes = downloaded
			item.Progress.TotalBytes = total
		}
		a.mu.Unlock()
		now := time.Now()
		if percent >= 100 || lastProgressEvent.IsZero() || now.Sub(lastProgressEvent) >= 200*time.Millisecond {
			lastProgressEvent = now
			a.emitItemUpdate(item.ID)
		}
	})
	if err != nil {
		if ctx.Err() != nil {
			a.cleanupInterruptedDownload(item.ID, "")
			return
		}
		a.updateError(item.ID, FormatOperationError("download", err, a.currentLanguage()))
		return
	}

	if !a.setActiveItemStatus(item.ID, StatusConverting) {
		return
	}
	if err := convertAndPublish(ctx, tempPath, outputPath, item.Quality, musicMeta, coverPath); err != nil {
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

	fileInfo, err := os.Stat(outputPath)
	if err != nil {
		os.Remove(outputPath)
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

func (a *App) processVideoDownload(ctx context.Context, item *DownloadItem, session *YouTubeSession, video *youtube.Video) {
	if item.VideoFormat == nil {
		a.updateError(item.ID, FormatOperationError("download", errors.New("formato de vídeo não selecionado"), a.currentLanguage()))
		return
	}
	format := *item.VideoFormat
	if format.Extension != "mp4" && format.Extension != "webm" && format.Extension != "mkv" {
		a.updateError(item.ID, FormatOperationError("download", errors.New("formato de vídeo inválido"), a.currentLanguage()))
		return
	}
	videoStream, ok := videoFormatByItag(video, format.VideoItag)
	if !ok {
		a.updateError(item.ID, FormatOperationError("download", errors.New("o formato de vídeo não está mais disponível"), a.currentLanguage()))
		return
	}
	var audioStream *youtube.Format
	if format.AudioItag != 0 {
		var found bool
		audioStream, found = videoFormatByItag(video, format.AudioItag)
		if !found {
			a.updateError(item.ID, FormatOperationError("download", errors.New("o formato de áudio não está mais disponível"), a.currentLanguage()))
			return
		}
	}

	a.mu.Lock()
	downloadDir := a.config.DownloadDir
	a.mu.Unlock()
	name := SanitizeFilename(video.Title)
	if format.Resolution != "" {
		name += " [" + SanitizeFilename(format.Resolution) + "]"
	}
	outputPath := a.resolveOutputPath(item, video.ID, filepath.Join(downloadDir, name+"."+format.Extension))
	if fileInfo, err := os.Stat(outputPath); err == nil {
		if !a.shouldSkipExistingOutput(item, outputPath) {
			if err := os.Remove(outputPath); err != nil {
				a.updateError(item.ID, FormatOperationError("finalize", err, a.currentLanguage()))
				return
			}
		} else {
			if !a.setActiveItemStatus(item.ID, StatusSkipped) {
				return
			}
			a.mu.Lock()
			item.FilePath = outputPath
			item.FileSize = fileInfo.Size()
			a.mu.Unlock()
			a.persistQueue()
			a.emitItemUpdate(item.ID)
			return
		}
	} else if !os.IsNotExist(err) {
		a.updateError(item.ID, FormatOperationError("finalize", err, a.currentLanguage()))
		return
	}

	if !a.setActiveItemStatus(item.ID, StatusDownloading) {
		return
	}
	videoPath := filepath.Join(a.cacheDir, item.ID+".video.tmp")
	audioPath := filepath.Join(a.cacheDir, item.ID+".audio.tmp")
	defer os.Remove(videoPath)
	defer os.Remove(audioPath)

	updateProgress := func(percent float64, downloaded, total int64) {
		a.mu.Lock()
		if a.isActiveItemLocked(item.ID) {
			item.Progress.Percent = percent
			item.Progress.DownloadedBytes = downloaded
			item.Progress.TotalBytes = total
		}
		a.mu.Unlock()
		a.emitItemUpdate(item.ID)
	}
	_, err := session.DownloadFormat(ctx, video, videoStream, videoPath, func(percent float64, downloaded, total int64) {
		if audioStream != nil {
			updateProgress(percent/2, downloaded, total+audioStream.ContentLength)
			return
		}
		updateProgress(percent, downloaded, total)
	})
	if err != nil {
		a.handleVideoDownloadError(ctx, item, err)
		return
	}
	if audioStream != nil {
		_, err = session.DownloadFormat(ctx, video, audioStream, audioPath, func(percent float64, downloaded, total int64) {
			updateProgress(50+percent/2, videoStream.ContentLength+downloaded, videoStream.ContentLength+total)
		})
		if err != nil {
			a.handleVideoDownloadError(ctx, item, err)
			return
		}
	}

	if !a.setActiveItemStatus(item.ID, StatusConverting) {
		return
	}
	partPath := outputPath + ".part"
	defer os.Remove(partPath)
	if audioStream == nil {
		err = copyFile(videoPath, partPath)
	} else {
		err = ConvertToVideo(ctx, videoPath, audioPath, partPath, format.Container)
	}
	if err == nil {
		err = publishConvertedFile(partPath, outputPath)
	}
	if err != nil {
		if ctx.Err() != nil {
			a.cleanupInterruptedDownload(item.ID, "")
			return
		}
		a.updateError(item.ID, FormatOperationError("conversion", err, a.currentLanguage()))
		return
	}

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

func (a *App) handleVideoDownloadError(ctx context.Context, item *DownloadItem, err error) {
	if ctx.Err() != nil {
		a.cleanupInterruptedDownload(item.ID, "")
		return
	}
	a.updateError(item.ID, FormatOperationError("download", err, a.currentLanguage()))
}

func copyFile(sourcePath, destinationPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()
	destination, err := os.OpenFile(destinationPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(destination, source); err != nil {
		destination.Close()
		return err
	}
	if err := destination.Sync(); err != nil {
		destination.Close()
		return err
	}
	return destination.Close()
}

func convertAndPublish(ctx context.Context, inputPath, outputPath, quality string, metadata *MusicMetadata, coverPath string) error {
	partPath := outputPath + ".part"
	defer os.Remove(partPath)
	if err := convertAudioToMP3(ctx, inputPath, partPath, quality, metadata, coverPath); err != nil {
		return err
	}
	if err := publishConvertedFile(partPath, outputPath); err != nil {
		return fmt.Errorf("%w: %v", errPublishConvertedFile, err)
	}
	return nil
}

func cleanupPartialFiles(downloadDir string) error {
	for _, extension := range []string{"mp3", "mp4", "webm", "mkv"} {
		matches, err := filepath.Glob(filepath.Join(downloadDir, "*."+extension+".part"))
		if err != nil {
			return err
		}
		for _, path := range matches {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}
	return nil
}

func publishConvertedFile(partPath, outputPath string) error {
	if _, err := os.Stat(outputPath); err == nil {
		return os.ErrExist
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.Rename(partPath, outputPath)
}

func (a *App) resolveOutputPath(item *DownloadItem, videoID, basePath string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, queuedItem := range a.items {
		if queuedItem.ID != item.ID && queuedItem.URL != item.URL && queuedItem.FilePath == basePath {
			extension := filepath.Ext(basePath)
			return strings.TrimSuffix(basePath, extension) + " [" + videoID + "]" + extension
		}
	}
	return basePath
}

func (a *App) shouldSkipExistingOutput(item *DownloadItem, outputPath string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, queuedItem := range a.items {
		if queuedItem.ID == item.ID || queuedItem.FilePath != outputPath {
			continue
		}
		if queuedItem.URL != item.URL {
			return true
		}
		if item.MediaType != queuedItem.MediaType {
			return false
		}
		if item.MediaType == MediaTypeVideo {
			return videoFormatsMatch(item.VideoFormat, queuedItem.VideoFormat)
		}
		return item.Quality == queuedItem.Quality
	}

	return true
}

func videoFormatsMatch(left, right *VideoFormat) bool {
	if left == nil || right == nil {
		return left == right
	}
	return left.VideoItag == right.VideoItag &&
		left.AudioItag == right.AudioItag &&
		left.Extension == right.Extension &&
		left.Resolution == right.Resolution
}
