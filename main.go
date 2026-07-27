package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// Config 設定結構體（映射 config.json 欄位）
type Config struct {
	TargetURLs     []string `json:"target_urls"`
	WebPort        int      `json:"web_port"`
	ProbeStart     string   `json:"probe_start"`
	ProbeEnd       string   `json:"probe_end"`
	ProbeInterval  int      `json:"probe_interval"`
	ProbeSleepDeep int      `json:"probe_sleep_deep"`
}

// StreamState 頻道專屬的隔離狀態與核心鎖控
type StreamState struct {
	mu           sync.Mutex
	TargetURL    string
	Prefix       string
	SaveDir      string
	IsRecording  bool
	IsProbing    bool
	ProbePaused  bool
	ProbeStatus  string
	LatestFile   string
	LatestSize   int64
	LatestMTime  string
	RecordCtx    context.Context
	RecordCancel context.CancelFunc
	ReloadChan   chan struct{}

	// Observability only. These fields never participate in recording decisions.
	SessionID              string
	RecordingStartedAt     time.Time
	ProbeAttempts          uint64
	ProbeSuccesses         uint64
	ProbeTotalDuration     time.Duration
	RecordingRestartCount  uint64
	FFmpegAbnormalExits    uint64
	RecordingStartFailures uint64
	LastSuccessfulWrite    time.Time
	LastError              string
	LastErrorAt            time.Time
	SegmentStartedAt       time.Time
	WriteBytesPerSecond    float64
	StreamlinkPID          int
	FFmpegPID              int
	FFmpegBitrate          string
	FFmpegSpeed            string
	PipelineState          string
	AvailableQualities     []string
	SelectedQuality        string
	SelectedStreamType     string
	SessionSegmentCount    uint64
	SessionRestartCount    uint64
	SessionRecordedBytes   int64
	SessionGapTotal        time.Duration
	SessionMaxGap          time.Duration
	RecoveryStartedAt      time.Time
	LastRecoveryDuration   time.Duration
	TotalRecoveryDuration  time.Duration
	VerifiedSegments       uint64
	BrokenSegments         uint64
	Events                 []StreamEvent
	Trend                  []TrendPoint
}

type StreamStateSnapshot struct {
	TargetURL              string   `json:"target_url"`
	Prefix                 string   `json:"prefix"`
	SaveDir                string   `json:"save_dir"`
	IsRecording            bool     `json:"is_recording"`
	IsProbing              bool     `json:"is_probing"`
	ProbePaused            bool     `json:"probe_paused"`
	ProbeStatus            string   `json:"probe_status"`
	LatestFile             string   `json:"latest_file"`
	LatestSize             int64    `json:"latest_size"`
	LatestMTime            string   `json:"latest_mtime"`
	SessionID              string   `json:"session_id"`
	RecordingStartedAt     string   `json:"recording_started_at"`
	ProbeAttempts          uint64   `json:"probe_attempts"`
	ProbeSuccesses         uint64   `json:"probe_successes"`
	ProbeSuccessRate       float64  `json:"probe_success_rate"`
	ProbeAverageDuration   float64  `json:"probe_average_duration_ms"`
	RecordingRestartCount  uint64   `json:"recording_restart_count"`
	FFmpegAbnormalExits    uint64   `json:"ffmpeg_abnormal_exits"`
	RecordingStartFailures uint64   `json:"recording_start_failures"`
	RecordedBytes          int64    `json:"recorded_bytes"`
	LastSuccessfulWrite    string   `json:"last_successful_write"`
	LastError              string   `json:"last_error"`
	LastErrorAt            string   `json:"last_error_at"`
	SegmentStartedAt       string   `json:"segment_started_at"`
	WriteBytesPerSecond    float64  `json:"write_bytes_per_second"`
	StreamlinkPID          int      `json:"streamlink_pid"`
	FFmpegPID              int      `json:"ffmpeg_pid"`
	FFmpegBitrate          string   `json:"ffmpeg_bitrate"`
	FFmpegSpeed            string   `json:"ffmpeg_speed"`
	PipelineState          string   `json:"pipeline_state"`
	AvailableQualities     []string `json:"available_qualities"`
	SelectedQuality        string   `json:"selected_quality"`
	SelectedStreamType     string   `json:"selected_stream_type"`
	SessionSegmentCount    uint64   `json:"session_segment_count"`
	SessionRestartCount    uint64   `json:"session_restart_count"`
	SessionRecordedBytes   int64    `json:"session_recorded_bytes"`
	SessionGapTotalMS      int64    `json:"session_gap_total_ms"`
	SessionMaxGapMS        int64    `json:"session_max_gap_ms"`
	LastRecoveryMS         int64    `json:"last_recovery_ms"`
	TotalRecoveryMS        int64    `json:"total_recovery_ms"`
	VerifiedSegments       uint64   `json:"verified_segments"`
	BrokenSegments         uint64   `json:"broken_segments"`
	SessionHealthPercent   float64  `json:"session_health_percent"`
}

type GlobalSystemState struct {
	DiskTotal  uint64  `json:"disk_total"`
	DiskAvail  uint64  `json:"disk_avail"`
	DiskUsed   uint64  `json:"disk_used"`
	CPULoad    string  `json:"cpu_load"`
	RAMPercent float64 `json:"ram_percent"`
	Uptime     string  `json:"uptime"`
}

type APIResponse struct {
	System                    GlobalSystemState              `json:"system"`
	Streams                   map[string]StreamStateSnapshot `json:"streams"`
	Files                     map[string][]FileRow           `json:"files"`
	Alerts                    []Alert                        `json:"alerts"`
	TotalWriteBytesPerSecond  float64                        `json:"total_write_bytes_per_second"`
	EstimatedRemainingSeconds int64                          `json:"estimated_remaining_seconds"`
	PipelineSelfCheck         PipelineSelfCheck              `json:"pipeline_self_check"`
}

type StreamlinkResponse struct {
	Streams map[string]json.RawMessage `json:"streams"`
}

type App struct {
	Config      Config
	configMu    sync.RWMutex
	BaseSaveDir string
	ConfigPath  string
	Streams     map[string]*StreamState
	StreamsMu   sync.RWMutex
	Auth        *authManager

	sysMu         sync.Mutex
	SysState      GlobalSystemState
	lastTotalTime uint64
	lastIdleTime  uint64

	obsMu        sync.RWMutex
	MediaQuality map[string]MediaQuality

	monitorMu           sync.Mutex
	lastMonitorRefresh  time.Time
	fileCacheMu         sync.Mutex
	lastFileScan        time.Time
	cachedFiles         map[string][]FileRow
	cachedRecordedBytes map[string]int64

	selfCheckMu      sync.Mutex
	selfCheckRunning bool
	SelfCheck        PipelineSelfCheck
}

func main() {
	if os.Getenv("LIVETOOL_LAUNCHD") != "1" || len(os.Args) != 1 {
		fmt.Fprintln(os.Stderr, "livetool 由 launchd 管理；請使用 ./scripts/manage-launchd.sh {install|start|stop|restart|status}")
		os.Exit(2)
	}
	configureLogging()
	log.Printf("[SYSTEM] process starting pid=%d version=%s", os.Getpid(), runtime.Version())

	numCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPU * 2)

	confFile, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatalf("[ERROR] 無法讀取 config.json: %v", err)
	}
	var config Config
	if err := json.Unmarshal(confFile, &config); err != nil {
		log.Fatalf("[ERROR] 解析 config.json 失敗: %v", err)
	}
	if err := validateConfig(config); err != nil {
		log.Fatalf("[ERROR] config.json 設定無效: %v", err)
	}

	log.Println("[SYSTEM] =========================================================")
	log.Println("[SYSTEM] 🚀 go_straight 多核心併發雷達核心啟動中...")
	startMacSleepPrevention()

	cwd, _ := os.Getwd()
	baseSaveDir := filepath.Join(cwd, "downloads")
	auth, generatedPassword, err := loadOrCreateAuth(authPath(cwd))
	if err != nil {
		log.Fatalf("[AUTH-ERROR] 無法初始化 Web 登入驗證: %v", err)
	}
	if generatedPassword != "" {
		log.Printf("[AUTH] INITIAL CREDENTIALS username=%s password=%s", auth.config.Username, generatedPassword)
		log.Printf("[AUTH] 請登入後妥善保存密碼；伺服器僅保存雜湊")
	}

	app := &App{
		Config:       config,
		BaseSaveDir:  baseSaveDir,
		ConfigPath:   "config.json",
		Streams:      make(map[string]*StreamState),
		MediaQuality: make(map[string]MediaQuality),
		Auth:         auth,
	}

	for _, url := range config.TargetURLs {
		parts := strings.Split(url, "@")
		if len(parts) < 2 {
			log.Printf("[WARNING] Target URL 格式錯誤 (必須包含 @): %s", url)
			continue
		}
		prefix := strings.Split(parts[1], "/")[0]
		saveDir := filepath.Join(baseSaveDir, prefix)
		_ = os.MkdirAll(saveDir, 0755)

		stream := &StreamState{
			TargetURL:   url,
			Prefix:      prefix,
			SaveDir:     saveDir,
			ProbeStatus: "⚪ 哨兵初始化，待命中...",
			ReloadChan:  make(chan struct{}, 1),
		}
		app.Streams[prefix] = stream
		log.Printf("[SYSTEM] 📡 成功載入目標雷達守備對象: @%s (儲存路徑: %s)", prefix, saveDir)
	}

	if fi, err := os.Stat(filepath.Join(cwd, "venv", "bin")); err == nil && fi.IsDir() {
		os.Setenv("PATH", filepath.Join(cwd, "venv", "bin")+":"+os.Getenv("PATH"))
		log.Println("[SYSTEM] 🐍 偵測到本地 Python venv，已成功併入 PATH 環境變數。")
	}

	app.updateDiskStatus()
	app.updateSystemResource()

	go app.configWatcher()

	for _, stream := range app.Streams {
		go app.templateProbeRadar(stream)
	}

	app.startHealthMonitoring()

	http.HandleFunc("/", app.handleIndex)
	http.HandleFunc("/api/status", app.handleAPIStatus)
	http.HandleFunc("/api/shutdown", app.handleAPIShutdown)
	http.HandleFunc("/api/probe", app.handleAPIProbe)
	http.HandleFunc("/api/probe_pause", app.handleAPIProbePause)
	http.HandleFunc("/api/restart", app.handleAPIRestart)
	http.HandleFunc("/api/restart_cluster", app.handleAPIRegionalRestart)
	http.HandleFunc("/api/logs", app.handleAPILogs)
	http.HandleFunc("/api/log_status", app.handleAPILogStatus) // 📥 註冊手動狀態紀錄 API
	http.HandleFunc("/api/session", app.handleAPISession)
	http.HandleFunc("/api/diagnostics", app.handleAPIDiagnostics)
	http.HandleFunc("/api/config", app.handleAPIConfig)
	http.HandleFunc("/login", auth.handleLogin)
	http.HandleFunc("/logout", auth.handleLogout)

	addr := fmt.Sprintf(":%d", app.Config.WebPort)
	log.Printf("=========================================================")
	log.Printf("🚀 總控儀表板監聽中: http://localhost%s", addr)
	log.Printf("=========================================================")

	server := &http.Server{
		Addr:              addr,
		Handler:           requestDiagnostics(sameOrigin(auth.middleware(http.DefaultServeMux))),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		// 錄影檔可能很大；全域 WriteTimeout 會截斷超過時限的下載。
		WriteTimeout: 0,
		IdleTimeout:  60 * time.Second,
	}
	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(shutdownSignal)
	go func() {
		sig := <-shutdownSignal
		log.Printf("[SYSTEM] 收到 %s，優雅停止所有錄影子程序", sig)
		app.StreamsMu.RLock()
		for _, stream := range app.Streams {
			stream.mu.Lock()
			if stream.RecordCancel != nil {
				stream.RecordCancel()
			}
			stream.mu.Unlock()
		}
		app.StreamsMu.RUnlock()

		deadline := time.Now().Add(4 * time.Second)
		for time.Now().Before(deadline) {
			allStopped := true
			app.StreamsMu.RLock()
			for _, stream := range app.Streams {
				stream.mu.Lock()
				if stream.StreamlinkPID != 0 || stream.FFmpegPID != 0 {
					allStopped = false
				}
				stream.mu.Unlock()
			}
			app.StreamsMu.RUnlock()
			if allStopped {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("[ERROR] Web 伺服器崩潰: %v", err)
	}
}

func (a *App) configWatcher() {
	var lastMod time.Time
	confPath := a.configFilePath()

	for {
		fi, err := os.Stat(confPath)
		if err == nil {
			if lastMod.IsZero() {
				lastMod = fi.ModTime()
			} else if fi.ModTime().After(lastMod) {
				data, err := os.ReadFile(confPath)
				if err == nil {
					var newCfg Config
					if json.Unmarshal(data, &newCfg) == nil {
						if err := validateConfig(newCfg); err != nil {
							log.Printf("[CONFIG] ⚠️ 設定檔熱載入失敗，保留目前設定: %v", err)
							lastMod = fi.ModTime()
							continue
						}
						a.configMu.Lock()
						a.Config.ProbeStart = newCfg.ProbeStart
						a.Config.ProbeEnd = newCfg.ProbeEnd
						a.Config.ProbeInterval = newCfg.ProbeInterval
						a.Config.ProbeSleepDeep = newCfg.ProbeSleepDeep
						a.configMu.Unlock()

						log.Printf("[CONFIG] 🔄 設定檔已熱載入！警戒時段變更為: %s ~ %s (Interval: %ds, SleepDeep: %ds)", newCfg.ProbeStart, newCfg.ProbeEnd, newCfg.ProbeInterval, newCfg.ProbeSleepDeep)

						a.StreamsMu.RLock()
						for _, s := range a.Streams {
							select {
							case s.ReloadChan <- struct{}{}:
							default:
							}
						}
						a.StreamsMu.RUnlock()

						lastMod = fi.ModTime()
					}
				}
			}
		}
		time.Sleep(5 * time.Second)
	}
}

func (a *App) getProbeParams() (interval int, sleepDeep int) {
	a.configMu.RLock()
	defer a.configMu.RUnlock()
	return a.Config.ProbeInterval, a.Config.ProbeSleepDeep
}

func (a *App) checkTimeWindowSafe() (bool, time.Duration) {
	a.configMu.RLock()
	pStart := a.Config.ProbeStart
	pEnd := a.Config.ProbeEnd
	a.configMu.RUnlock()

	inWindow, wait, err := calculateProbeWindow(time.Now(), pStart, pEnd)
	if err != nil {
		log.Printf("[CONFIG] ⚠️ 無法計算警戒時段: %v", err)
		return false, time.Minute
	}
	return inWindow, wait
}

func calculateProbeWindow(now time.Time, pStart, pEnd string) (bool, time.Duration, error) {
	if pStart == pEnd {
		return true, 0, nil
	}
	todayStr := now.Format("2006-01-02")

	start, err1 := time.ParseInLocation("2006-01-02 15:04", todayStr+" "+pStart, now.Location())
	end, err2 := time.ParseInLocation("2006-01-02 15:04", todayStr+" "+pEnd, now.Location())
	if err1 != nil || err2 != nil {
		return false, 0, fmt.Errorf("時間必須使用 HH:MM 格式（目前為 %q ~ %q）", pStart, pEnd)
	}

	if end.Before(start) {
		if now.Before(end) {
			start = start.AddDate(0, 0, -1)
		} else {
			end = end.AddDate(0, 0, 1)
		}
	}
	if !now.Before(start) && now.Before(end) {
		return true, 0, nil
	}

	var nextStart time.Time
	if now.Before(start) {
		nextStart = start
	} else {
		nextStart = start.AddDate(0, 0, 1)
	}

	diff := nextStart.Sub(now)
	if diff < 0 {
		return true, 0, nil
	}
	return false, diff, nil
}

func validateConfig(cfg Config) error {
	if err := validateTargetURLs(cfg.TargetURLs); err != nil {
		return err
	}
	if cfg.WebPort < 1 || cfg.WebPort > 65535 {
		return fmt.Errorf("web_port 必須介於 1 到 65535")
	}
	if cfg.ProbeInterval <= 0 {
		return fmt.Errorf("probe_interval 必須大於 0")
	}
	if cfg.ProbeSleepDeep <= 0 {
		return fmt.Errorf("probe_sleep_deep 必須大於 0")
	}
	reference := time.Date(2026, 1, 1, 12, 0, 0, 0, time.Local)
	for name, value := range map[string]string{"probe_start": cfg.ProbeStart, "probe_end": cfg.ProbeEnd} {
		if _, err := time.ParseInLocation("15:04", value, reference.Location()); err != nil {
			return fmt.Errorf("%s 必須使用 HH:MM 格式（目前為 %q）", name, value)
		}
	}
	_, _, err := calculateProbeWindow(reference, cfg.ProbeStart, cfg.ProbeEnd)
	return err
}

func (a *App) templateProbeRadar(s *StreamState) {
	for {
	RE_LOOP:
		s.mu.Lock()
		isRec := s.IsRecording
		isProb := s.IsProbing
		probePaused := s.ProbePaused
		s.mu.Unlock()

		if isRec {
			a.updateProbeStatus(s, "🟢 已交接錄影 (哨兵常駐監聽中)")
			time.Sleep(2 * time.Second)
			continue
		}

		if isProb {
			time.Sleep(1 * time.Second)
			continue
		}
		if probePaused {
			a.updateProbeStatus(s, "⏸ 自動刺探已暫停")
			time.Sleep(time.Second)
			continue
		}

		select {
		case <-s.ReloadChan:
			log.Printf("[RADAR] [@%s] 偵測到配置熱變更，重新對齊時間窗口。", s.Prefix)
		default:
		}

		inWindow, timeToStart := a.checkTimeWindowSafe()
		probeInterval, probeSleepDeep := a.getProbeParams()

		if !inWindow {
			totalSleepSec := int(timeToStart.Seconds())

			if totalSleepSec > probeSleepDeep {
				totalSleepSec = probeSleepDeep
			}

			if totalSleepSec <= 0 {
				time.Sleep(5 * time.Second)
				goto RE_LOOP
			}

			log.Printf("[RADAR] [@%s] 開播警戒時段外。雷達進入深度休眠 %d 秒 (設定最大深睡: %d 秒)。", s.Prefix, totalSleepSec, probeSleepDeep)

			for i := totalSleepSec; i > 0; {
				s.mu.Lock()
				if s.IsRecording || s.IsProbing {
					s.mu.Unlock()
					break
				}
				paused := s.ProbePaused
				s.mu.Unlock()

				select {
				case <-s.ReloadChan:
					log.Printf("[RADAR] [@%s] 深睡中收到排程熱載入變更！立刻打破長休眠重新計算。", s.Prefix)
					goto RE_LOOP
				default:
				}

				h, m, sTime := i/3600, (i%3600)/60, i%60
				if paused {
					a.updateProbeStatus(s, fmt.Sprintf("⏸ 非戰備倒數已暫停 (倒數 %02d:%02d:%02d)", h, m, sTime))
				} else {
					a.updateProbeStatus(s, fmt.Sprintf("💤 非戰備休眠中 (倒數 %02d:%02d:%02d)", h, m, sTime))
					i--
				}
				time.Sleep(1 * time.Second)
			}
			continue
		}

		a.updateProbeStatus(s, "🟣 發送網路請求中 (檢測開播狀態...)")
		log.Printf("[RADAR] [@%s] 警戒戰備期間，啟動常規流狀態刺探...", s.Prefix)

		if a.checkLiveStatusAndLog(s.Prefix, s.TargetURL) {
			log.Printf("[RADAR] [@%s] 🎯 命中！確認主播已開播，準備移交錄影核心管線。", s.Prefix)
			a.startRecordingWrapper(s)
		} else {
			waitTime := probeInterval + rand.Intn(21)
			log.Printf("[RADAR] [@%s] 🔎 刺探結果：尚未開播。隨機冷卻下一次檢測： %d 秒後。", s.Prefix, waitTime)

			for i := waitTime; i > 0; {
				s.mu.Lock()
				if s.IsRecording || s.IsProbing {
					s.mu.Unlock()
					break
				}
				paused := s.ProbePaused
				s.mu.Unlock()

				select {
				case <-s.ReloadChan:
					log.Printf("[RADAR] [@%s] 戰備倒數期間收到排程熱變更，立刻重新對齊。", s.Prefix)
					goto RE_LOOP
				default:
				}

				if paused {
					a.updateProbeStatus(s, fmt.Sprintf("⏸ 刺探倒數已暫停 (倒數 %d 秒)", i))
				} else {
					a.updateProbeStatus(s, fmt.Sprintf("🟡 刺探待命中 (倒數 %d 秒)", i))
					i--
				}
				time.Sleep(1 * time.Second)
			}
		}
	}
}

func (a *App) startRecordingWrapper(s *StreamState) {
	s.mu.Lock()
	if s.IsRecording {
		s.mu.Unlock()
		log.Printf("[PROBE] [@%s] 已有錄影管線運行，略過重複啟動。", s.Prefix)
		return
	}
	s.IsRecording = true
	s.SessionID = newRecordingSessionID(s.Prefix)
	s.RecordingStartedAt = time.Now()
	s.SessionSegmentCount = 0
	s.SessionRestartCount = 0
	s.SessionRecordedBytes = 0
	s.SessionGapTotal = 0
	s.SessionMaxGap = 0
	s.RecoveryStartedAt = time.Time{}
	s.LastRecoveryDuration = 0
	s.TotalRecoveryDuration = 0
	s.VerifiedSegments = 0
	s.BrokenSegments = 0
	s.LastSuccessfulWrite = time.Time{}
	sessionID := s.SessionID
	s.mu.Unlock()
	appendStreamEvent(s, "info", "recording_started", "開始新的錄影 Session")

	log.Printf("[PROBE] 發現目標正在直播 channel=%s recording_id=%s", s.Prefix, sessionID)

	for {
		s.mu.Lock()
		s.ProbeStatus = "🟢 已交接錄影 (哨兵常駐監聽中)"
		s.RecordCtx, s.RecordCancel = context.WithCancel(context.Background())
		ctx := s.RecordCtx
		cancel := s.RecordCancel
		s.mu.Unlock()

		a.runRecordEngine(ctx, s, sessionID)
		wasCanceled := ctx.Err() != nil
		cancel()

		if wasCanceled {
			log.Printf("[PROBE] [@%s] 錄影 Context 收到終止訊號，正式登出管線。", s.Prefix)
			goto END_RECORD
		} else {
			s.mu.Lock()
			s.RecoveryStartedAt = time.Now()
			s.mu.Unlock()
			a.updateProbeStatus(s, "🟡 管線意外斷開，2秒後確認是否為微斷流...")
			log.Printf("[PROBE] ⚠️ [@%s] 錄影管線意外中斷！正在等待 2 秒進行斷流判定...", s.Prefix)
			time.Sleep(2 * time.Second)

			if a.checkLiveStatusAndLog(s.Prefix, s.TargetURL) {
				s.mu.Lock()
				s.RecordingRestartCount++
				s.SessionRestartCount++
				s.mu.Unlock()
				appendStreamEvent(s, "warning", "pipeline_reconnect", "微斷流後重新建立錄影管線")
				log.Printf("[PROBE] 🔄 偵測到主播 @%s 仍在線 (確認為微斷流)，雷達立即接回重錄！", s.Prefix)
				continue
			} else {
				log.Printf("[PROBE] 🎬 主播 @%s 已確認下播，正式結束本次錄影任務。", s.Prefix)
				goto END_RECORD
			}
		}
	}

END_RECORD:
	s.mu.Lock()
	s.IsRecording = false
	if s.RecordCancel != nil {
		s.RecordCancel()
	}
	s.mu.Unlock()
	appendStreamEvent(s, "info", "recording_stopped", "錄影 Session 已結束")
	log.Printf("[PROBE] [@%s] 釋放鎖控，進入 10 秒緩衝期防止極端高頻迴圈。", s.Prefix)
	time.Sleep(10 * time.Second)
}

var recordingSessionSequence atomic.Uint64

func newRecordingSessionID(prefix string) string {
	return fmt.Sprintf("%s-%s-%d-%04d", time.Now().Format("20060102T150405"), prefix, os.Getpid(), recordingSessionSequence.Add(1))
}

func (a *App) runRecordEngine(ctx context.Context, s *StreamState, sessionID string) {
	if err := recordingPreflight(s.SaveDir); err != nil {
		s.mu.Lock()
		s.RecordingStartFailures++
		s.mu.Unlock()
		a.recordStreamError(s, "錄影預檢失敗: "+err.Error())
		appendStreamEvent(s, "error", "recording_preflight_failed", err.Error())
		log.Printf("[MINER-ERROR] channel=%s recording_id=%s stage=preflight error=%q", s.Prefix, sessionID, err)
		return
	}
	tsFile := newRecordingPath(s.SaveDir, time.Now())
	s.mu.Lock()
	s.SegmentStartedAt = time.Now()
	s.SessionSegmentCount++
	s.WriteBytesPerSecond = 0
	s.StreamlinkPID = 0
	s.FFmpegPID = 0
	s.FFmpegBitrate = ""
	s.FFmpegSpeed = ""
	s.PipelineState = "STARTING"
	s.SelectedQuality = ""
	s.SelectedStreamType = ""
	s.mu.Unlock()
	defer a.inspectCompletedSessionSegment(s, sessionID, tsFile)
	defer func() {
		s.mu.Lock()
		s.StreamlinkPID = 0
		s.FFmpegPID = 0
		s.WriteBytesPerSecond = 0
		s.PipelineState = "STOPPED"
		s.mu.Unlock()
	}()

	log.Printf("[MINER-CONNECT] channel=%s recording_id=%s target=%s", s.Prefix, sessionID, s.TargetURL)
	log.Printf("[MINER-JOB] channel=%s recording_id=%s output=%s", s.Prefix, sessionID, filepath.Base(tsFile))

	startTime := time.Now()

	// 跨平台調整：移除不相容的 ionice
	var streamlinkCmd, ffmpegCmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		streamlinkCmd = exec.CommandContext(ctx, "streamlink", streamlinkRecordingArgs(
			s.TargetURL,
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",
		)...)
		ffmpegCmd = exec.CommandContext(ctx,
			"ffmpeg", "-hide_banner", "-loglevel", "info", "-progress", "pipe:2", "-y", "-thread_queue_size", "1024", "-i", "pipe:0", "-c", "copy", "-f", "mpegts", tsFile,
		)
	} else {
		streamlinkArgs := append([]string{"-c", "2", "-n", "0", "streamlink"}, streamlinkRecordingArgs(
			s.TargetURL,
			"Mozilla/5.0 (X11; Linux x86_64; rv:126.0) Gecko/20100101 Firefox/126.0",
		)...)
		streamlinkCmd = exec.CommandContext(ctx, "ionice", streamlinkArgs...)
		ffmpegCmd = exec.CommandContext(ctx, "ionice", "-c", "2", "-n", "0",
			"ffmpeg", "-hide_banner", "-loglevel", "info", "-progress", "pipe:2", "-y", "-thread_queue_size", "1024", "-i", "pipe:0", "-c", "copy", "-f", "mpegts", tsFile,
		)
		// 移除 macOS 不支援的 LD_PRELOAD
		streamlinkCmd.Env = append(os.Environ(), "LD_PRELOAD=/usr/lib/libjemalloc.so")
		ffmpegCmd.Env = append(os.Environ(), "LD_PRELOAD=/usr/lib/libjemalloc.so")
	}

	// 🔥【進程組安全升級】設定獨立進程組 ID，防止 ffmpeg 脫管變成孤兒或殭屍
	if runtime.GOOS != "windows" {
		streamlinkCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		ffmpegCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}

	configureGracefulCancellation(streamlinkCmd)
	configureGracefulCancellation(ffmpegCmd)

	var lastStreamlinkMsg string = "Waiting for data..."
	var lastFfmpegMsg string = "bitrate=0kb/s speed=0x"
	var msgMu sync.Mutex
	streamlinkTail := newStderrTail(20)
	ffmpegTail := newStderrTail(20)
	var stderrWG sync.WaitGroup

	slStderr, err := streamlinkCmd.StderrPipe()
	if err != nil {
		a.recordStreamError(s, fmt.Sprintf("streamlink stderr pipe: %v", err))
		log.Printf("[MINER-ERROR] channel=%s recording_id=%s process=streamlink stage=stderr_pipe error=%q", s.Prefix, sessionID, err)
		return
	}
	ffStderr, err := ffmpegCmd.StderrPipe()
	if err != nil {
		a.recordStreamError(s, fmt.Sprintf("ffmpeg stderr pipe: %v", err))
		log.Printf("[MINER-ERROR] channel=%s recording_id=%s process=ffmpeg stage=stderr_pipe error=%q", s.Prefix, sessionID, err)
		return
	}
	stderrWG.Add(2)
	go func() {
		defer stderrWG.Done()
		scanner := bufio.NewScanner(slStderr)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			msgMu.Lock()
			streamlinkTail.Add(line)
			if strings.Contains(line, "[") {
				lastStreamlinkMsg = strings.TrimSpace(line)
			}
			msgMu.Unlock()
			if quality, streamType, ok := parseStreamlinkOpeningLine(line); ok {
				s.mu.Lock()
				s.SelectedQuality = quality
				s.SelectedStreamType = streamType
				s.mu.Unlock()
				log.Printf("[MINER-QUALITY] channel=%s recording_id=%s selected=%q stream_type=%q", s.Prefix, sessionID, quality, streamType)
			}
		}
		if scanErr := scanner.Err(); scanErr != nil {
			log.Printf("[MINER-WARNING] channel=%s recording_id=%s process=streamlink stderr_scanner_error=%q", s.Prefix, sessionID, scanErr)
		}
	}()
	go func() {
		defer stderrWG.Done()
		scanner := bufio.NewScanner(ffStderr)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		var currentBitrate, currentSpeed string
		for scanner.Scan() {
			line := scanner.Text()
			msgMu.Lock()
			ffmpegTail.Add(line)
			msgMu.Unlock()
			if strings.HasPrefix(line, "bitrate=") {
				currentBitrate = strings.TrimPrefix(line, "bitrate=")
			}
			if strings.HasPrefix(line, "speed=") {
				currentSpeed = strings.TrimPrefix(line, "speed=")
				msgMu.Lock()
				lastFfmpegMsg = fmt.Sprintf("bitrate=%s speed=%s", currentBitrate, currentSpeed)
				msgMu.Unlock()
				s.mu.Lock()
				s.FFmpegBitrate = strings.TrimSpace(currentBitrate)
				s.FFmpegSpeed = strings.TrimSpace(currentSpeed)
				s.mu.Unlock()
			}
		}
		if scanErr := scanner.Err(); scanErr != nil {
			log.Printf("[MINER-WARNING] channel=%s recording_id=%s process=ffmpeg stderr_scanner_error=%q", s.Prefix, sessionID, scanErr)
		}
	}()

	pipe, err := streamlinkCmd.StdoutPipe()
	if err != nil {
		_ = slStderr.Close()
		_ = ffStderr.Close()
		stderrWG.Wait()
		a.recordStreamError(s, fmt.Sprintf("stream pipe: %v", err))
		log.Printf("[MINER-ERROR] channel=%s recording_id=%s stage=stdout_pipe error=%q", s.Prefix, sessionID, err)
		return
	}
	ffmpegCmd.Stdin = pipe

	if err := streamlinkCmd.Start(); err != nil {
		_ = slStderr.Close()
		_ = ffStderr.Close()
		stderrWG.Wait()
		a.recordStreamError(s, fmt.Sprintf("streamlink start: %v", err))
		log.Printf("[MINER-ERROR] channel=%s recording_id=%s process=streamlink stage=start error=%q", s.Prefix, sessionID, err)
		return
	}
	if err := ffmpegCmd.Start(); err != nil {
		a.recordStreamError(s, fmt.Sprintf("ffmpeg start: %v", err))
		log.Printf("[MINER-ERROR] channel=%s recording_id=%s process=ffmpeg stage=start error=%q", s.Prefix, sessionID, err)
		killCommandGroup(streamlinkCmd)
		_ = streamlinkCmd.Wait()
		_ = ffStderr.Close()
		stderrWG.Wait()
		return
	}

	log.Printf("[MINER-START] channel=%s recording_id=%s streamlink_pid=%d ffmpeg_pid=%d",
		s.Prefix, sessionID, streamlinkCmd.Process.Pid, ffmpegCmd.Process.Pid)
	s.mu.Lock()
	s.StreamlinkPID = streamlinkCmd.Process.Pid
	s.FFmpegPID = ffmpegCmd.Process.Pid
	s.PipelineState = "RECORDING"
	s.mu.Unlock()
	appendStreamEvent(s, "info", "segment_started", "Streamlink 與 FFmpeg 管線建立完成")

	engineDone := make(chan struct{})
	var lastSize int64 = 0
	var firstWrite atomic.Bool
	var startupTimedOut atomic.Bool
	var writeStalled atomic.Bool
	watchdogDone := make(chan struct{})
	go func() {
		timer := time.NewTimer(75 * time.Second)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-watchdogDone:
			return
		case <-timer.C:
			if firstWrite.Load() {
				return
			}
			startupTimedOut.Store(true)
			message := "錄影管線啟動 75 秒後仍未寫入第一筆資料"
			s.mu.Lock()
			s.RecordingStartFailures++
			s.mu.Unlock()
			a.recordStreamError(s, message)
			s.mu.Lock()
			s.PipelineState = "START_FAILED"
			s.mu.Unlock()
			appendStreamEvent(s, "error", "first_write_timeout", message)
			log.Printf("[MINER-ERROR] channel=%s recording_id=%s stage=first_write_watchdog timeout_seconds=75", s.Prefix, sessionID)
			requestCommandStop(ffmpegCmd)
			requestCommandStop(streamlinkCmd)
		}
	}()

	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		lastGrowthAt := time.Now()
		delayReported := false

		for {
			select {
			case <-ctx.Done():
				return
			case <-engineDone:
				return
			case <-ticker.C:
				s.mu.Lock()
				isRec := s.IsRecording
				s.mu.Unlock()
				if !isRec {
					return
				}

				fi, err := os.Stat(tsFile)
				currentSize := lastSize
				if err == nil {
					currentSize = fi.Size()
					s.mu.Lock()
					s.LatestFile = filepath.Base(tsFile)
					s.LatestMTime = fi.ModTime().Format("2006-01-02 15:04:05")
					s.LatestSize = currentSize
					if currentSize > lastSize {
						now := time.Now()
						bytesDiff := currentSize - lastSize
						if !firstWrite.Load() && !s.RecoveryStartedAt.IsZero() {
							gap := now.Sub(s.LastSuccessfulWrite)
							if s.LastSuccessfulWrite.IsZero() || gap < 0 {
								gap = 0
							}
							recovery := now.Sub(s.RecoveryStartedAt)
							s.SessionGapTotal += gap
							if gap > s.SessionMaxGap {
								s.SessionMaxGap = gap
							}
							s.LastRecoveryDuration = recovery
							s.TotalRecoveryDuration += recovery
							s.RecoveryStartedAt = time.Time{}
						}
						s.SessionRecordedBytes += bytesDiff
						s.LastSuccessfulWrite = now
						lastGrowthAt = now
						firstWrite.Store(true)
						if delayReported {
							delayReported = false
							s.PipelineState = "RECORDING"
							log.Printf("[MINER-RECOVERED] channel=%s recording_id=%s stage=write_watchdog size_bytes=%d", s.Prefix, sessionID, currentSize)
						}
					}
					s.mu.Unlock()

					bytesDiff := currentSize - lastSize
					lastSize = currentSize
					speedMb := float64(bytesDiff) / (1024 * 1024) / 3.0
					s.mu.Lock()
					s.WriteBytesPerSecond = float64(bytesDiff) / 3.0
					s.mu.Unlock()
					appendTrendPoint(s, float64(bytesDiff)/3.0)
					totalMb := float64(currentSize) / (1024 * 1024)

					lifespanSec := int(time.Since(startTime).Seconds())
					runTimeStr := fmt.Sprintf("%d:%02d", lifespanSec/60, lifespanSec%60)

					msgMu.Lock()
					slInfo := lastStreamlinkMsg
					ffInfo := lastFfmpegMsg
					msgMu.Unlock()

					log.Printf("[MINER-PROGRESS] channel=%s recording_id=%s uptime=%s total_mb=%.2f speed_mb_s=%.2f streamlink=%q ffmpeg=%q",
						s.Prefix, sessionID, runTimeStr, totalMb, speedMb, slInfo, ffInfo)
				}

				delayed, stalled := recordingWriteHealth(firstWrite.Load(), lastGrowthAt, time.Now())
				if delayed && !delayReported {
					delayReported = true
					s.mu.Lock()
					s.PipelineState = "WRITE_DELAYED"
					s.mu.Unlock()
					appendStreamEvent(s, "warning", "write_delayed", "錄影檔已 30 秒沒有增加，持續確認中")
					log.Printf("[MINER-WARNING] channel=%s recording_id=%s stage=write_watchdog stalled_seconds=30 size_bytes=%d stat_error=%q", s.Prefix, sessionID, currentSize, err)
				}
				if stalled && writeStalled.CompareAndSwap(false, true) {
					message := "錄影檔已連續 60 秒沒有增加或已消失，立即重建錄影管線"
					s.mu.Lock()
					s.PipelineState = "WRITE_STALLED"
					s.mu.Unlock()
					a.recordStreamError(s, message)
					appendStreamEvent(s, "error", "write_stalled", message)
					log.Printf("[MINER-ERROR] channel=%s recording_id=%s stage=write_watchdog stalled_seconds=60 size_bytes=%d stat_error=%q", s.Prefix, sessionID, currentSize, err)
					requestCommandStop(ffmpegCmd)
					requestCommandStop(streamlinkCmd)
				}
			}
		}
	}()

	streamlinkErr, ffmpegErr := waitRecordingCommands(streamlinkCmd, ffmpegCmd)
	close(watchdogDone)
	stderrWG.Wait()
	expectedStop := ctx.Err() != nil || startupTimedOut.Load()

	msgMu.Lock()
	finalStreamlinkMsg := streamlinkTail.String()
	finalFFmpegMsg := ffmpegTail.String()
	if finalStreamlinkMsg == "" {
		finalStreamlinkMsg = lastStreamlinkMsg
	}
	if finalFFmpegMsg == "" {
		finalFFmpegMsg = lastFfmpegMsg
	}
	msgMu.Unlock()
	log.Printf("[MINER-EXIT] channel=%s recording_id=%s process=streamlink exit_code=%d expected_stop=%t duration_ms=%d error=%q last_stderr=%q",
		s.Prefix, sessionID, processExitCode(streamlinkErr), expectedStop, time.Since(startTime).Milliseconds(), streamlinkErr, finalStreamlinkMsg)
	log.Printf("[MINER-EXIT] channel=%s recording_id=%s process=ffmpeg exit_code=%d expected_stop=%t duration_ms=%d error=%q last_stderr=%q",
		s.Prefix, sessionID, processExitCode(ffmpegErr), expectedStop, time.Since(startTime).Milliseconds(), ffmpegErr, finalFFmpegMsg)
	if ffmpegErr != nil && !expectedStop {
		s.mu.Lock()
		s.FFmpegAbnormalExits++
		s.mu.Unlock()
		a.recordStreamError(s, fmt.Sprintf("ffmpeg exit code %d: %v", processExitCode(ffmpegErr), ffmpegErr))
		appendStreamEvent(s, "error", "ffmpeg_exit", fmt.Sprintf("FFmpeg 非正常退出，exit code %d", processExitCode(ffmpegErr)))
	} else if streamlinkErr != nil && !expectedStop {
		a.recordStreamError(s, fmt.Sprintf("streamlink exit code %d: %v", processExitCode(streamlinkErr), streamlinkErr))
		appendStreamEvent(s, "warning", "streamlink_exit", fmt.Sprintf("Streamlink 非正常退出，exit code %d", processExitCode(streamlinkErr)))
	}

	close(engineDone)

	lifespan := time.Since(startTime)

	fi, err := os.Stat(tsFile)
	if err != nil || fi.Size() == 0 || lifespan < 5*time.Second {
		_ = os.Remove(tsFile)
		log.Printf("\x1b[33m%s [MINER-RECYCLE] 🗑️ Session too short or empty. Junk file removed.\x1b[0m", time.Now().Format("15:04:05"))
	} else {
		log.Printf("\x1b[32m%s [MINER-SUMMARY] 🎉 Storage completed! Size: %.2f MB.\x1b[0m", time.Now().Format("15:04:05"), float64(fi.Size())/(1024*1024))
		a.inspectMediaAsync(tsFile)
	}
}

func processExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

func (a *App) recordStreamError(s *StreamState, message string) {
	s.mu.Lock()
	s.LastError = message
	s.LastErrorAt = time.Now()
	s.mu.Unlock()
}

func (a *App) updateProbeStatus(s *StreamState, status string) {
	s.mu.Lock()
	s.ProbeStatus = status
	s.mu.Unlock()
}

func (a *App) checkLiveStatusAndLog(prefix string, targetURL string) bool {
	started := time.Now()
	probeSucceeded := false
	defer func() { a.recordProbeMetrics(prefix, time.Since(started), probeSucceeded) }()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "streamlink", "--json", targetURL)
	output, err := cmd.CombinedOutput()
	trimmedOutput := strings.TrimSpace(string(output))

	log.Printf("[CLI-PROBE] [@%s] 執行指令: streamlink --json %s", prefix, targetURL)

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			a.setChannelError(prefix, "probe timed out after 90s")
			log.Printf("[CLI-PROBE] [@%s] 刺探超過 90 秒，已自動終止。", prefix)
		} else if trimmedOutput != "" {
			const maxErrorLog = 1000
			if len(trimmedOutput) > maxErrorLog {
				trimmedOutput = trimmedOutput[:maxErrorLog] + "…"
			}
			a.setChannelError(prefix, "probe failed: "+trimmedOutput)
			log.Printf("[CLI-PROBE] [@%s] 刺探失敗: %s", prefix, trimmedOutput)
		} else {
			a.setChannelError(prefix, fmt.Sprintf("probe failed: %v", err))
		}
		return false
	}

	var res StreamlinkResponse
	if err := json.Unmarshal(output, &res); err != nil {
		a.setChannelError(prefix, fmt.Sprintf("probe JSON parse: %v", err))
		log.Printf("[CLI-PARSE] [@%s] JSON 解析失敗 (可能非預期格式): %v", prefix, err)
		return false
	}
	probeSucceeded = true
	isLive := len(res.Streams) > 0
	qualities := make([]string, 0, len(res.Streams))
	for quality := range res.Streams {
		qualities = append(qualities, quality)
	}
	sort.Strings(qualities)
	a.StreamsMu.RLock()
	stream := a.Streams[prefix]
	a.StreamsMu.RUnlock()
	if stream != nil {
		stream.mu.Lock()
		stream.AvailableQualities = append(stream.AvailableQualities[:0], qualities...)
		stream.mu.Unlock()
	}
	log.Printf("[CLI-PROBE] [@%s] 刺探完成: live=%t, streams=%d, qualities=%q", prefix, isLive, len(res.Streams), qualities)
	return isLive
}

func (a *App) recordProbeMetrics(prefix string, duration time.Duration, succeeded bool) {
	a.StreamsMu.RLock()
	s := a.Streams[prefix]
	a.StreamsMu.RUnlock()
	if s == nil {
		return
	}
	s.mu.Lock()
	s.ProbeAttempts++
	s.ProbeTotalDuration += duration
	if succeeded {
		s.ProbeSuccesses++
	}
	s.mu.Unlock()
}

func (a *App) setChannelError(prefix, message string) {
	a.StreamsMu.RLock()
	s := a.Streams[prefix]
	a.StreamsMu.RUnlock()
	if s != nil {
		a.recordStreamError(s, message)
	}
}

func (a *App) updateDiskStatus() {
	var total, used, avail uint64

	if runtime.GOOS == "darwin" {
		var stat unix.Statfs_t
		if err := unix.Statfs(a.BaseSaveDir, &stat); err == nil {
			total = stat.Blocks * uint64(stat.Bsize)
			avail = stat.Bavail * uint64(stat.Bsize)
			used = total - (stat.Bfree * uint64(stat.Bsize))
		}
	} else {
		cmd := exec.Command("df", "-B1", a.BaseSaveDir)
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			if len(lines) >= 2 {
				fields := strings.Fields(lines[1])
				if len(fields) >= 4 {
					total, _ = strconv.ParseUint(fields[1], 10, 64)
					used, _ = strconv.ParseUint(fields[2], 10, 64)
					avail, _ = strconv.ParseUint(fields[3], 10, 64)
				}
			}
		}
	}

	a.sysMu.Lock()
	a.SysState.DiskTotal = total
	a.SysState.DiskUsed = used
	a.SysState.DiskAvail = avail
	a.sysMu.Unlock()
}

func (a *App) updateSystemResource() {
	a.updateSystemUptime()

	if runtime.GOOS == "darwin" {
		// macOS CPU 負載估算 (透過 top 抓取 CPU usage)
		cmd := exec.Command("top", "-l", "1", "-n", "0")
		if out, err := cmd.Output(); err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				if strings.Contains(line, "CPU usage:") {
					parts := strings.Split(line, ",")
					for _, p := range parts {
						if strings.Contains(p, "idle") {
							valStr := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(p, "idle", ""), "%", ""))
							if idleVal, err := strconv.ParseFloat(valStr, 64); err == nil {
								cpuPercent := 100.0 - idleVal
								a.sysMu.Lock()
								a.SysState.CPULoad = fmt.Sprintf("%.1f%%", cpuPercent)
								a.sysMu.Unlock()
							}
						}
					}
				}
			}
		}

		// macOS 記憶體使用率估算 (透過 vm_stat 與 sysctl hw.memsize)
		cmdMem := exec.Command("sysctl", "-n", "hw.memsize")
		if out, err := cmdMem.Output(); err == nil {
			if totalBytes, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64); err == nil && totalBytes > 0 {
				cmdVm := exec.Command("vm_stat")
				if outVm, err := cmdVm.Output(); err == nil {
					var freePages, inactivePages float64
					pageSize := 4096.0
					for _, line := range strings.Split(string(outVm), "\n") {
						parts := strings.Split(line, ":")
						if len(parts) == 2 {
							key := strings.TrimSpace(parts[0])
							valStr := strings.TrimSpace(strings.ReplaceAll(parts[1], ".", ""))
							val, _ := strconv.ParseFloat(valStr, 64)
							if key == "Pages free" {
								freePages = val
							} else if key == "Pages inactive" {
								inactivePages = val
							}
						}
					}
					freeBytes := (freePages + inactivePages) * pageSize
					usedBytes := totalBytes - freeBytes
					a.sysMu.Lock()
					a.SysState.RAMPercent = (usedBytes / totalBytes) * 100
					a.sysMu.Unlock()
				}
			}
		}
	} else {
		// Linux 邏輯維持不變
		statData, err := os.ReadFile("/proc/stat")
		if err == nil {
			lines := strings.Split(string(statData), "\n")
			if len(lines) > 0 && strings.HasPrefix(lines[0], "cpu ") {
				fields := strings.Fields(lines[0])
				if len(fields) >= 5 {
					var user, nice, system, idle, iowait, irq, softirq uint64
					user, _ = strconv.ParseUint(fields[1], 10, 64)
					nice, _ = strconv.ParseUint(fields[2], 10, 64)
					system, _ = strconv.ParseUint(fields[3], 10, 64)
					idle, _ = strconv.ParseUint(fields[4], 10, 64)
					if len(fields) > 5 {
						iowait, _ = strconv.ParseUint(fields[5], 10, 64)
					}
					if len(fields) > 6 {
						irq, _ = strconv.ParseUint(fields[6], 10, 64)
					}
					if len(fields) > 7 {
						softirq, _ = strconv.ParseUint(fields[7], 10, 64)
					}

					currentIdle := idle + iowait
					currentNonIdle := user + nice + system + irq + softirq
					currentTotal := currentIdle + currentNonIdle

					a.sysMu.Lock()
					if a.lastTotalTime > 0 && currentTotal > a.lastTotalTime {
						totalDiff := currentTotal - a.lastTotalTime
						idleDiff := currentIdle - a.lastIdleTime
						cpuPercent := (float64(totalDiff-idleDiff) / float64(totalDiff)) * 100
						a.SysState.CPULoad = fmt.Sprintf("%.1f%%", cpuPercent)
					} else {
						a.SysState.CPULoad = "計算中..."
					}
					a.sysMu.Unlock()

					a.lastTotalTime = currentTotal
					a.lastIdleTime = currentIdle
				}
			}
		}

		memData, err := os.ReadFile("/proc/meminfo")
		if err == nil {
			var memTotal, memAvail float64
			lines := strings.Split(string(memData), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "MemTotal:") {
					fields := strings.Fields(line)
					if len(fields) > 1 {
						memTotal, _ = strconv.ParseFloat(fields[1], 64)
					}
				}
				if strings.HasPrefix(line, "MemAvailable:") {
					fields := strings.Fields(line)
					if len(fields) > 1 {
						memAvail, _ = strconv.ParseFloat(fields[1], 64)
					}
					break
				}
			}
			if memTotal > 0 {
				a.sysMu.Lock()
				a.SysState.RAMPercent = ((memTotal - memAvail) / memTotal) * 100
				a.sysMu.Unlock()
			}
		}
	}
}

func (a *App) updateSystemUptime() {
	var seconds uint64

	if runtime.GOOS == "darwin" {
		cmd := exec.Command("sysctl", "-n", "kern.boottime")
		if out, err := cmd.Output(); err == nil {
			// 輸出格式類似: { sec = 1718000000, usec = ... }
			sStr := string(out)
			if idx := strings.Index(sStr, "sec = "); idx != -1 {
				sub := sStr[idx+6:]
				if commaIdx := strings.Index(sub, ","); commaIdx != -1 {
					if bootSec, err := strconv.ParseUint(strings.TrimSpace(sub[:commaIdx]), 10, 64); err == nil {
						currentSec := uint64(time.Now().Unix())
						if currentSec > bootSec {
							seconds = currentSec - bootSec
						}
					}
				}
			}
		}
	} else {
		uptimeData, err := os.ReadFile("/proc/uptime")
		if err != nil {
			return
		}

		fields := strings.Fields(string(uptimeData))
		if len(fields) == 0 {
			return
		}

		secondsFloat, err := strconv.ParseFloat(fields[0], 64)
		if err != nil {
			return
		}
		seconds = uint64(secondsFloat)
	}

	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60

	var uptime string
	if days > 0 {
		uptime = fmt.Sprintf("%dd %02d:%02d:%02d", days, hours, minutes, secs)
	} else {
		uptime = fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
	}

	a.sysMu.Lock()
	a.SysState.Uptime = uptime
	a.sysMu.Unlock()
}

func (a *App) getAPIResponseSnapshot() APIResponse {
	a.sysMu.Lock()
	sysState := a.SysState
	a.sysMu.Unlock()

	a.StreamsMu.RLock()
	defer a.StreamsMu.RUnlock()

	snapshots := make(map[string]StreamStateSnapshot, len(a.Streams))
	for prefix, s := range a.Streams {
		s.mu.Lock()
		probeRate := 0.0
		probeAverageMS := 0.0
		if s.ProbeAttempts > 0 {
			probeRate = float64(s.ProbeSuccesses) / float64(s.ProbeAttempts) * 100
			probeAverageMS = float64(s.ProbeTotalDuration.Microseconds()) / 1000 / float64(s.ProbeAttempts)
		}
		sessionHealth := sessionHealthPercent(s.RecordingStartedAt, s.SessionGapTotal, s.BrokenSegments, time.Now())
		snapshots[prefix] = StreamStateSnapshot{
			TargetURL:              s.TargetURL,
			Prefix:                 s.Prefix,
			SaveDir:                s.SaveDir,
			IsRecording:            s.IsRecording,
			IsProbing:              s.IsProbing,
			ProbePaused:            s.ProbePaused,
			ProbeStatus:            s.ProbeStatus,
			LatestFile:             s.LatestFile,
			LatestSize:             s.LatestSize,
			LatestMTime:            s.LatestMTime,
			SessionID:              s.SessionID,
			RecordingStartedAt:     formatOptionalTime(s.RecordingStartedAt),
			ProbeAttempts:          s.ProbeAttempts,
			ProbeSuccesses:         s.ProbeSuccesses,
			ProbeSuccessRate:       probeRate,
			ProbeAverageDuration:   probeAverageMS,
			RecordingRestartCount:  s.RecordingRestartCount,
			FFmpegAbnormalExits:    s.FFmpegAbnormalExits,
			RecordingStartFailures: s.RecordingStartFailures,
			LastSuccessfulWrite:    formatOptionalTime(s.LastSuccessfulWrite),
			LastError:              s.LastError,
			LastErrorAt:            formatOptionalTime(s.LastErrorAt),
			SegmentStartedAt:       formatOptionalTime(s.SegmentStartedAt),
			WriteBytesPerSecond:    s.WriteBytesPerSecond,
			StreamlinkPID:          s.StreamlinkPID,
			FFmpegPID:              s.FFmpegPID,
			FFmpegBitrate:          s.FFmpegBitrate,
			FFmpegSpeed:            s.FFmpegSpeed,
			PipelineState:          s.PipelineState,
			AvailableQualities:     append([]string(nil), s.AvailableQualities...),
			SelectedQuality:        s.SelectedQuality,
			SelectedStreamType:     s.SelectedStreamType,
			SessionSegmentCount:    s.SessionSegmentCount,
			SessionRestartCount:    s.SessionRestartCount,
			SessionRecordedBytes:   s.SessionRecordedBytes,
			SessionGapTotalMS:      s.SessionGapTotal.Milliseconds(),
			SessionMaxGapMS:        s.SessionMaxGap.Milliseconds(),
			LastRecoveryMS:         s.LastRecoveryDuration.Milliseconds(),
			TotalRecoveryMS:        s.TotalRecoveryDuration.Milliseconds(),
			VerifiedSegments:       s.VerifiedSegments,
			BrokenSegments:         s.BrokenSegments,
			SessionHealthPercent:   sessionHealth,
		}
		s.mu.Unlock()
	}

	return APIResponse{
		System:            sysState,
		Streams:           snapshots,
		PipelineSelfCheck: a.pipelineSelfCheckSnapshot(),
	}
}

func formatOptionalTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("2006-01-02 15:04:05")
}

func isVideoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".ts", ".mp4", ".mkv", ".mov", ".m4v":
		return true
	default:
		return false
	}
}

func (a *App) handleAPILogStatus(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	started := a.triggerPipelineSelfCheck()
	w.Header().Set("Content-Type", "application/json")
	if started {
		w.Write([]byte(`{"status":"running"}`))
	} else {
		w.Write([]byte(`{"status":"already_running"}`))
	}
}

func (a *App) handleAPILogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	buf, err := tailFile("livetool.log", 512*1024)
	if err != nil {
		w.Write([]byte("[ERROR] 無法打開日誌檔案 livetool.log"))
		return
	}
	lines := strings.Split(string(buf), "\n")
	level := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("level")))
	channel := strings.TrimPrefix(strings.TrimSpace(r.URL.Query().Get("channel")), "@")
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if level != "" && !strings.Contains(line, "level="+level) {
			continue
		}
		if channel != "" && !strings.Contains(line, "channel="+channel) && !strings.Contains(line, "@"+channel) {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(line), query) {
			continue
		}
		filtered = append(filtered, line)
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("lines"))
	if limit <= 0 || limit > 1000 {
		limit = 300
	}
	if len(filtered) > limit {
		filtered = filtered[len(filtered)-limit:]
	}
	w.Write([]byte(strings.Join(filtered, "\n")))
}

func (a *App) handleAPIProbe(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	prefix := strings.TrimPrefix(strings.TrimSpace(r.URL.Query().Get("prefix")), "@")
	a.StreamsMu.RLock()
	stream, exists := a.Streams[prefix]
	a.StreamsMu.RUnlock()

	if !exists {
		log.Printf("[API] ⚠️ 收到針對不存在頻道 @%s 的手動刺探請求", prefix)
		http.Error(w, `{"error":"頻道不存在"}`, http.StatusNotFound)
		return
	}

	stream.mu.Lock()
	if stream.IsRecording {
		stream.mu.Unlock()
		log.Printf("[API] [@%s] 手動刺探駁回：目前正在錄影中。", prefix)
		http.Error(w, `{"error":"該頻道正在錄影中，無需刺探"}`, http.StatusBadRequest)
		return
	}
	if stream.IsProbing {
		stream.mu.Unlock()
		log.Printf("[API] [@%s] 手動刺探駁回：上次刺探併發鎖尚未釋放。", prefix)
		http.Error(w, `{"error":"上一次刺探正在進行中，請稍候"}`, http.StatusTooManyRequests)
		return
	}
	stream.IsProbing = true
	stream.ProbeStatus = "🚀 收到手動指令，雷達全力刺探中..."
	stream.mu.Unlock()

	log.Printf("[API] ⚡ [手動刺探觸發] 正在為頻道 @%s 調度即時雷達偵測流...", prefix)

	go func(s *StreamState) {
		isLive := a.checkLiveStatusAndLog(s.Prefix, s.TargetURL)

		s.mu.Lock()
		s.IsProbing = false
		if isLive {
			log.Printf("[API] [@%s] 🎯 手動刺探命中！主播正處於開播狀態，立刻派遣管線接管！", s.Prefix)
			s.mu.Unlock()
			go a.startRecordingWrapper(s)
		} else {
			log.Printf("[API] [@%s] ❌ 手動刺探完畢：目標並未開播。", s.Prefix)
			s.ProbeStatus = "❌ 手動刺探：主播未開播"
			s.mu.Unlock()
			time.Sleep(6 * time.Second)
		}
	}(stream)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"triggered"}`))
}

func (a *App) handleAPIProbePause(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	prefix := strings.TrimPrefix(strings.TrimSpace(r.URL.Query().Get("prefix")), "@")
	paused, err := strconv.ParseBool(r.URL.Query().Get("paused"))
	if err != nil {
		http.Error(w, `{"error":"paused 必須是 true 或 false"}`, http.StatusBadRequest)
		return
	}
	a.StreamsMu.RLock()
	stream, exists := a.Streams[prefix]
	a.StreamsMu.RUnlock()
	if !exists {
		http.Error(w, `{"error":"頻道不存在"}`, http.StatusNotFound)
		return
	}
	stream.mu.Lock()
	stream.ProbePaused = paused
	isRecording := stream.IsRecording
	// 倒數中的狀態文字包含剩餘秒數，保留它讓前端與雷達從原位置繼續。
	if !isRecording && !strings.Contains(stream.ProbeStatus, "倒數") {
		if paused {
			stream.ProbeStatus = "⏸ 自動刺探已暫停"
		} else {
			stream.ProbeStatus = "▶ 自動刺探已繼續，重新對齊倒數中..."
		}
	}
	stream.mu.Unlock()
	state := "resumed"
	message := "自動刺探已繼續"
	if paused {
		state = "paused"
		message = "自動刺探與倒數已暫停"
	}
	appendStreamEvent(stream, "info", "probe_"+state, message)
	log.Printf("[RADAR] channel=%s automatic_probe=%s recording_untouched=%t", prefix, state, isRecording)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"status": state, "probe_paused": paused})
}

func (a *App) handleAPIRestart(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	prefix := strings.TrimPrefix(strings.TrimSpace(r.URL.Query().Get("prefix")), "@")
	a.StreamsMu.RLock()
	stream, exists := a.Streams[prefix]
	a.StreamsMu.RUnlock()

	if !exists {
		http.Error(w, `{"error":"頻道不存在"}`, http.StatusNotFound)
		return
	}

	stream.mu.Lock()
	if !stream.IsRecording {
		stream.mu.Unlock()
		http.Error(w, `{"error":"該頻道目前未在錄影中，無法重啟"}`, http.StatusBadRequest)
		return
	}

	log.Printf("[API] 🔄 收到網頁端要求強制中斷並重啟頻道 @%s 錄影之指令", prefix)
	stream.RecordingRestartCount++
	stream.ProbeStatus = "🔄 正在執行強制重啟..."
	if stream.RecordCancel != nil {
		stream.RecordCancel()
	}
	stream.mu.Unlock()

	select {
	case stream.ReloadChan <- struct{}{}:
	default:
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"restarting"}`))
}

func (a *App) handleAPIRegionalRestart(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"cluster_restarting"}`))

	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	log.Println("[SYSTEM] 🌐 收到網頁端全艦重啟指令，正在熔斷所有錄影管線並準備重構核心...")

	a.StreamsMu.Lock()
	for _, s := range a.Streams {
		s.mu.Lock()
		if s.RecordCancel != nil {
			s.RecordCancel()
		}
		s.mu.Unlock()
	}
	a.StreamsMu.Unlock()

	go func() {
		time.Sleep(1000 * time.Millisecond)
		log.Println("[SYSTEM] 交由 launchd 重新啟動服務")
		os.Exit(1)
	}()
}

func (a *App) handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	a.monitorMu.Lock()
	if time.Since(a.lastMonitorRefresh) >= 3*time.Second {
		a.updateDiskStatus()
		a.updateSystemResource()
		a.lastMonitorRefresh = time.Now()
	}
	a.monitorMu.Unlock()
	resp := a.getAPIResponseSnapshot()
	resp.Files = a.listVideoFiles(resp)
	resp.Alerts, resp.TotalWriteBytesPerSecond, resp.EstimatedRemainingSeconds = a.buildAlerts(resp)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (a *App) handleAPIShutdown(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"all_shutting_down"}`))
	log.Println("[SYSTEM] 收到集群關閉指令，正在優雅釋放所有錄影管線...")

	a.StreamsMu.Lock()
	for _, s := range a.Streams {
		s.mu.Lock()
		if s.RecordCancel != nil {
			s.RecordCancel()
		}
		s.mu.Unlock()
	}
	a.StreamsMu.Unlock()

	go func() {
		time.Sleep(1 * time.Second)
		os.Exit(0)
	}()
}

type FileRow struct {
	Channel   string        `json:"channel"`
	Name      string        `json:"name"`
	SizeBytes int64         `json:"size_bytes"`
	MTime     string        `json:"mtime"`
	IsGrowing bool          `json:"is_growing"`
	Quality   *MediaQuality `json:"quality,omitempty"`
}

func (a *App) listVideoFiles(resp APIResponse) map[string][]FileRow {
	a.fileCacheMu.Lock()
	defer a.fileCacheMu.Unlock()
	if a.cachedFiles != nil && time.Since(a.lastFileScan) < 5*time.Second {
		files := cloneFileRows(a.cachedFiles)
		enrichFileSnapshot(resp, files, a.cachedRecordedBytes)
		return files
	}

	filesByChannel := make(map[string][]FileRow)
	recordedBytes := make(map[string]int64)
	qualityCandidate := ""
	_ = filepath.Walk(a.BaseSaveDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && isVideoFile(path) {
			rel, _ := filepath.Rel(a.BaseSaveDir, path)
			parts := strings.Split(rel, string(os.PathSeparator))
			if len(parts) >= 2 {
				channel := parts[0]
				recordedBytes[channel] += info.Size()
				isGrowing := false
				if stream, ok := resp.Streams[channel]; ok && stream.IsRecording {
					if stream.LatestFile == info.Name() {
						isGrowing = true
					}
				}

				filesByChannel[channel] = append(filesByChannel[channel], FileRow{
					Channel:   channel,
					Name:      info.Name(),
					SizeBytes: info.Size(),
					MTime:     info.ModTime().Format("2006-01-02 15:04:05"),
					IsGrowing: isGrowing,
				})
				rowIndex := len(filesByChannel[channel]) - 1
				a.obsMu.RLock()
				quality, hasQuality := a.MediaQuality[path]
				a.obsMu.RUnlock()
				if hasQuality {
					filesByChannel[channel][rowIndex].Quality = &quality
				} else if !isGrowing && qualityCandidate == "" {
					qualityCandidate = path
				}
			}
		}
		return nil
	})

	for channel := range filesByChannel {
		sort.Slice(filesByChannel[channel], func(i, j int) bool {
			return filesByChannel[channel][i].Name > filesByChannel[channel][j].Name
		})
	}
	a.cachedFiles = cloneFileRows(filesByChannel)
	a.cachedRecordedBytes = recordedBytes
	a.lastFileScan = time.Now()
	enrichFileSnapshot(resp, filesByChannel, recordedBytes)
	if qualityCandidate != "" {
		a.inspectMediaAsync(qualityCandidate)
	}

	return filesByChannel
}

func cloneFileRows(source map[string][]FileRow) map[string][]FileRow {
	cloned := make(map[string][]FileRow, len(source))
	for channel, rows := range source {
		cloned[channel] = append([]FileRow(nil), rows...)
	}
	return cloned
}

func enrichFileSnapshot(resp APIResponse, files map[string][]FileRow, recordedBytes map[string]int64) {
	for channel, snapshot := range resp.Streams {
		snapshot.RecordedBytes = recordedBytes[channel]
		resp.Streams[channel] = snapshot
		if !snapshot.IsRecording || snapshot.LatestFile == "" {
			continue
		}
		for index := range files[channel] {
			if files[channel][index].Name == snapshot.LatestFile {
				files[channel][index].IsGrowing = true
				files[channel][index].SizeBytes = snapshot.LatestSize
				files[channel][index].MTime = snapshot.LatestMTime
				break
			}
		}
	}
}

type ChannelView struct {
	Prefix string
	State  StreamStateSnapshot
	Files  []FileRow
}

type PageData struct {
	ProbeStart string
	ProbeEnd   string
	Channels   []ChannelView
}

func (a *App) handleIndex(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/download/") {
		requestedPath := strings.TrimPrefix(r.URL.Path, "/download/")
		fullPath := filepath.Join(a.BaseSaveDir, requestedPath)

		rel, err := filepath.Rel(a.BaseSaveDir, fullPath)
		if err != nil || strings.HasPrefix(rel, "..") || rel == ".." {
			log.Printf("[SECURITY] 🚨 偵測到非法路徑訪問: %s", r.RemoteAddr)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		http.ServeFile(w, r, fullPath)
		return
	}

	resp := a.getAPIResponseSnapshot()
	filesByChannel := a.listVideoFiles(resp)

	prefixes := make([]string, 0, len(resp.Streams))
	for prefix := range resp.Streams {
		prefixes = append(prefixes, prefix)
	}
	sort.Strings(prefixes)

	channels := make([]ChannelView, 0, len(prefixes))
	for _, prefix := range prefixes {
		files := filesByChannel[prefix]
		channels = append(channels, ChannelView{
			Prefix: prefix,
			State:  resp.Streams[prefix],
			Files:  files,
		})
	}

	tmpl, err := template.New("index").Parse(htmlTemplate)
	if err != nil {
		http.Error(w, "模板解析錯誤", http.StatusInternalServerError)
		return
	}

	a.configMu.RLock()
	currentStart := a.Config.ProbeStart
	currentEnd := a.Config.ProbeEnd
	a.configMu.RUnlock()

	data := PageData{
		ProbeStart: currentStart,
		ProbeEnd:   currentEnd,
		Channels:   channels,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.Execute(w, data)
}

func killCommandGroup(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	if runtime.GOOS == "windows" {
		_ = cmd.Process.Kill()
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}

func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	w.Header().Set("Allow", method)
	http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	return false
}

func sameOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			if strings.EqualFold(r.Header.Get("Sec-Fetch-Site"), "cross-site") {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			if rawOrigin := r.Header.Get("Origin"); rawOrigin != "" {
				origin, err := url.Parse(rawOrigin)
				if err != nil || !strings.EqualFold(origin.Host, r.Host) {
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}
