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
		.logout-form { display: contents; margin: 0; }
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
		.selfcheck-strip { margin-top: 10px; padding: 10px 12px; display: flex; gap: 12px; align-items: center; justify-content: space-between; border: 1px solid var(--border-color); border-radius: 8px; background: rgba(0,0,0,.18); font: 11px monospace; }
		.selfcheck-strip.pass { border-color: rgba(166,227,161,.35); color: var(--accent-green); }
		.selfcheck-strip.fail { border-color: rgba(243,139,168,.4); color: var(--accent-red); }
		.selfcheck-strip.running { border-color: rgba(250,179,135,.35); color: var(--accent-orange); }
		.selfcheck-detail { color: var(--text-muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
        .alert-center { display: none; margin-bottom: 14px; border: 1px solid rgba(250,179,135,.35); background: rgba(250,179,135,.07); border-radius: 10px; padding: 10px; }
        .alert-center.active { display: block; }
        .alert-head { display: flex; justify-content: space-between; color: var(--accent-orange); font-size: 11px; font-weight: 900; margin-bottom: 7px; }
        .alert-list { display: flex; flex-direction: column; gap: 5px; }
        .alert-item { padding: 6px 8px; border-radius: 6px; background: rgba(0,0,0,.18); color: var(--text-main); font: 11px monospace; }
        .alert-item.error { color: var(--accent-red); border-left: 3px solid var(--accent-red); }
        .alert-item.warning { color: var(--accent-orange); border-left: 3px solid var(--accent-orange); }
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
        .channel-main { flex: 0 0 auto; }
        .channel-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
        .channel-name { font-weight: 700; font-size: 16px; color: var(--text-main); word-break: break-all; padding-right: 10px; }
        
        .badge { font-size: 11px; padding: 4px 8px; border-radius: 6px; font-weight: 700; display: inline-flex; align-items: center; white-space: nowrap; }
        .badge-offline { background: #313244; color: var(--text-muted); border: 1px solid #45475a; }
        .badge-live { background: rgba(243, 139, 168, 0.2); color: var(--accent-red); border: 1px solid rgba(243, 139, 168, 0.5); animation: pulse 2s infinite; }
        @keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.6; } }

        .channel-body { background: rgba(0,0,0,0.25); border-radius: 8px; padding: 10px; font-size: 13px; font-family: monospace; min-height: 76px; display: flex; flex-direction: column; justify-content: center; border: 1px solid rgba(255,255,255,0.02); }
        .channel-metrics { margin-top: 10px; display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 6px; font-family: monospace; }
        .metric-cell { min-width: 0; padding: 7px 8px; background: rgba(0,0,0,0.2); border: 1px solid rgba(255,255,255,0.04); border-radius: 6px; }
        .metric-label { display: block; color: var(--text-muted); font-size: 9px; letter-spacing: .06em; margin-bottom: 3px; }
        .metric-value { display: block; color: var(--text-main); font-size: 11px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
        .metric-error { grid-column: 1 / -1; border-color: rgba(243,139,168,.22); display: none; }
        .metric-error.has-error { display: block; }
        .metric-error .metric-value { color: var(--accent-red); }
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
        .sleep-state { color: var(--accent-green); font-size: 9px; font-weight: 900; letter-spacing: .08em; }
		.sleep-countdown.paused .sleep-label, .sleep-countdown.paused .sleep-state { color: var(--accent-orange); }
		.cooldown-layout { display: flex; flex-direction: column; align-items: center; width: 100%; }
		.cooldown-stage { position: relative; width: min(100%, 270px); min-height: 166px; display: grid; place-items: center; }
		.cooldown-ring { --progress: 0deg; width: 148px; height: 148px; border-radius: 50%; display: grid; place-items: center; position: relative; background: conic-gradient(from -90deg, var(--accent-green) var(--progress), rgba(255,255,255,.065) 0); box-shadow: 0 0 0 1px rgba(166,227,161,.14), 0 0 28px rgba(57,211,83,.17), inset 0 0 24px rgba(57,211,83,.07); transition: background .4s linear, box-shadow .3s; }
		.cooldown-ring::before { content: ""; position: absolute; inset: 10px; border-radius: 50%; background: radial-gradient(circle at 50% 38%, rgba(166,227,161,.09), transparent 47%), #0d0d16; border: 1px solid rgba(255,255,255,.08); box-shadow: inset 0 0 26px rgba(0,0,0,.65); }
		.cooldown-ring::after { content: ""; position: absolute; inset: 3px; border-radius: 50%; background: repeating-conic-gradient(from -90deg, rgba(255,255,255,.4) 0deg 1deg, transparent 1deg 9deg); mask: radial-gradient(transparent 0 65px, #000 66px); opacity: .28; }
		.sleep-countdown.paused .cooldown-ring { background: conic-gradient(from -90deg, var(--accent-orange) var(--progress), rgba(255,255,255,.065) 0); box-shadow: 0 0 0 1px rgba(250,179,135,.18), 0 0 28px rgba(250,179,135,.17); }
		.cooldown-ring-core { position: relative; z-index: 1; text-align: center; }
		.sleep-time { display: block; color: var(--accent-green); font-size: 27px; font-weight: 900; letter-spacing: -1.2px; white-space: nowrap; text-shadow: 0 0 14px rgba(166,227,161,.35); }
		.sleep-countdown.paused .sleep-time { color: var(--accent-orange); }
		.cooldown-unit { display: block; margin-top: 5px; color: var(--text-muted); font-size: 7px; letter-spacing: .13em; }
		.cooldown-detail { min-width: 0; width: min(100%, 310px); margin-top: 2px; }
		.cooldown-controls { position: absolute; inset: 0; pointer-events: none; }
		.cooldown-btn { position: absolute; top: 55px; width: 43px; height: 43px; padding: 0; border-radius: 50%; border: 1px solid rgba(137,180,250,.38); background: radial-gradient(circle at 35% 30%, rgba(137,180,250,.22), rgba(137,180,250,.07)); color: var(--accent-blue); font: 800 7px monospace; letter-spacing: .04em; cursor: pointer; pointer-events: auto; box-shadow: 0 5px 14px rgba(0,0,0,.32), inset 0 0 0 3px rgba(0,0,0,.18); transition: transform .18s, box-shadow .18s, opacity .18s; }
		.cooldown-btn:hover:not(:disabled) { transform: translateY(-2px) scale(1.04); box-shadow: 0 7px 20px rgba(137,180,250,.2); }
		.cooldown-btn.pause { left: 5px; color: var(--accent-orange); border-color: rgba(250,179,135,.42); background: radial-gradient(circle at 35% 30%, rgba(250,179,135,.22), rgba(250,179,135,.07)); }
		.cooldown-btn.start { right: 5px; }
		.cooldown-btn:disabled { opacity: .22; cursor: default; filter: grayscale(.5); }
        .sleep-sub { margin-top: 7px; color: var(--text-muted); font-size: 11px; display: flex; justify-content: space-between; gap: 8px; flex-wrap: wrap; }
        .sleep-sub span { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
        .live-vitals { display: none; width: 100%; }
        .live-vitals-head { display: flex; align-items: center; justify-content: space-between; gap: 8px; margin-bottom: 6px; }
        .live-vitals-label { color: var(--accent-green); font-size: 11px; font-weight: 900; letter-spacing: .7px; display: inline-flex; align-items: center; gap: 6px; }
        .live-vitals-label::before { content: ""; width: 7px; height: 7px; border-radius: 50%; background: var(--accent-green); box-shadow: 0 0 10px rgba(166,227,161,.75); animation: dotPulse 1.2s infinite; }
        .live-health { margin-left: 6px; padding: 2px 6px; border-radius: 5px; font-size: 9px; letter-spacing: .5px; border: 1px solid rgba(166,227,161,.3); color: var(--accent-green); background: rgba(166,227,161,.08); }
        .live-health.starting, .live-health.delayed { color: var(--accent-orange); border-color: rgba(250,179,135,.35); background: rgba(250,179,135,.08); }
        .live-health.stale { color: var(--accent-red); border-color: rgba(243,139,168,.4); background: rgba(243,139,168,.1); }
        .live-speed { color: var(--accent-green); font-size: 18px; font-weight: 900; letter-spacing: -.4px; white-space: nowrap; }
		.throughput-wrap { position: relative; height: 124px; box-sizing: border-box; border-radius: 10px; overflow: hidden; background: linear-gradient(rgba(166,227,161,.075) 1px, transparent 1px), linear-gradient(90deg, rgba(137,180,250,.06) 1px, transparent 1px), radial-gradient(circle at 50% 110%, rgba(137,180,250,.12), transparent 62%), #090b12; background-size: 100% 24px, 32px 100%, auto, auto; border: 1px solid rgba(166,227,161,.2); box-shadow: inset 0 0 25px rgba(0,0,0,.5), 0 0 18px rgba(166,227,161,.055); }
		.throughput-wrap::after { content: ""; position: absolute; top: -20%; bottom: -20%; width: 28%; left: -35%; z-index: 4; pointer-events: none; background: linear-gradient(90deg, transparent, rgba(166,227,161,.08), transparent); animation: throughputScan 3.6s linear infinite; }
		.throughput-wave { position: absolute; inset: 8px 8px 20px; z-index: 2; width: calc(100% - 16px); height: calc(100% - 28px); overflow: visible; }
		.throughput-area { fill: rgba(137,180,250,.11); }
		.throughput-line-glow { fill: none; stroke: rgba(166,227,161,.3); stroke-width: 7; filter: blur(3px); }
		.throughput-line { fill: none; stroke: var(--accent-green); stroke-width: 2.4; stroke-linecap: round; stroke-linejoin: round; vector-effect: non-scaling-stroke; }
		.throughput-bars { position: absolute; left: 8px; right: 8px; bottom: 5px; height: 27px; z-index: 1; display: flex; align-items: flex-end; gap: 2px; opacity: .58; }
		.throughput-bar { flex: 1 1 0; min-width: 2px; height: 3%; border-radius: 2px 2px 0 0; background: linear-gradient(to top, rgba(137,180,250,.25), rgba(166,227,161,.72)); transition: height .45s ease, opacity .45s ease; box-shadow: 0 0 5px rgba(166,227,161,.16); }
		.throughput-hud { position: absolute; top: 7px; left: 9px; right: 9px; z-index: 3; display: flex; justify-content: space-between; color: rgba(205,214,244,.55); font: 7px monospace; letter-spacing: .08em; pointer-events: none; }
		@keyframes throughputScan { from { left: -35%; } to { left: 110%; } }
        .throughput-empty { position: absolute; inset: 0; display: flex; align-items: center; justify-content: center; color: var(--text-muted); font: 10px monospace; letter-spacing: .08em; background: rgba(0,0,0,.2); }
        .throughput-empty.hidden { display: none; }
        .live-vitals-sub { margin-top: 6px; color: var(--text-muted); font-size: 11px; display: flex; justify-content: space-between; gap: 8px; flex-wrap: wrap; }
        .live-vitals-sub span { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
        .live-file { max-width: 62%; }
		.integrity-card { margin-top: 9px; padding: 9px; border-radius: 8px; border: 1px solid rgba(137,180,250,.25); background: linear-gradient(135deg, rgba(137,180,250,.08), rgba(166,227,161,.045)); }
		.integrity-head { display: flex; align-items: baseline; justify-content: space-between; gap: 8px; }
		.integrity-title { color: var(--accent-blue); font-size: 9px; font-weight: 900; letter-spacing: .08em; }
		.integrity-score { color: var(--accent-green); font-size: 20px; font-weight: 900; }
		.integrity-card.warning .integrity-score { color: var(--accent-orange); }
		.integrity-card.danger .integrity-score { color: var(--accent-red); }
		.integrity-track { height: 5px; margin: 7px 0; overflow: hidden; border-radius: 4px; background: rgba(0,0,0,.35); }
		.integrity-fill { height: 100%; width: 100%; background: var(--accent-green); transition: width .4s ease; }
		.integrity-card.warning .integrity-fill { background: var(--accent-orange); }
		.integrity-card.danger .integrity-fill { background: var(--accent-red); }
		.integrity-grid { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 6px; }
		.integrity-stat span { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
		.integrity-stat-label { color: var(--text-muted); font-size: 7px; letter-spacing: .05em; }
		.integrity-stat-value { margin-top: 2px; color: var(--text-main); font-size: 10px; font-weight: 700; }
        .pipeline-grid { margin-top: 8px; display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 5px; }
        .pipeline-item { min-width: 0; padding: 6px 7px; border-radius: 6px; background: rgba(0,0,0,.2); border: 1px solid rgba(166,227,161,.09); }
        .pipeline-label { display: block; color: var(--text-muted); font-size: 8px; letter-spacing: .06em; margin-bottom: 2px; }
        .pipeline-value { display: block; color: var(--text-main); font-size: 10px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
		.pipeline-watchdog { grid-column: 1 / -1; border-color: rgba(166,227,161,.25); }
		.pipeline-watchdog .pipeline-value { color: var(--accent-green); font-weight: 800; }
		.pipeline-watchdog.starting, .pipeline-watchdog.delayed { border-color: rgba(250,179,135,.4); background: rgba(250,179,135,.07); }
		.pipeline-watchdog.starting .pipeline-value, .pipeline-watchdog.delayed .pipeline-value { color: var(--accent-orange); }
		.pipeline-watchdog.stalled { border-color: rgba(243,139,168,.5); background: rgba(243,139,168,.09); }
		.pipeline-watchdog.stalled .pipeline-value { color: var(--accent-red); }
        .pipeline-processes { grid-column: 1 / -1; display: flex; justify-content: space-between; gap: 8px; }
        .history-block { margin-top: 12px; border-top: 1px solid var(--border-color); padding-top: 10px; flex: 0 0 auto; }
        .history-title { color: var(--text-muted); font-size: 12px; font-weight: 700; margin-bottom: 8px; display: flex; justify-content: space-between; align-items: center; }
        .history-count { color: var(--accent-blue); font-family: monospace; font-size: 11px; }
        .history-list { max-height: 220px; overflow-y: auto; display: flex; flex-direction: column; gap: 6px; padding-right: 2px; }
        .history-item { background: rgba(0,0,0,0.22); border: 1px solid rgba(255,255,255,0.04); border-radius: 8px; padding: 8px; }
        .history-item.row-growing { border-color: rgba(166, 227, 161, 0.45); background: rgba(166, 227, 161, 0.06); }
        .history-link { color: var(--accent-blue); text-decoration: none; font-size: 12px; font-family: monospace; font-weight: 700; word-break: break-all; line-height: 1.4; display: flex; align-items: flex-start; gap: 6px; }
        .history-meta { margin-top: 5px; color: var(--text-muted); font-size: 11px; font-family: monospace; display: flex; justify-content: space-between; gap: 8px; flex-wrap: wrap; }
        .quality-badge { display: inline-flex; margin-top: 5px; padding: 2px 5px; border-radius: 4px; font: 9px monospace; color: var(--accent-green); border: 1px solid rgba(166,227,161,.25); }
        .quality-badge.BROKEN, .quality-badge.WARNING { color: var(--accent-red); border-color: rgba(243,139,168,.35); }
        .history-empty { color: var(--text-muted); font-size: 12px; background: rgba(0,0,0,0.18); border: 1px dashed var(--border-color); border-radius: 8px; padding: 10px; text-align: center; }
        .channel-actions { margin-top: auto; padding-top: 12px; display: flex; justify-content: stretch; gap: 8px; }
        .channel-actions .btn { margin-top: 0; }

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
        .log-filters { padding: 9px 15px; display: grid; grid-template-columns: 90px 1fr 1.5fr auto; gap: 7px; border-bottom: 1px solid var(--border-color); }
        .log-filters input, .log-filters select { min-width: 0; background: #11111b; color: var(--text-main); border: 1px solid var(--border-color); border-radius: 5px; padding: 6px; font: 11px monospace; }
        .detail-body { flex: 1; overflow: auto; padding: 15px; color: var(--text-main); font: 12px monospace; }
        .detail-summary { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 7px; margin-bottom: 14px; }
        .detail-cell { padding: 8px; border-radius: 6px; background: rgba(0,0,0,.2); border: 1px solid var(--border-color); overflow: hidden; text-overflow: ellipsis; }
        .settings-form { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 12px; }
        .settings-field { min-width: 0; display: flex; flex-direction: column; gap: 5px; color: var(--text-muted); font: 10px monospace; }
        .settings-field.full { grid-column: 1 / -1; }
        .settings-field input, .settings-field textarea { width: 100%; box-sizing: border-box; background: #11111b; color: var(--text-main); border: 1px solid var(--border-color); border-radius: 6px; padding: 9px; font: 12px monospace; }
        .settings-field textarea { min-height: 110px; resize: vertical; }
        .settings-note { grid-column: 1 / -1; color: var(--accent-orange); font: 10px monospace; }
        .settings-actions { grid-column: 1 / -1; display: flex; justify-content: flex-end; gap: 8px; }
        .trend-chart { width: 100%; height: 100px; background: rgba(0,0,0,.2); border: 1px solid var(--border-color); border-radius: 7px; margin-bottom: 14px; }
        .event-list { display: flex; flex-direction: column; gap: 6px; }
        .event-row { padding: 7px; border-left: 3px solid var(--accent-blue); background: rgba(0,0,0,.18); }
        .event-row.error { border-color: var(--accent-red); }
        .event-row.warning { border-color: var(--accent-orange); }
		@media (max-width: 600px) {
			.log-filters { grid-template-columns: 1fr 1fr; }
			.settings-form { grid-template-columns: 1fr; }
			.settings-field.full, .settings-note, .settings-actions { grid-column: 1; }
			.header-btn-group { justify-content: flex-start; margin-left: 0; }
			.btn-mini { min-width: 0; padding: 9px 10px; font-size: 11px; }
		}

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
                    <button onclick="openSettings()" class="btn-mini btn-mini-status">SETTINGS</button>
                    <a href="/api/diagnostics" class="btn-mini btn-mini-status" style="text-decoration:none">DIAGNOSTICS</a>
                    <button onclick="logCurrentStatus()" class="btn-mini btn-mini-status">CHECK</button>
                    <button onclick="restartCluster()" class="btn-mini btn-mini-restart">RESTART</button>
                    <button onclick="shutdownCluster()" class="btn-mini btn-mini-danger">SHUTDOWN</button>
					<form class="logout-form" method="post" action="/logout"><button type="submit" class="btn-mini btn-mini-status">LOGOUT</button></form>
                </div>
            </div>
        </header>

        <div id="alertCenter" class="alert-center">
            <div class="alert-head"><span>ACTIVE ALERTS</span><span id="alertCount">0</span></div>
            <div id="alertList" class="alert-list"></div>
        </div>

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
                            <div id="systemConnection" class="btop-terminal"><span class="ok">●</span> probe / record / web server running</div>
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
				<div id="pipelineSelfCheck" class="selfcheck-strip running">
					<span><b>RECORDING PIPELINE SELF-TEST</b> · <span class="selfcheck-status">RUNNING</span></span>
					<span class="selfcheck-detail">Streamlink → pipe → FFmpeg → TS → ffprobe</span>
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
								<span class="sleep-state">RUNNING</span>
                            </div>
							<div class="cooldown-layout">
								<div class="cooldown-stage">
									<div class="cooldown-ring"><div class="cooldown-ring-core"><span class="sleep-time">--:--</span><span class="cooldown-unit">UNTIL NEXT PROBE</span></div></div>
									<div class="cooldown-controls">
										<button type="button" class="cooldown-btn pause" onclick="setProbePaused(this, true)" title="Pause automatic probes">PAUSE</button>
										<button type="button" class="cooldown-btn start" onclick="setProbePaused(this, false)" title="Resume automatic probes" disabled>START</button>
									</div>
								</div>
								<div class="cooldown-detail">
									<div class="sleep-sub"><span class="sleep-mode">outside battle window</span><span class="sleep-percent">-- remaining</span></div>
								</div>
							</div>
                        </div>
                        <div class="live-vitals" {{if $state.IsRecording}}style="display:block"{{end}}>
                            <div class="live-vitals-head">
                                <span class="live-vitals-label">LIVE THROUGHPUT <span class="live-health starting">STARTING</span></span>
                                <span class="live-speed">-- MB/s</span>
                            </div>
                            <div class="throughput-wrap">
								<div class="throughput-hud"><span class="throughput-average">AVG --</span><span class="throughput-peak">PEAK --</span></div>
								<svg class="throughput-wave" viewBox="0 0 320 100" preserveAspectRatio="none" aria-hidden="true">
									<polygon class="throughput-area" points="0,100 320,100"></polygon>
									<polyline class="throughput-line-glow" points=""></polyline>
									<polyline class="throughput-line" points=""></polyline>
								</svg>
                                <div class="throughput-bars"></div>
                                <div class="throughput-empty">WAITING FOR FIRST WRITE</div>
                            </div>
                            <div class="live-vitals-sub">
                                <span class="live-file">waiting file</span>
                                <span class="live-size">--</span>
                            </div>
							<div class="integrity-card">
								<div class="integrity-head"><span class="integrity-title">SESSION INTEGRITY</span><span class="integrity-score">100.0%</span></div>
								<div class="integrity-track"><div class="integrity-fill"></div></div>
								<div class="integrity-grid">
									<div class="integrity-stat"><span class="integrity-stat-label">RECORDED</span><span class="integrity-stat-value integrity-recorded">--</span></div>
									<div class="integrity-stat"><span class="integrity-stat-label">SEGMENTS</span><span class="integrity-stat-value integrity-segments">0</span></div>
									<div class="integrity-stat"><span class="integrity-stat-label">RESTARTS</span><span class="integrity-stat-value integrity-restarts">0</span></div>
									<div class="integrity-stat"><span class="integrity-stat-label">TOTAL GAP</span><span class="integrity-stat-value integrity-gap">0.0s</span></div>
									<div class="integrity-stat"><span class="integrity-stat-label">LONGEST GAP</span><span class="integrity-stat-value integrity-max-gap">0.0s</span></div>
									<div class="integrity-stat"><span class="integrity-stat-label">LAST RECOVERY</span><span class="integrity-stat-value integrity-recovery">--</span></div>
									<div class="integrity-stat"><span class="integrity-stat-label">VERIFIED FILES</span><span class="integrity-stat-value integrity-verified">0 / 0</span></div>
									<div class="integrity-stat"><span class="integrity-stat-label">SESSION DATA</span><span class="integrity-stat-value integrity-bytes">0 B</span></div>
									<div class="integrity-stat"><span class="integrity-stat-label">DISK LEFT</span><span class="integrity-stat-value integrity-disk">--</span></div>
								</div>
							</div>
                            <div class="pipeline-grid">
                                <div class="pipeline-item"><span class="pipeline-label">RECORDING TIME</span><span class="pipeline-value live-recording-time">--:--:--</span></div>
                                <div class="pipeline-item"><span class="pipeline-label">SEGMENT TIME</span><span class="pipeline-value live-segment-time">--:--:--</span></div>
                                <div class="pipeline-item"><span class="pipeline-label">LAST WRITE</span><span class="pipeline-value live-write-age">waiting</span></div>
                                <div class="pipeline-item"><span class="pipeline-label">FFMPEG</span><span class="pipeline-value live-ffmpeg">--</span></div>
								<div class="pipeline-item pipeline-watchdog starting"><span class="pipeline-label">WRITE WATCHDOG · WARN 30s · RESTART 60s</span><span class="pipeline-value live-watchdog">STARTUP · waiting for first write</span></div>
								<div class="pipeline-item"><span class="pipeline-label">SELECTED QUALITY</span><span class="pipeline-value live-selected-quality">--</span></div>
								<div class="pipeline-item"><span class="pipeline-label">AVAILABLE QUALITIES</span><span class="pipeline-value live-available-qualities">--</span></div>
                                <div class="pipeline-item pipeline-processes"><span class="pipeline-value live-streamlink-pid">streamlink: --</span><span class="pipeline-value live-ffmpeg-pid">ffmpeg: --</span></div>
                            </div>
                        </div>
                    </div>
                    <div class="channel-metrics">
                        <div class="metric-cell"><span class="metric-label">PROBE SUCCESS</span><span class="metric-value metric-probe">{{$state.ProbeSuccessRate | printf "%.1f"}}% / {{$state.ProbeAttempts}}</span></div>
                        <div class="metric-cell"><span class="metric-label">AVG PROBE</span><span class="metric-value metric-probe-avg">{{$state.ProbeAverageDuration | printf "%.0f"}} ms</span></div>
                        <div class="metric-cell"><span class="metric-label">REC RESTARTS</span><span class="metric-value metric-restarts">{{$state.RecordingRestartCount}}</span></div>
                        <div class="metric-cell"><span class="metric-label">FFMPEG ABNORMAL</span><span class="metric-value metric-ffmpeg-errors">{{$state.FFmpegAbnormalExits}}</span></div>
                        <div class="metric-cell"><span class="metric-label">START FAILURES</span><span class="metric-value metric-start-failures">{{$state.RecordingStartFailures}}</span></div>
                        <div class="metric-cell"><span class="metric-label">RECORDED</span><span class="metric-value metric-recorded" data-bytes="{{$state.RecordedBytes}}">--</span></div>
                        <div class="metric-cell"><span class="metric-label">LAST WRITE</span><span class="metric-value metric-last-write">{{if $state.LastSuccessfulWrite}}{{$state.LastSuccessfulWrite}}{{else}}--{{end}}</span></div>
                        <div class="metric-cell"><span class="metric-label">SESSION ID</span><span class="metric-value metric-session" title="{{$state.SessionID}}">{{if $state.SessionID}}{{$state.SessionID}}{{else}}--{{end}}</span></div>
                        <div class="metric-cell"><span class="metric-label">RECORDING SINCE</span><span class="metric-value metric-recording-since">{{if $state.RecordingStartedAt}}{{$state.RecordingStartedAt}}{{else}}--{{end}}</span></div>
                        <div class="metric-cell metric-error {{if $state.LastError}}has-error{{end}}"><span class="metric-label">LAST ERROR · <span class="metric-error-at">{{$state.LastErrorAt}}</span></span><span class="metric-value metric-last-error" title="{{$state.LastError}}">{{$state.LastError}}</span></div>
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
                            <a class="history-link" href="/download/{{.Channel}}/{{.Name}}" download="{{.Name}}">
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
                    <button onclick="openSessionDetail(this)" class="btn btn-probe">DETAILS</button>
                    {{if $state.IsRecording}}
                    <button onclick="restartStream(this)" class="btn btn-restart action-btn">RESTART REC</button>
                    {{else}}
                    <button onclick="forceProbe(this)" class="btn btn-probe action-btn" {{if $state.IsProbing}}disabled{{end}}>SCAN NOW</button>
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
            <div class="log-filters">
                <select id="logLevel" onchange="fetchLogs()"><option value="">ALL</option><option>INFO</option><option>WARN</option><option>ERROR</option></select>
                <input id="logChannel" placeholder="channel">
                <input id="logQuery" placeholder="keyword / session id">
                <button onclick="fetchLogs()" class="log-close">FILTER</button>
            </div>
            <div id="logBody" class="log-body">Connecting to log stream...</div>
        </div>
    </div>

    <div id="sessionModal" class="log-modal">
        <div class="log-box">
            <div class="log-header"><div class="log-title">SESSION DETAIL</div><button onclick="closeSessionDetail()" class="log-close">CLOSE</button></div>
            <div id="sessionBody" class="detail-body">Loading session...</div>
        </div>
    </div>

    <div id="settingsModal" class="log-modal">
        <div class="log-box">
            <div class="log-header"><div class="log-title">CONFIG SETTINGS</div><button onclick="closeSettings()" class="log-close">CLOSE</button></div>
            <div class="detail-body">
                <div class="settings-form">
                    <label class="settings-field full">TARGET URLS · ONE PER LINE<textarea id="cfgTargets"></textarea></label>
                    <label class="settings-field">WEB PORT<input id="cfgPort" type="number" min="1" max="65535"></label>
                    <label class="settings-field">PROBE INTERVAL · SECONDS<input id="cfgInterval" type="number" min="1"></label>
                    <label class="settings-field">WINDOW START<input id="cfgStart" type="time"></label>
                    <label class="settings-field">WINDOW END<input id="cfgEnd" type="time"></label>
                    <label class="settings-field">MAX DEEP SLEEP · SECONDS<input id="cfgSleep" type="number" min="1"></label>
                    <div class="settings-note">Changing target URLs or Web Port requires a Web restart. Schedule values apply immediately after save.</div>
                    <div id="settingsMessage" class="settings-note"></div>
                    <div class="settings-actions"><button onclick="saveSettings()" class="log-close">SAVE CONFIG</button></div>
                </div>
            </div>
        </div>
    </div>

    <script>
        console.log("go_straight template: scan-fix-v2");
        let logInterval = null;
        const streamVitals = {};
        const sleepTimers = {};
        const manualProbeHolds = {};
		const liveHistoryEstimates = {};
		let statusRequestInFlight = false;

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

        function escapeHtml(text) {
            return String(text || "")
                .replace(/&/g, "&amp;")
                .replace(/</g, "&lt;")
                .replace(/>/g, "&gt;")
                .replace(/"/g, "&quot;")
                .replace(/'/g, "&#039;");
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
			document.querySelectorAll(".log-modal").forEach(modal => {
				modal.addEventListener("click", event => {
					if (event.target === modal) closeModal(modal);
				});
			});
        });

		document.addEventListener("keydown", event => {
			if (event.key === "Escape") {
				const openModal = Array.from(document.querySelectorAll(".log-modal")).find(modal => modal.style.display === "block");
				if (openModal) closeModal(openModal);
			}
		});

		function closeModal(modal) {
			if (!modal) return;
			modal.style.display = "none";
			document.body.style.overflow = "auto";
			if (modal.id === "logModal" && logInterval) {
				clearInterval(logInterval);
				logInterval = null;
			}
		}

        function openLogViewer() {
            const modal = document.getElementById("logModal");
            modal.style.display = "block";
            document.body.style.overflow = "hidden";
            
            fetchLogs();
            logInterval = setInterval(fetchLogs, 2000); 
        }

        function closeLogViewer() {
			closeModal(document.getElementById("logModal"));
        }

        function fetchLogs() {
            const logBody = document.getElementById("logBody");
			const params = new URLSearchParams();
			const level = document.getElementById("logLevel").value;
			const channel = document.getElementById("logChannel").value;
			const query = document.getElementById("logQuery").value;
			if (level) params.set("level", level);
			if (channel) params.set("channel", channel);
			if (query) params.set("q", query);
            fetch('/api/logs?' + params.toString())
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

		function closeSessionDetail() {
			closeModal(document.getElementById("sessionModal"));
		}

		function closeSettings() {
			closeModal(document.getElementById("settingsModal"));
		}

		function openSettings() {
			const modal = document.getElementById("settingsModal");
			const message = document.getElementById("settingsMessage");
			modal.style.display = "block";
			document.body.style.overflow = "hidden";
			message.innerText = "Loading config...";
			fetch('/api/config').then(r => {
				if (!r.ok) return r.json().then(e => { throw new Error(e.error || 'Config load failed'); });
				return r.json();
			}).then(cfg => {
				document.getElementById("cfgTargets").value = (cfg.target_urls || []).join("\n");
				document.getElementById("cfgPort").value = cfg.web_port || "";
				document.getElementById("cfgStart").value = cfg.probe_start || "";
				document.getElementById("cfgEnd").value = cfg.probe_end || "";
				document.getElementById("cfgInterval").value = cfg.probe_interval || "";
				document.getElementById("cfgSleep").value = cfg.probe_sleep_deep || "";
				message.innerText = "";
			}).catch(err => message.innerText = err.message);
		}

		function saveSettings() {
			const message = document.getElementById("settingsMessage");
			const cfg = {
				target_urls: document.getElementById("cfgTargets").value.split(/\r?\n/).map(v => v.trim()).filter(Boolean),
				web_port: Number(document.getElementById("cfgPort").value),
				probe_start: document.getElementById("cfgStart").value,
				probe_end: document.getElementById("cfgEnd").value,
				probe_interval: Number(document.getElementById("cfgInterval").value),
				probe_sleep_deep: Number(document.getElementById("cfgSleep").value)
			};
			message.innerText = "Saving...";
			fetch('/api/config', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(cfg)}).then(r => r.json().then(data => {
				if (!r.ok) throw new Error(data.error || 'Config save failed');
				return data;
			})).then(result => {
				message.innerText = result.restart_required ? "Saved. Restart required." : "Saved and applied.";
				if (result.restart_required && confirm("Config saved. Restart now to apply channel or port changes?")) {
					fetch('/api/restart_cluster', {method:'POST'}).finally(() => {
						setTimeout(() => { window.location.href = 'http://' + window.location.hostname + ':' + cfg.web_port + '/'; }, 4500);
					});
				}
			}).catch(err => message.innerText = err.message);
		}

		function openSessionDetail(btn) {
			const prefix = getChannelPrefix(btn, "");
			const modal = document.getElementById("sessionModal");
			const body = document.getElementById("sessionBody");
			modal.style.display = "block";
			document.body.style.overflow = "hidden";
			body.innerText = "Loading session...";
			fetch('/api/session?prefix=' + encodeURIComponent(prefix)).then(r => r.json()).then(data => {
				const s = data.state || {};
				const trend = Array.isArray(data.trend) ? data.trend : [];
				const values = trend.map(p => Number(p.write_speed_bytes || 0) / 1024 / 1024);
				const max = Math.max(.01, ...values);
				const points = values.map((v,i) => ((i / Math.max(1, values.length-1))*600).toFixed(1) + ',' + (95-(v/max)*85).toFixed(1)).join(' ');
				const events = (data.events || []).slice().reverse().map(e => '<div class="event-row '+escapeHtml(e.level)+'"><b>'+escapeHtml(e.time)+' · '+escapeHtml(e.type)+'</b><br>'+escapeHtml(e.message)+'</div>').join('') || '<div class="history-empty">No events yet</div>';
				body.innerHTML = '<div class="detail-summary">' +
					'<div class="detail-cell">CHANNEL<br><b>@'+escapeHtml(prefix)+'</b></div>' +
					'<div class="detail-cell">SESSION<br><b>'+escapeHtml(s.session_id || '--')+'</b></div>' +
					'<div class="detail-cell">PIPELINE<br><b>'+escapeHtml(s.pipeline_state || '--')+'</b></div>' +
					'<div class="detail-cell">TOTAL FILES<br><b>'+String((data.files || []).length)+'</b></div></div>' +
					'<div class="metric-label">WRITE SPEED TREND · '+values.length+' SAMPLES</div><svg class="trend-chart" viewBox="0 0 600 100" preserveAspectRatio="none"><polyline points="'+points+'" fill="none" stroke="#a6e3a1" stroke-width="2"/></svg>' +
					'<div class="metric-label">EVENT TIMELINE</div><div class="event-list">'+events+'</div>';
			}).catch(err => body.innerText = "Session unavailable: " + err);
		}

		function renderAlerts(data) {
			const alerts = Array.isArray(data.alerts) ? data.alerts : [];
			const center = document.getElementById("alertCenter");
			document.getElementById("alertCount").innerText = String(alerts.length);
			document.getElementById("alertList").innerHTML = alerts.map(a => '<div class="alert-item '+escapeHtml(a.level)+'">'+(a.channel ? '@'+escapeHtml(a.channel)+' · ' : '')+escapeHtml(a.message)+'</div>').join('');
			center.classList.toggle("active", alerts.length > 0);
		}

        function logCurrentStatus() {
            fetch('/api/log_status', { method: 'POST' })
                .then(r => r.json())
                .then(d => {
					const strip = document.getElementById("pipelineSelfCheck");
					if (strip) strip.querySelector(".selfcheck-status").innerText = "RUNNING";
                })
				.catch(err => alert("Pipeline self-test failed to start: " + err));
        }

        function restartCluster() {
            if (!confirm("WARNING: this will interrupt all active recordings and restart the Go core service. Continue?")) return;
            
            const controller = new AbortController();
            const timeoutId = setTimeout(() => controller.abort(), 1200);

            alert("Restart command sent. Please refresh this dashboard in 5 seconds.");

            fetch('/api/restart_cluster', { method: 'POST', signal: controller.signal })
                .then(r => r.json())
                .then(d => {
                    clearTimeout(timeoutId);
                    location.reload();
                })
                .catch(e => {
                    console.log("Core restart in progress...");
                });
        }

        function getChannelPrefix(btn, prefix) {
            let p = String(prefix || "").trim();
            if (!p && btn && typeof btn.closest === "function") {
                const box = btn.closest(".channel-box");
                if (box) p = String(box.getAttribute("data-channel") || "").trim();
            }
            return p.replace(/^@+/, "");
        }

        function forceProbe(btn, prefix) {
            prefix = getChannelPrefix(btn, prefix);
            if (!prefix) {
                alert("Missing channel prefix");
                return;
            }
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

            fetch('/api/probe?prefix=' + encodeURIComponent(prefix), { method: 'POST' })
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
            prefix = getChannelPrefix(btn, prefix);
            if (!prefix) {
                alert("Missing channel prefix");
                return;
            }

            if (!confirm("Force restart the current recording for @" + prefix + "?\nThe current segment will be closed and a new file will be opened.")) return;
            btn.disabled = true;
            btn.innerText = "RESTARTING...";
            fetch('/api/restart?prefix=' + encodeURIComponent(prefix), { method: 'POST' })
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

        function parseServerTime(value) {
            if (!value) return null;
            const parsed = new Date(String(value).replace(" ", "T"));
            return Number.isNaN(parsed.getTime()) ? null : parsed;
        }

        function elapsedSeconds(value) {
            const parsed = parseServerTime(value);
            return parsed ? Math.max(0, Math.floor((Date.now() - parsed.getTime()) / 1000)) : null;
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

        function paintSleepCountdown(box, prefix, remaining, status, paused) {
            const sleepBox = box.querySelector(".sleep-countdown");
            if (!sleepBox) return;

            const state = sleepTimers[prefix];
            const total = Math.max(1, state ? state.total : inferCountdownTotal(remaining));
            const pct = Math.max(0, Math.min(100, ((total - remaining) / total) * 100));

            const label = sleepBox.querySelector(".sleep-label");
            const time = sleepBox.querySelector(".sleep-time");
			const ring = sleepBox.querySelector(".cooldown-ring");
            const mode = sleepBox.querySelector(".sleep-mode");
            const percent = sleepBox.querySelector(".sleep-percent");
			const stateLabel = sleepBox.querySelector(".sleep-state");
			const pauseButton = sleepBox.querySelector(".cooldown-btn.pause");
			const startButton = sleepBox.querySelector(".cooldown-btn.start");

            if (label) {
                label.innerText = String(status || "").includes("刺探") ? "PROBE COOLDOWN" : "SLEEP TIMER";
            }
            if (time) time.innerText = formatCountdown(remaining);
			if (ring) ring.style.setProperty("--progress", (pct * 3.6).toFixed(1) + "deg");
			sleepBox.classList.toggle("paused", !!paused);
			if (stateLabel) stateLabel.innerText = paused ? "PAUSED" : "RUNNING";
			if (pauseButton) pauseButton.disabled = !!paused;
			if (startButton) startButton.disabled = !paused;
            if (mode) {
                mode.innerText = String(status || "").includes("非戰備") ? "outside battle window" : "next probe cooldown";
            }
			if (percent) percent.innerText = paused ? "countdown frozen" : Math.max(0, 100 - pct).toFixed(0) + "% remaining";
        }

        function renderSleepCountdown(box, prefix, stream, status) {
            const msgCell = box.querySelector(".probe-msg");
            const sleepBox = box.querySelector(".sleep-countdown");
            if (!msgCell || !sleepBox) return false;

            const serverRemaining = parseCountdownSeconds(status);
			const paused = !!(stream && stream.probe_paused);
            if (serverRemaining === null && !paused) {
                sleepBox.style.display = "none";
                delete sleepTimers[prefix];
                return false;
            }

            const now = Date.now();
            let state = sleepTimers[prefix];

			const initialRemaining = serverRemaining === null ? 0 : serverRemaining;
            if (!state) {
                state = {
					total: inferCountdownTotal(initialRemaining),
					baseRemaining: initialRemaining,
                    baseAt: now,
					status: status,
					paused: paused
                };
            } else {
                const smoothRemaining = Math.max(
                    0,
                    state.baseRemaining - Math.floor((now - state.baseAt) / 1000)
                );

                // Do not let a slower / stale backend status pull the UI countdown backwards.
				if (serverRemaining !== null && (paused || serverRemaining <= smoothRemaining || serverRemaining > smoothRemaining + 5)) {
                    state.baseRemaining = serverRemaining;
                    state.baseAt = now;
                }

				if (serverRemaining !== null && (serverRemaining > state.total || serverRemaining > smoothRemaining + 5)) {
                    state.total = inferCountdownTotal(serverRemaining);
                }

                state.status = status;
				if (state.paused && !paused) state.baseAt = now;
				state.paused = paused;
            }

            sleepTimers[prefix] = state;

			const displayRemaining = state.paused ? state.baseRemaining : Math.max(0, state.baseRemaining - Math.floor((Date.now() - state.baseAt) / 1000));

			paintSleepCountdown(box, prefix, displayRemaining, status, state.paused);

            msgCell.style.display = "none";
            sleepBox.style.display = "block";
            return true;
        }

        function tickSleepCountdowns() {
            Object.entries(sleepTimers).forEach(([prefix, state]) => {
                const box = document.querySelector('div[data-channel="' + prefix + '"]');
                if (!box) return;

                const sleepBox = box.querySelector(".sleep-countdown");
                if (!sleepBox || sleepBox.style.display === "none") return;

				const remaining = state.paused ? state.baseRemaining : Math.max(0, state.baseRemaining - Math.floor((Date.now() - state.baseAt) / 1000));

				paintSleepCountdown(box, prefix, remaining, state.status || "", state.paused);
            });
        }

		function setProbePaused(button, paused) {
			const prefix = getChannelPrefix(button, "");
			if (!prefix || button.disabled) return;
			button.disabled = true;
			fetch('/api/probe_pause?prefix=' + encodeURIComponent(prefix) + '&paused=' + String(paused), {method:'POST'})
				.then(r => r.json().then(data => {
					if (!r.ok) throw new Error(data.error || 'Unable to change probe state');
					const state = sleepTimers[prefix];
					if (state) {
						const now = Date.now();
						if (paused && !state.paused) state.baseRemaining = Math.max(0, state.baseRemaining - Math.floor((now - state.baseAt) / 1000));
						state.baseAt = now;
						state.paused = paused;
						const box = button.closest('.channel-box');
						if (box) paintSleepCountdown(box, prefix, state.baseRemaining, state.status || '', paused);
					} else {
						const sleepBox = button.closest('.sleep-countdown');
						if (sleepBox) {
							sleepBox.classList.toggle('paused', paused);
							const stateLabel = sleepBox.querySelector('.sleep-state');
							const pauseButton = sleepBox.querySelector('.cooldown-btn.pause');
							const startButton = sleepBox.querySelector('.cooldown-btn.start');
							if (stateLabel) stateLabel.innerText = paused ? 'PAUSED' : 'RUNNING';
							if (pauseButton) pauseButton.disabled = paused;
							if (startButton) startButton.disabled = !paused;
						}
					}
				}))
				.catch(error => { alert(error.message); button.disabled = false; });
		}

		function renderThroughput(liveVitals, samples, hasData) {
			const chart = liveVitals.querySelector(".throughput-bars");
			const empty = liveVitals.querySelector(".throughput-empty");
			const line = liveVitals.querySelector(".throughput-line");
			const glow = liveVitals.querySelector(".throughput-line-glow");
			const area = liveVitals.querySelector(".throughput-area");
			const averageLabel = liveVitals.querySelector(".throughput-average");
			const peakLabel = liveVitals.querySelector(".throughput-peak");
			if (!chart || !empty || !line || !glow || !area) return;
			empty.classList.toggle("hidden", hasData);
			if (!hasData) {
				chart.innerHTML = "";
				line.setAttribute("points", "");
				glow.setAttribute("points", "");
				area.setAttribute("points", "0,100 320,100");
				if (averageLabel) averageLabel.innerText = "AVG --";
				if (peakLabel) peakLabel.innerText = "PEAK --";
				return;
			}
			const values = samples.slice(-32);
			const max = Math.max(0.01, ...values);
			const average = values.reduce((sum, value) => sum + value, 0) / Math.max(1, values.length);
			const padded = values.length > 1 ? values : [values[0] || 0, values[0] || 0];
			const points = padded.map((value, index) => {
				const x = (index / Math.max(1, padded.length - 1)) * 320;
				const normalized = Math.max(0, Math.min(1, value / max));
				const y = 91 - normalized * 72;
				return x.toFixed(1) + "," + y.toFixed(1);
			}).join(" ");
			line.setAttribute("points", points);
			glow.setAttribute("points", points);
			area.setAttribute("points", "0,100 " + points + " 320,100");
			if (averageLabel) averageLabel.innerText = "AVG " + average.toFixed(2) + " MB/s";
			if (peakLabel) peakLabel.innerText = "PEAK " + max.toFixed(2) + " MB/s";
			chart.innerHTML = values.map(value => {
				const height = Math.max(4, Math.min(100, (value / max) * 100));
				const opacity = value > 0 ? 1 : .22;
				return '<i class="throughput-bar" style="height:'+height.toFixed(1)+'%;opacity:'+opacity+'"></i>';
			}).join("");
        }

		function updateLiveVitals(box, prefix, stream, system, estimatedRemainingSeconds) {
            const msgCell = box.querySelector(".probe-msg");
            const liveVitals = box.querySelector(".live-vitals");
            if (!msgCell || !liveVitals) return;

            const sleepBox = box.querySelector(".sleep-countdown");
            const probeBox = box.querySelector(".probe-scan");

            if (!stream.is_recording) {
                liveVitals.style.display = "none";
                if (renderProbeScan(box, prefix, stream, stream.probe_status)) {
                    if (sleepBox) sleepBox.style.display = "none";
				} else if (!renderSleepCountdown(box, prefix, stream, stream.probe_status)) {
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
					lastChangeTime: now,
					lastSampleTime: 0,
					lastSpeed: 0,
					hasData: false,
					speeds: []
                };
            }

            const diff = currentSize - state.lastSize;
			const seconds = Math.max(0.001, (now - state.lastChangeTime) / 1000);
			let measuredSpeed = 0;
			if (diff > 0) {
				measuredSpeed = diff / 1024 / 1024 / seconds;
				state.lastChangeTime = now;
				state.lastSize = currentSize;
				state.hasData = true;
			}
			const backendSpeed = Number(stream.write_bytes_per_second || 0) / 1024 / 1024;
			let speed = backendSpeed > 0 ? backendSpeed : measuredSpeed;
			if (speed > 0) {
				state.lastSpeed = state.lastSpeed > 0 ? state.lastSpeed * .55 + speed * .45 : speed;
				state.hasData = true;
			} else if (now - state.lastChangeTime < 6500) {
				speed = state.lastSpeed;
			} else {
				state.lastSpeed *= .65;
				speed = state.lastSpeed;
			}
			if (now - state.lastSampleTime >= 1000) {
				state.speeds.push(Math.max(0, speed));
				if (state.speeds.length > 32) state.speeds.shift();
				state.lastSampleTime = now;
			}
            streamVitals[prefix] = state;

            const speedText = liveVitals.querySelector(".live-speed");
            const fileText = liveVitals.querySelector(".live-file");
            const sizeText = liveVitals.querySelector(".live-size");

			if (speedText) speedText.innerText = state.hasData ? Math.max(0, speed).toFixed(2) + " MB/s" : "WAITING";
            if (fileText) fileText.innerText = stream.latest_file || "waiting file";
            if (sizeText) sizeText.innerText = formatBytes(currentSize);
			renderThroughput(liveVitals, state.speeds, state.hasData);

			const recordingAge = elapsedSeconds(stream.recording_started_at);
			const segmentAge = elapsedSeconds(stream.segment_started_at);
			const writeAge = elapsedSeconds(stream.last_successful_write);
			const health = liveVitals.querySelector(".live-health");
			let healthState = "healthy", healthText = "HEALTHY";
			if (writeAge === null || stream.pipeline_state === "STARTING") {
				healthState = "starting";
				healthText = "STARTING";
			} else if (writeAge >= 60 || stream.pipeline_state === "WRITE_STALLED") {
				healthState = "stale";
				healthText = "RESTARTING";
			} else if (writeAge >= 30 || stream.pipeline_state === "WRITE_DELAYED") {
				healthState = "delayed";
				healthText = "DELAYED";
			}
			if (health) {
				health.className = "live-health " + healthState;
				health.innerText = healthText;
			}
			const setLiveText = (selector, value) => {
				const element = liveVitals.querySelector(selector);
				if (element) element.innerText = value;
			};
			setLiveText(".live-recording-time", recordingAge === null ? "--:--:--" : formatCountdown(recordingAge));
			setLiveText(".live-segment-time", segmentAge === null ? "--:--:--" : formatCountdown(segmentAge));
			setLiveText(".live-write-age", writeAge === null ? "waiting" : writeAge + "s ago");
			const watchdog = liveVitals.querySelector(".pipeline-watchdog");
			let watchdogState = "healthy";
			let watchdogText = "HEALTHY · last write " + (writeAge === null ? "waiting" : writeAge + "s ago");
			if (writeAge === null) {
				watchdogState = "starting";
				const startupRemaining = Math.max(0, 75 - Number(segmentAge || 0));
				watchdogText = "STARTUP · first-write timeout in " + startupRemaining + "s";
			} else if (writeAge >= 60 || stream.pipeline_state === "WRITE_STALLED") {
				watchdogState = "stalled";
				watchdogText = "STALLED · rebuilding recording pipeline";
			} else if (writeAge >= 30 || stream.pipeline_state === "WRITE_DELAYED") {
				watchdogState = "delayed";
				watchdogText = "DELAYED · automatic restart in " + Math.max(0, 60 - writeAge) + "s";
			}
			if (watchdog) watchdog.className = "pipeline-item pipeline-watchdog " + watchdogState;
			setLiveText(".live-watchdog", watchdogText);
			const ffmpegInfo = [stream.ffmpeg_bitrate, stream.ffmpeg_speed].filter(Boolean).join(" · ");
			setLiveText(".live-ffmpeg", ffmpegInfo || stream.pipeline_state || "--");
			const selectedQuality = [stream.selected_quality, stream.selected_stream_type ? '(' + stream.selected_stream_type + ')' : ''].filter(Boolean).join(' ');
			setLiveText(".live-selected-quality", selectedQuality || "waiting");
			setLiveText(".live-available-qualities", Array.isArray(stream.available_qualities) && stream.available_qualities.length ? stream.available_qualities.join(', ') : "--");
			setLiveText(".live-streamlink-pid", "streamlink: " + (stream.streamlink_pid || "--"));
			setLiveText(".live-ffmpeg-pid", "ffmpeg: " + (stream.ffmpeg_pid || "--"));

			const integrity = liveVitals.querySelector(".integrity-card");
			const integrityScore = Math.max(0, Math.min(100, Number(stream.session_health_percent ?? 100)));
			if (integrity) integrity.className = "integrity-card " + (integrityScore < 95 || Number(stream.broken_segments || 0) > 0 ? "danger" : (integrityScore < 99.5 ? "warning" : "healthy"));
			const integrityScoreEl = liveVitals.querySelector(".integrity-score");
			if (integrityScoreEl) integrityScoreEl.innerText = integrityScore.toFixed(1) + "%";
			const integrityFill = liveVitals.querySelector(".integrity-fill");
			if (integrityFill) integrityFill.style.width = integrityScore.toFixed(1) + "%";
			const msText = value => Number(value || 0) > 0 ? (Number(value) / 1000).toFixed(1) + "s" : "0.0s";
			setLiveText(".integrity-recorded", recordingAge === null ? "--" : formatCountdown(recordingAge));
			setLiveText(".integrity-segments", String(stream.session_segment_count || 0));
			setLiveText(".integrity-restarts", String(stream.session_restart_count || 0));
			setLiveText(".integrity-gap", msText(stream.session_gap_total_ms));
			setLiveText(".integrity-max-gap", msText(stream.session_max_gap_ms));
			setLiveText(".integrity-recovery", Number(stream.last_recovery_ms || 0) > 0 ? msText(stream.last_recovery_ms) : "--");
			setLiveText(".integrity-verified", String(stream.verified_segments || 0) + " / " + String(stream.broken_segments || 0) + " bad");
			setLiveText(".integrity-bytes", formatBytes(Number(stream.session_recorded_bytes || 0)));
			const diskText = Number(estimatedRemainingSeconds || 0) > 0 ? formatCountdown(estimatedRemainingSeconds) : formatBytes(Number(system?.disk_avail || 0));
			setLiveText(".integrity-disk", diskText);
        }

        function updateChannelMetrics(box, stream) {
            const setText = (selector, value) => {
                const element = box.querySelector(selector);
                if (element) element.innerText = value;
                return element;
            };
            const attempts = Number(stream.probe_attempts || 0);
            setText(".metric-probe", Number(stream.probe_success_rate || 0).toFixed(1) + "% / " + attempts);
            setText(".metric-probe-avg", Number(stream.probe_average_duration_ms || 0).toFixed(0) + " ms");
            setText(".metric-restarts", String(stream.recording_restart_count || 0));
            setText(".metric-ffmpeg-errors", String(stream.ffmpeg_abnormal_exits || 0));
			setText(".metric-start-failures", String(stream.recording_start_failures || 0));
            setText(".metric-recorded", formatBytes(Number(stream.recorded_bytes || 0)));
            setText(".metric-last-write", stream.last_successful_write || "--");
            const session = setText(".metric-session", stream.session_id || "--");
            if (session) session.title = stream.session_id || "";
            setText(".metric-recording-since", stream.recording_started_at || "--");

            const errorCell = box.querySelector(".metric-error");
            const errorText = setText(".metric-last-error", stream.last_error || "");
            setText(".metric-error-at", stream.last_error_at || "");
            if (errorText) errorText.title = stream.last_error || "";
            if (errorCell) errorCell.classList.toggle("has-error", !!stream.last_error);
        }

        function renderHistoryList(box, prefix, files, stream) {
            const count = box.querySelector(".history-count");
            const list = box.querySelector(".history-list");
            if (!list) return;

            files = Array.isArray(files) ? files : [];
			const growingFile = files.find(file => !!(file.is_growing ?? file.IsGrowing));
			if (growingFile && stream && stream.is_recording) {
				const actualSize = Number(growingFile.size_bytes ?? growingFile.SizeBytes ?? stream.latest_size ?? 0);
				const speed = Math.max(0, Number(stream.write_bytes_per_second || 0));
				const previous = liveHistoryEstimates[prefix];
				liveHistoryEstimates[prefix] = {
					file: growingFile.name || growingFile.Name || "",
					baseSize: actualSize,
					baseAt: Date.now(),
					speed: speed > 0 ? speed : (previous && previous.file === (growingFile.name || growingFile.Name || "") ? previous.speed : 0)
				};
			} else {
				delete liveHistoryEstimates[prefix];
			}
			const signature = files.map(file => [file.name, file.size_bytes, file.mtime, file.is_growing, file.quality && file.quality.status].join(':')).join('|');
			if (list.dataset.signature === signature) return;
			const previousScroll = list.scrollTop;
			list.dataset.signature = signature;
            if (count) count.innerText = files.length + " files";

            if (files.length === 0) {
                list.innerHTML = '<div class="history-empty">No recorded segments</div>';
                return;
            }

            list.innerHTML = files.map(function(file) {
                const channel = file.channel || file.Channel || prefix;
                const name = file.name || file.Name || "";
                const sizeBytes = Number(file.size_bytes ?? file.SizeBytes ?? 0);
                const mtime = file.mtime || file.MTime || "";
                const isGrowing = !!(file.is_growing ?? file.IsGrowing);
				const quality = file.quality || file.Quality || null;
				const qualityText = quality ? [quality.status, quality.resolution, quality.video_codec, quality.audio_codec, quality.duration_seconds ? Number(quality.duration_seconds).toFixed(0)+'s' : ''].filter(Boolean).join(' · ') : (isGrowing ? 'RECORDING' : 'PENDING CHECK');
				const qualityClass = quality ? String(quality.status || '') : '';
                const href = "/download/" + encodeURIComponent(channel) + "/" + encodeURIComponent(name);
                return '' +
                    '<div class="history-item ' + (isGrowing ? 'row-growing' : '') + '" data-channel="' + escapeHtml(channel) + '" data-filename="' + escapeHtml(name) + '">' +
                        '<a class="history-link" href="' + href + '" download="' + escapeHtml(name) + '">' +
                            (isGrowing ? '<span class="pulse-dot"></span>' : '') +
                            '<span>' + escapeHtml(name) + '</span>' +
                        '</a>' +
                        '<div class="history-meta">' +
                            '<span class="file-size" data-bytes="' + sizeBytes + '">' + formatBytes(sizeBytes) + '</span>' +
                            '<span class="file-mtime">' + escapeHtml(mtime) + '</span>' +
                        '</div>' +
						'<span class="quality-badge '+escapeHtml(qualityClass)+'" title="'+escapeHtml(quality && quality.error || '')+'">'+escapeHtml(qualityText)+'</span>' +
                    '</div>';
            }).join("");
			list.scrollTop = previousScroll;
        }

		function tickLiveHistorySizes() {
			const now = Date.now();
			Object.entries(liveHistoryEstimates).forEach(([prefix, state]) => {
				if (!state || !state.file) return;
				const box = document.querySelector('div[data-channel="' + prefix + '"]');
				if (!box) return;
				const rows = box.querySelectorAll('.history-item.row-growing');
				let row = null;
				rows.forEach(candidate => { if (candidate.getAttribute('data-filename') === state.file) row = candidate; });
				if (!row) return;
				const elapsed = Math.max(0, (now - state.baseAt) / 1000);
				const estimatedSize = Math.max(state.baseSize, state.baseSize + state.speed * elapsed);
				const size = row.querySelector('.file-size');
				if (size) {
					size.setAttribute('data-bytes', String(Math.round(estimatedSize)));
					size.innerText = formatBytes(estimatedSize) + ' · LIVE';
				}
			});
		}

        function renderActionButton(box, prefix, stream) {
            prefix = String(prefix || "").trim().replace(/^@+/, "");

            const actions = box.querySelector(".channel-actions");
            if (!actions) return;
			const held = !!getManualProbeHold(prefix);
			const disabled = stream.is_probing || held;
			const actionState = stream.is_recording ? "recording" : (disabled ? "probing" : "idle");
			if (actions.dataset.state === actionState) return;
			actions.dataset.state = actionState;

            if (stream.is_recording) {
				actions.innerHTML = '<button onclick="openSessionDetail(this)" class="btn btn-probe">DETAILS</button><button onclick="restartStream(this)" class="btn btn-restart action-btn">RESTART REC</button>';
                return;
            }

			actions.innerHTML = '<button onclick="openSessionDetail(this)" class="btn btn-probe">DETAILS</button><button onclick="forceProbe(this)" class="btn btn-probe action-btn" ' + (disabled ? 'disabled' : '') + '>' + (disabled ? 'SCANNING...' : 'SCAN NOW') + '</button>';
        }

        function shutdownCluster() {
            if (confirm("WARNING: this will stop all recordings and shut down the Go core service. Continue?")) {
                fetch('/api/shutdown', { method: 'POST' }).then(r => r.json()).then(d => {
                    alert("Shutdown command sent. Core service is shutting down.");
                    window.close();
                }).catch(e => alert("Connection interrupted. The service may already be offline."));
            }
        }

        document.querySelectorAll('.file-size').forEach(function(td) {
            var b = parseInt(td.getAttribute('data-bytes'));
            if(!isNaN(b)) td.innerHTML = formatBytes(b);
        });

        setInterval(tickSleepCountdowns, 1000);
		setInterval(tickLiveHistorySizes, 1000);

		function pollStatus() {
			if (statusRequestInFlight || document.hidden) return;
			statusRequestInFlight = true;
            fetch('/api/status').then(r => r.json()).then(data => {
				const connection = document.getElementById("systemConnection");
				if (connection) connection.innerHTML = '<span class="ok">●</span> probe / record / web server running';
				renderAlerts(data);
				const selfCheck = data.pipeline_self_check || {};
				const selfCheckStrip = document.getElementById("pipelineSelfCheck");
				if (selfCheckStrip) {
					const status = String(selfCheck.status || "PENDING").toUpperCase();
					selfCheckStrip.className = "selfcheck-strip " + status.toLowerCase();
					selfCheckStrip.querySelector(".selfcheck-status").innerText = status;
					const detail = selfCheckStrip.querySelector(".selfcheck-detail");
					if (detail) detail.innerText = status === "PASS"
						? [selfCheck.resolution, selfCheck.video_codec, selfCheck.audio_codec, formatBytes(Number(selfCheck.output_bytes || 0)), Number(selfCheck.duration_ms || 0) + "ms"].filter(Boolean).join(" · ")
						: (selfCheck.error || "Streamlink → pipe → FFmpeg → TS → ffprobe");
				}
                var uptime = data.system.uptime || "--";
                var cpuUptimeText = document.getElementById("cpuUptimeText");
                if (cpuUptimeText) cpuUptimeText.innerText = "uptime: " + uptime;

                var totalGB = data.system.disk_total / (1024*1024*1024), availGB = data.system.disk_avail / (1024*1024*1024), usedGB = data.system.disk_used / (1024*1024*1024);
                var pct = totalGB > 0 ? (usedGB / totalGB) * 100 : 0;
                document.getElementById("diskText").innerText = pct.toFixed(1) + "%";
                document.getElementById("diskUsedText").innerText = usedGB.toFixed(2) + " GB";
                document.getElementById("diskFreeText").innerText = availGB.toFixed(2) + " GB";
                document.getElementById("diskSubText").innerHTML = "<span>" + usedGB.toFixed(2) + " / " + totalGB.toFixed(2) + " GB</span><span>free " + availGB.toFixed(2) + " GB</span>";
				if (Number(data.estimated_remaining_seconds || 0) > 0) {
					document.getElementById("diskSubText").innerHTML = "<span>" + usedGB.toFixed(2) + " / " + totalGB.toFixed(2) + " GB</span><span>recording left " + formatCountdown(data.estimated_remaining_seconds) + "</span>";
				}
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

                Object.entries(data.streams).forEach(([prefix, stream]) => {
                    var box = document.querySelector('div[data-channel="' + prefix + '"]');
                    if (!box) {
                        // New channel added by config hot reload: full refresh once to create its card.
                        location.reload();
                        return;
                    }

                    if(stream.is_recording) {
                        box.classList.add("recording");
                    } else {
                        box.classList.remove("recording");
                    }

                    const channelFiles = (data.files && data.files[prefix]) || [];
                    var badge = box.querySelector(".status-badge");
                    if (badge) {
						badge.className = stream.is_recording ? "badge badge-live" : "badge badge-offline";
						badge.innerHTML = stream.is_recording ? "REC" : "IDLE";
                    }

                    updateLiveVitals(box, prefix, stream, data.system, data.estimated_remaining_seconds);
                    updateChannelMetrics(box, stream);
                    renderActionButton(box, prefix, stream);
                    renderHistoryList(box, prefix, channelFiles, stream);
                });
			}).catch(e => {
				console.error("Radar API error:", e);
				const connection = document.getElementById("systemConnection");
				if (connection) connection.innerHTML = '<span style="color:var(--accent-red)">●</span> dashboard connection interrupted';
			}).finally(() => { statusRequestInFlight = false; });
		}
		pollStatus();
		setInterval(pollStatus, 2000);
    </script>
</body>
</html>
`
