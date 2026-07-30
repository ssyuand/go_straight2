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
