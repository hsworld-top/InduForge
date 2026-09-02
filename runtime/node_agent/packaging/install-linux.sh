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
HOST_DATA_DIR=""
NODE_IP=""
RUNTIME_DATA_DIR="/var/lib/induforge/node-agent"
RELEASE_SIGNING_KEY_ID=""
RELEASE_SIGNING_PUBLIC_KEY=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --prefix) PREFIX="$2"; shift 2;;
    --config-dir) CONFIG_DIR="$2"; shift 2;;
    --no-service) NO_SERVICE=true; shift;;
    --enable-collector) ENABLE_COLLECTOR=true; shift;;
    --server-url) SERVER_URL="$2"; shift 2;;
    --enrollment-code) ENROLLMENT_CODE="$2"; shift 2;;
    --node-data-dir) HOST_DATA_DIR="$2"; shift 2;;
    --node-ip) NODE_IP="$2"; shift 2;;
    --release-signing-key-id) RELEASE_SIGNING_KEY_ID="$2"; shift 2;;
    --release-signing-public-key) RELEASE_SIGNING_PUBLIC_KEY="$2"; shift 2;;
    *) echo "unknown argument: $1" >&2; exit 2;;
  esac
done
validate_install_root() {
  case "$1" in /*) ;; *) echo "$2 must be an absolute path" >&2; exit 2;; esac
  case "$1" in /|/bin|/boot|/dev|/etc|/home|/opt|/proc|/root|/run|/sys|/tmp|/usr|/var) echo "$2 cannot be a system directory" >&2; exit 2;; esac
  if ! printf '%s' "$1" | grep -Eq '^/[A-Za-z0-9._/-]+$'; then echo "$2 contains unsupported characters" >&2; exit 2; fi
}
validate_install_root "$PREFIX" "--prefix"
validate_install_root "$CONFIG_DIR" "--config-dir"
case "$(uname -m)" in x86_64|amd64) ARCH=amd64;; aarch64|arm64) ARCH=arm64;; *) echo "unsupported Linux architecture: $(uname -m)" >&2; exit 1;; esac
if [ ! -f "$SCRIPT_DIR/BUILD_VERSION" ]; then echo "package BUILD_VERSION is missing" >&2; exit 1; fi
BUILD_VERSION="$(tr -d '\r\n' < "$SCRIPT_DIR/BUILD_VERSION")"
if [ -z "$BUILD_VERSION" ]; then echo "package BUILD_VERSION is empty" >&2; exit 1; fi
if [ ! -d "$SCRIPT_DIR/capabilities/$ARCH" ]; then echo "capabilities for $ARCH are missing from this package" >&2; exit 1; fi
if [ ! -x "$SCRIPT_DIR/bin/node-hostd-linux-$ARCH" ] || [ ! -x "$SCRIPT_DIR/bin/node-hostctl-linux-$ARCH" ]; then echo "hostd binaries for $ARCH are missing from this package" >&2; exit 1; fi
if [ ! -f "$SCRIPT_DIR/k3s/$ARCH/k3s" ] || [ ! -f "$SCRIPT_DIR/k3s/$ARCH/k3s-airgap-images-$ARCH.tar.zst" ] || [ ! -f "$SCRIPT_DIR/k3s/$ARCH/induforge-foundation-images-$ARCH.tar.gz" ] || [ ! -f "$SCRIPT_DIR/k3s/$ARCH/SHA256SUMS" ]; then echo "verified K3s and foundation assets for $ARCH are missing from this package" >&2; exit 1; fi
if ! compgen -G "$SCRIPT_DIR/time-sync/ubuntu-24.04/$ARCH/chrony_*.deb" >/dev/null || [ ! -f "$SCRIPT_DIR/time-sync/ubuntu-24.04/$ARCH/SHA256SUMS" ]; then echo "verified Chrony assets for Ubuntu 24.04 $ARCH are missing from this package" >&2; exit 1; fi
if [ -z "$HOST_DATA_DIR" ]; then HOST_DATA_DIR="$PREFIX/data/k3s"; fi
case "$HOST_DATA_DIR" in /*) ;; *) echo "--node-data-dir must be an absolute path" >&2; exit 2;; esac
case "$HOST_DATA_DIR" in /|/bin|/boot|/dev|/etc|/home|/opt|/proc|/root|/run|/sys|/tmp|/usr|/var) echo "--node-data-dir cannot be a system directory" >&2; exit 2;; esac
if ! printf '%s' "$HOST_DATA_DIR" | grep -Eq '^/[A-Za-z0-9._/-]+$'; then echo "--node-data-dir contains unsupported characters" >&2; exit 2; fi
if [ -n "$NODE_IP" ] && ! printf '%s' "$NODE_IP" | grep -Eq '^([0-9]{1,3}\.){3}[0-9]{1,3}$|^[0-9A-Fa-f:]+$'; then echo "--node-ip must be an IP address" >&2; exit 2; fi
if [ -z "$RELEASE_SIGNING_KEY_ID" ] || ! printf '%s' "$RELEASE_SIGNING_KEY_ID" | grep -Eq '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'; then echo "--release-signing-key-id is required and invalid" >&2; exit 2; fi
if [ -z "$RELEASE_SIGNING_PUBLIC_KEY" ] || [ "$(printf '%s' "$RELEASE_SIGNING_PUBLIC_KEY" | base64 -d 2>/dev/null | wc -c | tr -d ' ')" != 32 ]; then echo "--release-signing-public-key must be a base64 Ed25519 public key" >&2; exit 2; fi
RUN_USER="${NODE_AGENT_USER:-induforge}"
RUN_GROUP="$RUN_USER"
if [ "$(id -u)" -eq 0 ]; then
  if ! id -u "$RUN_USER" >/dev/null 2>&1; then
    if command -v useradd >/dev/null 2>&1; then useradd --system --home "$PREFIX" --shell /usr/sbin/nologin "$RUN_USER"; else adduser --system --home "$PREFIX" --disabled-login "$RUN_USER"; fi
  fi
  if ! getent group "$RUN_GROUP" >/dev/null 2>&1; then RUN_GROUP="$(id -gn "$RUN_USER")"; fi
	install -d -m 0700 -o "$RUN_USER" -g "$RUN_GROUP" "$RUNTIME_DATA_DIR"
	# 旧包把 Agent 身份与 Release 放在 PREFIX/data；首次升级时只复制到标准
	# hostPath 根，保留旧目录作为可恢复备份，不在安装阶段删除用户数据。
	if [ -f "$PREFIX/data/ops-agent-identity.json" ] && [ ! -f "$RUNTIME_DATA_DIR/ops-agent-identity.json" ]; then
		cp -a "$PREFIX/data/." "$RUNTIME_DATA_DIR/"
		chown -R "$RUN_USER:$RUN_GROUP" "$RUNTIME_DATA_DIR"
	fi
fi
# 能力和离线资产是版本化制品，不保留上一包的未知文件；身份、运行数据、
# Release 与日志目录单独保留，升级不会误删用户数据。
rm -rf -- "$PREFIX/capabilities/$ARCH" "$PREFIX/k3s/$ARCH"
install -d -m 0755 "$PREFIX/bin" "$PREFIX/capabilities/$ARCH" "$PREFIX/k3s/$ARCH" "$PREFIX/data" "$PREFIX/runtime" "$PREFIX/releases" "$PREFIX/logs" "$CONFIG_DIR"
install -m 0755 "$SCRIPT_DIR/bin/node-agent-linux-$ARCH" "$PREFIX/bin/node-agent"
install -m 0755 "$SCRIPT_DIR/bin/node-hostd-linux-$ARCH" "$PREFIX/bin/node-hostd"
install -m 0755 "$SCRIPT_DIR/bin/node-hostctl-linux-$ARCH" "$PREFIX/bin/node-hostctl"
cp -R "$SCRIPT_DIR/capabilities/$ARCH/." "$PREFIX/capabilities/$ARCH/"
cp "$SCRIPT_DIR/k3s/$ARCH/k3s" "$SCRIPT_DIR/k3s/$ARCH/k3s-airgap-images-$ARCH.tar.zst" "$SCRIPT_DIR/k3s/$ARCH/induforge-foundation-images-$ARCH.tar.gz" "$SCRIPT_DIR/k3s/$ARCH/SHA256SUMS" "$PREFIX/k3s/$ARCH/"
find "$PREFIX/capabilities/$ARCH" -type f -exec chmod 0755 {} +
chmod 0755 "$PREFIX/k3s/$ARCH/k3s"
chmod 0600 "$PREFIX/k3s/$ARCH/k3s-airgap-images-$ARCH.tar.zst" "$PREFIX/k3s/$ARCH/induforge-foundation-images-$ARCH.tar.gz" "$PREFIX/k3s/$ARCH/SHA256SUMS"
if [ ! -f "$CONFIG_DIR/config.yaml" ]; then
  install -m 0600 "$SCRIPT_DIR/config.yaml" "$CONFIG_DIR/config.yaml"
  SAFE_BUILD_VERSION="$(printf '%s' "$BUILD_VERSION" | sed 's/[\\&|]/\\&/g')"
  sed -i.bak "s|__BUILD_VERSION__|$SAFE_BUILD_VERSION|" "$CONFIG_DIR/config.yaml" && rm -f "$CONFIG_DIR/config.yaml.bak"
  SAFE_HOST_DATA_DIR="$(printf '%s' "$HOST_DATA_DIR" | sed 's/[\\&|]/\\&/g')"
  sed -i.bak "s|__HOST_DATA_DIR__|$SAFE_HOST_DATA_DIR|" "$CONFIG_DIR/config.yaml" && rm -f "$CONFIG_DIR/config.yaml.bak"
	SAFE_NODE_IP="$(printf '%s' "$NODE_IP" | sed 's/[\\&|]/\\&/g')"
	sed -i.bak "s|__NODE_IP__|$SAFE_NODE_IP|" "$CONFIG_DIR/config.yaml" && rm -f "$CONFIG_DIR/config.yaml.bak"
fi
SAFE_BUILD_VERSION="$(printf '%s' "$BUILD_VERSION" | sed 's/[\\&|]/\\&/g')"
SAFE_HOST_DATA_DIR="$(printf '%s' "$HOST_DATA_DIR" | sed 's/[\\&|]/\\&/g')"
sed -i.bak "s|agentVersion:.*|agentVersion: '$SAFE_BUILD_VERSION'|" "$CONFIG_DIR/config.yaml" && rm -f "$CONFIG_DIR/config.yaml.bak"
sed -i.bak "s|runtimeVersion:.*|runtimeVersion: '$SAFE_BUILD_VERSION'|" "$CONFIG_DIR/config.yaml" && rm -f "$CONFIG_DIR/config.yaml.bak"
# 老版本配置由 Go YAML 序列化器生成时可能使用 8 空格缩进，发布模板使用 4
# 空格。插入字段必须沿用 dataDir 所在层级，不能写死缩进。
insert_after_yaml_key() {
  local source_key="$1" first_line="$2" second_line="${3:-}" third_line="${4:-}" temporary
  temporary="$(mktemp "$CONFIG_DIR/config.yaml.XXXXXX")"
  if ! awk -v source_key="$source_key" -v first_line="$first_line" -v second_line="$second_line" -v third_line="$third_line" '
    $0 ~ "^[[:space:]]*" source_key ":[[:space:]]*" && !inserted {
      match($0, /^[[:space:]]*/); indentation=substr($0, RSTART, RLENGTH)
      print; print indentation first_line
      if (second_line != "") print indentation second_line
      if (third_line != "") print indentation third_line
      inserted=1; next
    }
    { print }
    END { if (!inserted) exit 42 }
  ' "$CONFIG_DIR/config.yaml" > "$temporary"; then
    rm -f "$temporary"
    echo "cannot locate $source_key in existing config" >&2
    exit 1
  fi
  mv "$temporary" "$CONFIG_DIR/config.yaml"
}
# 老版本配置升级时原地补齐 Hostd 边界，保留已领取的节点身份与用户数据。
if ! grep -q '^[[:space:]]*hostdSocket:' "$CONFIG_DIR/config.yaml"; then
  insert_after_yaml_key dataDir "hostdSocket: /run/induforge/hostd.sock" "hostDataDir: '$SAFE_HOST_DATA_DIR'"
else
  sed -E -i.bak "s|^([[:space:]]*)hostDataDir:.*|\\1hostDataDir: '$SAFE_HOST_DATA_DIR'|" "$CONFIG_DIR/config.yaml" && rm -f "$CONFIG_DIR/config.yaml.bak"
fi
if [ -n "$NODE_IP" ]; then
  SAFE_NODE_IP="$(printf '%s' "$NODE_IP" | sed 's/[\\&|]/\\&/g')"
  if grep -q '^[[:space:]]*nodeIp:' "$CONFIG_DIR/config.yaml"; then sed -E -i.bak "s|^([[:space:]]*)nodeIp:.*|\\1nodeIp: '$SAFE_NODE_IP'|" "$CONFIG_DIR/config.yaml"; else insert_after_yaml_key hostDataDir "nodeIp: '$SAFE_NODE_IP'"; fi
  rm -f "$CONFIG_DIR/config.yaml.bak"
fi
if [[ "$SERVER_URL$ENROLLMENT_CODE" == *$'\n'* || "$SERVER_URL$ENROLLMENT_CODE" == *$'\r'* ]]; then echo "invalid enrollment value" >&2; exit 2; fi
escape_sed_replacement() { printf '%s' "$1" | sed 's/[\\&|]/\\&/g'; }
if [ -n "$SERVER_URL" ]; then SAFE_SERVER_URL="$(escape_sed_replacement "$SERVER_URL")"; sed -i.bak "s|serverUrl:.*|serverUrl: \"$SAFE_SERVER_URL\"|" "$CONFIG_DIR/config.yaml" && rm -f "$CONFIG_DIR/config.yaml.bak"; fi
if [ -n "$ENROLLMENT_CODE" ]; then SAFE_ENROLLMENT_CODE="$(escape_sed_replacement "$ENROLLMENT_CODE")"; sed -i.bak "s|enrollmentCode:.*|enrollmentCode: \"$SAFE_ENROLLMENT_CODE\"|" "$CONFIG_DIR/config.yaml" && rm -f "$CONFIG_DIR/config.yaml.bak"; fi
SAFE_RELEASE_SIGNING_KEY_ID="$(escape_sed_replacement "$RELEASE_SIGNING_KEY_ID")"
SAFE_RELEASE_SIGNING_PUBLIC_KEY="$(escape_sed_replacement "$RELEASE_SIGNING_PUBLIC_KEY")"
if grep -q '^[[:space:]]*trustKeys:' "$CONFIG_DIR/config.yaml"; then
  sed -E -i.bak "s|^([[:space:]]*)trustKeys:.*|\1trustKeys:\n\1  - keyId: '$SAFE_RELEASE_SIGNING_KEY_ID'\n\1    publicKey: '$SAFE_RELEASE_SIGNING_PUBLIC_KEY'|" "$CONFIG_DIR/config.yaml" && rm -f "$CONFIG_DIR/config.yaml.bak"
else
  insert_after_yaml_key dataDir "trustKeys:" "  - keyId: '$SAFE_RELEASE_SIGNING_KEY_ID'" "    publicKey: '$SAFE_RELEASE_SIGNING_PUBLIC_KEY'"
fi
if [ "$ENABLE_COLLECTOR" = true ]; then
  sed -i.bak '/group: collector/,/enabled: false/ s/installed: false/installed: true/' "$CONFIG_DIR/config.yaml" && rm -f "$CONFIG_DIR/config.yaml.bak"
fi
if [ "$(id -u)" -eq 0 ]; then
	if [ ! -f /etc/os-release ]; then echo "cannot identify Linux distribution for offline Chrony installation" >&2; exit 1; fi
	# shellcheck disable=SC1091
	. /etc/os-release
	if [ "${ID:-}" != "ubuntu" ] || [ "${VERSION_ID:-}" != "24.04" ]; then echo "this NodeAgent package currently supports offline Chrony installation on Ubuntu 24.04 only" >&2; exit 1; fi
	(cd "$SCRIPT_DIR/time-sync/ubuntu-24.04/$ARCH" && shasum -a 256 -c SHA256SUMS)
	if ! command -v chronyc >/dev/null 2>&1; then
		# 安装期间保持 Chrony 被屏蔽，避免软件包携带的 Ubuntu 公网时间源在
		# 平台写入固定拓扑前短暂启动并修改中心服务器时间。
		systemctl mask chrony.service
		if dpkg-query -W -f='${db:Status-Abbrev}' systemd-timesyncd 2>/dev/null | grep -q '^ii'; then
			dpkg --remove systemd-timesyncd
		fi
		dpkg --unpack "$SCRIPT_DIR/time-sync/ubuntu-24.04/$ARCH"/*.deb
		DEBIAN_FRONTEND=noninteractive dpkg --configure -a
		systemctl unmask chrony.service
	fi
  # Agent 领取身份后要以“临时文件 + rename”原子清除一次性接入码，因此配置目录
  # 也必须由运行账户独占；仅修改文件所有权会导致 rename 被父目录权限拒绝。
  chown "$RUN_USER:$RUN_GROUP" "$CONFIG_DIR"
  chmod 0700 "$CONFIG_DIR"
  chown "$RUN_USER:$RUN_GROUP" "$CONFIG_DIR/config.yaml"
  chmod 0600 "$CONFIG_DIR/config.yaml"
  chown -R "$RUN_USER:$RUN_GROUP" "$PREFIX/data" "$PREFIX/runtime" "$PREFIX/releases" "$PREFIX/logs"
  chmod 0750 "$PREFIX/data" "$PREFIX/runtime" "$PREFIX/releases" "$PREFIX/logs"
  chown -R root:root "$PREFIX/bin/node-hostd" "$PREFIX/bin/node-hostctl" "$PREFIX/k3s"
fi
START_SERVICE=false
if [ -n "$SERVER_URL" ] && { [ -n "$ENROLLMENT_CODE" ] || [ -f "$PREFIX/data/ops-agent-identity.json" ]; }; then START_SERVICE=true; fi
if [ "$NO_SERVICE" != true ] && command -v systemctl >/dev/null 2>&1; then
  UNIT_NAME="induforge-node-agent"
  UNIT_PATH="/etc/systemd/system/$UNIT_NAME.service"
  if [ "$(id -u)" -ne 0 ]; then echo "systemd installation requires root; retry with --no-service for container verification" >&2; exit 1; fi
  install -d -m 0700 "$PREFIX/data/hostd" "$HOST_DATA_DIR" /etc/rancher/induforge-k3s
  install -d -m 0755 /etc/chrony
  install -d -o root -g "$RUN_GROUP" -m 0750 /run/induforge
  HOSTD_UNIT_NAME="induforge-node-hostd"
  HOSTD_UNIT_PATH="/etc/systemd/system/$HOSTD_UNIT_NAME.service"
  cat > "$HOSTD_UNIT_PATH" <<EOF
[Unit]
Description=InduForge privileged node host service
After=network-online.target
[Service]
Type=simple
User=root
Group=root
Environment=INDUFORGE_HOSTD_ASSETS_DIR=$PREFIX/k3s
Environment=INDUFORGE_HOSTD_STATE_DIR=$PREFIX/data/hostd
ExecStart=$PREFIX/bin/node-hostd
Restart=always
RestartSec=3
NoNewPrivileges=true
ProtectSystem=strict
ReadWritePaths=$PREFIX/data/hostd $RUNTIME_DATA_DIR $HOST_DATA_DIR /etc/rancher/induforge-k3s /etc/chrony /etc/systemd/system /usr/local/bin /run/induforge
[Install]
WantedBy=multi-user.target
EOF
  cat > "$UNIT_PATH" <<EOF
[Unit]
Description=InduForge NodeAgent
After=network-online.target induforge-node-hostd.service
Requires=induforge-node-hostd.service
[Service]
Type=simple
User=$RUN_USER
Group=$RUN_GROUP
Environment=NODE_AGENT_WORKDIR=$PREFIX
Environment=NODE_AGENT_CONFIG=$CONFIG_DIR/config.yaml
Environment=NODE_AGENT_DATA_DIR=$RUNTIME_DATA_DIR
ExecStart=$PREFIX/bin/node-agent --daemon
Restart=always
RestartSec=3
[Install]
WantedBy=multi-user.target
EOF
  systemctl daemon-reload
  systemctl enable --now "$HOSTD_UNIT_NAME.service"
  if systemctl is-active --quiet "$HOSTD_UNIT_NAME.service"; then systemctl restart "$HOSTD_UNIT_NAME.service"; fi
  if [ "$START_SERVICE" = true ]; then
    systemctl enable "$UNIT_NAME.service"
    if systemctl is-active --quiet "$UNIT_NAME.service"; then systemctl restart "$UNIT_NAME.service"; else systemctl start "$UNIT_NAME.service"; fi
  else echo "Service installed but not started: pass --server-url and --enrollment-code after approval."; fi
fi
echo "Installed NodeAgent $BUILD_VERSION under $PREFIX. Capability templates can claim a node but stay disabled until local release configuration is complete."
