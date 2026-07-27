package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAppLogWriterAddsLevelAndComponent(t *testing.T) {
	var output bytes.Buffer
	w := &appLogWriter{out: &output}
	if _, err := w.Write([]byte("[HEALTH-WARNING] disk space low\n")); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	if !strings.Contains(got, "level=WARN") || !strings.Contains(got, "component=HEALTH-WARNING") {
		t.Fatalf("unexpected formatted log: %s", got)
	}
}

func TestParseStreamlinkOpeningLine(t *testing.T) {
	quality, streamType, ok := parseStreamlinkOpeningLine("[cli][info] Opening stream: hd (hls)")
	if !ok || quality != "hd" || streamType != "hls" {
		t.Fatalf("quality=%q streamType=%q ok=%t", quality, streamType, ok)
	}
	if _, _, ok := parseStreamlinkOpeningLine("[cli][info] Available streams: ld, hd"); ok {
		t.Fatal("available-stream line must not be parsed as selected quality")
	}
}

func TestRecordingWriteHealth(t *testing.T) {
	now := time.Date(2026, 7, 27, 20, 0, 0, 0, time.Local)
	tests := []struct {
		name        string
		firstWrite  bool
		age         time.Duration
		wantDelayed bool
		wantStalled bool
	}{
		{name: "before first write", firstWrite: false, age: time.Minute},
		{name: "healthy", firstWrite: true, age: 29 * time.Second},
		{name: "delayed", firstWrite: true, age: 30 * time.Second, wantDelayed: true},
		{name: "stalled", firstWrite: true, age: 60 * time.Second, wantDelayed: true, wantStalled: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delayed, stalled := recordingWriteHealth(tt.firstWrite, now.Add(-tt.age), now)
			if delayed != tt.wantDelayed || stalled != tt.wantStalled {
				t.Fatalf("delayed=%t stalled=%t", delayed, stalled)
			}
		})
	}
}

func TestSessionHealthPercent(t *testing.T) {
	now := time.Date(2026, 7, 27, 20, 10, 0, 0, time.Local)
	started := now.Add(-10 * time.Minute)
	if got := sessionHealthPercent(started, time.Minute, 0, now); got != 90 {
		t.Fatalf("health=%v, want 90", got)
	}
	if got := sessionHealthPercent(started, 0, 1, now); got != 0 {
		t.Fatalf("broken segment health=%v, want 0", got)
	}
	if got := sessionHealthPercent(started, 20*time.Minute, 0, now); got != 0 {
		t.Fatalf("clamped health=%v, want 0", got)
	}
}

func TestRequestDiagnosticsRecordsRequest(t *testing.T) {
	var output bytes.Buffer
	previous := log.Writer()
	previousFlags := log.Flags()
	log.SetOutput(&appLogWriter{out: &output})
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(previous)
		log.SetFlags(previousFlags)
	})

	h := requestDiagnostics(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "http://recorder.local/api/probe?prefix=test", nil))

	if w.Code != http.StatusCreated || w.Header().Get("X-Request-ID") == "" {
		t.Fatalf("status=%d request_id=%q", w.Code, w.Header().Get("X-Request-ID"))
	}
	got := output.String()
	if !strings.Contains(got, "component=HTTP") || !strings.Contains(got, "status=201") || !strings.Contains(got, "path=\\\"/api/probe\\\"") {
		t.Fatalf("request log missing diagnostics: %s", got)
	}
}

func TestRotatingFileWriter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	w, err := newRotatingFileWriter(path, 10, 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("12345678")); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("abcdefgh")); err != nil {
		t.Fatal(err)
	}
	current, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	backup, err := os.ReadFile(path + ".1")
	if err != nil {
		t.Fatal(err)
	}
	if string(current) != "abcdefgh" || string(backup) != "12345678" {
		t.Fatalf("current=%q backup=%q", current, backup)
	}
}

func TestProcessExitCodeNil(t *testing.T) {
	if got := processExitCode(nil); got != 0 {
		t.Fatalf("got %d, want 0", got)
	}
}

func TestProbeMetricsSnapshot(t *testing.T) {
	stream := &StreamState{Prefix: "demo"}
	app := &App{Streams: map[string]*StreamState{"demo": stream}}
	app.recordProbeMetrics("demo", 100*time.Millisecond, true)
	app.recordProbeMetrics("demo", 300*time.Millisecond, false)

	snapshot := app.getAPIResponseSnapshot().Streams["demo"]
	if snapshot.ProbeAttempts != 2 || snapshot.ProbeSuccesses != 1 {
		t.Fatalf("attempts=%d successes=%d", snapshot.ProbeAttempts, snapshot.ProbeSuccesses)
	}
	if snapshot.ProbeSuccessRate != 50 || snapshot.ProbeAverageDuration != 200 {
		t.Fatalf("rate=%f average_ms=%f", snapshot.ProbeSuccessRate, snapshot.ProbeAverageDuration)
	}
}

func TestRecordingTelemetrySnapshot(t *testing.T) {
	started := time.Date(2026, 7, 26, 10, 0, 0, 0, time.Local)
	stream := &StreamState{
		Prefix:              "demo",
		IsRecording:         true,
		SegmentStartedAt:    started,
		WriteBytesPerSecond: 1024,
		StreamlinkPID:       101,
		FFmpegPID:           202,
		FFmpegBitrate:       "2500kbits/s",
		FFmpegSpeed:         "1.0x",
		PipelineState:       "RECORDING",
	}
	app := &App{Streams: map[string]*StreamState{"demo": stream}}
	snapshot := app.getAPIResponseSnapshot().Streams["demo"]
	if snapshot.SegmentStartedAt == "" || snapshot.WriteBytesPerSecond != 1024 {
		t.Fatalf("segment_started_at=%q write_speed=%f", snapshot.SegmentStartedAt, snapshot.WriteBytesPerSecond)
	}
	if snapshot.StreamlinkPID != 101 || snapshot.FFmpegPID != 202 || snapshot.PipelineState != "RECORDING" {
		t.Fatalf("unexpected process telemetry: %+v", snapshot)
	}
}

func TestBuildAlertsAndDiskEstimate(t *testing.T) {
	app := &App{}
	resp := APIResponse{
		System: GlobalSystemState{DiskAvail: 1024 * 1024},
		Streams: map[string]StreamStateSnapshot{
			"demo": {IsRecording: true, WriteBytesPerSecond: 1024},
		},
	}
	alerts, speed, estimate := app.buildAlerts(resp)
	if len(alerts) == 0 || speed != 1024 || estimate != 1024 {
		t.Fatalf("alerts=%v speed=%f estimate=%d", alerts, speed, estimate)
	}
}

func TestStreamEventsAreBounded(t *testing.T) {
	stream := &StreamState{SessionID: "session-test"}
	for i := 0; i < 120; i++ {
		appendStreamEvent(stream, "info", "test", fmt.Sprintf("event %d", i))
	}
	if len(stream.Events) != 100 || stream.Events[0].Message != "event 20" {
		t.Fatalf("unexpected bounded events: len=%d first=%q", len(stream.Events), stream.Events[0].Message)
	}
}

func TestTailFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tail.log")
	if err := os.WriteFile(path, []byte("0123456789"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := tailFile(path, 4)
	if err != nil || string(got) != "6789" {
		t.Fatalf("got=%q err=%v", got, err)
	}
}

func TestFileListCacheKeepsLiveFileFresh(t *testing.T) {
	base := t.TempDir()
	channelDir := filepath.Join(base, "demo")
	if err := os.MkdirAll(channelDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(channelDir, "segment.ts"), []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	app := &App{BaseSaveDir: base, MediaQuality: make(map[string]MediaQuality)}
	resp := APIResponse{Streams: map[string]StreamStateSnapshot{"demo": {IsRecording: true, LatestFile: "segment.ts", LatestSize: 3}}}
	first := app.listVideoFiles(resp)
	if len(first["demo"]) != 1 || !first["demo"][0].IsGrowing {
		t.Fatalf("unexpected first scan: %+v", first)
	}
	resp.Streams["demo"] = StreamStateSnapshot{IsRecording: true, LatestFile: "segment.ts", LatestSize: 99, LatestMTime: "now"}
	second := app.listVideoFiles(resp)
	if second["demo"][0].SizeBytes != 99 || second["demo"][0].MTime != "now" {
		t.Fatalf("cached live row was stale: %+v", second["demo"][0])
	}
}

func TestNewRecordingPathIsUnique(t *testing.T) {
	now := time.Date(2026, 7, 26, 21, 5, 30, 482000000, time.Local)
	first := newRecordingPath("/recordings", now)
	second := newRecordingPath("/recordings", now)
	if first == second || !strings.Contains(first, "20260726-210530-482-s") || filepath.Ext(first) != ".ts" {
		t.Fatalf("unexpected recording paths: %q %q", first, second)
	}
}

func TestStderrTailIsBounded(t *testing.T) {
	tail := newStderrTail(3)
	for _, line := range []string{"one", "two", "three", "four"} {
		tail.Add(line)
	}
	if got := tail.String(); got != "two | three | four" {
		t.Fatalf("got %q", got)
	}
}

func TestCheckRecordingDirectory(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "new", "channel")
	if err := checkRecordingDirectory(directory); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("preflight left temporary files: %v", entries)
	}
}

func TestCalculateProbeWindowAcrossMidnight(t *testing.T) {
	loc := time.FixedZone("Asia/Taipei", 8*60*60)
	tests := []struct {
		name       string
		hour       int
		minute     int
		wantActive bool
		wantWait   time.Duration
	}{
		{name: "before start", hour: 19, minute: 19, wantActive: false, wantWait: time.Minute},
		{name: "exact start is active", hour: 19, minute: 20, wantActive: true},
		{name: "before midnight", hour: 23, minute: 59, wantActive: true},
		{name: "exact end is inactive", hour: 0, minute: 0, wantActive: false, wantWait: 19*time.Hour + 20*time.Minute},
		{name: "after midnight is inactive", hour: 0, minute: 1, wantActive: false, wantWait: 19*time.Hour + 19*time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Date(2026, 7, 26, tt.hour, tt.minute, 0, 0, loc)
			active, wait, err := calculateProbeWindow(now, "19:20", "00:00")
			if err != nil {
				t.Fatal(err)
			}
			if active != tt.wantActive {
				t.Fatalf("active=%v, want %v", active, tt.wantActive)
			}
			if wait != tt.wantWait {
				t.Fatalf("wait=%v, want %v", wait, tt.wantWait)
			}
		})
	}
}

func TestCalculateProbeWindowSameTimeMeansAllDay(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.Local)
	active, wait, err := calculateProbeWindow(now, "00:00", "00:00")
	if err != nil || !active || wait != 0 {
		t.Fatalf("active=%v wait=%v err=%v", active, wait, err)
	}
}

func TestValidateConfigRejectsInvalidHotReloadValues(t *testing.T) {
	cfg := Config{TargetURLs: []string{"https://example.com/@demo/live"}, WebPort: 36591, ProbeStart: "19:20", ProbeEnd: "00:00", ProbeInterval: 120, ProbeSleepDeep: 3600}
	if err := validateConfig(cfg); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	cfg.ProbeStart = "19:99"
	if err := validateConfig(cfg); err == nil {
		t.Fatal("invalid time was accepted")
	}
	cfg.ProbeStart = "bad"
	cfg.ProbeEnd = "bad"
	if err := validateConfig(cfg); err == nil {
		t.Fatal("matching invalid times were accepted as an all-day window")
	}
}

func TestValidateTargetURLs(t *testing.T) {
	if err := validateTargetURLs([]string{"https://example.com/@demo/live"}); err != nil {
		t.Fatal(err)
	}
	if err := validateTargetURLs([]string{"not-a-url"}); err == nil {
		t.Fatal("invalid target URL accepted")
	}
}

func TestHandleAPIConfigSavesAtomically(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	current := Config{TargetURLs: []string{"https://example.com/@old/live"}, WebPort: 36591, ProbeStart: "19:20", ProbeEnd: "00:00", ProbeInterval: 120, ProbeSleepDeep: 3600}
	if err := saveConfigAtomically(path, current); err != nil {
		t.Fatal(err)
	}
	app := &App{Config: current, ConfigPath: path, Streams: map[string]*StreamState{"old": {ReloadChan: make(chan struct{}, 1)}}}
	next := `{"target_urls":["https://example.com/@new/live"],"web_port":36592,"probe_start":"20:00","probe_end":"01:00","probe_interval":90,"probe_sleep_deep":1800}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader(next))
	app.handleAPIConfig(w, r)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"restart_required":true`) {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	saved, err := readConfigFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if saved.WebPort != 36592 || saved.ProbeInterval != 90 || len(saved.TargetURLs) != 1 {
		t.Fatalf("unexpected saved config: %+v", saved)
	}
}

func TestHandleAPIProbePause(t *testing.T) {
	stream := &StreamState{Prefix: "demo"}
	app := &App{Streams: map[string]*StreamState{"demo": stream}}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/probe_pause?prefix=demo&paused=true", nil)
	app.handleAPIProbePause(w, r)
	if w.Code != http.StatusOK || !stream.ProbePaused || !strings.Contains(w.Body.String(), `"probe_paused":true`) {
		t.Fatalf("pause status=%d paused=%t body=%s", w.Code, stream.ProbePaused, w.Body.String())
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/api/probe_pause?prefix=demo&paused=false", nil)
	app.handleAPIProbePause(w, r)
	if w.Code != http.StatusOK || stream.ProbePaused {
		t.Fatalf("resume status=%d paused=%t body=%s", w.Code, stream.ProbePaused, w.Body.String())
	}
}

func TestHTMLTemplateParses(t *testing.T) {
	if _, err := template.New("index").Parse(htmlTemplate); err != nil {
		t.Fatalf("dashboard template does not parse: %v", err)
	}
}

func TestHTMLJavaScriptSyntax(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed")
	}
	tmpl, err := template.New("index").Parse(htmlTemplate)
	if err != nil {
		t.Fatal(err)
	}
	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, PageData{}); err != nil {
		t.Fatal(err)
	}
	html := rendered.String()
	start := strings.Index(html, "<script>")
	end := strings.LastIndex(html, "</script>")
	if start < 0 || end <= start {
		t.Fatal("dashboard script block not found")
	}
	cmd := exec.Command(node, "--check")
	cmd.Stdin = strings.NewReader(html[start+len("<script>") : end])
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("dashboard JavaScript syntax error: %v\n%s", err, out)
	}
}

func TestRequireMethod(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/shutdown", nil)
	if requireMethod(w, r, http.MethodPost) {
		t.Fatal("GET must not be accepted for a destructive endpoint")
	}
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestSameOrigin(t *testing.T) {
	called := false
	h := sameOrigin(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "http://recorder.local/api/shutdown", nil)
	r.Header.Set("Origin", "http://attacker.invalid")
	h.ServeHTTP(w, r)

	if called {
		t.Fatal("cross-origin POST reached the handler")
	}
	if w.Code != http.StatusForbidden {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestSameOriginAllowsLocalDashboard(t *testing.T) {
	called := false
	h := sameOrigin(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "http://recorder.local/api/probe", nil)
	r.Header.Set("Origin", "http://recorder.local")
	h.ServeHTTP(w, r)

	if !called {
		t.Fatal("same-origin dashboard request was blocked")
	}
}
