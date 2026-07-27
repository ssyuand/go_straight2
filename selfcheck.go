package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type PipelineSelfCheck struct {
	Status            string `json:"status"`
	StartedAt         string `json:"started_at,omitempty"`
	CompletedAt       string `json:"completed_at,omitempty"`
	DurationMS        int64  `json:"duration_ms,omitempty"`
	StreamlinkVersion string `json:"streamlink_version,omitempty"`
	FFmpegVersion     string `json:"ffmpeg_version,omitempty"`
	OutputBytes       int64  `json:"output_bytes,omitempty"`
	Resolution        string `json:"resolution,omitempty"`
	VideoCodec        string `json:"video_codec,omitempty"`
	AudioCodec        string `json:"audio_codec,omitempty"`
	Error             string `json:"error,omitempty"`
}

func (a *App) startHealthMonitoring() {
	log.Println("[SYSTEM] 真實錄影管線自檢與系統監控已啟用")
	a.triggerPipelineSelfCheck()
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			a.updateDiskStatus()
			a.updateSystemResource()
			a.sysMu.Lock()
			availGB := float64(a.SysState.DiskAvail) / (1024 * 1024 * 1024)
			a.sysMu.Unlock()
			if availGB < 5 {
				log.Printf("[HEALTH-WARNING] 儲存剩餘空間極低 remaining_gb=%.2f", availGB)
			}
		}
	}()
}

func (a *App) pipelineSelfCheckSnapshot() PipelineSelfCheck {
	a.selfCheckMu.Lock()
	defer a.selfCheckMu.Unlock()
	return a.SelfCheck
}

func (a *App) triggerPipelineSelfCheck() bool {
	a.selfCheckMu.Lock()
	if a.selfCheckRunning {
		a.selfCheckMu.Unlock()
		return false
	}
	a.selfCheckRunning = true
	started := time.Now()
	a.SelfCheck = PipelineSelfCheck{Status: "RUNNING", StartedAt: started.Format("2006-01-02 15:04:05")}
	a.selfCheckMu.Unlock()

	go func() {
		result := runPipelineSelfCheck(a.BaseSaveDir)
		result.StartedAt = started.Format("2006-01-02 15:04:05")
		result.CompletedAt = time.Now().Format("2006-01-02 15:04:05")
		result.DurationMS = time.Since(started).Milliseconds()
		a.selfCheckMu.Lock()
		a.SelfCheck = result
		a.selfCheckRunning = false
		a.selfCheckMu.Unlock()
		if result.Status == "PASS" {
			log.Printf("[SELFTEST] recording pipeline passed duration_ms=%d output_bytes=%d resolution=%s video=%s audio=%s", result.DurationMS, result.OutputBytes, result.Resolution, result.VideoCodec, result.AudioCodec)
		} else {
			log.Printf("[SELFTEST-ERROR] recording pipeline failed duration_ms=%d error=%q", result.DurationMS, result.Error)
		}
	}()
	return true
}

func runPipelineSelfCheck(baseSaveDir string) PipelineSelfCheck {
	result := PipelineSelfCheck{Status: "FAIL"}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	if err := checkRecordingDirectory(baseSaveDir); err != nil {
		result.Error = err.Error()
		return result
	}
	streamlinkPath, err := exec.LookPath("streamlink")
	if err != nil {
		result.Error = "找不到 streamlink: " + err.Error()
		return result
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		result.Error = "找不到 ffmpeg: " + err.Error()
		return result
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		result.Error = "找不到 ffprobe: " + err.Error()
		return result
	}
	result.StreamlinkVersion = firstCommandLine(ctx, streamlinkPath, "--version")
	result.FFmpegVersion = firstCommandLine(ctx, "ffmpeg", "-version")

	testDir, err := os.MkdirTemp(baseSaveDir, ".pipeline-selftest-")
	if err != nil {
		result.Error = "無法建立自檢目錄: " + err.Error()
		return result
	}
	defer os.RemoveAll(testDir)

	playlist := filepath.Join(testDir, "index.m3u8")
	generate := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "testsrc2=size=320x180:rate=15",
		"-f", "lavfi", "-i", "sine=frequency=1000:sample_rate=48000",
		"-t", "3", "-c:v", "mpeg2video", "-c:a", "mp2",
		"-f", "hls", "-hls_time", "1", "-hls_list_size", "0", playlist)
	if out, err := generate.CombinedOutput(); err != nil {
		result.Error = fmt.Sprintf("測試 HLS 產生失敗: %v: %s", err, strings.TrimSpace(string(out)))
		return result
	}

	server := httptest.NewServer(http.FileServer(http.Dir(testDir)))
	defer server.Close()
	outputPath := filepath.Join(testDir, "recorded.ts")
	streamlinkCmd := exec.CommandContext(ctx, streamlinkPath, "hls://"+server.URL+"/index.m3u8", "best", "--loglevel", "error", "--stdout")
	ffmpegCmd := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error", "-y", "-i", "pipe:0", "-c", "copy", "-f", "mpegts", outputPath)
	pipe, err := streamlinkCmd.StdoutPipe()
	if err != nil {
		result.Error = "Streamlink pipe 建立失敗: " + err.Error()
		return result
	}
	var streamlinkErr, ffmpegErr bytes.Buffer
	streamlinkCmd.Stderr = &streamlinkErr
	ffmpegCmd.Stderr = &ffmpegErr
	ffmpegCmd.Stdin = pipe
	if err := streamlinkCmd.Start(); err != nil {
		result.Error = "Streamlink 自檢啟動失敗: " + err.Error()
		return result
	}
	if err := ffmpegCmd.Start(); err != nil {
		_ = streamlinkCmd.Process.Kill()
		_ = streamlinkCmd.Wait()
		result.Error = "FFmpeg 自檢啟動失敗: " + err.Error()
		return result
	}
	streamErr := streamlinkCmd.Wait()
	ffErr := ffmpegCmd.Wait()
	if streamErr != nil || ffErr != nil {
		result.Error = fmt.Sprintf("自檢管線退出: streamlink=%v (%s), ffmpeg=%v (%s)", streamErr, strings.TrimSpace(streamlinkErr.String()), ffErr, strings.TrimSpace(ffmpegErr.String()))
		return result
	}
	info, err := os.Stat(outputPath)
	if err != nil || info.Size() == 0 {
		result.Error = "自檢沒有產生有效錄影檔"
		return result
	}
	result.OutputBytes = info.Size()
	quality := inspectMedia(outputPath)
	if quality.Status != "VALID" || !quality.HasVideo || !quality.HasAudio {
		result.Error = "ffprobe 驗證失敗: " + quality.Error
		return result
	}
	result.Resolution = quality.Resolution
	result.VideoCodec = quality.VideoCodec
	result.AudioCodec = quality.AudioCodec
	result.Status = "PASS"
	return result
}

func firstCommandLine(ctx context.Context, command string, args ...string) string {
	out, err := exec.CommandContext(ctx, command, args...).CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
}
