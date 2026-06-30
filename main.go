package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
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
	ProbeStatus  string
	LatestFile   string
	LatestSize   int64
	LatestMTime  string
	RecordCtx    context.Context
	RecordCancel context.CancelFunc
	ReloadChan   chan struct{}

	// 🩺 新增：工業級自檢健康雷達專用欄位
	LastCheckFile string
	LastCheckSize int64
	GrowthFailCnt int
}

type StreamStateSnapshot struct {
	TargetURL   string `json:"target_url"`
	Prefix      string `json:"prefix"`
	SaveDir     string `json:"save_dir"`
	IsRecording bool   `json:"is_recording"`
	IsProbing   bool   `json:"is_probing"`
	ProbeStatus string `json:"probe_status"`
	LatestFile  string `json:"latest_file"`
	LatestSize  int64  `json:"latest_size"`
	LatestMTime string `json:"latest_mtime"`
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
	System  GlobalSystemState              `json:"system"`
	Streams map[string]StreamStateSnapshot `json:"streams"`
}

type StreamlinkResponse struct {
	Streams map[string]json.RawMessage `json:"streams"`
}

type App struct {
	Config      Config
	configMu    sync.RWMutex
	BaseSaveDir string
	Streams     map[string]*StreamState
	StreamsMu   sync.RWMutex

	sysMu         sync.Mutex
	SysState      GlobalSystemState
	lastTotalTime uint64
	lastIdleTime  uint64
}

func main() {
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

	if len(os.Args) > 1 {
		action := os.Args[1]
		switch action {
		case "status":
			showCLIStatus(config.WebPort)
			return
		case "stop", "shutdown":
			terminateService(config.WebPort)
			return
		case "restart":
			fmt.Printf("🔄 正在停止舊的核心服務與錄影管線...\n")
			terminateService(config.WebPort)
			time.Sleep(1 * time.Second)

			fmt.Printf("🚀 正在重新啟動服務...\n")
			logFile, err := os.OpenFile("livetool.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err != nil {
				log.Fatalf("[ERROR] 無法建立日誌檔案: %v", err)
			}
			cmd := exec.Command(os.Args[0], "start")
			cmd.Env = append(os.Environ(), "LIVETOOL_DAEMON=1")
			cmd.Stdout = logFile
			cmd.Stderr = logFile

			if err := cmd.Start(); err != nil {
				log.Fatalf("[ERROR] 自動切換至背景運行失敗: %v", err)
			}
			fmt.Printf("\x1b[32m[LAUNCH] 🚀 go_straight 多核心併發雷達已成功重啟並運行於背景！\x1b[0m\n")
			fmt.Printf("📊 總控儀表板 -> http://localhost:%d\n", config.WebPort)
			return
		case "start":
			if os.Getenv("LIVETOOL_DAEMON") != "1" {
				logFile, err := os.OpenFile("livetool.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
				if err != nil {
					log.Fatalf("[ERROR] 無法建立日誌檔案: %v", err)
				}
				cmd := exec.Command(os.Args[0], "start")
				cmd.Env = append(os.Environ(), "LIVETOOL_DAEMON=1")
				cmd.Stdout = logFile
				cmd.Stderr = logFile

				if err := cmd.Start(); err != nil {
					log.Fatalf("[ERROR] 自動切換至背景運行失敗: %v", err)
				}
				fmt.Printf("\x1b[32m[LAUNCH] 🚀 go_straight 多核心併發雷達已成功射入背景運行！\x1b[0m\n")
				fmt.Printf("📊 總控儀表板 -> http://localhost:%d\n", config.WebPort)
				return
			}
		default:
			fmt.Printf("未知參數: %s\n可用指令:\n  ./livetool start\n  ./livetool status\n  ./livetool stop\n  ./livetool restart\n", action)
			return
		}
	} else {
		fmt.Printf("提示: 請使用 ./livetool start 啟動服務。\n")
		return
	}

	// 配置全域 Log 輸出格式
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.Println("[SYSTEM] =========================================================")
	log.Println("[SYSTEM] 🚀 go_straight 多核心併發雷達核心啟動中...")

	cwd, _ := os.Getwd()
	baseSaveDir := filepath.Join(cwd, "downloads")

	app := &App{
		Config:      config,
		BaseSaveDir: baseSaveDir,
		Streams:     make(map[string]*StreamState),
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

	// 📥 掛載背景 1 分鐘自動核心健康檢查自檢守護進程 (更新為 1 分鐘)
	app.startSelfCheck()

	http.HandleFunc("/", app.handleIndex)
	http.HandleFunc("/api/status", app.handleAPIStatus)
	http.HandleFunc("/api/shutdown", app.handleAPIShutdown)
	http.HandleFunc("/api/probe", app.handleAPIProbe)
	http.HandleFunc("/api/restart", app.handleAPIRestart)
	http.HandleFunc("/api/restart_cluster", app.handleAPIRegionalRestart)
	http.HandleFunc("/api/logs", app.handleAPILogs)
	http.HandleFunc("/api/log_status", app.handleAPILogStatus) // 📥 註冊手動狀態紀錄 API

	addr := fmt.Sprintf(":%d", app.Config.WebPort)
	log.Printf("=========================================================")
	log.Printf("🚀 總控儀表板監聽中: http://localhost%s", addr)
	log.Printf("=========================================================")

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("[ERROR] Web 伺服器崩潰: %v", err)
	}
}

func (a *App) configWatcher() {
	var lastMod time.Time
	confPath := "config.json"

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

	if pStart == pEnd {
		return true, 0
	}
	now := time.Now()
	todayStr := now.Format("2006-01-02")

	start, err1 := time.ParseInLocation("2006-01-02 15:04", todayStr+" "+pStart, time.Local)
	end, err2 := time.ParseInLocation("2006-01-02 15:04", todayStr+" "+pEnd, time.Local)
	if err1 != nil || err2 != nil {
		return true, 0
	}

	if end.Before(start) {
		if now.Before(end) {
			start = start.AddDate(0, 0, -1)
		} else {
			end = end.AddDate(0, 0, 1)
		}
	}
	if now.After(start) && now.Before(end) {
		return true, 0
	}

	var nextStart time.Time
	if now.Before(start) {
		nextStart = start
	} else {
		nextStart = start.AddDate(0, 0, 1)
	}

	diff := nextStart.Sub(now)
	if diff < 0 {
		return true, 0
	}
	return false, diff
}

func (a *App) templateProbeRadar(s *StreamState) {
	for {
	RE_LOOP:
		s.mu.Lock()
		isRec := s.IsRecording
		isProb := s.IsProbing
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

			for i := totalSleepSec; i > 0; i-- {
				s.mu.Lock()
				if s.IsRecording || s.IsProbing {
					s.mu.Unlock()
					break
				}
				s.mu.Unlock()

				select {
				case <-s.ReloadChan:
					log.Printf("[RADAR] [@%s] 深睡中收到排程熱載入變更！立刻打破長休眠重新計算。", s.Prefix)
					goto RE_LOOP
				default:
				}

				h, m, sTime := i/3600, (i%3600)/60, i%60
				a.updateProbeStatus(s, fmt.Sprintf("💤 非戰備休眠中 (倒數 %02d:%02d:%02d)", h, m, sTime))
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

			for i := waitTime; i > 0; i-- {
				s.mu.Lock()
				if s.IsRecording || s.IsProbing {
					s.mu.Unlock()
					break
				}
				s.mu.Unlock()

				select {
				case <-s.ReloadChan:
					log.Printf("[RADAR] [@%s] 戰備倒數期間收到排程熱變更，立刻重新對齊。", s.Prefix)
					goto RE_LOOP
				default:
				}

				a.updateProbeStatus(s, fmt.Sprintf("🟡 刺探待命中 (倒數 %d 秒)", i))
				time.Sleep(1 * time.Second)
			}
		}
	}
}

func (a *App) startRecordingWrapper(s *StreamState) {
	log.Printf("[PROBE] ⚠️ 發現目標 @%s 正在直播！派遣錄影核心...", s.Prefix)

	for {
		s.mu.Lock()
		s.IsRecording = true
		s.ProbeStatus = "🟢 已交接錄影 (哨兵常駐監聽中)"
		s.RecordCtx, s.RecordCancel = context.WithCancel(context.Background())
		ctx := s.RecordCtx
		s.mu.Unlock()

		a.runRecordEngine(ctx, s)

		select {
		case <-ctx.Done():
			log.Printf("[PROBE] [@%s] 錄影 Context 收到終止訊號，正式登出管線。", s.Prefix)
			goto END_RECORD
		default:
			a.updateProbeStatus(s, "🟡 管線意外斷開，2秒後確認是否為微斷流...")
			log.Printf("[PROBE] ⚠️ [@%s] 錄影管線意外中斷！正在等待 2 秒進行斷流判定...", s.Prefix)
			time.Sleep(2 * time.Second)

			if a.checkLiveStatusAndLog(s.Prefix, s.TargetURL) {
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
	log.Printf("[PROBE] [@%s] 釋放鎖控，進入 10 秒緩衝期防止極端高頻迴圈。", s.Prefix)
	time.Sleep(10 * time.Second)
}

func (a *App) runRecordEngine(ctx context.Context, s *StreamState) {
	tsFile := filepath.Join(s.SaveDir, time.Now().Format("20060102-150405")+".ts")

	nowStr := time.Now().Format("15:04:05")
	log.Printf("\x1b[36m%s [MINER-CONNECT] 📡 Target: %s\x1b[0m", nowStr, s.TargetURL)
	log.Printf("\x1b[36m%s [MINER-JOB] 🔨 Output: %s\x1b[0m", nowStr, filepath.Base(tsFile))

	startTime := time.Now()

	streamlinkCmd := exec.CommandContext(ctx, "ionice", "-c", "2", "-n", "0",
		"streamlink", s.TargetURL, "hd,ld,best",
		"--loglevel", "info",
		"--ringbuffer-size", "512M",
		"--stream-segment-threads", "1",
		"--stream-timeout", "60",
		"--http-header", "Referer=https://www.tiktok.com/",
		"--http-header", "Origin=https://www.tiktok.com",
		"--http-header", "User-Agent=Mozilla/5.0 (X11; Linux x86_64; rv:126.0) Gecko/20100101 Firefox/126.0",
		"-O",
	)

	ffmpegCmd := exec.CommandContext(ctx, "ionice", "-c", "2", "-n", "0",
		"ffmpeg", "-hide_banner", "-loglevel", "info", "-progress", "pipe:2", "-y", "-thread_queue_size", "1024", "-i", "pipe:0", "-c", "copy", "-f", "mpegts", tsFile,
	)

	streamlinkCmd.Env = append(os.Environ(), "LD_PRELOAD=/usr/lib/libjemalloc.so")
	ffmpegCmd.Env = append(os.Environ(), "LD_PRELOAD=/usr/lib/libjemalloc.so")

	// 🔥【進程組安全升級】設定獨立進程組 ID，防止 ffmpeg 脫管變成孤兒或殭屍
	if runtime.GOOS != "windows" {
		streamlinkCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		ffmpegCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}

	// 監聽 Context 熔斷。一旦觸發，強行發送 SIGKILL 至負的進程 PID（整組蒸發）
	go func() {
		<-ctx.Done()
		if runtime.GOOS != "windows" {
			if streamlinkCmd.Process != nil {
				_ = syscall.Kill(-streamlinkCmd.Process.Pid, syscall.SIGKILL)
			}
			if ffmpegCmd.Process != nil {
				_ = syscall.Kill(-ffmpegCmd.Process.Pid, syscall.SIGKILL)
			}
		}
	}()

	var lastStreamlinkMsg string = "Waiting for data..."
	var lastFfmpegMsg string = "bitrate=0kb/s speed=0x"
	var msgMu sync.Mutex

	slStderr, _ := streamlinkCmd.StderrPipe()
	go func() {
		scanner := bufio.NewScanner(slStderr)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "[") {
				msgMu.Lock()
				lastStreamlinkMsg = strings.TrimSpace(line)
				msgMu.Unlock()
			}
		}
	}()

	ffStderr, _ := execCommandStderrPipe(ffmpegCmd)
	go func() {
		scanner := bufio.NewScanner(ffStderr)
		var currentBitrate, currentSpeed string
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "bitrate=") {
				currentBitrate = strings.TrimPrefix(line, "bitrate=")
			}
			if strings.HasPrefix(line, "speed=") {
				currentSpeed = strings.TrimPrefix(line, "speed=")
				msgMu.Lock()
				lastFfmpegMsg = fmt.Sprintf("bitrate=%s speed=%s", currentBitrate, currentSpeed)
				msgMu.Unlock()
			}
		}
	}()

	pipe, err := streamlinkCmd.StdoutPipe()
	if err != nil {
		log.Printf("\x1b[31m%s [MINER-ERROR] ❌ Pipe setup failed: %v\x1b[0m", time.Now().Format("15:04:05"), err)
		return
	}
	ffmpegCmd.Stdin = pipe

	if err := streamlinkCmd.Start(); err != nil {
		log.Printf("\x1b[31m%s [MINER-ERROR] ❌ Streamlink start failed: %v\x1b[0m", time.Now().Format("15:04:05"), err)
		return
	}
	if err := ffmpegCmd.Start(); err != nil {
		log.Printf("\x1b[31m%s [MINER-ERROR] ❌ FFmpeg start failed: %v\x1b[0m", time.Now().Format("15:04:05"), err)
		return
	}

	log.Printf("\x1b[32m%s [MINER-START] 🚀 Pipeline established! Streamlink(PID:%d) ffmpeg(PID:%d)\x1b[0m",
		time.Now().Format("15:04:05"), streamlinkCmd.Process.Pid, ffmpegCmd.Process.Pid)

	engineDone := make(chan struct{})
	var lastSize int64 = 0

	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

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
				if err == nil {
					currentSize := fi.Size()
					s.mu.Lock()
					s.LatestFile = filepath.Base(tsFile)
					s.LatestMTime = fi.ModTime().Format("2006-01-02 15:04:05")
					s.LatestSize = currentSize
					s.mu.Unlock()

					bytesDiff := currentSize - lastSize
					lastSize = currentSize
					speedMb := float64(bytesDiff) / (1024 * 1024) / 3.0
					totalMb := float64(currentSize) / (1024 * 1024)

					lifespanSec := int(time.Since(startTime).Seconds())
					runTimeStr := fmt.Sprintf("%d:%02d", lifespanSec/60, lifespanSec%60)

					msgMu.Lock()
					slInfo := lastStreamlinkMsg
					ffInfo := lastFfmpegMsg
					msgMu.Unlock()

					tNow := time.Now().Format("15:04:05")
					fmt.Printf(
						"\n[%s] @%s | Up:%s | Total:%.2fMB | Speed:%.2fMB/s | SL:%s | FFmpeg:%s\n",
						tNow, s.Prefix, runTimeStr, totalMb, speedMb, slInfo, ffInfo,
					)
				} else {
					return
				}
			}
		}
	}()

	_ = streamlinkCmd.Wait()
	_ = ffmpegCmd.Wait()

	close(engineDone)

	lifespan := time.Since(startTime)

	fi, err := os.Stat(tsFile)
	if err != nil || fi.Size() == 0 || lifespan < 5*time.Second {
		_ = os.Remove(tsFile)
		log.Printf("\x1b[33m%s [MINER-RECYCLE] 🗑️ Session too short or empty. Junk file removed.\x1b[0m", time.Now().Format("15:04:05"))
	} else {
		log.Printf("\x1b[32m%s [MINER-SUMMARY] 🎉 Storage completed! Size: %.2f MB.\x1b[0m", time.Now().Format("15:04:05"), float64(fi.Size())/(1024*1024))
	}
}

func execCommandStderrPipe(cmd *exec.Cmd) (io.ReadCloser, error) {
	return cmd.StderrPipe()
}

func (a *App) updateProbeStatus(s *StreamState, status string) {
	s.mu.Lock()
	s.ProbeStatus = status
	s.mu.Unlock()
}

func (a *App) checkLiveStatusAndLog(prefix string, targetURL string) bool {
	cmd := exec.Command("streamlink", "--json", targetURL)
	output, err := cmd.CombinedOutput()
	trimmedOutput := strings.TrimSpace(string(output))

	log.Printf("[CLI-PROBE] [@%s] 執行指令: streamlink --json %s", prefix, targetURL)
	log.Printf("[CLI-OUTPUT] [@%s] 完整 CLI 輸出如下:\n%s\n----------------------------------------", prefix, trimmedOutput)

	if err != nil {
		return false
	}

	var res StreamlinkResponse
	if err := json.Unmarshal(output, &res); err != nil {
		log.Printf("[CLI-PARSE] [@%s] JSON 解析失敗 (可能非預期格式): %v", prefix, err)
		return false
	}
	return len(res.Streams) > 0
}

func (a *App) updateDiskStatus() {
	cmd := exec.Command("df", "-B1", a.BaseSaveDir)
	output, err := cmd.Output()
	if err != nil {
		return
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 2 {
		return
	}
	fields := strings.Fields(lines[1])
	if len(fields) < 4 {
		return
	}
	total, _ := strconv.ParseUint(fields[1], 10, 64)
	used, _ := strconv.ParseUint(fields[2], 10, 64)
	avail, _ := strconv.ParseUint(fields[3], 10, 64)

	a.sysMu.Lock()
	a.SysState.DiskTotal = total
	a.SysState.DiskUsed = used
	a.SysState.DiskAvail = avail
	a.sysMu.Unlock()
}

func (a *App) updateSystemResource() {
	a.updateSystemUptime()

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

func (a *App) updateSystemUptime() {
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

	seconds := uint64(secondsFloat)
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

	snapshots := make(map[string]StreamStateSnapshot)
	for prefix, s := range a.Streams {
		s.mu.Lock()
		snapshots[prefix] = StreamStateSnapshot{
			TargetURL:   s.TargetURL,
			Prefix:      s.Prefix,
			SaveDir:     s.SaveDir,
			IsRecording: s.IsRecording,
			IsProbing:   s.IsProbing,
			ProbeStatus: s.ProbeStatus,
			LatestFile:  s.LatestFile,
			LatestSize:  s.LatestSize,
			LatestMTime: s.LatestMTime,
		}
		s.mu.Unlock()
	}

	return APIResponse{
		System:  sysState,
		Streams: snapshots,
	}
}

// 🩺 startSelfCheck 啟動每 1 分鐘一次的全艦常駐自檢程序
func (a *App) startSelfCheck() {
	go func() {
		log.Println("[SYSTEM] 🩺 全艦 1 分鐘定時核心健康自檢雷達已成功上線！")

		// 🚀 啟動時立即跑一次自檢建立基準值
		a.performSelfCheck()

		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			a.performSelfCheck()
		}
	}()
}

// 🩺 performSelfCheck 執行歷史檔案狀態比對（揪出被迫斷流、卡死、無效錄製程序）
func (a *App) performSelfCheck() {
	a.updateDiskStatus()
	a.updateSystemResource()

	a.sysMu.Lock()
	availGB := float64(a.SysState.DiskAvail) / (1024 * 1024 * 1024)
	a.sysMu.Unlock()

	if availGB < 5.0 {
		log.Printf("[HEALTH-WARNING] ⚠️ 【緊急空間告警】儲存剩餘空間極低！僅剩 %.2f GB", availGB)
	}

	a.StreamsMu.RLock()
	defer a.StreamsMu.RUnlock()

	for prefix, s := range a.Streams {
		s.mu.Lock()
		isRec := s.IsRecording
		latestFile := s.LatestFile
		saveDir := s.SaveDir
		s.mu.Unlock()

		// 如果目前沒在錄影，順手清空檢測計數避免殘留舊狀態
		if !isRec || latestFile == "" {
			s.mu.Lock()
			s.LastCheckFile = ""
			s.LastCheckSize = 0
			s.GrowthFailCnt = 0
			s.mu.Unlock()
			continue
		}

		// 常規增長自檢：讀取實際檔案大小
		fullPath := filepath.Join(saveDir, latestFile)
		fi, err := os.Stat(fullPath)
		if err == nil {
			currentSize := fi.Size()
			modTimeStr := fi.ModTime().Format("2006-01-02 15:04:05")

			s.mu.Lock()
			// 順便為 Web UI 常規更新最新大小與時間
			s.LatestSize = currentSize
			s.LatestMTime = modTimeStr

			// 1. 防禦換檔誤判：若發現正在檢查全新的檔案，先建立新基準
			if s.LastCheckFile != latestFile {
				s.LastCheckFile = latestFile
				s.LastCheckSize = currentSize
				s.GrowthFailCnt = 0
			} else {
				// 2. 🚀 精準零增長判定：只有完全卡死（增長量 <= 0）才視為異常
				growth := currentSize - s.LastCheckSize
				if growth <= 0 {
					s.GrowthFailCnt++
					log.Printf("[HEALTH-WARNING] ⚠️ 警告：頻道 @%s 檔案大小完全停滯，累計未達標次數: %d/2", prefix, s.GrowthFailCnt)

					// 連續 2 次未達標 (約經過 2 分鐘)
					if s.GrowthFailCnt >= 2 {
						log.Printf("[HEALTH-ERROR] ❌ 確定卡死！頻道 @%s 檔案連續兩次無任何寫入，發動熔斷重錄。", prefix)
						if s.RecordCancel != nil {
							s.RecordCancel()
						}
						// 同步清空追蹤欄位
						s.LastCheckFile = ""
						s.LastCheckSize = 0
						s.GrowthFailCnt = 0
					}
				} else {
					// 檔案健康增長中
					s.GrowthFailCnt = 0
					s.LastCheckSize = currentSize
				}
			}
			s.mu.Unlock()
		} else {
			// os.Stat 失敗（檔案被刪除等）
			s.mu.Lock()
			s.GrowthFailCnt++
			if s.GrowthFailCnt >= 2 {
				log.Printf("[HEALTH-ERROR] ❌ 找不到檔案：頻道 @%s 錄影檔無法讀取狀態，主動重置線路。", prefix)
				if s.RecordCancel != nil {
					s.RecordCancel()
				}
				s.LastCheckFile = ""
				s.LastCheckSize = 0
				s.GrowthFailCnt = 0
			}
			s.mu.Unlock()
		}
	}
}

// 📥 handleAPILogStatus 網頁點擊「紀錄狀態」時呼叫此 API，將詳細狀態美化輸出至日誌
func (a *App) handleAPILogStatus(w http.ResponseWriter, r *http.Request) {
	a.updateDiskStatus()
	a.updateSystemResource()

	a.sysMu.Lock()
	diskTotal := a.SysState.DiskTotal
	diskUsed := a.SysState.DiskUsed
	diskAvail := a.SysState.DiskAvail
	cpuLoad := a.SysState.CPULoad
	ramPercent := a.SysState.RAMPercent
	a.sysMu.Unlock()

	gb := float64(1024 * 1024 * 1024)
	pct := 0.0
	if diskTotal > 0 {
		pct = (float64(diskUsed) / float64(diskTotal)) * 100
	}

	log.Println("[DIAGNOSTIC] 📊 === 手動核心實時狀態快照觸發 ===")
	log.Printf("[DIAGNOSTIC] 💾 磁碟空間: 已用 %.2f GB / 總共 %.2f GB (使用 %.1f%%) | 剩餘 %.2f GB", float64(diskUsed)/gb, float64(diskTotal)/gb, pct, float64(diskAvail)/gb)
	log.Printf("[DIAGNOSTIC] ⚡ 系統效能: CPU 負載: %s | RAM 使用率: %.1f%%", cpuLoad, ramPercent)
	log.Println("[DIAGNOSTIC] 📡 --- 各頻道哨兵實時狀態盤點 ---")

	a.StreamsMu.RLock()
	for name, s := range a.Streams {
		s.mu.Lock()
		isRec := s.IsRecording
		probStatus := s.ProbeStatus
		latFile := s.LatestFile
		latSize := s.LatestSize
		s.mu.Unlock()

		if isRec {
			log.Printf("[DIAGNOSTIC] 🎬 頻道 @%-12s: ● 錄影中 | 📂 當前檔案: %s (容量: %.2f MB)", name, latFile, float64(latSize)/(1024*1024))
		} else {
			log.Printf("[DIAGNOSTIC] 🎬 頻道 @%-12s: ○ 待命中 | 📡 雷達狀態: %s", name, probStatus)
		}
	}
	a.StreamsMu.RUnlock()
	log.Println("[DIAGNOSTIC] ==========================================")

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"success"}`))
}

func (a *App) handleAPILogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	file, err := os.Open("livetool.log")
	if err != nil {
		w.Write([]byte("[ERROR] 無法打開日誌檔案 livetool.log"))
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		w.Write([]byte("[ERROR] 無法獲取日誌屬性"))
		return
	}

	var bufSize int64 = 128 * 1024
	if stat.Size() < bufSize {
		bufSize = stat.Size()
	}

	buf := make([]byte, bufSize)
	_, err = file.ReadAt(buf, stat.Size()-bufSize)
	if err != nil {
		w.Write(buf)
		return
	}

	lines := strings.Split(string(buf), "\n")
	if len(lines) > 300 {
		lines = lines[len(lines)-300:]
	}
	w.Write([]byte(strings.Join(lines, "\n")))
}

func (a *App) handleAPIProbe(w http.ResponseWriter, r *http.Request) {
	prefix := r.URL.Query().Get("prefix")
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

func (a *App) handleAPIRestart(w http.ResponseWriter, r *http.Request) {
	prefix := r.URL.Query().Get("prefix")
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

	_ = exec.Command("pkill", "-f", "streamlink").Run()
	_ = exec.Command("pkill", "-f", "ffmpeg").Run()

	go func() {
		time.Sleep(1000 * time.Millisecond)

		execPath, err := os.Executable()
		if err != nil {
			execPath = os.Args[0]
		}

		logFile, err := os.OpenFile("livetool.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			os.Exit(1)
		}

		cmd := exec.Command("sh", "-c", fmt.Sprintf("%s start", execPath))
		cmd.Env = append(os.Environ(), "LIVETOOL_DAEMON=1")
		cmd.Stdout = logFile
		cmd.Stderr = logFile
		cwd, _ := os.Getwd()
		cmd.Dir = cwd

		if err := cmd.Start(); err != nil {
			log.Printf("[ERROR] 網頁熱重啟派生新核心失敗: %v", err)
			os.Exit(1)
		}

		log.Println("[SYSTEM] 🚀 新核心已成功在背景更迭，舊主程式安全退出。")
		os.Exit(0)
	}()
}

func (a *App) handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	a.updateDiskStatus()
	a.updateSystemResource()
	resp := a.getAPIResponseSnapshot()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (a *App) handleAPIShutdown(w http.ResponseWriter, r *http.Request) {
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

	_ = exec.Command("pkill", "-f", "streamlink").Run()
	_ = exec.Command("pkill", "-f", "ffmpeg").Run()

	go func() {
		time.Sleep(1 * time.Second)
		os.Exit(0)
	}()
}

type FileRow struct {
	Channel   string
	Name      string
	SizeBytes int64
	MTime     string
	IsGrowing bool
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

	filesByChannel := make(map[string][]FileRow)
	_ = filepath.Walk(a.BaseSaveDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && filepath.Ext(path) == ".ts" {
			rel, _ := filepath.Rel(a.BaseSaveDir, path)
			parts := strings.Split(rel, string(os.PathSeparator))
			if len(parts) >= 2 {
				channel := parts[0]
				isGrowing := false
				if stream, ok := resp.Streams[channel]; ok && stream.LatestFile == info.Name() && stream.IsRecording {
					isGrowing = true
				}

				filesByChannel[channel] = append(filesByChannel[channel], FileRow{
					Channel:   channel,
					Name:      info.Name(),
					SizeBytes: info.Size(),
					MTime:     info.ModTime().Format("2006-01-02 15:04:05"),
					IsGrowing: isGrowing,
				})
			}
		}
		return nil
	})

	prefixes := make([]string, 0, len(resp.Streams))
	for prefix := range resp.Streams {
		prefixes = append(prefixes, prefix)
	}
	sort.Strings(prefixes)

	channels := make([]ChannelView, 0, len(prefixes))
	for _, prefix := range prefixes {
		files := filesByChannel[prefix]
		sort.Slice(files, func(i, j int) bool {
			return files[i].Name > files[j].Name
		})
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

func showCLIStatus(port int) {
	url := fmt.Sprintf("http://localhost:%d/api/status", port)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("\x1b[31m[OFFLINE] 核心叢集服務未啟動！\x1b[0m")
		return
	}
	defer resp.Body.Close()

	var r APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		fmt.Printf("\x1b[31m[ERROR] 狀態資料解析失敗: %v\x1b[0m\n", err)
		return
	}

	gb := float64(1024 * 1024 * 1024)
	pct := 0.0
	if r.System.DiskTotal > 0 {
		pct = (float64(r.System.DiskUsed) / float64(r.System.DiskTotal)) * 100
	}

	fmt.Println("\x1b[36m==================== 📊 核心叢集實時狀態 ====================\x1b[0m")
	fmt.Printf(" 💾 磁碟空間: 已用 %.2f GB / 總共 %.2f GB (使用 %.1f%%)\n", float64(r.System.DiskUsed)/gb, float64(r.System.DiskTotal)/gb, pct)
	fmt.Printf(" ⚡ 系統效能: CPU: %s | RAM: %.1f%% | Uptime: %s\n", r.System.CPULoad, r.System.RAMPercent, r.System.Uptime)
	fmt.Println("\x1b[36m---------------------------------------------------------\x1b[0m")

	for name, s := range r.Streams {
		if s.IsRecording {
			fmt.Printf(" 🎬 頻道 @%-12s: \x1b[31m● 錄影中\x1b[0m | 📂 檔案: %s (%.2f MB)\n", name, s.LatestFile, float64(s.LatestSize)/(1024*1024))
		} else {
			fmt.Printf(" 🎬 頻道 @%-12s: \x1b[90m○ 待命中\x1b[0m | 📡 雷達: %s\n", name, s.ProbeStatus)
		}
	}
	fmt.Println("\x1b[36m=========================================================\x1b[0m")
}

func terminateService(port int) {
	url := fmt.Sprintf("http://localhost:%d/api/shutdown", port)
	_, _ = http.Get(url)
	_ = exec.Command("pkill", "-9", "-f", "streamlink").Run()
	_ = exec.Command("pkill", "-9", "-f", "ffmpeg").Run()
	time.Sleep(300 * time.Millisecond)
	_ = exec.Command("pkill", "-9", "-f", "livetool start").Run()
	fmt.Println("\x1b[32m[CLEAN] ✨ 背景叢集所有殘留程序已徹底清空！\x1b[0m")
}

// ====================================================================================
// 🎨 HTML/CSS 樣式模板
// ====================================================================================
