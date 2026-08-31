#!/usr/bin/env bash
# 支持 PREFIX/CONFIG_DIR，方便容器和无 root 验收；默认路径适用于正式 Linux 安装。
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PREFIX="${PREFIX:-/opt/induforge/node-agent}"
CONFIG_DIR="${CONFIG_DIR:-/etc/induforge/node-agent}"
NO_SERVICE="${NO_SERVICE:-false}"
ENABLE_COLLECTOR=false
SERVER_URL=""
ENROLLMENT_CODE=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --prefix) PREFIX="$2"; shift 2;;
    --config-dir) CONFIG_DIR="$2"; shift 2;;
    --no-service) NO_SERVICE=true; shift;;
    --enable-collector) ENABLE_COLLECTOR=true; shift;;
    --server-url) SERVER_URL="$2"; shift 2;;
    --enrollment-code) ENROLLMENT_CODE="$2"; shift 2;;
    *) echo "unknown argument: $1" >&2; exit 2;;
  esac
done
case "$(uname -m)" in x86_64|amd64) ARCH=amd64;; aarch64|arm64) ARCH=arm64;; *) echo "unsupported Linux architecture: $(uname -m)" >&2; exit 1;; esac
if [ ! -f "$SCRIPT_DIR/BUILD_VERSION" ]; then echo "package BUILD_VERSION is missing" >&2; exit 1; fi
BUILD_VERSION="$(tr -d '\r\n' < "$SCRIPT_DIR/BUILD_VERSION")"
if [ -z "$BUILD_VERSION" ]; then echo "package BUILD_VERSION is empty" >&2; exit 1; fi
if [ ! -d "$SCRIPT_DIR/capabilities/$ARCH" ]; then echo "capabilities for $ARCH are missing from this package" >&2; exit 1; fi
RUN_USER="${NODE_AGENT_USER:-induforge}"
RUN_GROUP="$RUN_USER"
if [ "$(id -u)" -eq 0 ]; then
  if ! id -u "$RUN_USER" >/dev/null 2>&1; then
    if command -v useradd >/dev/null 2>&1; then useradd --system --home "$PREFIX" --shell /usr/sbin/nologin "$RUN_USER"; else adduser --system --home "$PREFIX" --disabled-login "$RUN_USER"; fi
  fi
  if ! getent group "$RUN_GROUP" >/dev/null 2>&1; then RUN_GROUP="$(id -gn "$RUN_USER")"; fi
fi
install -d -m 0755 "$PREFIX/bin" "$PREFIX/capabilities/$ARCH" "$PREFIX/data" "$PREFIX/runtime" "$PREFIX/releases" "$PREFIX/logs" "$CONFIG_DIR"
install -m 0755 "$SCRIPT_DIR/bin/node-agent-linux-$ARCH" "$PREFIX/bin/node-agent"
cp -R "$SCRIPT_DIR/capabilities/$ARCH/." "$PREFIX/capabilities/$ARCH/"
find "$PREFIX/capabilities/$ARCH" -type f -exec chmod 0755 {} +
if [ ! -f "$CONFIG_DIR/config.yaml" ]; then
  install -m 0600 "$SCRIPT_DIR/config.yaml" "$CONFIG_DIR/config.yaml"
  SAFE_BUILD_VERSION="$(printf '%s' "$BUILD_VERSION" | sed 's/[\\&|]/\\&/g')"
  sed -i.bak "s|__BUILD_VERSION__|$SAFE_BUILD_VERSION|" "$CONFIG_DIR/config.yaml" && rm -f "$CONFIG_DIR/config.yaml.bak"
fi
if [[ "$SERVER_URL$ENROLLMENT_CODE" == *$'\n'* || "$SERVER_URL$ENROLLMENT_CODE" == *$'\r'* ]]; then echo "invalid enrollment value" >&2; exit 2; fi
escape_sed_replacement() { printf '%s' "$1" | sed 's/[\\&|]/\\&/g'; }
if [ -n "$SERVER_URL" ]; then SAFE_SERVER_URL="$(escape_sed_replacement "$SERVER_URL")"; sed -i.bak "s|serverUrl:.*|serverUrl: \"$SAFE_SERVER_URL\"|" "$CONFIG_DIR/config.yaml" && rm -f "$CONFIG_DIR/config.yaml.bak"; fi
if [ -n "$ENROLLMENT_CODE" ]; then SAFE_ENROLLMENT_CODE="$(escape_sed_replacement "$ENROLLMENT_CODE")"; sed -i.bak "s|enrollmentCode:.*|enrollmentCode: \"$SAFE_ENROLLMENT_CODE\"|" "$CONFIG_DIR/config.yaml" && rm -f "$CONFIG_DIR/config.yaml.bak"; fi
if [ "$ENABLE_COLLECTOR" = true ]; then
  sed -i.bak '/group: collector/,/enabled: false/ s/installed: false/installed: true/' "$CONFIG_DIR/config.yaml" && rm -f "$CONFIG_DIR/config.yaml.bak"
fi
if [ "$(id -u)" -eq 0 ]; then
  chown -R "$RUN_USER:$RUN_GROUP" "$PREFIX/data" "$PREFIX/runtime" "$PREFIX/releases" "$PREFIX/logs"
  chmod 0750 "$PREFIX/data" "$PREFIX/runtime" "$PREFIX/releases" "$PREFIX/logs"
fi
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
User=$RUN_USER
Group=$RUN_GROUP
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
echo "Installed NodeAgent $BUILD_VERSION under $PREFIX. Capability templates can claim a node but stay disabled until local release configuration is complete."
