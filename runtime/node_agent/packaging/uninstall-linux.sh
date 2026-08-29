#!/usr/bin/env bash
# 默认保留 config/data/log，避免误删节点身份和诊断证据；--purge 才会清除。
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
  rm -f /etc/systemd/system/induforge-node-agent.service
  systemctl daemon-reload
fi
rm -f "$PREFIX/bin/node-agent"
if [ "$PURGE" = true ]; then
  safe_purge_path "$PREFIX" && safe_purge_path "$CONFIG_DIR" || { echo "refusing unsafe purge target" >&2; exit 1; }
  rm -rf "$PREFIX" "$CONFIG_DIR"
fi
echo "NodeAgent binary removed. Use --purge to remove retained config and state."
