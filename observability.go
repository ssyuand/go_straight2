package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type StreamEvent struct {
	Time      string `json:"time"`
	Level     string `json:"level"`
	Type      string `json:"type"`
	Message   string `json:"message"`
	SessionID string `json:"session_id,omitempty"`
}

type TrendPoint struct {
	Time       int64   `json:"time"`
	WriteSpeed float64 `json:"write_speed_bytes"`
}

type Alert struct {
	Level   string `json:"level"`
	Channel string `json:"channel,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type MediaQuality struct {
	Status     string  `json:"status"`
	Duration   float64 `json:"duration_seconds,omitempty"`
	Resolution string  `json:"resolution,omitempty"`
	FPS        string  `json:"fps,omitempty"`
	VideoCodec string  `json:"video_codec,omitempty"`
	AudioCodec string  `json:"audio_codec,omitempty"`
	SampleRate string  `json:"sample_rate,omitempty"`
	HasVideo   bool    `json:"has_video"`
	HasAudio   bool    `json:"has_audio"`
	CheckedAt  string  `json:"checked_at,omitempty"`
	Error      string  `json:"error,omitempty"`
}

type SessionDetail struct {
	Channel string              `json:"channel"`
	State   StreamStateSnapshot `json:"state"`
	Events  []StreamEvent       `json:"events"`
	Trend   []TrendPoint        `json:"trend"`
	Files   []FileRow           `json:"files"`
}

func appendStreamEvent(s *StreamState, level, eventType, message string) {
	s.mu.Lock()
	event := StreamEvent{Time: time.Now().Format("2006-01-02 15:04:05"), Level: level, Type: eventType, Message: message, SessionID: s.SessionID}
	s.Events = append(s.Events, event)
	if len(s.Events) > 100 {
		s.Events = append([]StreamEvent(nil), s.Events[len(s.Events)-100:]...)
	}
	s.mu.Unlock()
}

func appendTrendPoint(s *StreamState, speed float64) {
	s.mu.Lock()
	s.Trend = append(s.Trend, TrendPoint{Time: time.Now().Unix(), WriteSpeed: speed})
	if len(s.Trend) > 300 {
		s.Trend = append([]TrendPoint(nil), s.Trend[len(s.Trend)-300:]...)
	}
	s.mu.Unlock()
}

func (a *App) buildAlerts(resp APIResponse) ([]Alert, float64, int64) {
	alerts := make([]Alert, 0)
	if resp.PipelineSelfCheck.Status == "FAIL" {
		alerts = append(alerts, Alert{Level: "error", Code: "pipeline_selftest_failed", Message: "真實錄影管線自檢失敗：" + resp.PipelineSelfCheck.Error})
	}
	var totalSpeed float64
	for prefix, stream := range resp.Streams {
		totalSpeed += stream.WriteBytesPerSecond
		if stream.IsRecording {
			lastWrite, _ := time.ParseInLocation("2006-01-02 15:04:05", stream.LastSuccessfulWrite, time.Local)
			if !lastWrite.IsZero() && time.Since(lastWrite) > 30*time.Second {
				alerts = append(alerts, Alert{Level: "error", Channel: prefix, Code: "write_stale", Message: "錄影超過 30 秒沒有成功寫入"})
			}
			if stream.WriteBytesPerSecond > 0 && stream.WriteBytesPerSecond < 32*1024 {
				alerts = append(alerts, Alert{Level: "warning", Channel: prefix, Code: "low_write_speed", Message: "錄影寫入速度異常偏低"})
			}
		}
		if stream.FFmpegAbnormalExits > 0 {
			alerts = append(alerts, Alert{Level: "warning", Channel: prefix, Code: "ffmpeg_exit", Message: fmt.Sprintf("FFmpeg 已非正常退出 %d 次", stream.FFmpegAbnormalExits)})
		}
		if stream.RecordingStartFailures > 0 {
			alerts = append(alerts, Alert{Level: "warning", Channel: prefix, Code: "recording_start_failure", Message: fmt.Sprintf("錄影啟動失敗 %d 次，請查看最近錯誤", stream.RecordingStartFailures)})
		}
	}
	if resp.System.DiskAvail > 0 && resp.System.DiskAvail < 10*1024*1024*1024 {
		alerts = append(alerts, Alert{Level: "error", Code: "disk_low", Message: "磁碟剩餘空間低於 10 GB"})
	}
	estimate := int64(0)
	if totalSpeed > 0 {
		estimate = int64(float64(resp.System.DiskAvail) / totalSpeed)
	}
	return alerts, totalSpeed, estimate
}

func (a *App) inspectMediaAsync(path string) {
	a.obsMu.Lock()
	if _, exists := a.MediaQuality[path]; exists {
		a.obsMu.Unlock()
		return
	}
	a.MediaQuality[path] = MediaQuality{Status: "CHECKING"}
	a.obsMu.Unlock()
	go func() {
		quality := inspectMedia(path)
		a.obsMu.Lock()
		a.MediaQuality[path] = quality
		a.obsMu.Unlock()
		a.fileCacheMu.Lock()
		a.lastFileScan = time.Time{}
		a.fileCacheMu.Unlock()
	}()
}

func inspectMedia(path string) MediaQuality {
	quality := MediaQuality{Status: "CHECKING"}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-show_entries", "format=duration:stream=codec_type,codec_name,width,height,r_frame_rate,sample_rate", "-of", "json", path)
	out, err := cmd.Output()
	quality.CheckedAt = time.Now().Format("2006-01-02 15:04:05")
	if err != nil {
		quality.Status = "WARNING"
		quality.Error = truncateText(err.Error(), 300)
		return quality
	}
	var payload struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
		Streams []struct {
			CodecType  string `json:"codec_type"`
			CodecName  string `json:"codec_name"`
			Width      int    `json:"width"`
			Height     int    `json:"height"`
			FrameRate  string `json:"r_frame_rate"`
			SampleRate string `json:"sample_rate"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		quality.Status = "BROKEN"
		quality.Error = truncateText(err.Error(), 300)
		return quality
	}
	quality.Duration, _ = strconv.ParseFloat(payload.Format.Duration, 64)
	for _, stream := range payload.Streams {
		switch stream.CodecType {
		case "video":
			quality.HasVideo = true
			quality.VideoCodec = stream.CodecName
			quality.Resolution = fmt.Sprintf("%dx%d", stream.Width, stream.Height)
			quality.FPS = stream.FrameRate
		case "audio":
			quality.HasAudio = true
			quality.AudioCodec = stream.CodecName
			quality.SampleRate = stream.SampleRate
		}
	}
	if quality.HasVideo && quality.Duration > 0 {
		quality.Status = "VALID"
	} else {
		quality.Status = "BROKEN"
		quality.Error = "找不到有效影片軌或影片時長"
	}
	return quality
}

func (a *App) inspectCompletedSessionSegment(s *StreamState, sessionID, path string) {
	info, err := os.Stat(path)
	if err != nil || info.Size() == 0 {
		return
	}
	go func() {
		quality := inspectMedia(path)
		a.obsMu.Lock()
		a.MediaQuality[path] = quality
		a.obsMu.Unlock()
		s.mu.Lock()
		if s.SessionID == sessionID {
			s.VerifiedSegments++
			if quality.Status != "VALID" {
				s.BrokenSegments++
			}
		}
		s.mu.Unlock()
		a.fileCacheMu.Lock()
		a.lastFileScan = time.Time{}
		a.fileCacheMu.Unlock()
	}()
}

func truncateText(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max] + "…"
}

func (a *App) handleAPISession(w http.ResponseWriter, r *http.Request) {
	prefix := strings.TrimPrefix(strings.TrimSpace(r.URL.Query().Get("prefix")), "@")
	resp := a.getAPIResponseSnapshot()
	resp.Files = a.listVideoFiles(resp)
	state, ok := resp.Streams[prefix]
	if !ok {
		http.Error(w, `{"error":"頻道不存在"}`, http.StatusNotFound)
		return
	}
	a.StreamsMu.RLock()
	stream := a.Streams[prefix]
	a.StreamsMu.RUnlock()
	stream.mu.Lock()
	events := append([]StreamEvent(nil), stream.Events...)
	trend := append([]TrendPoint(nil), stream.Trend...)
	stream.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(SessionDetail{Channel: prefix, State: state, Events: events, Trend: trend, Files: resp.Files[prefix]})
}

func (a *App) handleAPIDiagnostics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="livetool-diagnostics.zip"`)
	zw := zip.NewWriter(w)
	defer zw.Close()
	resp := a.getAPIResponseSnapshot()
	resp.Files = a.listVideoFiles(resp)
	for prefix, stream := range resp.Streams {
		stream.TargetURL = ""
		stream.SaveDir = ""
		resp.Streams[prefix] = stream
	}
	writeZipJSON(zw, "status.json", resp)
	a.configMu.RLock()
	safeConfig := a.Config
	safeConfig.TargetURLs = nil
	a.configMu.RUnlock()
	writeZipJSON(zw, "config-sanitized.json", safeConfig)
	if data, err := tailFile("livetool.log", 512*1024); err == nil {
		if entry, createErr := zw.Create("livetool.log"); createErr == nil {
			_, _ = entry.Write(data)
		}
	}
	versions := []string{"go=" + runtime.Version()}
	for _, command := range []string{"streamlink", "ffmpeg", "ffprobe"} {
		versionFlag := "--version"
		if command != "streamlink" {
			versionFlag = "-version"
		}
		if out, err := exec.Command(command, versionFlag).CombinedOutput(); err == nil {
			versions = append(versions, command+"="+strings.SplitN(string(out), "\n", 2)[0])
		}
	}
	if entry, err := zw.Create("versions.txt"); err == nil {
		_, _ = io.WriteString(entry, strings.Join(versions, "\n"))
	}
}

func writeZipJSON(zw *zip.Writer, name string, value any) {
	entry, err := zw.Create(name)
	if err != nil {
		return
	}
	encoder := json.NewEncoder(entry)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(value)
}

func tailFile(path string, maxBytes int64) ([]byte, error) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	start := stat.Size() - maxBytes
	if start < 0 {
		start = 0
	}
	if _, err := file.Seek(start, io.SeekStart); err != nil {
		return nil, err
	}
	return io.ReadAll(io.LimitReader(file, maxBytes))
}
