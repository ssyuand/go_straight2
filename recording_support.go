package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

var recordingFileSequence atomic.Uint64

const recordingWriteDelayTimeout = 30 * time.Second
const recordingWriteStallTimeout = 60 * time.Second

func recordingWriteHealth(firstWrite bool, lastGrowthAt, now time.Time) (delayed, stalled bool) {
	if !firstWrite || lastGrowthAt.IsZero() {
		return false, false
	}
	age := now.Sub(lastGrowthAt)
	return age >= recordingWriteDelayTimeout, age >= recordingWriteStallTimeout
}

func sessionHealthPercent(startedAt time.Time, gap time.Duration, brokenSegments uint64, now time.Time) float64 {
	if brokenSegments > 0 {
		return 0
	}
	if startedAt.IsZero() || !now.After(startedAt) {
		return 100
	}
	elapsed := now.Sub(startedAt)
	health := (1 - float64(gap)/float64(elapsed)) * 100
	if health < 0 {
		return 0
	}
	if health > 100 {
		return 100
	}
	return health
}

func newRecordingPath(saveDir string, now time.Time) string {
	name := fmt.Sprintf("%s-%03d-s%04d.ts", now.Format("20060102-150405"), now.Nanosecond()/int(time.Millisecond), recordingFileSequence.Add(1)%10000)
	return filepath.Join(saveDir, name)
}

func streamlinkRecordingArgs(targetURL, userAgent string) []string {
	return []string{
		targetURL, "hd,ld,best",
		"--loglevel", "info",
		"--ringbuffer-size", "512M",
		"--retry-open", "3",
		"--stream-segment-attempts", "5",
		"--stream-segment-threads", "1",
		"--stream-segment-timeout", "20",
		"--hls-playlist-reload-attempts", "5",
		"--http-header", "Referer=https://www.tiktok.com/",
		"--http-header", "Origin=https://www.tiktok.com",
		"--http-header", "User-Agent=" + userAgent,
		"-O",
	}
}

func parseStreamlinkOpeningLine(line string) (quality, streamType string, ok bool) {
	const marker = "Opening stream:"
	markerIndex := strings.Index(line, marker)
	if markerIndex < 0 {
		return "", "", false
	}
	value := strings.TrimSpace(line[markerIndex+len(marker):])
	if value == "" {
		return "", "", false
	}
	if open := strings.LastIndex(value, " ("); open >= 0 && strings.HasSuffix(value, ")") {
		quality = strings.TrimSpace(value[:open])
		streamType = strings.TrimSpace(value[open+2 : len(value)-1])
	} else {
		quality = value
	}
	return quality, streamType, quality != ""
}

func recordingPreflight(saveDir string) error {
	required := []string{"streamlink", "ffmpeg"}
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		required = append(required, "ionice")
	}
	for _, command := range required {
		if _, err := exec.LookPath(command); err != nil {
			return fmt.Errorf("找不到必要指令 %s: %w", command, err)
		}
	}
	return checkRecordingDirectory(saveDir)
}

func startMacSleepPrevention() {
	if runtime.GOOS != "darwin" {
		return
	}
	cmd := exec.Command("/usr/bin/caffeinate", "-i", "-m", "-s", "-w", strconv.Itoa(os.Getpid()))
	if err := cmd.Start(); err != nil {
		log.Printf("[SYSTEM-WARNING] 無法啟用 macOS 防睡眠保護: %v", err)
		return
	}
	log.Printf("[SYSTEM] macOS 防睡眠保護已啟用 caffeinate_pid=%d", cmd.Process.Pid)
	go func() {
		if err := cmd.Wait(); err != nil {
			log.Printf("[SYSTEM-WARNING] macOS 防睡眠保護已結束: %v", err)
		}
	}()
}

func checkRecordingDirectory(saveDir string) error {
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return fmt.Errorf("無法建立錄影目錄: %w", err)
	}
	file, err := os.CreateTemp(saveDir, ".recording-write-check-*")
	if err != nil {
		return fmt.Errorf("錄影目錄不可寫入: %w", err)
	}
	path := file.Name()
	defer os.Remove(path)
	if _, err := file.Write([]byte{0}); err != nil {
		_ = file.Close()
		return fmt.Errorf("錄影目錄寫入測試失敗: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("錄影目錄同步測試失敗: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("錄影目錄關閉測試失敗: %w", err)
	}
	return nil
}

type stderrTail struct {
	lines []string
	limit int
}

func newStderrTail(limit int) *stderrTail {
	return &stderrTail{limit: limit}
}

func (t *stderrTail) Add(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	t.lines = append(t.lines, line)
	if len(t.lines) > t.limit {
		copy(t.lines, t.lines[len(t.lines)-t.limit:])
		t.lines = t.lines[:t.limit]
	}
}

func (t *stderrTail) String() string {
	return strings.Join(t.lines, " | ")
}

func configureGracefulCancellation(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.WaitDelay = 5 * time.Second
	if runtime.GOOS == "windows" {
		return
	}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
}

func requestCommandStop(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	if runtime.GOOS == "windows" {
		_ = cmd.Process.Kill()
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
}

type commandWaitResult struct {
	process string
	err     error
}

func waitRecordingCommands(streamlinkCmd, ffmpegCmd *exec.Cmd) (streamlinkErr, ffmpegErr error) {
	results := make(chan commandWaitResult, 2)
	go func() { results <- commandWaitResult{process: "streamlink", err: streamlinkCmd.Wait()} }()
	go func() { results <- commandWaitResult{process: "ffmpeg", err: ffmpegCmd.Wait()} }()

	first := <-results
	if first.process == "ffmpeg" && streamlinkCmd.Process != nil {
		// The pipe consumer is gone; ask the producer to stop so it cannot block
		// forever writing to a pipe that nobody reads.
		if runtime.GOOS == "windows" {
			_ = streamlinkCmd.Process.Kill()
		} else {
			_ = syscall.Kill(-streamlinkCmd.Process.Pid, syscall.SIGTERM)
		}
	}

	waitLimit := 15 * time.Second
	if first.process == "ffmpeg" {
		waitLimit = 5 * time.Second
	}
	timer := time.NewTimer(waitLimit)
	defer timer.Stop()
	var second commandWaitResult
	select {
	case second = <-results:
	case <-timer.C:
		killCommandGroup(streamlinkCmd)
		killCommandGroup(ffmpegCmd)
		second = <-results
	}

	for _, result := range []commandWaitResult{first, second} {
		if result.process == "streamlink" {
			streamlinkErr = result.err
		} else {
			ffmpegErr = result.err
		}
	}
	return streamlinkErr, ffmpegErr
}
