package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// CheckAndDownloadFFmpeg garante que o FFmpeg esteja disponível na pasta local do app
func CheckAndDownloadFFmpeg() (string, error) {
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}

	if executablePath, err := os.Executable(); err == nil {
		bundledPath := filepath.Join(filepath.Dir(executablePath), "ffmpeg"+ext)
		if _, err := os.Stat(bundledPath); err == nil {
			return bundledPath, nil
		}
	}

	home, _ := os.UserHomeDir()
	// Pasta de binários privada do App
	binDir := filepath.Join(home, ".yt-mp3-downloader-bin")
	os.MkdirAll(binDir, 0755)

	ffmpegPath := filepath.Join(binDir, "ffmpeg"+ext)

	// Se já existe, retorna o caminho absoluto
	if _, err := os.Stat(ffmpegPath); err == nil {
		return ffmpegPath, nil
	}

	fmt.Printf(">>> FFmpeg não encontrado em %s. Usando versão do sistema...\n", ffmpegPath)

	if systemPath, err := exec.LookPath("ffmpeg" + ext); err == nil {
		return systemPath, nil
	}

	return "", fmt.Errorf("ffmpeg não encontrado")
}
