package main

// htmlTemplate contains the full Web UI template.
// Split out from main.go so UI changes stay away from the recording core.
const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
    <title>go_straight Console</title>
    <style>
        :root {
            --bg-main: #1e1e2e;         
            --bg-card: #181825;         
            --bg-strip: #11111b;        
            --border-color: #313244;    
            --text-main: #cdd6f4;       
            --text-muted: #a6adc8;      
            --accent-blue: #89b4fa;     
            --accent-green: #a6e3a1;    
            --accent-red: #f38ba8;      
            --accent-orange: #fab387;   
            --accent-lavender: #b4befe; 
        }

        body { background-color: var(--bg-main); color: var(--text-main); font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; margin: 0; padding: 15px; display: flex; justify-content: center; -webkit-font-smoothing: antialiased; }
        .container { width: 100%; max-width: 1560px; }
        
        header { display: flex; flex-direction: row; align-items: center; justify-content: space-between; gap: 24px; margin-bottom: 20px; border-bottom: 2px solid var(--border-color); padding-bottom: 15px; }
        .header-title-area { display: flex; flex-direction: row; align-items: center; justify-content: space-between; gap: 24px; width: 100%; flex-wrap: wrap; }
        h2 { margin: 0; font-size: 20px; font-weight: 800; color: var(--accent-lavender); letter-spacing: -0.5px; line-height: 1.3; white-space: nowrap; }
        .header-btn-group { display: flex; gap: 10px; width: auto; flex-wrap: wrap; justify-content: flex-end; margin-left: auto; }
        .btn-mini { flex: none; min-width: 80px; padding: 12px; border-radius: 8px; font-size: 14px; font-weight: 600; cursor: pointer; transition: all 0.2s; border: 1px solid transparent; text-align: center; }
        .btn-mini-log { 
            background: rgba(166, 227, 161, 0.12); 
            color: var(--accent-green);            
            border-color: rgba(166, 227, 161, 0.35); 
        }
        .btn-mini-log:active { 
            background: var(--accent-green);       
            color: #11111b;                        
        }
        .btn-mini-status {
            background: rgba(137, 180, 250, 0.12);
            color: var(--accent-blue);
            border-color: rgba(137, 180, 250, 0.25);
        }
        .btn-mini-status:active { background: var(--accent-blue); color: #11111b; }
        .btn-mini-restart { background: rgba(250, 179, 135, 0.12); color: var(--accent-orange); border-color: rgba(250, 179, 135, 0.25); }
        .btn-mini-restart:active { background: var(--accent-orange); color: #11111b; }
        .btn-mini-danger { background: rgba(243, 139, 168, 0.12); color: var(--accent-red); border-color: rgba(243, 139, 168, 0.25); }
        .btn-mini-danger:active { background: var(--accent-red); color: #11111b; }

        .monitor-grid { margin-bottom: 25px; }
        .system-panel { background: #11111b; border: 1px solid var(--border-color); border-radius: 14px; padding: 12px; box-shadow: 0 8px 28px rgba(0,0,0,0.28); position: relative; overflow: hidden; }
        .system-panel::before { content: ""; position: absolute; inset: 0; pointer-events: none; background: radial-gradient(circle at top left, rgba(137,180,250,0.08), transparent 34%), radial-gradient(circle at bottom right, rgba(166,227,161,0.06), transparent 38%); }
        .panel-head { position: relative; display: flex; align-items: center; justify-content: space-between; gap: 10px; margin-bottom: 10px; color: var(--text-muted); font-size: 12px; font-family: monospace; font-weight: 800; letter-spacing: 0.5px; }
        .panel-title { color: var(--accent-lavender); }
        .panel-chip { border: 1px solid rgba(180,190,254,0.25); background: rgba(180,190,254,0.08); color: var(--text-muted); border-radius: 999px; padding: 3px 9px; }
        .btop-grid { position: relative; display: grid; grid-template-columns: 1fr; gap: 10px; }
        .btop-box { background: rgba(24,24,37,0.92); border: 1px solid rgba(69,71,90,0.9); border-radius: 12px; padding: 12px; min-height: 116px; display: flex; flex-direction: column; justify-content: space-between; }
        .btop-topline { display: flex; align-items: center; justify-content: space-between; gap: 10px; margin-bottom: 8px; }
        .btop-title { color: var(--text-muted); font-size: 12px; font-weight: 800; font-family: monospace; text-transform: uppercase; letter-spacing: .6px; }
        .btop-value { color: var(--text-main); font-family: monospace; font-size: 24px; font-weight: 900; letter-spacing: -.5px; line-height: 1; }
        .btop-value.blue { color: var(--accent-blue); }
        .btop-value.green { color: var(--accent-green); }
        .btop-value.orange { color: var(--accent-orange); }
        .btop-sub { color: var(--text-muted); font-family: monospace; font-size: 11px; margin-top: 6px; display: flex; justify-content: space-between; gap: 8px; flex-wrap: wrap; }
        .btop-meter { width: 100%; height: 14px; background: #0b0b12; border: 1px solid rgba(255,255,255,0.05); border-radius: 5px; overflow: hidden; margin-top: 10px; box-shadow: inset 0 0 8px rgba(0,0,0,0.45); }
        .btop-fill { height: 100%; width: 0%; border-radius: 4px; transition: width .45s ease, background-color .45s ease; background-color: var(--accent-blue); background-image: repeating-linear-gradient(90deg, rgba(255,255,255,.18) 0 1px, transparent 1px 9px); }
        .btop-fill.green { background-color: var(--accent-green); }
        .btop-fill.orange { background-color: var(--accent-orange); }
        .btop-fill.red { background-color: var(--accent-red); }
        .btop-mini-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 8px; margin-top: 10px; }
        .btop-mini { background: rgba(0,0,0,.22); border: 1px solid rgba(255,255,255,.04); border-radius: 8px; padding: 8px; }
        .btop-mini-label { color: var(--text-muted); font-size: 10px; font-family: monospace; margin-bottom: 3px; }
        .btop-mini-value { color: var(--text-main); font-size: 12px; font-family: monospace; font-weight: 800; }
        .btop-terminal { margin-top: 8px; color: var(--text-muted); font-family: monospace; font-size: 11px; line-height: 1.35; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
        .btop-terminal .ok { color: var(--accent-green); }
        .progress-bg { width: 100%; height: 6px; background: #11111b; border-radius: 3px; overflow: hidden; border: 1px solid rgba(255,255,255,0.02); }
        .progress-fill { height: 100%; width: 0%; background: var(--accent-blue); border-radius: 3px; transition: width 0.5s ease-in-out; }

        .section-title-row { display: flex; flex-direction: row; flex-wrap: nowrap; align-items: center; justify-content: space-between; gap: 24px; margin-bottom: 12px; margin-top: 10px; width: 100%; }
        .section-title { font-size: 16px; color: var(--accent-lavender); font-weight: 900; display: flex; align-items: center; gap: 8px; margin: 0; letter-spacing: .7px; text-transform: uppercase; }
        .window-tag { margin-left: auto; background: rgba(180, 190, 254, 0.1); color: var(--accent-lavender); border: 1px solid rgba(180, 190, 254, 0.3); padding: 4px 10px; border-radius: 6px; font-size: 12px; font-family: monospace; font-weight: 700; white-space: nowrap; }

        .channel-grid { display: grid; grid-template-columns: 1fr; gap: 15px; margin-bottom: 30px; }
        .channel-box { background: var(--bg-card); border: 1px solid var(--border-color); border-radius: 12px; padding: 15px; display: flex; flex-direction: column; justify-content: flex-start; position: relative; box-shadow: 0 4px 15px rgba(0,0,0,0.15); min-width: 0; width: 100%; box-sizing: border-box; }
        .channel-box.recording { border-color: rgba(243, 139, 168, 0.5); background: linear-gradient(145deg, #241b2f, #181825); box-shadow: 0 0 20px rgba(243, 139, 168, 0.1); }
        .channel-box.recording { border-color: rgba(243, 139, 168, 0.5); background: linear-gradient(145deg, #241b2f, #181825); box-shadow: 0 0 20px rgba(243, 139, 168, 0.1); }
        .channel-main { flex: 0 0 auto; }
        .channel-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
        .channel-name { font-weight: 700; font-size: 16px; color: var(--text-main); word-break: break-all; padding-right: 10px; }
        
        .badge { font-size: 11px; padding: 4px 8px; border-radius: 6px; font-weight: 700; display: inline-flex; align-items: center; white-space: nowrap; }
        .badge-offline { background: #313244; color: var(--text-muted); border: 1px solid #45475a; }
        .badge-live { background: rgba(243, 139, 168, 0.2); color: var(--accent-red); border: 1px solid rgba(243, 139, 168, 0.5); animation: pulse 2s infinite; }
        @keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.6; } }

        .channel-body { background: rgba(0,0,0,0.25); border-radius: 8px; padding: 10px; font-size: 13px; font-family: monospace; min-height: 76px; display: flex; flex-direction: column; justify-content: center; border: 1px solid rgba(255,255,255,0.02); }
        .probe-msg { color: var(--text-muted); line-height: 1.5; word-break: break-all; }
        .probe-scan { display: none; width: 100%; }
        .probe-scan-head { display: flex; align-items: center; justify-content: space-between; gap: 10px; margin-bottom: 8px; }
        .probe-scan-label { color: var(--accent-blue); font-size: 11px; font-weight: 900; letter-spacing: .8px; }
        .probe-scan-state { color: var(--text-muted); font-size: 11px; text-transform: uppercase; white-space: nowrap; }
        .probe-scan-panel { position: relative; height: 42px; border-radius: 8px; overflow: hidden; background: linear-gradient(90deg, rgba(137,180,250,.07) 1px, transparent 1px), linear-gradient(rgba(137,180,250,.05) 1px, transparent 1px), rgba(0,0,0,.22); background-size: 18px 100%, 100% 10px, auto; border: 1px solid rgba(137,180,250,.16); }
        .probe-scan-beam { position: absolute; top: 0; bottom: 0; width: 32%; left: -34%; background: linear-gradient(90deg, transparent, rgba(137,180,250,.16), rgba(137,180,250,.34), transparent); animation: probeSweep 1.9s linear infinite; }
        .probe-scan-line { position: absolute; left: 10px; right: 10px; top: 50%; height: 1px; background: rgba(137,180,250,.24); }
        .probe-scan-bars { position: absolute; right: 12px; bottom: 9px; display: flex; align-items: end; gap: 4px; height: 20px; }
        .probe-scan-bars i { width: 5px; border-radius: 3px 3px 0 0; background: rgba(137,180,250,.72); animation: probeBars 1.1s ease-in-out infinite; }
        .probe-scan-bars i:nth-child(1) { height: 7px; animation-delay: 0s; }
        .probe-scan-bars i:nth-child(2) { height: 12px; animation-delay: .12s; }
        .probe-scan-bars i:nth-child(3) { height: 18px; animation-delay: .24s; }
        .probe-scan-sub { margin-top: 7px; color: var(--text-muted); font-size: 11px; display: flex; justify-content: space-between; gap: 8px; }
        .probe-scan-sub span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
        @keyframes probeSweep { 0% { left: -34%; opacity: .2; } 18% { opacity: 1; } 100% { left: 102%; opacity: .25; } }
        @keyframes probeBars { 0%, 100% { opacity: .35; transform: scaleY(.65); } 50% { opacity: 1; transform: scaleY(1); } }
        .sleep-countdown { display: none; width: 100%; }
        .sleep-head { display: flex; align-items: center; justify-content: space-between; gap: 10px; margin-bottom: 8px; }
        .sleep-label { color: var(--accent-green); font-size: 11px; font-weight: 900; letter-spacing: .7px; display: inline-flex; align-items: center; gap: 6px; }
        .sleep-time { color: var(--accent-green); font-size: 19px; font-weight: 900; letter-spacing: -.4px; white-space: nowrap; }
        .sleep-track { position: relative; height: 10px; border-radius: 6px; overflow: hidden; background: #0b0b12; border: 1px solid rgba(57,211,83,.18); box-shadow: inset 0 0 10px rgba(0,0,0,.45); }
        .sleep-fill { height: 100%; width: 0%; border-radius: 6px; background-color: #39d353; background-image: repeating-linear-gradient(90deg, rgba(255,255,255,.14) 0 1px, transparent 1px 9px); box-shadow: 0 0 12px rgba(57,211,83,.28); transition: width .45s ease; }
        .sleep-sub { margin-top: 7px; color: var(--text-muted); font-size: 11px; display: flex; justify-content: space-between; gap: 8px; flex-wrap: wrap; }
        .sleep-sub span { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
        .live-vitals { display: none; width: 100%; }
        .live-vitals-head { display: flex; align-items: center; justify-content: space-between; gap: 8px; margin-bottom: 6px; }
        .live-vitals-label { color: var(--accent-green); font-size: 11px; font-weight: 900; letter-spacing: .7px; display: inline-flex; align-items: center; gap: 6px; }
        .live-vitals-label::before { content: ""; width: 7px; height: 7px; border-radius: 50%; background: var(--accent-green); box-shadow: 0 0 10px rgba(166,227,161,.75); animation: dotPulse 1.2s infinite; }
        .live-speed { color: var(--accent-green); font-size: 18px; font-weight: 900; letter-spacing: -.4px; white-space: nowrap; }
        .ecg-wrap { height: 46px; border-radius: 8px; overflow: hidden; background: linear-gradient(rgba(166,227,161,.06) 1px, transparent 1px), linear-gradient(90deg, rgba(166,227,161,.06) 1px, transparent 1px), rgba(0,0,0,.22); background-size: 100% 12px, 18px 100%, auto; border: 1px solid rgba(166,227,161,.15); }
        .ecg-svg { width: 100%; height: 46px; display: block; }
        .ecg-line { fill: none; stroke: var(--accent-green); stroke-width: 2.4; stroke-linecap: round; stroke-linejoin: round; filter: drop-shadow(0 0 5px rgba(166,227,161,.65)); }
        .live-vitals-sub { margin-top: 6px; color: var(--text-muted); font-size: 11px; display: flex; justify-content: space-between; gap: 8px; flex-wrap: wrap; }
        .live-vitals-sub span { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
        .live-file { max-width: 62%; }
        .history-block { margin-top: 12px; border-top: 1px solid var(--border-color); padding-top: 10px; flex: 0 0 auto; }
        .history-title { color: var(--text-muted); font-size: 12px; font-weight: 700; margin-bottom: 8px; display: flex; justify-content: space-between; align-items: center; }
        .history-count { color: var(--accent-blue); font-family: monospace; font-size: 11px; }
        .history-list { max-height: 220px; overflow-y: auto; display: flex; flex-direction: column; gap: 6px; padding-right: 2px; }
        .history-item { background: rgba(0,0,0,0.22); border: 1px solid rgba(255,255,255,0.04); border-radius: 8px; padding: 8px; }
        .history-item.row-growing { border-color: rgba(166, 227, 161, 0.45); background: rgba(166, 227, 161, 0.06); }
        .history-link { color: var(--accent-blue); text-decoration: none; font-size: 12px; font-family: monospace; font-weight: 700; word-break: break-all; line-height: 1.4; display: flex; align-items: flex-start; gap: 6px; }
        .history-meta { margin-top: 5px; color: var(--text-muted); font-size: 11px; font-family: monospace; display: flex; justify-content: space-between; gap: 8px; flex-wrap: wrap; }
        .history-empty { color: var(--text-muted); font-size: 12px; background: rgba(0,0,0,0.18); border: 1px dashed var(--border-color); border-radius: 8px; padding: 10px; text-align: center; }
        .channel-actions { margin-top: auto; padding-top: 12px; display: flex; justify-content: stretch; }

        .btn { width: 100%; padding: 12px; border-radius: 8px; font-size: 14px; font-weight: 700; cursor: pointer; transition: all 0.2s; border: 1px solid transparent; margin-top: 12px; letter-spacing: .2px; }
        .btn-probe { background: rgba(137,180,250,.10); color: var(--accent-blue); border-color: rgba(137,180,250,.32); }
        .btn-probe:hover { background: rgba(137,180,250,.16); border-color: rgba(137,180,250,.5); }
        .btn-probe:active { background: rgba(137,180,250,.24); color: #dbeafe; }
        .btn-probe:disabled { opacity: 0.45; cursor: not-allowed; filter: saturate(.6); }
        .btn-restart { background: rgba(250, 179, 135, 0.15); color: var(--accent-orange); border-color: rgba(250, 179, 135, 0.4); }
        .btn-restart:active { background: var(--accent-orange); color: #11111b; }

        .table-container { width: 100%; }
        table { width: 100%; border-collapse: collapse; background: transparent; }
        thead { display: none; }
        tbody tr { display: flex; flex-direction: column; background: var(--bg-card); border: 1px solid var(--border-color); border-radius: 12px; margin-bottom: 15px; padding: 15px; box-shadow: 0 4px 15px rgba(0,0,0,0.15); }
        tbody td { display: flex; justify-content: space-between; align-items: center; padding: 6px 0; border: none; font-size: 13px; color: var(--text-main); }
        tbody td::before { content: attr(data-label); color: var(--text-muted); font-size: 12px; font-weight: 600; min-width: 70px; }
        
        tbody td.file-name-cell { flex-direction: column; align-items: flex-start; gap: 6px; border-top: 1px solid var(--border-color); border-bottom: 1px solid var(--border-color); margin: 8px 0; padding: 10px 0; }
        tbody td.file-name-cell::before { display: block; margin-bottom: 4px; }
        .file-link { color: var(--accent-blue); text-decoration: none; font-weight: 600; display: inline-flex; align-items: flex-start; gap: 6px; word-break: break-all; line-height: 1.4; font-size: 14px; }
        
        .channel-tag { background: rgba(137, 180, 250, 0.15); color: var(--accent-blue); border: 1px solid rgba(137, 180, 250, 0.3); padding: 3px 8px; border-radius: 6px; font-size: 12px; font-weight: 700; }
        .row-growing { background: rgba(166, 227, 161, 0.05) !important; border-color: rgba(166, 227, 161, 0.3) !important; }
        .row-growing td { color: var(--accent-green) !important; }
        .row-growing .file-link { color: var(--accent-green) !important; }
        .pulse-dot { min-width: 8px; width: 8px; height: 8px; background: var(--accent-green); border-radius: 50%; display: inline-block; position: relative; top: 4px; animation: dotPulse 1.2s infinite; }
        @keyframes dotPulse { 0% { transform: scale(0.8); opacity: 0.5; } 50% { transform: scale(1.2); opacity: 1; } 100% { transform: scale(0.8); opacity: 0.5; } }
        .empty-row td { justify-content: center; color: var(--text-muted); padding: 30px 10px; text-align: center; }
        .empty-row td::before { display: none; }

        .log-modal { display: none; position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(17,17,27,0.85); z-index: 9999; box-sizing: border-box; padding: 10px; }
        .log-box { display: flex; flex-direction: column; background: #11111b; border: 1px solid var(--border-color); width: 100%; height: 100%; border-radius: 12px; box-shadow: 0 10px 30px rgba(0,0,0,0.5); overflow: hidden; }
        .log-header { background: var(--bg-card); padding: 12px 15px; display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid var(--border-color); }
        .log-title { font-weight: bold; color: var(--accent-lavender); font-size: 15px; display: flex; align-items: center; gap: 8px; }
        .log-close { background: rgba(243, 139, 168, 0.2); color: var(--accent-red); border: 1px solid rgba(243, 139, 168, 0.4); padding: 6px 14px; border-radius: 6px; font-weight: bold; cursor: pointer; font-size: 13px; }
        .log-body { flex: 1; padding: 15px; overflow-y: auto; font-family: 'Courier New', Courier, monospace; font-size: 12px; line-height: 1.5; color: #a6e3a1; white-space: pre-wrap; word-break: break-all; scroll-behavior: smooth; }

        @media (min-width: 768px) {
            body { padding: 30px; }
            header { flex-direction: row; justify-content: space-between; align-items: center; }
            .header-title-area { flex-direction: row; align-items: center; justify-content: space-between; width: 100%; }
            h2 { font-size: 24px; }
            .header-btn-group { width: auto; margin-left: auto; }
            .btn-mini { flex: none; padding: 6px 14px; font-size: 12px; }
            .btn-mini:hover { filter: brightness(1.2); }
            .btop-grid { grid-template-columns: 1fr 1fr 1.35fr; }
            .btop-box { min-height: 128px; }
            
            .channel-grid { grid-template-columns: repeat(auto-fit, minmax(min(340px, 100%), 1fr)); align-items: start; }
            .btn { width: auto; padding: 8px 16px; margin-top: 0; font-size: 13px; }
            .channel-actions { justify-content: flex-end; }

            table { background: var(--bg-card); border: 1px solid var(--border-color); border-radius: 12px; box-shadow: 0 4px 20px rgba(0,0,0,0.2); overflow: hidden; }
            thead { display: table-header-group; background: var(--bg-strip); }
            th { color: var(--text-muted); font-size: 13px; font-weight: 600; padding: 14px 20px; text-align: left; border-bottom: 1px solid var(--border-color); }
            tbody tr { display: table-row; background: transparent; border: none; margin: 0; padding: 0; box-shadow: none; border-radius: 0; }
            tbody tr:hover td { background: rgba(255,255,255,0.02); }
            tbody td { display: table-cell; padding: 14px 20px; border-bottom: 1px solid var(--border-color); font-size: 14px; }
            tbody td::before { display: none; }
            tbody td.file-name-cell { flex-direction: row; align-items: center; border-top: none; margin: 0; padding: 14px 20px; }
            .file-link:hover { color: #b4befe; text-decoration: underline; }
            .history-link:hover { color: #b4befe; text-decoration: underline; }
            .pulse-dot { top: -1px; }
            tbody tr:last-child td { border-bottom: none; }
            
            .log-modal { padding: 40px; }
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <div class="header-title-area">
                <h2>go_straight Console</h2>
                <div class="header-btn-group">
                    <button onclick="openLogViewer()" class="btn-mini btn-mini-log">LOGS</button>
                    <button onclick="logCurrentStatus()" class="btn-mini btn-mini-status">CHECK</button>
                    <button onclick="restartCluster()" class="btn-mini btn-mini-restart">RESTART</button>
                    <button onclick="shutdownCluster()" class="btn-mini btn-mini-danger">SHUTDOWN</button>
                </div>
            </div>
        </header>

        <div class="monitor-grid">
            <div class="system-panel">
                <div class="panel-head">
                    <span class="panel-title">SYS MONITOR</span>
                </div>
                <div class="btop-grid">
                    <div class="btop-box">
                        <div>
                            <div class="btop-topline">
                                <span class="btop-title">CPU</span>
                                <span id="cpuText" class="btop-value green">--</span>
                            </div>
                            <div class="btop-meter"><div id="cpuBarFill" class="btop-fill green"></div></div>
                            <div class="btop-terminal"><span class="ok">●</span> probe / record / web server running</div>
                        </div>
                        <div class="btop-sub">
                            <span>load source: /proc/stat</span>
                            <span id="cpuUptimeText">uptime: --</span>
                        </div>
                    </div>

                    <div class="btop-box">
                        <div>
                            <div class="btop-topline">
                                <span class="btop-title">MEM</span>
                                <span id="ramText" class="btop-value blue">--</span>
                            </div>
                            <div class="btop-meter"><div id="ramBarFill" class="btop-fill"></div></div>
                            <div class="btop-mini-grid">
                                <div class="btop-mini">
                                    <div class="btop-mini-label">RAM USED</div>
                                    <div id="ramUsedText" class="btop-mini-value">--</div>
                                </div>
                                <div class="btop-mini">
                                    <div class="btop-mini-label">STATUS</div>
                                    <div id="ramStatusText" class="btop-mini-value">NORMAL</div>
                                </div>
                            </div>
                        </div>
                    </div>

                    <div class="btop-box">
                        <div>
                            <div class="btop-topline">
                                <span class="btop-title">DISK</span>
                                <span id="diskText" class="btop-value orange">--</span>
                            </div>
                            <div class="btop-meter"><div id="diskBarFill" class="btop-fill orange"></div></div>
                            <div class="btop-mini-grid">
                                <div class="btop-mini">
                                    <div class="btop-mini-label">USED</div>
                                    <div id="diskUsedText" class="btop-mini-value">--</div>
                                </div>
                                <div class="btop-mini">
                                    <div class="btop-mini-label">FREE</div>
                                    <div id="diskFreeText" class="btop-mini-value">--</div>
                                </div>
                            </div>
                        </div>
                        <div id="diskSubText" class="btop-sub">
                            <span>mount: downloads</span>
                            <span>--</span>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <div class="section-title-row">
            <h3 class="section-title">CHANNEL RADAR</h3>
            <span class="window-tag">WINDOW {{.ProbeStart}} ~ {{.ProbeEnd}}</span>
        </div>
        
        <div class="channel-grid">
            {{range .Channels}}
            {{$prefix := .Prefix}}
            {{$state := .State}}
            <div class="channel-box {{if $state.IsRecording}}recording{{end}}" data-channel="{{$prefix}}">
                <div class="channel-main">
                    <div class="channel-header">
                        <span class="channel-name" title="@{{$prefix}}">@{{$prefix}}</span>
                        <span class="badge {{if $state.IsRecording}}badge-live{{else}}badge-offline{{end}} status-badge">
                            {{if $state.IsRecording}}REC{{else}}IDLE{{end}}
                        </span>
                    </div>
                    <div class="channel-body">
                        <div class="probe-msg" {{if $state.IsRecording}}style="display:none"{{end}}>{{$state.ProbeStatus}}</div>
                        <div class="probe-scan">
                            <div class="probe-scan-head">
                                <span class="probe-scan-label">RADAR PROBE</span>
                                <span class="probe-scan-state">SCANNING</span>
                            </div>
                            <div class="probe-scan-panel">
                                <div class="probe-scan-line"></div>
                                <div class="probe-scan-beam"></div>
                                <div class="probe-scan-bars"><i></i><i></i><i></i></div>
                            </div>
                            <div class="probe-scan-sub">
                                <span class="probe-scan-mode">checking stream endpoint</span>
                                <span>streamlink --json</span>
                            </div>
                        </div>
                        <div class="sleep-countdown">
                            <div class="sleep-head">
                                <span class="sleep-label">SLEEP TIMER</span>
                                <span class="sleep-time">--:--:--</span>
                            </div>
                            <div class="sleep-track"><div class="sleep-fill"></div></div>
                            <div class="sleep-sub">
                                <span class="sleep-mode">outside battle window</span>
                                <span class="sleep-percent">-- remaining</span>
                            </div>
                        </div>
                        <div class="live-vitals" {{if $state.IsRecording}}style="display:block"{{end}}>
                            <div class="live-vitals-head">
                                <span class="live-vitals-label">WRITE PULSE</span>
                                <span class="live-speed">-- MB/s</span>
                            </div>
                            <div class="ecg-wrap">
                                <svg class="ecg-svg" viewBox="0 0 240 46" preserveAspectRatio="none">
                                    <polyline class="ecg-line" points="0,38 30,38 42,14 54,38 84,38 96,22 108,38 150,38 162,10 174,38 240,38"></polyline>
                                </svg>
                            </div>
                            <div class="live-vitals-sub">
                                <span class="live-file">waiting file</span>
                                <span class="live-size">--</span>
                            </div>
                        </div>
                    </div>
                </div>
                
                <div class="history-block">
                    <div class="history-title">
                        <span>HISTORY</span>
                        <span class="history-count">{{len .Files}} files</span>
                    </div>
                    <div class="history-list">
                        {{range .Files}}
                        <div class="history-item {{if .IsGrowing}}row-growing{{end}}" data-channel="{{.Channel}}" data-filename="{{.Name}}">
                            <a class="history-link" href="/download/{{.Channel}}/{{.Name}}" download>
                                {{if .IsGrowing}}<span class="pulse-dot"></span>{{end}}
                                <span>{{.Name}}</span>
                            </a>
                            <div class="history-meta">
                                <span class="file-size" data-bytes="{{.SizeBytes}}">Loading...</span>
                                <span class="file-mtime">{{.MTime}}</span>
                            </div>
                        </div>
                        {{else}}
                        <div class="history-empty">No recorded segments</div>
                        {{end}}
                    </div>
                </div>

                <div class="channel-actions">
                    {{if $state.IsRecording}}
                    <button onclick="restartStream(this, '{{$prefix}}')" class="btn btn-restart action-btn">RESTART REC</button>
                    {{else}}
                    <button onclick="forceProbe(this, '{{$prefix}}')" class="btn btn-probe action-btn" {{if $state.IsProbing}}disabled{{end}}>SCAN NOW</button>
                    {{end}}
                </div>
            </div>
            {{end}}
        </div>

    <div id="logModal" class="log-modal">
        <div class="log-box">
            <div class="log-header">
                <div class="log-title">LOG TERMINAL</div>
                <button onclick="closeLogViewer()" class="log-close">CLOSE</button>
            </div>
            <div id="logBody" class="log-body">Connecting to log stream...</div>
        </div>
    </div>

    <script>
        let logInterval = null;
        const streamVitals = {};
        const sleepTimers = {};
        const manualProbeHolds = {};

        function getManualProbeHold(prefix) {
            const hold = manualProbeHolds[prefix];
            if (!hold) return null;
            if (Date.now() > hold.until) {
                delete manualProbeHolds[prefix];
                return null;
            }
            return hold;
        }

        function formatBytes(bytes) {
            if (bytes === 0) return "0.00 MB";
            var k = 1024, sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'], i = Math.floor(Math.log(bytes) / Math.log(k));
            if (i < 2) i = 2; 
            var val = bytes / Math.pow(k, i);
            return (sizes[i] === 'GB' || sizes[i] === 'TB') ? val.toFixed(2) + " " + sizes[i] : val.toFixed(2) + " " + sizes[i];
        }

        function cleanStatusText(text) {
            const raw = String(text || "");

            if (raw.includes("發送網路請求") || raw.includes("檢測開播")) return "probing live status...";
            if (raw.includes("手動指令") || raw.includes("全力刺探")) return "manual probe running...";
            if (raw.includes("手動刺探") && raw.includes("未開播")) return "manual probe: stream offline";
            if (raw.includes("正在執行強制重啟")) return "restarting recording pipeline...";
            if (raw.includes("哨兵初始化")) return "radar initialized, standing by";
            if (raw.includes("已交接錄影")) return "recording pipeline active";
            if (raw.includes("管線意外斷開")) return "pipeline interrupted, verifying stream...";
            if (raw.includes("尚未開播")) return "stream offline, waiting for next probe";
            if (raw.includes("非戰備休眠")) return "outside battle window";
            if (raw.includes("刺探待命")) return "probe cooldown";

            return raw
                .replace(/[\u{1F300}-\u{1FAFF}\u{2600}-\u{27BF}]/gu, "")
                .replace(/^\s*[\-:：|]+\s*/, "")
                .replace(/\s{2,}/g, " ")
                .trim();
        }

        document.addEventListener("DOMContentLoaded", function() {
            document.querySelectorAll(".probe-msg").forEach(function(el) {
                el.innerText = cleanStatusText(el.innerText);
            });
        });

        function openLogViewer() {
            const modal = document.getElementById("logModal");
            modal.style.display = "block";
            document.body.style.overflow = "hidden";
            
            fetchLogs();
            logInterval = setInterval(fetchLogs, 2000); 
        }

        function closeLogViewer() {
            document.getElementById("logModal").style.display = "none";
            document.body.style.overflow = "auto";
            if (logInterval) {
                clearInterval(logInterval);
                logInterval = null;
            }
        }

        function fetchLogs() {
            const logBody = document.getElementById("logBody");
            fetch('/api/logs')
                .then(r => r.text())
                .then(text => {
                    const isAtBottom = logBody.scrollHeight - logBody.clientHeight <= logBody.scrollTop + 100;
                    logBody.innerText = text;
                    if (isAtBottom) {
                        logBody.scrollTop = logBody.scrollHeight;
                    }
                })
                .catch(err => {
                    logBody.innerText = "[ERROR] Log channel unavailable: " + err;
                });
        }

        function logCurrentStatus() {
            fetch('/api/log_status')
                .then(r => r.json())
                .then(d => {
                    if(d.status === "success") {
                        openLogViewer();
                    }
                })
                .catch(err => alert("Status snapshot failed: " + err));
        }

        function restartCluster() {
            if (!confirm("WARNING: this will interrupt all active recordings and restart the Go core service. Continue?")) return;
            
            const controller = new AbortController();
            const timeoutId = setTimeout(() => controller.abort(), 1200);

            alert("Restart command sent. Please refresh this dashboard in 5 seconds.");

            fetch('/api/restart_cluster', { signal: controller.signal })
                .then(r => r.json())
                .then(d => {
                    clearTimeout(timeoutId);
                    location.reload();
                })
                .catch(e => {
                    console.log("Core restart in progress...");
                });
        }

        function forceProbe(btn, prefix) {
            btn.disabled = true;
            btn.innerText = "SCANNING...";

            manualProbeHolds[prefix] = {
                until: Date.now() + 4500,
                state: "SCANNING",
                mode: "manual probe request"
            };

            const box = document.querySelector('div[data-channel="' + prefix + '"]');
            if (box) {
                renderProbeScan(box, prefix, { is_probing: true }, "手動指令");
            }

            fetch('/api/probe?prefix=' + prefix)
                .then(r => {
                    if(!r.ok) return r.json().then(e => { throw new Error(e.error); });
                    return r.json();
                })
                .then(() => {
                    manualProbeHolds[prefix] = {
                        until: Date.now() + 4500,
                        state: "SCANNING",
                        mode: "manual probe request"
                    };
                })
                .catch(e => {
                    delete manualProbeHolds[prefix];
                    alert(e.message);
                    btn.disabled = false;
                    btn.innerText = "SCAN NOW";
                });
        }

        function restartStream(btn, prefix) {
            if (!confirm("Force restart the current recording for @" + prefix + "?\nThe current segment will be closed and a new file will be opened.")) return;
            btn.disabled = true;
            btn.innerText = "RESTARTING...";
            fetch('/api/restart?prefix=' + prefix)
                .then(r => {
                    if(!r.ok) return r.json().then(e => { throw new Error(e.error); });
                    return r.json();
                })
                .catch(e => {
                    alert(e.message);
                    btn.disabled = false;
                    btn.innerText = "RESTART REC";
                });
        }

        function parseCountdownSeconds(status) {
            const text = String(status || "");
            let m = text.match(/倒數\s*(\d+):(\d{2}):(\d{2})/);
            if (m) {
                return parseInt(m[1], 10) * 3600 + parseInt(m[2], 10) * 60 + parseInt(m[3], 10);
            }
            m = text.match(/倒數\s*(\d+)\s*秒/);
            if (m) {
                return parseInt(m[1], 10);
            }
            return null;
        }

        function formatCountdown(seconds) {
            seconds = Math.max(0, Math.floor(Number(seconds) || 0));
            const h = Math.floor(seconds / 3600);
            const m = Math.floor((seconds % 3600) / 60);
            const s = seconds % 60;
            return String(h).padStart(2, "0") + ":" + String(m).padStart(2, "0") + ":" + String(s).padStart(2, "0");
        }

        function inferCountdownTotal(seconds) {
            if (seconds <= 60) return 60;
            if (seconds <= 300) return 300;
            if (seconds <= 900) return 900;
            if (seconds <= 1800) return 1800;
            if (seconds <= 3600) return 3600;
            return Math.ceil(seconds / 3600) * 3600;
        }

        function isProbeActive(prefix, stream, status) {
            const raw = String(status || "");
            return !!getManualProbeHold(prefix) ||
                !!(stream && stream.is_probing) ||
                raw.includes("發送網路請求") ||
                raw.includes("檢測開播") ||
                raw.includes("手動指令") ||
                raw.includes("全力刺探");
        }

        function isManualOfflineResult(status) {
            const raw = String(status || "");
            return raw.includes("手動刺探") && raw.includes("未開播");
        }

        function renderProbeScan(box, prefix, stream, status) {
            const msgCell = box.querySelector(".probe-msg");
            const probeBox = box.querySelector(".probe-scan");
            if (!msgCell || !probeBox) return false;

            const raw = String(status || "");
            const hold = getManualProbeHold(prefix);
            const offline = isManualOfflineResult(raw);

            if (!isProbeActive(prefix, stream, status) && !offline) {
                probeBox.style.display = "none";
                return false;
            }

            const mode = probeBox.querySelector(".probe-scan-mode");
            const state = probeBox.querySelector(".probe-scan-state");

            if (offline) {
                if (mode) mode.innerText = "stream offline / no live source";
                if (state) state.innerText = "OFFLINE";
            } else {
                if (mode) mode.innerText = hold ? hold.mode : (raw.includes("手動") ? "manual probe request" : "checking stream endpoint");
                if (state) state.innerText = hold ? hold.state : "SCANNING";
            }

            msgCell.style.display = "none";
            probeBox.style.display = "block";
            return true;
        }

        function renderSleepCountdown(box, prefix, status) {
            const msgCell = box.querySelector(".probe-msg");
            const sleepBox = box.querySelector(".sleep-countdown");
            if (!msgCell || !sleepBox) return false;

            const remaining = parseCountdownSeconds(status);
            if (remaining === null) {
                sleepBox.style.display = "none";
                return false;
            }

            let state = sleepTimers[prefix];
            if (!state || remaining > state.total || remaining > state.lastRemaining + 10) {
                state = {
                    total: inferCountdownTotal(remaining),
                    lastRemaining: remaining
                };
            }
            state.lastRemaining = remaining;
            sleepTimers[prefix] = state;

            const total = Math.max(1, state.total);
            const pct = Math.max(0, Math.min(100, ((total - remaining) / total) * 100));
            const label = sleepBox.querySelector(".sleep-label");
            const time = sleepBox.querySelector(".sleep-time");
            const fill = sleepBox.querySelector(".sleep-fill");
            const mode = sleepBox.querySelector(".sleep-mode");
            const percent = sleepBox.querySelector(".sleep-percent");

            if (label) {
                label.innerText = String(status || "").includes("刺探") ? "PROBE COOLDOWN" : "SLEEP TIMER";
            }
            if (time) time.innerText = formatCountdown(remaining);
            if (fill) fill.style.width = pct.toFixed(1) + "%";
            if (mode) {
                mode.innerText = String(status || "").includes("非戰備") ? "outside battle window" : "next probe cooldown";
            }
            if (percent) percent.innerText = pct.toFixed(0) + "% elapsed";

            msgCell.style.display = "none";
            sleepBox.style.display = "block";
            return true;
        }

        function renderECG(polyline, samples) {
            if (!polyline) return;
            const w = 240, h = 46;
            const values = samples && samples.length ? samples.slice(-28) : [0];
            const max = Math.max(0.05, ...values);
            const step = values.length > 1 ? w / (values.length - 1) : w;
            const points = values.map(function(v, i) {
                const normalized = Math.max(0, Math.min(v / max, 1));
                const x = i * step;
                const y = h - 6 - normalized * (h - 14);
                return x.toFixed(1) + "," + y.toFixed(1);
            }).join(" ");
            polyline.setAttribute("points", points);
        }

        function updateLiveVitals(box, prefix, stream) {
            const msgCell = box.querySelector(".probe-msg");
            const liveVitals = box.querySelector(".live-vitals");
            if (!msgCell || !liveVitals) return;

            const sleepBox = box.querySelector(".sleep-countdown");
            const probeBox = box.querySelector(".probe-scan");

            if (!stream.is_recording) {
                liveVitals.style.display = "none";
                if (renderProbeScan(box, prefix, stream, stream.probe_status)) {
                    if (sleepBox) sleepBox.style.display = "none";
                } else if (!renderSleepCountdown(box, prefix, stream.probe_status)) {
                    if (sleepBox) sleepBox.style.display = "none";
                    if (probeBox) probeBox.style.display = "none";
                    msgCell.style.display = "block";
                    msgCell.innerText = cleanStatusText(stream.probe_status);
                }
                return;
            }

            msgCell.style.display = "none";
            if (sleepBox) sleepBox.style.display = "none";
            if (probeBox) probeBox.style.display = "none";
            liveVitals.style.display = "block";

            const now = Date.now();
            const currentSize = Number(stream.latest_size || 0);
            let state = streamVitals[prefix];
            if (!state || state.file !== stream.latest_file) {
                state = {
                    file: stream.latest_file || "",
                    lastSize: currentSize,
                    lastTime: now,
                    speeds: [0]
                };
            }

            const diff = currentSize - state.lastSize;
            const seconds = Math.max(0.001, (now - state.lastTime) / 1000);
            const speed = diff >= 0 ? diff / 1024 / 1024 / seconds : 0;
            state.speeds.push(speed);
            if (state.speeds.length > 28) state.speeds.shift();
            state.lastSize = currentSize;
            state.lastTime = now;
            streamVitals[prefix] = state;

            const speedText = liveVitals.querySelector(".live-speed");
            const fileText = liveVitals.querySelector(".live-file");
            const sizeText = liveVitals.querySelector(".live-size");
            const polyline = liveVitals.querySelector(".ecg-line");

            if (speedText) speedText.innerText = speed.toFixed(2) + " MB/s";
            if (fileText) fileText.innerText = stream.latest_file || "waiting file";
            if (sizeText) sizeText.innerText = formatBytes(currentSize);
            renderECG(polyline, state.speeds);
        }

        function shutdownCluster() {
            if (confirm("WARNING: this will stop all recordings and shut down the Go core service. Continue?")) {
                fetch('/api/shutdown').then(r => r.json()).then(d => {
                    alert("Shutdown command sent. Core service is shutting down.");
                    window.close();
                }).catch(e => alert("Connection interrupted. The service may already be offline."));
            }
        }

        document.querySelectorAll('.file-size').forEach(function(td) {
            var b = parseInt(td.getAttribute('data-bytes'));
            if(!isNaN(b)) td.innerHTML = formatBytes(b);
        });

        setInterval(function() {
            fetch('/api/status').then(r => r.json()).then(data => {
                var uptime = data.system.uptime || "--";
                var cpuUptimeText = document.getElementById("cpuUptimeText");
                if (cpuUptimeText) cpuUptimeText.innerText = "uptime: " + uptime;

                var totalGB = data.system.disk_total / (1024*1024*1024), availGB = data.system.disk_avail / (1024*1024*1024), usedGB = data.system.disk_used / (1024*1024*1024);
                var pct = totalGB > 0 ? (usedGB / totalGB) * 100 : 0;
                document.getElementById("diskText").innerText = pct.toFixed(1) + "%";
                document.getElementById("diskUsedText").innerText = usedGB.toFixed(2) + " GB";
                document.getElementById("diskFreeText").innerText = availGB.toFixed(2) + " GB";
                document.getElementById("diskSubText").innerHTML = "<span>" + usedGB.toFixed(2) + " / " + totalGB.toFixed(2) + " GB</span><span>free " + availGB.toFixed(2) + " GB</span>";
                var bar = document.getElementById("diskBarFill");
                bar.style.width = pct.toFixed(1) + "%";
                bar.className = "btop-fill " + (pct > 90 ? "red" : (pct > 75 ? "orange" : ""));

                var cpuRaw = data.system.cpu_load || "Loading...";
                var cpuPct = parseFloat(String(cpuRaw).replace("%", ""));
                document.getElementById("cpuText").innerText = isNaN(cpuPct) ? cpuRaw : cpuPct.toFixed(1) + "%";
                var cpuBar = document.getElementById("cpuBarFill");
                if (!isNaN(cpuPct)) {
                    cpuBar.style.width = Math.max(0, Math.min(cpuPct, 100)).toFixed(1) + "%";
                    cpuBar.className = "btop-fill " + (cpuPct > 85 ? "red" : (cpuPct > 60 ? "orange" : "green"));
                }

                if (data.system.ram_percent > 0) {
                    var ramPct = data.system.ram_percent;
                    document.getElementById("ramText").innerText = ramPct.toFixed(1) + "%";
                    document.getElementById("ramUsedText").innerText = ramPct.toFixed(1) + "% used";
                    document.getElementById("ramStatusText").innerText = ramPct > 85 ? "HIGH" : (ramPct > 65 ? "BUSY" : "NORMAL");
                    var ramBar = document.getElementById("ramBarFill");
                    ramBar.style.width = ramPct.toFixed(1) + "%";
                    ramBar.className = "btop-fill " + (ramPct > 85 ? "red" : (ramPct > 65 ? "orange" : ""));
                }

                let reloadRequired = false;
                Object.entries(data.streams).forEach(([prefix, stream]) => {
                    var box = document.querySelector('div[data-channel="' + prefix + '"]');
                    if (box) {
                        var wasRecording = box.classList.contains("recording");
                        if(stream.is_recording) {
                            box.classList.add("recording");
                        } else {
                            box.classList.remove("recording");
                        }

                        if (wasRecording !== stream.is_recording) { reloadRequired = true; }

                        var badge = box.querySelector(".status-badge");
                        if (badge) {
                            badge.className = stream.is_recording ? "badge badge-live" : "badge badge-offline";
                            badge.innerHTML = stream.is_recording ? "REC" : "IDLE";
                        }
                        updateLiveVitals(box, prefix, stream);
                        
                        var btn = box.querySelector(".action-btn");
                        if (btn && !reloadRequired) {
                            if (!stream.is_recording) {
                                const held = !!getManualProbeHold(prefix);
                                btn.disabled = stream.is_probing || held;
                                btn.innerText = (stream.is_probing || held) ? "SCANNING..." : "SCAN NOW";
                            }
                        }

                        if (stream.is_recording && stream.latest_file) {
                        }
                    }

                    if (stream.is_recording && stream.latest_file) {
                        var targetRow = document.querySelector('[data-channel="' + prefix + '"][data-filename="' + stream.latest_file + '"]');
                        if (targetRow) {
                            targetRow.classList.add("row-growing");
                            var sizeCell = targetRow.querySelector(".file-size");
                            if (sizeCell) {
                                sizeCell.setAttribute("data-bytes", stream.latest_size);
                                sizeCell.innerHTML = formatBytes(stream.latest_size);
                            }
                            var mtimeCell = targetRow.querySelector(".file-mtime");
                            if (mtimeCell) { mtimeCell.innerHTML = stream.latest_mtime; }
                        } else {
                            reloadRequired = true;
                        }
                    }
                });

                if (reloadRequired) { location.reload(); }
            }).catch(e => { console.error("Radar API error:", e); });
        }, 2000);
    </script>
</body>
</html>
`

