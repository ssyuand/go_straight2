```mermaid
flowchart LR
    INIT["Daemon Init<br/>Load Config<br/>Start Web API<br/>Start Health Guard"] --> RADAR

    RADAR["Radar Loop<br/>Per Stream Worker"] --> WINDOW{"Probe Window?"}

    WINDOW -- "Outside" --> SLEEP["Sleep Timer<br/>Wait Until Active Window"]
    SLEEP --> RADAR

    WINDOW -- "Inside" --> PROBE["Live Probe<br/>streamlink --json"]
    PROBE --> LIVE{"Live?"}

    LIVE -- "No" --> COOLDOWN["Jitter Cooldown<br/>ProbeInterval + Random Delay"]
    COOLDOWN --> RADAR

    LIVE -- "Yes" --> RECORD["Recording Loop<br/>Create Record Context"]

    RECORD --> PIPE["Stream Pipeline<br/>streamlink -> ffmpeg -> .ts"]
    PIPE --> STATE["Update StreamState<br/>File / Size / MTime / Speed"]

    STATE --> HEALTH["Health Check<br/>Every 60 Seconds"]
    HEALTH --> GROW{"File Growing?"}

    GROW -- "Yes" --> PIPE

    GROW -- "No<br/>2 Strikes" --> RESET["Circuit Breaker<br/>RecordCancel<br/>Kill Process Group"]
    RESET --> RADAR

    PIPE --> END{"Pipeline Ended?"}

    END -- "No" --> PIPE

    END -- "Yes" --> VERIFY["Re-Verify Stream<br/>streamlink --json"]
    VERIFY --> STILL{"Still Live?"}

    STILL -- "Yes<br/>Micro Disconnect" --> RECORD
    STILL -- "No<br/>Stream Ended" --> RELEASE["Release State<br/>10s Storm Shield"]
    RELEASE --> RADAR

    classDef core fill:#1f2937,stroke:#89b4fa,color:#cdd6f4,stroke-width:1px;
    classDef loop fill:#181825,stroke:#b4befe,color:#cdd6f4,stroke-width:1px;
    classDef pipe fill:#111827,stroke:#a6e3a1,color:#cdd6f4,stroke-width:1px;
    classDef guard fill:#1f1b24,stroke:#fab387,color:#cdd6f4,stroke-width:1px;
    classDef danger fill:#2a1118,stroke:#f38ba8,color:#f38ba8,stroke-width:1px;

    class INIT core;
    class RADAR,WINDOW,SLEEP,PROBE,LIVE,COOLDOWN,RECORD,VERIFY,STILL,RELEASE loop;
    class PIPE,STATE pipe;
    class HEALTH,GROW guard;
    class RESET danger;
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
