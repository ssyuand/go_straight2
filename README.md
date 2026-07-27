## go_straight 快速操作手冊

### 安裝依賴

```bash
# 首次下載
git clone https://github.com/ssyuand/go_straight2.git
cd go_straight2

# macOS（Homebrew）
brew install go ffmpeg

python3 -m venv venv
source venv/bin/activate
pip install streamlink
deactivate
```

### 安裝與操作

```bash
./scripts/manage-launchd.sh install
./scripts/manage-launchd.sh status
./scripts/manage-launchd.sh start
./scripts/manage-launchd.sh stop
./scripts/manage-launchd.sh restart
```

`install` 會自動建置、安裝並啟動服務，不需要再手動執行 `./livetool`。

### Web 登入驗證

第一次啟動會自動建立 `auth.json`，產生隨機管理員密碼。使用以下指令查看首次登入資料：

```bash
rg "INITIAL CREDENTIALS" livetool.log | tail -n 1
```

預設帳號為 `admin`。`auth.json` 只保存加鹽密碼雜湊，且已排除 Git；每台新電腦第一次啟動都會產生不同密碼。控制台、API、設定、診斷與錄影檔下載皆需要登入，右上角 `LOGOUT` 可清除目前 Session。

### macOS 自動恢復與防睡眠

程式在 macOS 運行期間會透過 `caffeinate` 防止系統閒置睡眠；顯示器仍可正常關閉。若要在登入後自動啟動，並在程序異常退出時自動拉起：

停止或移除登入自啟服務：

```bash
./scripts/manage-launchd.sh stop
./scripts/manage-launchd.sh uninstall
```

`launchd` 管理模式下，Web 的 `RESTART` 會交由 `launchd` 重啟。Web 的 `SHUTDOWN` 是正常退出，因此不會立即自動拉起；重新登入、重新開機或執行 `./scripts/manage-launchd.sh start` 後會恢復。

啟動後，控制台位於 `http://localhost:<config.json 的 web_port>`。錄影會儲存在 `downloads/<頻道>/`。

修改 `probe_start`、`probe_end`、`probe_interval` 或 `probe_sleep_deep` 後，服務會自動重新載入設定。

### Web 診斷功能

控制台提供即時告警、寫入速度趨勢、錄影事件時間軸、Session 詳情、磁碟剩餘錄影時間預估、日誌篩選及錄影檔 `ffprobe` 完整性資訊。`DIAGNOSTICS` 按鈕會下載包含狀態、版本、遮蔽後設定與近期日誌的診斷 ZIP；所有功能皆在本機 Web 控制台運作，不會發送外部通知。

服務每次啟動會執行一次真實錄影管線自檢：產生本機 HLS 測試來源，依序通過 Streamlink、stdout pipe、FFmpeg、TS 寫入及 ffprobe 驗證，完成後刪除測試檔。Web 的 `CHECK` 可隨時重新執行，自檢結果會直接顯示在 `SYS MONITOR`，不會連線到目標頻道或影響正式錄影。

`SETTINGS` 可直接在 Web 控制台編輯 `config.json`。警戒時段與探測間隔會立即套用；頻道 URL 或 Web Port 變更後，控制台會提示從 Web 端重啟並自動前往新 Port。
