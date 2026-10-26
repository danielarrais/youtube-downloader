package main

type DownloadStatus string

type MediaType string

const (
	MediaTypeAudio MediaType = "audio"
	MediaTypeVideo MediaType = "video"
)

const (
	StatusPending     DownloadStatus = "pending"
	StatusFetching    DownloadStatus = "fetching_info"
	StatusDownloading DownloadStatus = "downloading"
	StatusConverting  DownloadStatus = "converting"
	StatusCompleted   DownloadStatus = "completed"
	StatusFailed      DownloadStatus = "failed"
	StatusCancelled   DownloadStatus = "cancelled"
	StatusSkipped     DownloadStatus = "skipped"
)

func isRunningStatus(status DownloadStatus) bool {
	switch status {
	case StatusFetching, StatusDownloading, StatusConverting:
		return true
	default:
		return false
	}
}

func isCancellableStatus(status DownloadStatus) bool {
	return status == StatusPending || isRunningStatus(status)
}

func isCompletedStatus(status DownloadStatus) bool {
	return status == StatusCompleted || status == StatusSkipped
}

func canUpdateActiveStatus(status DownloadStatus) bool {
	return status != StatusCancelled && status != StatusPending
}

type DownloadProgress struct {
	Percent         float64 `json:"percent"`
	DownloadedBytes int64   `json:"downloaded_bytes"`
	TotalBytes      int64   `json:"total_bytes"`
	Speed           string  `json:"speed"`
	ETA             string  `json:"eta"`
}

type DownloadItem struct {
	ID           string           `json:"id"`
	URL          string           `json:"url"`
	Title        string           `json:"title"`
	Quality      string           `json:"quality"`
	MediaType    MediaType        `json:"media_type,omitempty"`
	AudioFormat  *AudioFormat     `json:"audio_format,omitempty"`
	VideoFormat  *VideoFormat     `json:"video_format,omitempty"`
	Status       DownloadStatus   `json:"status"`
	Progress     DownloadProgress `json:"progress"`
	Error        string           `json:"error,omitempty"`
	FilePath     string           `json:"file_path,omitempty"`
	FileSize     int64            `json:"file_size,omitempty"`
	ThumbnailURL string           `json:"thumbnail_url,omitempty"`
	CreatedAt    string           `json:"created_at"`
	StartedAt    string           `json:"started_at,omitempty"`
	CompletedAt  string           `json:"completed_at,omitempty"`
}

type AudioFormat struct {
	FormatID   string `json:"format_id"`
	Container  string `json:"container"`
	Extension  string `json:"extension"`
	AudioCodec string `json:"audio_codec,omitempty"`
	Bitrate    int    `json:"bitrate,omitempty"`
	SampleRate int    `json:"sample_rate,omitempty"`
	Protocol   string `json:"protocol,omitempty"`
	Label      string `json:"label"`
}

// VideoFormat identifies a video stream and the audio stream needed to play it.
// AudioItag is zero when the selected video stream already contains audio.
type VideoFormat struct {
	VideoItag  int    `json:"video_itag"`
	AudioItag  int    `json:"audio_itag,omitempty"`
	FormatID   string `json:"format_id,omitempty"`
	Container  string `json:"container"`
	Extension  string `json:"extension"`
	Resolution string `json:"resolution"`
	Width      int    `json:"width,omitempty"`
	Height     int    `json:"height,omitempty"`
	FPS        int    `json:"fps,omitempty"`
	VideoCodec string `json:"video_codec,omitempty"`
	AudioCodec string `json:"audio_codec,omitempty"`
	Size       int64  `json:"size,omitempty"`
	Label      string `json:"label"`
}

type VideoInfo struct {
	Title        string        `json:"title"`
	ThumbnailURL string        `json:"thumbnail_url,omitempty"`
	Formats      []VideoFormat `json:"formats"`
	AudioFormats []AudioFormat `json:"audio_formats,omitempty"`
}

type AudioDownloadRequest struct {
	URL    string      `json:"url"`
	Format AudioFormat `json:"format"`
	Error  string      `json:"error,omitempty"`
}

type VideoDownloadRequest struct {
	URL    string      `json:"url"`
	Format VideoFormat `json:"format"`
	Error  string      `json:"error,omitempty"`
}

type QueueStats struct {
	Total       int  `json:"total"`
	Pending     int  `json:"pending"`
	Downloading int  `json:"downloading"`
	Completed   int  `json:"completed"`
	Failed      int  `json:"failed"`
	Paused      bool `json:"paused"`
}

type PlaylistInfo struct {
	ID     string          `json:"id"`
	Title  string          `json:"title"`
	Author string          `json:"author"`
	Videos []PlaylistVideo `json:"videos"`
}

type PlaylistVideo struct {
	ID                string `json:"id"`
	URL               string `json:"url"`
	Title             string `json:"title"`
	Author            string `json:"author"`
	DurationSeconds   int    `json:"duration_seconds"`
	ThumbnailURL      string `json:"thumbnail_url"`
	Available         bool   `json:"available"`
	UnavailableReason string `json:"unavailable_reason,omitempty"`
	Index             int    `json:"index"`
}
