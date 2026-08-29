#!/usr/bin/env bash
# 支持 PREFIX/CONFIG_DIR，方便容器和无 root 验收；默认路径适用于正式 Linux 安装。
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PREFIX="${PREFIX:-/opt/induforge/node-agent}"
CONFIG_DIR="${CONFIG_DIR:-/etc/induforge/node-agent}"
NO_SERVICE="${NO_SERVICE:-false}"
SERVER_URL=""
ENROLLMENT_CODE=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --prefix) PREFIX="$2"; shift 2;;
    --config-dir) CONFIG_DIR="$2"; shift 2;;
    --no-service) NO_SERVICE=true; shift;;
    --server-url) SERVER_URL="$2"; shift 2;;
    --enrollment-code) ENROLLMENT_CODE="$2"; shift 2;;
    *) echo "unknown argument: $1" >&2; exit 2;;
  esac
done
case "$(uname -m)" in x86_64|amd64) ARCH=amd64;; aarch64|arm64) ARCH=arm64;; *) echo "unsupported Linux architecture: $(uname -m)" >&2; exit 1;; esac
install -d -m 0755 "$PREFIX/bin" "$PREFIX/data" "$PREFIX/runtime" "$PREFIX/logs" "$CONFIG_DIR"
install -m 0755 "$SCRIPT_DIR/bin/node-agent-linux-$ARCH" "$PREFIX/bin/node-agent"
if [ ! -f "$CONFIG_DIR/config.yaml" ]; then install -m 0600 "$SCRIPT_DIR/config.yaml" "$CONFIG_DIR/config.yaml"; fi
if [[ "$SERVER_URL$ENROLLMENT_CODE" == *$'\n'* || "$SERVER_URL$ENROLLMENT_CODE" == *$'\r'* ]]; then echo "invalid enrollment value" >&2; exit 2; fi
escape_sed_replacement() { printf '%s' "$1" | sed 's/[\\&|]/\\&/g'; }
if [ -n "$SERVER_URL" ]; then SAFE_SERVER_URL="$(escape_sed_replacement "$SERVER_URL")"; sed -i.bak "s|serverUrl:.*|serverUrl: \"$SAFE_SERVER_URL\"|" "$CONFIG_DIR/config.yaml" && rm -f "$CONFIG_DIR/config.yaml.bak"; fi
if [ -n "$ENROLLMENT_CODE" ]; then SAFE_ENROLLMENT_CODE="$(escape_sed_replacement "$ENROLLMENT_CODE")"; sed -i.bak "s|enrollmentCode:.*|enrollmentCode: \"$SAFE_ENROLLMENT_CODE\"|" "$CONFIG_DIR/config.yaml" && rm -f "$CONFIG_DIR/config.yaml.bak"; fi
START_SERVICE=false
if [ -n "$SERVER_URL" ] && [ -n "$ENROLLMENT_CODE" ]; then START_SERVICE=true; fi
if [ "$NO_SERVICE" != true ] && command -v systemctl >/dev/null 2>&1; then
  UNIT_NAME="induforge-node-agent"
  UNIT_PATH="/etc/systemd/system/$UNIT_NAME.service"
  if [ "$(id -u)" -ne 0 ]; then echo "systemd installation requires root; retry with --no-service for container verification" >&2; exit 1; fi
  cat > "$UNIT_PATH" <<EOF
[Unit]
Description=InduForge NodeAgent
After=network-online.target
[Service]
Type=simple
Environment=NODE_AGENT_WORKDIR=$PREFIX
Environment=NODE_AGENT_CONFIG=$CONFIG_DIR/config.yaml
Environment=NODE_AGENT_DATA_DIR=$PREFIX/data
ExecStart=$PREFIX/bin/node-agent --daemon
Restart=always
RestartSec=3
[Install]
WantedBy=multi-user.target
EOF
  systemctl daemon-reload
  if [ "$START_SERVICE" = true ]; then systemctl enable --now "$UNIT_NAME.service"; else echo "Service installed but not started: pass --server-url and --enrollment-code after approval."; fi
fi
echo "Installed NodeAgent under $PREFIX. Configure $CONFIG_DIR/config.yaml or pass --server-url/--enrollment-code before starting."
