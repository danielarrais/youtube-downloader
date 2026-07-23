package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	defaultVideoContainer = "mp4"
	defaultVideoQuality   = "1080p"
	defaultTheme          = "dark"
	FileDeletionDelete    = "delete"
	FileDeletionAsk       = "ask"
	FileDeletionKeep      = "keep"
)

type Config struct {
	DownloadDir    string `json:"download_dir"`
	Quality        string `json:"quality"`
	VideoContainer string `json:"video_container"`
	VideoQuality   string `json:"video_quality"`
	FileDeletion   string `json:"file_deletion"`
	Language       string `json:"language"`
	Theme          string `json:"theme"`
}

func defaultConfig(home string) Config {
	return Config{
		DownloadDir:    filepath.Join(home, "Downloads", "YouTube-MP3"),
		Quality:        "192k",
		VideoContainer: defaultVideoContainer,
		VideoQuality:   defaultVideoQuality,
		FileDeletion:   FileDeletionAsk,
		Language:       "pt-BR",
		Theme:          defaultTheme,
	}
}

func normalizeConfig(config, defaults Config) Config {
	if config.DownloadDir == "" {
		config.DownloadDir = defaults.DownloadDir
	}
	switch config.Quality {
	case "128k", "192k", "320k":
	default:
		config.Quality = defaults.Quality
	}
	switch config.VideoContainer {
	case "mp4", "webm", "mkv":
	default:
		config.VideoContainer = defaults.VideoContainer
	}
	if config.VideoContainer == "" {
		config.VideoContainer = defaultVideoContainer
	}
	switch config.VideoQuality {
	case "144p", "240p", "360p", "480p", "720p", "1080p", "1440p", "2160p":
	default:
		config.VideoQuality = defaults.VideoQuality
	}
	if config.VideoQuality == "" {
		config.VideoQuality = defaultVideoQuality
	}
	switch config.FileDeletion {
	case FileDeletionDelete, FileDeletionAsk, FileDeletionKeep:
	default:
		config.FileDeletion = defaults.FileDeletion
	}
	if config.FileDeletion == "" {
		config.FileDeletion = FileDeletionAsk
	}
	if config.Language != "en-US" && config.Language != "pt-BR" {
		config.Language = defaults.Language
	}
	switch config.Theme {
	case "dark", "light":
	default:
		config.Theme = defaults.Theme
	}
	if config.Theme == "" {
		config.Theme = defaultTheme
	}
	return config
}

func loadConfigFile(path string, defaults Config) (Config, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return defaults, nil
	}
	if err != nil {
		return Config{}, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("configuração inválida: %w", err)
	}
	return normalizeConfig(config, defaults), nil
}

func saveConfigFile(path string, config Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
