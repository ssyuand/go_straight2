#!/bin/zsh
set -euo pipefail

label="com.syuan.gostraight"
script_dir="${0:A:h}"
project_dir="${script_dir:h}"
plist_dir="${HOME}/Library/LaunchAgents"
plist_path="${plist_dir}/${label}.plist"
service_target="gui/$(id -u)/${label}"
binary_path="${project_dir}/livetool"

write_plist() {
  mkdir -p "${plist_dir}"
  temp_plist="$(mktemp "${TMPDIR:-/tmp}/${label}.XXXXXX")"
  trap 'rm -f "${temp_plist}"' EXIT
  cat > "${temp_plist}" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>${label}</string>
  <key>ProgramArguments</key>
  <array>
    <string>${binary_path}</string>
  </array>
  <key>WorkingDirectory</key>
  <string>${project_dir}</string>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <dict>
    <key>SuccessfulExit</key>
    <false/>
  </dict>
  <key>ThrottleInterval</key>
  <integer>10</integer>
  <key>EnvironmentVariables</key>
  <dict>
    <key>LIVETOOL_LAUNCHD</key>
    <string>1</string>
    <key>PATH</key>
    <string>${project_dir}/venv/bin:/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
  </dict>
  <key>StandardOutPath</key>
  <string>${project_dir}/launchd.log</string>
  <key>StandardErrorPath</key>
  <string>${project_dir}/launchd.log</string>
</dict>
</plist>
PLIST
  plutil -lint "${temp_plist}"
  mv "${temp_plist}" "${plist_path}"
	chmod 644 "${plist_path}"
  trap - EXIT
}

wait_for_unload() {
	local attempt
	for attempt in {1..50}; do
		if ! launchctl print "${service_target}" >/dev/null 2>&1; then
			return 0
		fi
		sleep 0.1
	done
	echo "Timed out waiting for ${label} to unload" >&2
	return 1
}

bootstrap_service() {
	local attempt
	for attempt in {1..5}; do
		if launchctl bootstrap "gui/$(id -u)" "${plist_path}"; then
			return 0
		fi
		if launchctl print "${service_target}" >/dev/null 2>&1; then
			return 0
		fi
		echo "launchd bootstrap attempt ${attempt}/5 failed; retrying..." >&2
		sleep 0.5
	done
	return 1
}

case "${1:-status}" in
  install)
    cd "${project_dir}"
    go build -o "${binary_path}" .
    launchctl bootout "${service_target}" 2>/dev/null || true
	wait_for_unload
    write_plist
	bootstrap_service
    launchctl enable "${service_target}"
    launchctl kickstart -k "${service_target}"
    echo "Installed and started ${label}"
    ;;
  uninstall)
    launchctl bootout "${service_target}" 2>/dev/null || true
    if [[ -f "${plist_path}" ]]; then
      mv "${plist_path}" "${HOME}/.Trash/${label}.$(date +%Y%m%d-%H%M%S).plist"
    fi
    echo "Stopped and moved the LaunchAgent plist to Trash"
    ;;
  start)
    if ! launchctl print "${service_target}" >/dev/null 2>&1; then
		bootstrap_service
    fi
    launchctl kickstart -k "${service_target}"
    ;;
  stop)
    launchctl bootout "${service_target}" 2>/dev/null || true
    ;;
  restart)
    launchctl kickstart -k "${service_target}"
    ;;
  status)
    launchctl print "${service_target}"
    ;;
  *)
    echo "Usage: $0 {install|uninstall|start|stop|restart|status}" >&2
    exit 2
    ;;
esac
