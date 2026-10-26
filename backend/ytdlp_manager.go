package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	ytDLPReleaseBaseURL = "https://github.com/yt-dlp/yt-dlp/releases/latest/download"
	maxYTDLPBinarySize  = 100 << 20
)

var (
	ytDLPMu             sync.Mutex
	cachedYTDLPPath     string
	ytDLPExecutablePath = os.Executable
	ytDLPLookPath       = exec.LookPath
	ytDLPValidateBinary = validateYTDLP
)

type ytDLPArtifact struct {
	filename   string
	binaryName string
}

func CheckAndDownloadYTDLP() (string, error) {
	ytDLPMu.Lock()
	defer ytDLPMu.Unlock()

	if cachedYTDLPPath != "" {
		return cachedYTDLPPath, nil
	}

	artifact, artifactErr := ytDLPArtifactFor(runtime.GOOS, runtime.GOARCH)
	binaryName := ytDLPBinaryName(runtime.GOOS)

	if executablePath, err := ytDLPExecutablePath(); err == nil {
		if resolvedPath, err := filepath.EvalSymlinks(executablePath); err == nil {
			executablePath = resolvedPath
		}
		bundledPath := filepath.Join(filepath.Dir(executablePath), binaryName)
		if ytDLPValidateBinary(bundledPath) == nil {
			return cacheYTDLPPath(bundledPath), nil
		}
	}

	binDir, err := appBinaryCacheDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", fmt.Errorf("não foi possível criar a pasta de binários: %w", err)
	}

	ytDLPPath := filepath.Join(binDir, binaryName)
	if err := ytDLPValidateBinary(ytDLPPath); err == nil {
		return cacheYTDLPPath(ytDLPPath), nil
	}

	if systemPath, err := ytDLPLookPath(binaryName); err == nil {
		if ytDLPValidateBinary(systemPath) == nil {
			return cacheYTDLPPath(systemPath), nil
		}
	}

	if artifactErr != nil {
		return "", artifactErr
	}

	fmt.Printf(">>> yt-dlp não encontrado. Baixando %s...\n", artifact.filename)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	if err := downloadAndInstallYTDLP(ctx, newYTDLPHTTPClient(), ytDLPReleaseBaseURL, artifact, ytDLPPath); err != nil {
		return "", fmt.Errorf("não foi possível baixar o yt-dlp: %w", err)
	}

	fmt.Printf(">>> yt-dlp instalado em %s\n", ytDLPPath)
	return cacheYTDLPPath(ytDLPPath), nil
}

func cacheYTDLPPath(path string) string {
	cachedYTDLPPath = path
	if version, err := ytDLPVersion(path); err == nil {
		log.Printf("[yt-dlp] using executable path=%s version=%s", path, version)
	}
	return path
}

func ytDLPBinaryName(goos string) string {
	if goos == "windows" {
		return "yt-dlp.exe"
	}
	return "yt-dlp"
}

func ytDLPArtifactFor(goos, goarch string) (ytDLPArtifact, error) {
	switch {
	case goos == "linux" && goarch == "amd64":
		return ytDLPArtifact{filename: "yt-dlp_linux", binaryName: "yt-dlp"}, nil
	case goos == "linux" && goarch == "arm64":
		return ytDLPArtifact{filename: "yt-dlp_linux_aarch64", binaryName: "yt-dlp"}, nil
	case goos == "windows" && goarch == "amd64":
		return ytDLPArtifact{filename: "yt-dlp.exe", binaryName: "yt-dlp.exe"}, nil
	case goos == "windows" && goarch == "arm64":
		return ytDLPArtifact{filename: "yt-dlp_arm64.exe", binaryName: "yt-dlp.exe"}, nil
	default:
		return ytDLPArtifact{}, fmt.Errorf(
			"download automático do yt-dlp não suportado em %s/%s; instale o yt-dlp no sistema",
			goos,
			goarch,
		)
	}
}

func newYTDLPHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
			TLSHandshakeTimeout:   30 * time.Second,
			ResponseHeaderTimeout: 60 * time.Second,
			IdleConnTimeout:       90 * time.Second,
		},
	}
}

func downloadAndInstallYTDLP(ctx context.Context, client *http.Client, baseURL string, artifact ytDLPArtifact, targetPath string) error {
	expectedChecksum, err := fetchYTDLPChecksum(ctx, client, baseURL, artifact.filename)
	if err != nil {
		return err
	}

	binaryFile, err := os.CreateTemp(filepath.Dir(targetPath), "yt-dlp-binary-*")
	if err != nil {
		return fmt.Errorf("não foi possível criar o binário temporário: %w", err)
	}
	binaryPath := binaryFile.Name()
	defer os.Remove(binaryPath)

	checksum := sha256.New()
	limitedDestination := io.MultiWriter(binaryFile, checksum)
	if err := downloadURLWithLimit(ctx, client, baseURL+"/"+artifact.filename, limitedDestination, maxYTDLPBinarySize); err != nil {
		binaryFile.Close()
		return err
	}
	if err := binaryFile.Sync(); err != nil {
		binaryFile.Close()
		return fmt.Errorf("não foi possível sincronizar o binário: %w", err)
	}
	if err := binaryFile.Close(); err != nil {
		return fmt.Errorf("não foi possível fechar o binário: %w", err)
	}

	actualChecksum := hex.EncodeToString(checksum.Sum(nil))
	if !strings.EqualFold(actualChecksum, expectedChecksum) {
		return fmt.Errorf("checksum inválido para %s", artifact.filename)
	}
	if err := os.Chmod(binaryPath, 0755); err != nil {
		return fmt.Errorf("não foi possível tornar o yt-dlp executável: %w", err)
	}
	if err := ytDLPValidateBinary(binaryPath); err != nil {
		return fmt.Errorf("o yt-dlp baixado é inválido: %w", err)
	}

	if err := os.Remove(targetPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("não foi possível substituir o yt-dlp existente: %w", err)
	}
	if err := os.Rename(binaryPath, targetPath); err != nil {
		return fmt.Errorf("não foi possível instalar o yt-dlp: %w", err)
	}
	return nil
}

func fetchYTDLPChecksum(ctx context.Context, client *http.Client, baseURL string, filename string) (string, error) {
	var checksums bytes.Buffer
	if err := downloadURLWithLimit(ctx, client, baseURL+"/SHA2-256SUMS", &checksums, 1<<20); err != nil {
		return "", fmt.Errorf("não foi possível baixar os checksums: %w", err)
	}
	checksum, err := parseYTDLPChecksum(checksums.String(), filename)
	if err != nil {
		return "", err
	}
	return checksum, nil
}

func parseYTDLPChecksum(contents, filename string) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(contents))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		entryName := strings.TrimPrefix(fields[len(fields)-1], "*")
		if entryName == filename {
			if len(fields[0]) != sha256.Size*2 {
				return "", fmt.Errorf("checksum inválido para %s", filename)
			}
			if _, err := hex.DecodeString(fields[0]); err != nil {
				return "", fmt.Errorf("checksum inválido para %s", filename)
			}
			return strings.ToLower(fields[0]), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("não foi possível ler os checksums: %w", err)
	}
	return "", fmt.Errorf("checksum não encontrado para %s", filename)
}

func downloadURLWithLimit(ctx context.Context, client *http.Client, url string, destination io.Writer, maxBytes int64) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("não foi possível criar a requisição: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("falha ao baixar %s: %w", url, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("falha ao baixar %s: status HTTP %d", url, response.StatusCode)
	}
	reader := io.LimitReader(response.Body, maxBytes+1)
	written, err := io.Copy(destination, reader)
	if err != nil {
		return fmt.Errorf("falha ao salvar %s: %w", url, err)
	}
	if written > maxBytes {
		return fmt.Errorf("download de %s excedeu o limite de tamanho", url)
	}
	return nil
}

func validateYTDLP(path string) error {
	_, err := ytDLPVersion(path)
	return err
}

func ytDLPVersion(path string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	version := strings.TrimSpace(string(output))
	if version == "" {
		return "", errors.New("yt-dlp não retornou versão")
	}
	return version, nil
}
