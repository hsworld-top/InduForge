#!/usr/bin/env bash
# K3s、容器、网络和本地 PVC 始终清理；默认只保留 NodeAgent 身份与日志，
# --purge 再清除这些诊断和重新接入所需的数据。
set -euo pipefail
PREFIX="${PREFIX:-/opt/induforge/node-agent}"
CONFIG_DIR="${CONFIG_DIR:-/etc/induforge/node-agent}"
PURGE=false
while [ "$#" -gt 0 ]; do case "$1" in --prefix) PREFIX="$2"; shift 2;; --config-dir) CONFIG_DIR="$2"; shift 2;; --purge) PURGE=true; shift;; *) exit 2;; esac; done
safe_purge_path() {
  case "$1" in ""|/|/opt|/etc|/var|/usr|/home) return 1;; esac
  case "$1" in *[Ii]ndu[Ff]orge*|*[Nn]ode[Aa]gent*) return 0;; *) return 1;; esac
}
if command -v systemctl >/dev/null 2>&1 && [ "$(id -u)" -eq 0 ]; then
  systemctl disable --now induforge-node-agent.service >/dev/null 2>&1 || true
	if [ -x "$PREFIX/bin/node-hostctl" ] && systemctl is-active --quiet induforge-node-hostd.service; then
		if [ "$PURGE" = true ]; then "$PREFIX/bin/node-hostctl" uninstall-current --purge-data; else "$PREFIX/bin/node-hostctl" uninstall-current; fi
	fi
	systemctl disable --now induforge-node-hostd.service >/dev/null 2>&1 || true
  rm -f /etc/systemd/system/induforge-node-agent.service /etc/systemd/system/induforge-node-hostd.service
  systemctl daemon-reload
fi
rm -f "$PREFIX/bin/node-agent" "$PREFIX/bin/node-hostd" "$PREFIX/bin/node-hostctl"
if [ "$PURGE" = true ]; then
  safe_purge_path "$PREFIX" && safe_purge_path "$CONFIG_DIR" || { echo "refusing unsafe purge target" >&2; exit 1; }
  rm -rf "$PREFIX" "$CONFIG_DIR"
fi
echo "NodeAgent and managed K3s runtime removed. Use --purge to also remove retained node identity and logs."
