# 🎥 Live Recorder / Go Straight²

一個專為 **24/7 無人值守直播錄製** 設計的極簡守護程序。

基於 Go 實作。

單 Binary 部署。

無資料庫。

無外部服務。

無額外狀態檔。

---

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

## 核心指令
./launcher.sh start   # 啟動系統 (清空舊進程，將雷達與 Web 丟入背景)
./launcher.sh status  # 打開動態監控儀表板 (按 Ctrl+C 退出面板，不影響背景錄影)
./launcher.sh stop    # 優雅關閉 (發送 pkill 清空所有相關進程，不留殭屍)
./launcher.sh log     # 選擇查看不同組件的系統日誌

## 設計理念

遵循 Unix Philosophy：

```text
Keep It Simple.
Keep It Running.
```

系統負責：

```text
偵測
↓
錄製
↓
監控
↓
自癒
```

而不是堆疊大量依賴。

---

## 核心能力

### 📡 多頻道獨立雷達

每個主播皆擁有完全隔離的：

```text
Probe Radar
↓
Record Engine
↓
State Machine
↓
Health Monitor
```

互不干擾。

即使某一路錄製異常：

```text
@aaa
```

也不會影響：

```text
@bbb
@ccc
```

正常運作。

---

### 🎬 自動錄製

偵測開播後自動建立錄製管線：

```text
streamlink
     ↓
   pipe
     ↓
 ffmpeg
     ↓
   .ts
```

支援：

- TikTok Live
- Streamlink 支援之平台

---

### 🩺 工業級健康檢查

背景常駐健康雷達：

```text
每分鐘
↓
檢查錄影檔增長
↓
偵測卡死
↓
自動熔斷
↓
重新接管錄製
```

避免：

- Streamlink 卡死
- FFmpeg 假存活
- 檔案停止增長
- 長時間空轉

---

### 🔥 自動自癒

當錄影管線異常中斷：

```text
錄影斷開
↓
二次確認直播狀態
↓
仍在線
↓
立即重錄
```

主播不需要重新開播。

系統自行接回。

---

### 🔄 Config 熱載入

修改：

```json
config.json
```

後無需重啟服務。

新設定自動同步至所有雷達。

---

### 🌐 Web Dashboard

內建控制台：

```text
http://localhost:PORT
```

可查看：

- 即時錄製狀態
- 最新錄影檔
- 檔案大小
- CPU 使用率
- RAM 使用率
- 磁碟空間
- 即時日誌

---

### ⚡ Web API

提供控制介面：

```text
/api/status
/api/probe
/api/restart
/api/restart_cluster
/api/shutdown
/api/logs
```

支援：

- 手動刺探
- 單頻道重啟
- 全艦重啟
- 安全關閉
- 狀態查詢

---

## 系統架構

```text
                 config.json
                       │
                       ▼

              Go Straight² Core
                       │
        ┌──────────────┼──────────────┐
        ▼              ▼              ▼

     Channel A      Channel B      Channel C
        │              │              │

   Probe Radar    Probe Radar    Probe Radar
        │              │              │

  Record Engine  Record Engine  Record Engine
        │              │              │

 Health Monitor Health Monitor Health Monitor
        │              │              │

    streamlink     streamlink     streamlink
        │              │              │

      ffmpeg         ffmpeg         ffmpeg
        │              │              │

       .ts            .ts            .ts
```

---

## 適用場景

- TikTok Live 長期監控
- 錄播伺服器
- NAS
- Linux Server
- Mini PC
- VPS
- 家用錄播機

---

## 特性

✅ 單 Binary

✅ 多頻道並發

✅ 自動錄製

✅ 自動重錄

✅ 熱載入設定

✅ Web Dashboard

✅ 健康自檢

✅ 卡死自癒

✅ 24/7 無人值守

---

## 一句話總結

> 一個 Binary，一套控制台，多個獨立錄影核心，自動監控、自動錄製、自動自癒，為長期 24/7 掛機而生。
