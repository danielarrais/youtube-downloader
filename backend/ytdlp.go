package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var errYTDLPUnavailable = errors.New("yt-dlp is not available")

var ytDLPPercentPattern = regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)%`)

type mediaDownloadProgress struct {
	Percent         float64
	DownloadedBytes int64
	TotalBytes      int64
	Speed           string
	ETA             string
}

type ytDLPDownloadRequest struct {
	URL               string
	FormatSelector    string
	OutputTemplate    string
	MergeOutputFormat string
	ExtraArgs         []string
}

type ytDLPInfo struct {
	ID           string
	Title        string
	ThumbnailURL string
	AudioFormats []AudioFormat
	VideoFormats []VideoFormat
}

func downloadAudioWithYTDLP(ctx context.Context, url string, selected *AudioFormat, outputTemplate string, onProgress func(mediaDownloadProgress)) (string, error) {
	selector := "bestaudio/best"
	if selected != nil && selected.FormatID != "" {
		selector = selected.FormatID + "/bestaudio/best"
	}
	return runYTDLPDownload(ctx, ytDLPDownloadRequest{
		URL:            url,
		FormatSelector: selector,
		OutputTemplate: outputTemplate,
	}, onProgress)
}

func downloadVideoWithYTDLP(ctx context.Context, url string, format VideoFormat, outputTemplate string, onProgress func(mediaDownloadProgress)) (string, error) {
	return runYTDLPDownload(ctx, ytDLPDownloadRequest{
		URL:               url,
		FormatSelector:    ytDLPVideoSelector(format),
		OutputTemplate:    outputTemplate,
		MergeOutputFormat: format.Extension,
	}, onProgress)
}

func runYTDLPDownload(ctx context.Context, request ytDLPDownloadRequest, onProgress func(mediaDownloadProgress)) (string, error) {
	ytDLPPath, err := CheckAndDownloadYTDLP()
	if err != nil {
		return "", fmt.Errorf("%w: %v", errYTDLPUnavailable, err)
	}

	var lastErr error
	for _, strategy := range ytDLPStrategies() {
		strategyName := ytDLPStrategyName(strategy)
		args := append(buildYTDLPBaseArgs(), strategy...)
		args = append(args,
			"--progress",
			"--newline",
			"--progress-template", "download:%(progress._percent_str)s|%(progress.downloaded_bytes)s|%(progress.total_bytes)s|%(progress.total_bytes_estimate)s|%(progress.speed)s|%(progress.eta)s",
			"--print", "after_move:filepath",
			"--no-part",
			"--force-overwrites",
			"-f", request.FormatSelector,
			"-o", request.OutputTemplate,
		)
		if request.MergeOutputFormat != "" {
			args = append(args, "--merge-output-format", request.MergeOutputFormat)
		}
		args = append(args, request.ExtraArgs...)
		args = append(args, request.URL)

		log.Printf("[yt-dlp] starting download strategy=%s format=%q url=%s", strategyName, request.FormatSelector, request.URL)
		path, err := execYTDLP(ctx, ytDLPPath, args, onProgress)
		if err == nil {
			log.Printf("[yt-dlp] download completed strategy=%s path=%s url=%s", strategyName, path, request.URL)
			return path, nil
		}
		log.Printf("[yt-dlp] download failed strategy=%s error=%v url=%s", strategyName, err, request.URL)
		lastErr = err
		if !shouldRetryYTDLPStrategy(err) {
			break
		}
	}
	if lastErr == nil {
		lastErr = errors.New("yt-dlp download failed")
	}
	return "", lastErr
}

func buildYTDLPBaseArgs() []string {
	args := []string{"--no-update", "--remote-components", "ejs:github"}
	if nodePath, err := exec.LookPath("node"); err == nil {
		args = append(args, "--js-runtimes", "node:"+nodePath)
	}
	return args
}

func execYTDLP(ctx context.Context, ytDLPPath string, args []string, onProgress func(mediaDownloadProgress)) (string, error) {
	cmd := exec.CommandContext(ctx, ytDLPPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	var actualPath string
	var stderrBuffer bytes.Buffer
	var stderrMu sync.Mutex

	if err := cmd.Start(); err != nil {
		return "", err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			log.Printf("[yt-dlp] stdout: %s", line)
			if onProgress != nil {
				if progress, ok := parseYTDLPProgress(line); ok {
					onProgress(progress)
					continue
				}
			}
			actualPath = line
		}
	}()
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			log.Printf("[yt-dlp] stderr: %s", line)
			stderrMu.Lock()
			stderrBuffer.WriteString(line)
			stderrBuffer.WriteByte('\n')
			stderrMu.Unlock()
			if onProgress != nil {
				if progress, ok := parseYTDLPProgress(line); ok {
					onProgress(progress)
				}
			}
		}
	}()

	err = cmd.Wait()
	wg.Wait()
	if err != nil {
		stderrMu.Lock()
		detail := strings.TrimSpace(stderrBuffer.String())
		stderrMu.Unlock()
		if detail == "" {
			detail = err.Error()
		}
		return "", fmt.Errorf("yt-dlp error: %s", detail)
	}
	if actualPath == "" {
		actualPath = inferOutputPathFromTemplate(args)
	}
	if actualPath == "" {
		return "", errors.New("yt-dlp did not report an output path")
	}
	if _, err := os.Stat(actualPath); err != nil {
		return "", err
	}
	return actualPath, nil
}

func inferOutputPathFromTemplate(args []string) string {
	for index := 0; index < len(args)-1; index++ {
		if args[index] == "-o" {
			return strings.ReplaceAll(args[index+1], "%(ext)s", "")
		}
	}
	return ""
}

func parseYTDLPProgress(line string) (mediaDownloadProgress, bool) {
	if !strings.HasPrefix(line, "download:") {
		if strings.Contains(line, "|") {
			return parseYTDLPProgressTemplateFields(line)
		}
		return parseYTDLPDownloadLine(line)
	}
	return parseYTDLPProgressTemplateFields(strings.TrimPrefix(line, "download:"))
}

func parseYTDLPProgressTemplateFields(value string) (mediaDownloadProgress, bool) {
	parts := strings.SplitN(value, "|", 6)
	if len(parts) != 6 {
		return mediaDownloadProgress{}, false
	}
	percent, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(parts[0], "%")), 64)
	if err != nil {
		return mediaDownloadProgress{}, false
	}
	downloaded := parseProgressInt(parts[1])
	total := parseProgressInt(parts[2])
	if total == 0 {
		total = parseProgressInt(parts[3])
	}
	return mediaDownloadProgress{
		Percent:         percent,
		DownloadedBytes: downloaded,
		TotalBytes:      total,
		Speed:           sanitizeYTDLPField(parts[4]),
		ETA:             sanitizeYTDLPField(parts[5]),
	}, true
}

func parseYTDLPDownloadLine(line string) (mediaDownloadProgress, bool) {
	if !strings.Contains(line, "[download]") {
		return mediaDownloadProgress{}, false
	}
	match := ytDLPPercentPattern.FindStringSubmatch(line)
	if len(match) < 2 {
		return mediaDownloadProgress{}, false
	}
	percent, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return mediaDownloadProgress{}, false
	}
	progress := mediaDownloadProgress{
		Percent: percent,
		Speed:   "---",
		ETA:     "---",
	}
	if index := strings.Index(line, " at "); index >= 0 {
		rest := line[index+4:]
		if etaIndex := strings.Index(rest, " ETA "); etaIndex >= 0 {
			progress.Speed = strings.TrimSpace(rest[:etaIndex])
			progress.ETA = strings.TrimSpace(rest[etaIndex+5:])
		} else {
			progress.Speed = strings.TrimSpace(rest)
		}
	}
	return progress, true
}

func parseProgressInt(value string) int64 {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || trimmed == "NA" {
		return 0
	}
	parsed, _ := strconv.ParseInt(trimmed, 10, 64)
	if parsed > 0 {
		return parsed
	}
	floatValue, _ := strconv.ParseFloat(trimmed, 64)
	return int64(floatValue)
}

func sanitizeYTDLPField(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || trimmed == "NA" {
		return "---"
	}
	return trimmed
}

func shouldRetryYTDLPStrategy(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "cookies") ||
		strings.Contains(message, "impersonate") ||
		strings.Contains(message, "requested format is not available") ||
		strings.Contains(message, "sign in to confirm") ||
		strings.Contains(message, "too many requests")
}

func ytDLPVideoSelector(format VideoFormat) string {
	height := videoHeight(format.Resolution)
	videoFilter := "bestvideo"
	audioFilter := "bestaudio"
	progressive := "best"
	if height > 0 {
		videoFilter += "[height<=" + strconv.Itoa(height) + "]"
		progressive += "[height<=" + strconv.Itoa(height) + "]"
	}
	switch format.Extension {
	case "mp4":
		videoFilter += "[ext=mp4]"
		audioFilter += "[ext=m4a]/bestaudio"
		progressive += "[ext=mp4]"
	case "webm":
		videoFilter += "[ext=webm]"
		audioFilter += "[ext=webm]/bestaudio"
		progressive += "[ext=webm]"
	}
	selector := videoFilter + "+" + audioFilter + "/" + progressive
	if height > 0 {
		selector += "/best[height<=" + strconv.Itoa(height) + "]"
	}
	return selector + "/best"
}

func ytDLPOutputTemplate(dir, baseName string) string {
	return filepath.Join(dir, baseName+".%(ext)s")
}

func listAudioFormatsWithYTDLP(ctx context.Context, url string) (ytDLPInfo, error) {
	return fetchYTDLPInfo(ctx, url)
}

func fetchYTDLPInfo(ctx context.Context, url string, extraArgs ...string) (ytDLPInfo, error) {
	ytDLPPath, err := CheckAndDownloadYTDLP()
	if err != nil {
		return ytDLPInfo{}, fmt.Errorf("%w: %v", errYTDLPUnavailable, err)
	}
	var lastErr error
	for _, strategy := range ytDLPStrategies() {
		strategyName := ytDLPStrategyName(strategy)
		args := append(buildYTDLPBaseArgs(), strategy...)
		args = append(args, "-J", "--no-download")
		args = append(args, extraArgs...)
		args = append(args, url)
		log.Printf("[yt-dlp] listing formats strategy=%s url=%s", strategyName, url)
		cmd := exec.CommandContext(ctx, ytDLPPath, args...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		data, err := cmd.Output()
		if err == nil {
			log.Printf("[yt-dlp] listed formats strategy=%s url=%s", strategyName, url)
			return parseYTDLPInfo(data)
		}
		if detail := strings.TrimSpace(stderr.String()); detail != "" {
			err = fmt.Errorf("yt-dlp error: %s", detail)
		}
		log.Printf("[yt-dlp] list formats failed strategy=%s error=%v url=%s", strategyName, err, url)
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("yt-dlp metadata failed")
	}
	return ytDLPInfo{}, lastErr
}

type ytDLPFormat struct {
	FormatID       string  `json:"format_id"`
	Ext            string  `json:"ext"`
	AudioCodec     string  `json:"acodec"`
	VideoCodec     string  `json:"vcodec"`
	ABR            float64 `json:"abr"`
	ASR            float64 `json:"asr"`
	TBR            float64 `json:"tbr"`
	FPS            float64 `json:"fps"`
	Height         int     `json:"height"`
	Filesize       float64 `json:"filesize"`
	FilesizeApprox float64 `json:"filesize_approx"`
	Protocol       string  `json:"protocol"`
	FormatNote     string  `json:"format_note"`
	Resolution     string  `json:"resolution"`
}

type ytDLPPayload struct {
	ID        string         `json:"id"`
	Title     string         `json:"title"`
	Thumbnail string         `json:"thumbnail"`
	Formats   []ytDLPFormat  `json:"formats"`
	Entries   []ytDLPPayload `json:"entries"`
}

func parseYTDLPInfo(data []byte) (ytDLPInfo, error) {
	var info ytDLPPayload
	if err := json.Unmarshal(data, &info); err != nil {
		return ytDLPInfo{}, err
	}
	if len(info.Formats) == 0 && len(info.Entries) > 0 {
		entry := info.Entries[0]
		if entry.Title == "" {
			entry.Title = info.Title
		}
		if entry.Thumbnail == "" {
			entry.Thumbnail = info.Thumbnail
		}
		info = entry
	}
	result := ytDLPInfo{ID: info.ID, Title: info.Title, ThumbnailURL: info.Thumbnail}
	seen := map[string]struct{}{}
	for _, format := range info.Formats {
		if format.FormatID == "" || format.AudioCodec == "" || format.AudioCodec == "none" {
			continue
		}
		if format.VideoCodec != "" && format.VideoCodec != "none" {
			continue
		}
		if _, ok := seen[format.FormatID]; ok {
			continue
		}
		seen[format.FormatID] = struct{}{}
		abr, asr := int(format.ABR), int(format.ASR)
		audioFormat := AudioFormat{
			FormatID:   format.FormatID,
			Container:  format.Ext,
			Extension:  format.Ext,
			AudioCodec: format.AudioCodec,
			Bitrate:    abr,
			SampleRate: asr,
			Protocol:   format.Protocol,
			Label:      audioFormatLabel(format.Ext, format.AudioCodec, abr, asr, format.Protocol, format.FormatNote),
		}
		result.AudioFormats = append(result.AudioFormats, audioFormat)
	}
	sort.SliceStable(result.AudioFormats, func(i, j int) bool {
		if result.AudioFormats[i].Bitrate != result.AudioFormats[j].Bitrate {
			return result.AudioFormats[i].Bitrate > result.AudioFormats[j].Bitrate
		}
		return result.AudioFormats[i].Label < result.AudioFormats[j].Label
	})
	result.VideoFormats = genericVideoFormats(info.Formats)
	return result, nil
}

// genericVideoFormats groups yt-dlp video formats by height. The download only
// uses the resolution and extension, so VideoItag is just a stable index.
func genericVideoFormats(formats []ytDLPFormat) []VideoFormat {
	best := map[int]ytDLPFormat{}
	for _, format := range formats {
		if format.VideoCodec == "none" || (format.VideoCodec == "" && format.Height == 0) {
			continue
		}
		current, ok := best[format.Height]
		if !ok || ytDLPFormatSize(format) > ytDLPFormatSize(current) || (ytDLPFormatSize(format) == ytDLPFormatSize(current) && format.TBR > current.TBR) {
			best[format.Height] = format
		}
	}
	heights := make([]int, 0, len(best))
	for height := range best {
		heights = append(heights, height)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(heights)))

	result := make([]VideoFormat, 0, len(heights))
	for index, height := range heights {
		format := best[height]
		videoFormat := VideoFormat{
			VideoItag:  index + 1,
			Container:  "mp4",
			Extension:  "mp4",
			FPS:        int(format.FPS),
			VideoCodec: strings.TrimSpace(format.VideoCodec),
			Size:       ytDLPFormatSize(format),
		}
		if height > 0 {
			videoFormat.Resolution = strconv.Itoa(height) + "p"
		}
		if format.AudioCodec != "" && format.AudioCodec != "none" {
			videoFormat.AudioCodec = format.AudioCodec
		}
		videoFormat.Label = videoFormatLabel(videoFormat)
		if height == 0 {
			videoFormat.Label = "Melhor qualidade (MP4)"
		}
		result = append(result, videoFormat)
	}
	return result
}

func ytDLPFormatSize(format ytDLPFormat) int64 {
	if format.Filesize > 0 {
		return int64(format.Filesize)
	}
	return int64(format.FilesizeApprox)
}

func audioFormatLabel(container, codec string, bitrate, sampleRate int, protocol, note string) string {
	parts := []string{strings.ToUpper(container)}
	if bitrate > 0 {
		parts = append(parts, fmt.Sprintf("%dk", bitrate))
	}
	if codec != "" {
		parts = append(parts, codec)
	}
	if sampleRate > 0 {
		parts = append(parts, fmt.Sprintf("%dHz", sampleRate))
	}
	if protocol != "" {
		parts = append(parts, protocol)
	}
	if note != "" {
		parts = append(parts, note)
	}
	return strings.Join(parts, " - ")
}

func ytDLPStrategyName(strategy []string) string {
	if len(strategy) == 0 {
		return "plain"
	}
	parts := make([]string, 0, len(strategy)/2+1)
	for i := 0; i < len(strategy); i++ {
		switch strategy[i] {
		case "--impersonate":
			if i+1 < len(strategy) {
				parts = append(parts, "impersonate:"+strategy[i+1])
				i++
			}
		}
	}
	if len(parts) == 0 {
		return "custom"
	}
	return strings.Join(parts, "+")
}

func ytDLPStrategies() [][]string {
	return [][]string{
		{"--impersonate", "chrome-136"},
		nil,
	}
}
