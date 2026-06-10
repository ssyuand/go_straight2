# 🎥 Live Recorder - Go Straight² / Multi-Radar Edition

基於 **Unix / Suckless Philosophy** 打造的極簡多頻道直播錄製守護進程（Daemon）。

專為：

* 1~N 路直播長期監控
* 雙主播同步錄製
* 24/7 無人值守運行

而設計。

---

## 核心開發準則

不造多餘的輪子。

不依賴肥大的框架。

不寫無意義的狀態檔。

不把記憶體問題丟給硬碟。

一切向 Linux Kernel 借力。

---

過去需要：

* 多個 Shell Script
* Cron
* 額外監控程序
* 外部 API
* 多份設定檔

才能完成的工作。

現在全部收攏於：

```text
main.go
```

單一 Binary。

---

## 🚀 Go Straight² 的進化

相較於初代版本：

```text
單主播
↓
單雷達
↓
單錄影核心
```

Go Straight² 已升級為：

```text
多主播
↓
多雷達
↓
多錄影核心
↓
完全隔離運行
```

每個主播皆擁有獨立：

* Probe Radar
* Record Engine
* State Machine
* Hot Reload Channel

互不影響。

即使：

```text
@aaa 開播
```

正在錄影。

也不會阻塞：

```text
@bbb
@ccc
```

的偵測與錄製。

---

## 🎯 設計目標

打造一套：

```text
低資源
低維護
低依賴
高穩定
```

的直播錄製系統。

適合：

* TikTok Live
* 長時間直播監控
* 私人錄播伺服器
* NAS
* Linux Server
* Mini PC

長期掛機使用。

---

## 系統架構

```text
          ┌──────────────┐
          │ config.json  │
          └──────┬───────┘
                 │
                 ▼

        ┌──────────────────┐
        │   Go Straight²   │
        │ Master Control   │
        └────────┬─────────┘
                 │

      ┌──────────┼──────────┐
      ▼          ▼          ▼

   Radar A    Radar B    Radar C
      │          │          │
      ▼          ▼          ▼

 Record A   Record B   Record C
      │          │          │
      ▼          ▼          ▼

 streamlink streamlink streamlink
      │          │          │
      ▼          ▼          ▼

   ffmpeg     ffmpeg     ffmpeg
      │          │          │
      ▼          ▼          ▼

    .ts         .ts        .ts
```

---

## 一句話總結

> 一個 Binary，多個雷達，多個錄影核心，24/7 自動監控與錄製。
