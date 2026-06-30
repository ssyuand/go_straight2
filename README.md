```text
======================================================================================================
                         GO-STRAIGHT CONCURRENT RADAR SYSTEM LIFECYCLE (V2)
======================================================================================================

 [GLOBAL CORE INIT] -> Spawns 1x Global app.configWatcher() (Hot-reload monitor)
                    -> Spawns 1x Global app.startSelfCheck() [Every 60s Global Cron Guard]
                    -> Spawns Nx Parallel app.templateProbeRadar(s) per defensive channel
                                 |
                                 v
+----------------------------------------------------------------------------------------------------+
| STAGE 1: PARALLEL RADAR PROBE & JITTER COOL DOWN (app.templateProbeRadar)                          |
+----------------------------------------------------------------------------------------------------+
  RE_LOOP Entry Point
    |-- [ LOCK ] -> mu.Lock() -> Read: isRec, isProb -> mu.Unlock()
    |     |-- If isRec  -> time.Sleep(2 * time.Second) -> goto RE_LOOP
    |     +-- If isProb -> time.Sleep(1 * time.Second) -> goto RE_LOOP
    |
    |-- Select Intercept: case <-s.ReloadChan -> Log Config Hot-Reload Action (Align Windows)
    |-- Call: a.checkTimeWindowSafe()
    |     +-- [ State: Outside Window ] --------------------------------------+
    |     |     |-- Calculate: totalSleep = Min(timeToStart, probeSleepDeep)  |
    |     |     |-- Countdown Loop: updates s.ProbeStatus ("💤 非戰備休眠中")   |
    |     |     |     |-- Intercept: <-s.ReloadChan -> Break & Reset Loop     |
    |     |     +-------------------------------------------------------------|--> goto RE_LOOP
    |     |
    |     +-- [ State: Inside Window ]
    |           |-- Set: s.ProbeStatus = "🟣 發送網路請求中 (檢測開播狀態...)"
    |           |-- Call: a.checkLiveStatusAndLog() -> exec.Command("streamlink", "--json", ...)
    |                 |
    |                 +-- [ Result: Offline / JSON Parse Fail ] --------------+
    |                 |     |-- Calculate Jitter: waitTime = ProbeInterval + rand.Intn(21)
    |                 |     |-- Countdown Loop: updates s.ProbeStatus ("🟡 刺探待命中")
    |                 |     |     |-- Intercept: <-s.ReloadChan -> Break & Reset Loop
    |                 |     +-------------------------------------------------|--> goto RE_LOOP
    |                 |
    |                 +-- [ Result: Online / Valid Streams ]
    |                       |
    |                       v (Dispatches Core Recording Wrapper Pipeline)

+----------------------------------------------------------------------------------------------------+
| STAGE 2: RECORDING WRAPPER & MICRO-DISCONNECTION RETRY LOOP (startRecordingWrapper)                |
+----------------------------------------------------------------------------------------------------+
  INNER_RETRY_LOOP Entry Point
    |-- [ LOCK ] -> mu.Lock()
    |     |-- Set: IsRecording = true, ProbeStatus = "🟢 已交接錄影 (哨兵常駐監聽中)"
    |     |-- Set: RecordCtx, RecordCancel = context.WithCancel(context.Background())
    |     |-- Extract: ctx = s.RecordCtx
    |-- [ UNLOCK ] -> mu.Unlock()
    |
    |-- Call: a.runRecordEngine(ctx, s)  ---> [ Blocks here until OS pipelines close ]
    |
    |-- Select Process Breakdown (Engine termination discovery):
          |
          +-- case <-ctx.Done(): ---------------------------------------------> [ Go to END_RECORD ]
          |     (Triggered by manual intervention or Global Health Monitor killing the context)
          |
          +-- default (Pipeline broke naturally / Stream server dropped connection):
                |-- Set: s.ProbeStatus = "🟡 管線意外斷開，2秒後確認是否為微斷流..."
                |-- Exec: time.Sleep(2 * time.Second)
                |-- Call: a.checkLiveStatusAndLog()
                |     |
                |     +-- [ Re-Verification: Stream Still Online (Micro-Disconnection Event) ]
                |     |     |-- Log: "🔄 偵測到主播仍在線 (確認為微斷流)，雷達立即接回重錄！"
                |     |     +-------------------------------------------------> [ Loop Back to INNER_RETRY_LOOP ]
                |     |
                |     +-- [ Re-Verification: Stream Offline (Legitimate Stream End) ]
                |           |-- Log: "🎬 主播已確認下播，正式結束本次錄影任務。"
                |           +-------------------------------------------------> [ Go to END_RECORD ]
  
  END_RECORD:
    |-- [ LOCK ] -> mu.Lock() 
    |     |-- Set: IsRecording = false
    |     |-- Exec: s.RecordCancel() (Clean cleanup of context tree)
    |-- [ UNLOCK ] -> mu.Unlock()
    |-- Storm Shield: time.Sleep(10 * time.Second) (Protects radar against extreme high-freq loops)
    +-------------------------------------------------------------------------> [ Loop Out to STAGE 1 RE_LOOP ]

+----------------------------------------------------------------------------------------------------+
| STAGE 3: RUNTIME PROCESS MINER ENGINE (a.runRecordEngine)                                          |
+----------------------------------------------------------------------------------------------------+
  Architecture Deployment:
    |-- Gen Target Filename: YYYYMMDD-HHMMSS.ts
    |-- Inject Memory Optimizer: Env += "LD_PRELOAD=/usr/lib/libjemalloc.so"
    |-- Cmd1: exec.CommandContext(ctx, "ionice", "-c", "2", "-n", "0", "streamlink", ..., "-O")
    |-- Cmd2: exec.CommandContext(ctx, "ionice", "-c", "2", "-n", "0", "ffmpeg", "-i", "pipe:0", ...)
    |-- SetPGID: Cmd1.SysProcAttr, Cmd2.SysProcAttr -> {Setpgid: true} (Linux Process Isolation)
    |-- Link OS Pipe: ffmpegCmd.Stdin = streamlinkCmd.StdoutPipe()
    |
    |-- Fork Asynchronous Core Handlers:
    |     |-- Go Routine 1 [Context Circuit Breaker]: 
    |     |     |-- <-ctx.Done() -> syscall.Kill(-streamlinkCmd.Process.Pid, SIGKILL) -> Evaporate Group
    |     |-- Go Routine 2 [Streamlink Stderr Scanner]: Capture logs into lastStreamlinkMsg
    |     |-- Go Routine 3 [FFmpeg Stderr Scanner]: Parse bitrate & speed into lastFfmpegMsg
    |     |-- Go Routine 4 [3-Second Progress Reporter]: 
    |     |     |-- Every 3s Ticker: os.Stat(tsFile) -> Sync fields to StreamState (LatestSize/MTime)
    |     |     |-- Compute: SpeedMb/s, TotalMb, RunTimeStr -> Formatted stdout dump to console
    |
    |-- Pipeline Execution: streamlinkCmd.Start() -> ffmpegCmd.Start()
    |-- Pipeline Blocking: streamlinkCmd.Wait() -> ffmpegCmd.Wait() -> Close(engineDone)
    |
    +-- Session Post-Recycler Check:
          |-- Lifespan = time.Since(startTime)
          |-- If Lifespan < 5s OR File Size == 0:
          |     |-- Exec: os.Remove(tsFile) -> Log: "🗑️ Session too short or empty. Junk file removed."
          +-- Else -> Log: "🎉 Storage completed! Size: ... MB."

+----------------------------------------------------------------------------------------------------+
| STAGE 4: GLOBAL HEALTH MONITOR GUARDIAN (app.startSelfCheck - Independent Cron Goroutine)          |
+----------------------------------------------------------------------------------------------------+
  Loop Event: Triggered Every 1 Minute (time.NewTicker)
    |-- Call: a.updateDiskStatus() & a.updateSystemResource()
    |-- Space Check: If DiskAvail < 5GB -> Log: "⚠️ 【緊急空間告警】儲存剩餘空間極低！"
    |-- Iterate Global App.Streams map (Thread-Safe RLock):
          |
          +-- State Profile A: IsRecording == false OR LatestFile == ""
          |     |-- [ LOCK ] -> mu.Lock() -> Reset tracking fields (GrowthFailCnt=0) -> mu.Unlock()
          |     |-- Proceed to Next Stream in Map
          |
          +-- State Profile B: IsRecording == true AND LatestFile != ""
                |-- Call: os.Stat(SaveDir/LatestFile)
                |
                +-- [ Case: os.Stat Fails / File Missing ]
                |     |-- [ LOCK ] -> mu.Lock() -> GrowthFailCnt++
                |     |     |-- If GrowthFailCnt >= 2:
                |     |     |     |-- Log: "❌ 找不到檔案... 主動重置線路。" -> s.RecordCancel()
                |     |     |     |-- Reset tracking fields
                |     |     |-- mu.Unlock()
                |
                +-- [ Case: os.Stat Succeeds ]
                      |-- [ LOCK ] -> mu.Lock()
                      |-- Update UI Fields: s.LatestSize = currentSize, s.LatestMTime = modTimeStr
                      |-- If s.LastCheckFile != latestFile (File Rotation Guard)
                      |     |-- Reset Reference: LastCheckFile=latestFile, LastCheckSize=Size, GrowthFailCnt=0
                      |-- Else (Same File Monitor)
                      |     |-- Calculate: growth = currentSize - s.LastCheckSize
                      |     |-- If growth <= 0 (STAGNANT DEADLOCK / PIPE FROZEN DETECTED)
                      |     |     |-- GrowthFailCnt++
                      |     |     |-- Log: "⚠️ 警告：檔案大小完全停滯，累計未達標次數: X/2"
                      |     |     |-- If GrowthFailCnt >= 2:
                      |     |     |     |-- Log: "❌ 確定卡死！發動熔斷重錄。" -> s.RecordCancel()
                      |     |     |     |-- Reset tracking fields
                      |     |-- Else (growth > 0 - Healthy Pipeline IO)
                      |     |     |-- Reset: GrowthFailCnt = 0, LastCheckSize = currentSize
                      |-- [ UNLOCK ] -> mu.Unlock()
======================================================================================================
```
## 🛠️ 快速操作手冊

**1. 啟動與修改設定**
```bash
# 1. 打造一個全新的工具箱 (建立新的 venv)
python3 -m venv venv

# 2. 打開工具箱
source venv/bin/activate

# 3. 把 streamlink 這個工具裝進去
pip install streamlink

# 4. 把工具箱關起來 (裝完就好了)
deactivate

# 5. build
go build -ldflags="-s -w" -o livetool .



## 核心指令
./launcher.sh start   # 啟動系統 (清空舊進程，將雷達與 Web 丟入背景)
./launcher.sh status  # 打開動態監控儀表板 (按 Ctrl+C 退出面板，不影響背景錄影)
./launcher.sh stop    # 優雅關閉 (發送 pkill 清空所有相關進程，不留殭屍)
./launcher.sh log     # 選擇查看不同組件的系統日誌
```
