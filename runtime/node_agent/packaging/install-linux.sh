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
# 动画只负责显示，实际命令仍在前台执行，保留退出码和中断行为。
PROGRESS_PID=""
PROGRESS_LABEL=""
progress_stop() {
  if [ -n "$PROGRESS_PID" ]; then
    kill "$PROGRESS_PID" 2>/dev/null || true
    wait "$PROGRESS_PID" 2>/dev/null || true
    PROGRESS_PID=""
    printf '\r\033[K' >&2
  fi
}
progress_start() {
  progress_stop
  PROGRESS_LABEL="$1"
  printf '%s…\n' "$PROGRESS_LABEL" >&2
  if [ -t 2 ]; then
    (
      started=$SECONDS
      frames='|/-\'
      while :; do
        for ((frame=0; frame<4; frame++)); do
          printf '\r\033[K%s %s · 已等待 %s 秒' "${frames:frame:1}" "$PROGRESS_LABEL" "$((SECONDS-started))" >&2
          sleep 0.2
        done
      done
    ) &
    PROGRESS_PID=$!
  fi
}
progress_done() {
  progress_stop
  printf '✓ %s\n' "$PROGRESS_LABEL" >&2
  PROGRESS_LABEL=""
}
progress_exit() {
  local result=$?
  progress_stop
  if [ "$result" -ne 0 ] && [ -n "$PROGRESS_LABEL" ]; then
    printf '未完成：%s\n' "$PROGRESS_LABEL" >&2
  fi
  return "$result"
}
trap progress_exit EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
# 只接受中心对本次启动的在线心跳确认，不以本地身份文件或进程存活代替。
wait_for_center() {
  progress_start "等待中心确认节点在线"
  for attempt in $(seq 1 45); do
    if python3 - "$RUNTIME_DATA_DIR/ops-center-connection.json" "$PREFIX/data/ops-center-connection.json" "$CONNECTION_STARTED_AT" <<'ACK'
import json,sys,pathlib
for name in sys.argv[1:3]:
 try:
  p=pathlib.Path(name); data=json.loads(p.read_text()); identity=json.loads((p.parent/'ops-agent-identity.json').read_text())
  if data.get('nodeId') and data['nodeId']==identity.get('nodeId') and data.get('confirmedAt',0)>=int(sys.argv[3]): sys.exit(0)
 except (OSError,ValueError,TypeError): pass
sys.exit(1)
ACK
    then progress_done; echo "节点已接入中心，首次心跳已确认。请在 Studio 节点管理中查看并关联运行环境。"; return 0; fi
    sleep 1
  done
  progress_stop
  PROGRESS_LABEL=""
  echo "接入未完成。再次运行 ./install.sh 可重试连接，不会重新安装。" >&2
  python3 - "$RUNTIME_DATA_DIR/ops-center-connection.json" <<'ERROR'
import json,sys
try:
 message=json.load(open(sys.argv[1])).get('error')
 if message: print('原因：'+str(message))
except (OSError,ValueError): pass
ERROR
  return 1
}
# 校验用户选择或明确确认创建的空目录，不沿符号链接选择数据位置。
validate_existing_data_dir() {
  python3 - "$1" <<'DIRECTORY'
import pathlib,sys
p=pathlib.Path(sys.argv[1])
if not p.is_absolute() or '..' in p.parts or any(x.is_symlink() for x in [p,*p.parents]):
 raise SystemExit('请选择无符号链接的绝对路径。')
if not p.is_dir(): raise SystemExit('目录不存在，请先创建空目录，再输入或按 Tab 补全。')
if any(p.iterdir()): raise SystemExit('目录不是空目录，请选择其他目录（包括隐藏文件也必须为空）。')
DIRECTORY
}
# 无参数启动进入向导；命令行参数保留给自动化发布与隔离测试。
INTERACTIVE=false
if [ "$#" -eq 0 ]; then
  if [ ! -t 0 ]; then echo "请在终端中直接运行 ./install.sh，按提示完成安装。" >&2; exit 2; fi
  INTERACTIVE=true
  if [ "$(id -u)" -ne 0 ]; then
    echo "正在获取管理员权限，系统可能要求输入当前 Linux 用户的密码。"
    exec sudo -- bash "$SCRIPT_DIR/$(basename "${BASH_SOURCE[0]}")"
  fi
fi
# 信任材料由中心管理员随安装包提供，只读取两行数据，绝不执行配置文件。
if [ -f "$SCRIPT_DIR/release-trust.txt" ]; then
  RELEASE_SIGNING_KEY_ID="$(sed -n '1p' "$SCRIPT_DIR/release-trust.txt")"
  RELEASE_SIGNING_PUBLIC_KEY="$(sed -n '2p' "$SCRIPT_DIR/release-trust.txt")"
fi
if [ "$INTERACTIVE" = true ]; then
  NODE_UNIT_STATE="$(systemctl show induforge-node-agent.service -p LoadState --value 2>/dev/null || true)"
  if [ "$NODE_UNIT_STATE" = "not-found" ] || [ -z "$NODE_UNIT_STATE" ]; then
    if [ -d "$CONFIG_DIR" ] || [ -d "$PREFIX" ] || [ -d "$RUNTIME_DATA_DIR" ]; then
      echo "节点服务已卸载，但仍有上次保留的文件。"
      echo "请先运行 ./uninstall.sh，按向导备份并清理残留，再重新安装。"
      exit 2
    fi
  elif [ ! -x "$PREFIX/bin/node-agent" ]; then
    echo "节点服务存在但程序缺失，请先运行卸载向导处理残留。" >&2
    exit 2
  fi
  if [ "$NODE_UNIT_STATE" != "not-found" ] && [ -n "$NODE_UNIT_STATE" ]; then
    echo "检测到已安装的节点，已停止重复安装，不会覆盖配置或数据目录。"
    echo "查看状态：sudo systemctl status induforge-node-agent"
    read -r -p "是否重试连接中心？[y/N]：" RETRY || exit 1
    case "$RETRY" in y|Y|yes|YES)
      CONNECTION_STARTED_AT=$(date +%s)
      systemctl restart induforge-node-agent.service
      wait_for_center
      exit $? ;;
      *) exit 2 ;;
    esac
  fi
  echo ""
  echo "InduFrame 节点安装向导"
  echo "请先在 Studio 的节点管理中创建接入码。按 Ctrl+C 可退出。"
  if [ -z "$RELEASE_SIGNING_KEY_ID" ] || [ -z "$RELEASE_SIGNING_PUBLIC_KEY" ]; then
    echo "安装包缺少中心信任信息 release-trust.txt，请向中心管理员获取配套文件。" >&2
    exit 2
  fi
  while :; do
    read -r -e -p "节点连接地址（请复制 Studio 显示的 HTTPS 地址）：" SERVER_URL || exit 1
    SERVER_URL="${SERVER_URL%/}"
    if printf '%s' "$SERVER_URL" | grep -Eq '^https://([A-Za-z0-9._-]+|\[[0-9A-Fa-f:]+\])(:[0-9]+)?$'; then
      if ! command -v curl >/dev/null 2>&1; then
        echo "缺少 curl，请先安装 curl 后重新运行安装向导。" >&2; exit 2
      fi
      # 从目标节点检查实际网络和证书，不以浏览器可达作为安装依据。
      ca_options=()
      if [ -f "$SCRIPT_DIR/center-ca.crt" ]; then ca_options=(--cacert "$SCRIPT_DIR/center-ca.crt"); fi
      progress_start "检查中心连接与证书"
      check_status=0
      http_status=$(curl "${ca_options[@]}" --silent --output /dev/null --write-out '%{http_code}' --connect-timeout 5 --max-time 10 "$SERVER_URL/health") || check_status=$?
      progress_stop
      PROGRESS_LABEL=""
      if [ "$check_status" -eq 0 ] && [ "$http_status" = 200 ]; then
        echo "中心连接正常。"; break
      fi
      case "$check_status" in
        60|51) echo "中心证书校验失败，请检查域名和系统信任证书。" ;;
        6) echo "无法解析中心域名，请检查地址和 DNS。" ;;
        7|28) echo "无法连接中心，请检查地址、端口及节点网络。" ;;
        *) echo "中心健康检查失败（HTTP $http_status），请确认地址指向中心服务。" ;;
      esac
      continue
    fi
    echo "请输入完整的 https:// 节点连接地址，不包含路径或空格。"
  done
  while :; do
    read -r -s -p "接入码（输入不显示）：" ENROLLMENT_CODE || exit 1
    echo ""
    if [ -n "$ENROLLMENT_CODE" ] && printf '%s' "$ENROLLMENT_CODE" | grep -Eq '^[A-Za-z0-9._:-]+$'; then break; fi
    echo "接入码不能为空，且只能包含字母、数字、点、下划线、冒号或短横线。"
  done
  command -v python3 >/dev/null || { echo "缺少 python3，无法检查目录。" >&2; exit 2; }
  while :; do
    read -r -e -p "节点运行数据目录（空目录，Tab 补全）：" HOST_DATA_DIR || exit 1
    HOST_DATA_DIR="$(printf '%s' "$HOST_DATA_DIR" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"
    case "$HOST_DATA_DIR" in /|/bin|/boot|/dev|/etc|/home|/opt|/proc|/root|/run|/sys|/tmp|/usr|/var) echo "请指定独立子目录。"; continue;; esac
    if printf '%s' "$HOST_DATA_DIR" | grep -Eq '^/[A-Za-z0-9._/-]+$'; then
      if [ ! -e "$HOST_DATA_DIR" ] && [ ! -L "$HOST_DATA_DIR" ]; then
        # 创建前检查父路径，避免通过符号链接或 .. 创建到用户未预期的位置。
        if ! python3 - "$HOST_DATA_DIR" <<'CREATE_CHECK'
import pathlib,sys
p=pathlib.Path(sys.argv[1])
if '..' in p.parts or any(x.is_symlink() for x in [p,*p.parents]):
 raise SystemExit('请选择无符号链接的绝对路径。')
CREATE_CHECK
        then continue; fi
        read -r -p "目录不存在，是否创建 $HOST_DATA_DIR？[y/N]：" CREATE_DIRECTORY || exit 1
        case "$CREATE_DIRECTORY" in
          y|Y|yes|YES)
            if ! mkdir -p -- "$HOST_DATA_DIR"; then
              echo "目录创建失败，请重新选择。"
              continue
            fi
            echo "目录已创建：$HOST_DATA_DIR"
            ;;
          *) echo "未创建，请重新选择目录。"; continue;;
        esac
      fi
      if validate_existing_data_dir "$HOST_DATA_DIR"; then break; fi
      continue
    fi
    echo "请输入绝对路径，仅支持英文字母、数字、点、下划线和短横线。"
  done
  ENABLE_COLLECTOR=true

fi
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
if [ ! -f "$SCRIPT_DIR/k3s/$ARCH/k3s" ] || [ ! -f "$SCRIPT_DIR/k3s/$ARCH/k3s-airgap-images-$ARCH.tar.zst" ] || [ ! -f "$SCRIPT_DIR/k3s/$ARCH/SHA256SUMS" ]; then echo "verified K3s assets for $ARCH are missing from this package" >&2; exit 1; fi
if ! compgen -G "$SCRIPT_DIR/time-sync/ubuntu-24.04/$ARCH/chrony_*.deb" >/dev/null || [ ! -f "$SCRIPT_DIR/time-sync/ubuntu-24.04/$ARCH/SHA256SUMS" ]; then echo "verified Chrony assets for Ubuntu 24.04 $ARCH are missing from this package" >&2; exit 1; fi
if [ -z "$HOST_DATA_DIR" ]; then HOST_DATA_DIR="$PREFIX/data/k3s"; fi
case "$HOST_DATA_DIR" in /*) ;; *) echo "--node-data-dir must be an absolute path" >&2; exit 2;; esac
case "$HOST_DATA_DIR" in /|/bin|/boot|/dev|/etc|/home|/opt|/proc|/root|/run|/sys|/tmp|/usr|/var) echo "--node-data-dir cannot be a system directory" >&2; exit 2;; esac
if ! printf '%s' "$HOST_DATA_DIR" | grep -Eq '^/[A-Za-z0-9._/-]+$'; then echo "--node-data-dir contains unsupported characters" >&2; exit 2; fi
if [ -n "$NODE_IP" ] && ! printf '%s' "$NODE_IP" | grep -Eq '^([0-9]{1,3}\.){3}[0-9]{1,3}$|^[0-9A-Fa-f:]+$'; then echo "--node-ip must be an IP address" >&2; exit 2; fi
if [ -z "$RELEASE_SIGNING_KEY_ID" ] || ! printf '%s' "$RELEASE_SIGNING_KEY_ID" | grep -Eq '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'; then echo "--release-signing-key-id is required and invalid" >&2; exit 2; fi
if [ -z "$RELEASE_SIGNING_PUBLIC_KEY" ] || [ "$(printf '%s' "$RELEASE_SIGNING_PUBLIC_KEY" | base64 -d 2>/dev/null | wc -c | tr -d ' ')" != 32 ]; then echo "--release-signing-public-key must be a base64 Ed25519 public key" >&2; exit 2; fi
if [ "$INTERACTIVE" = true ]; then
  command -v python3 >/dev/null || { echo "缺少 python3，无法执行安装前检查。" >&2; exit 2; }
  . /etc/os-release
  [ "${ID:-}" = ubuntu ] && [ "${VERSION_ID:-}" = 24.04 ] || { echo "当前安装包支持 Ubuntu 24.04。" >&2; exit 2; }
  # 在创建用户、目录或服务之前验证全部制品；损坏包不进入安装阶段。
  [ -f "$SCRIPT_DIR/SHA256SUMS" ] || { echo "安装包缺少完整性清单，请重新下载。" >&2; exit 2; }
  progress_start "校验安装包完整性（文件较大，请稍候）"
  (cd "$SCRIPT_DIR" && sha256sum --status -c SHA256SUMS) || { echo "安装包完整性校验失败，请重新下载。" >&2; exit 2; }
  progress_done
  validate_existing_data_dir "$HOST_DATA_DIR"
  progress_start "检查目录权限与磁盘空间"
  python3 - "$HOST_DATA_DIR" "$PREFIX" <<'CHECK'
import os,sys,shutil,pathlib
for value in sys.argv[1:]:
 p=pathlib.Path(value)
 if '..' in p.parts or any(q.is_symlink() for q in [p,*p.parents]): sys.exit('数据目录不能包含上级路径或符号链接')
 if p.exists() and (not p.is_dir() or any(p.iterdir())): sys.exit('目标目录已被占用：'+value)
 while not p.exists(): p=p.parent
 if not os.access(p,os.W_OK|os.X_OK): sys.exit('目录不可写：'+str(p))
 if shutil.disk_usage(p).free < 4*1024**3: sys.exit('目标磁盘可用空间不足 4 GB：'+str(p))
CHECK
  progress_done
  while :; do
    # 凭据从标准输入传递，避免出现在进程参数和日志中。
    progress_start "验证接入码"
    result=$(printf '%s' "$ENROLLMENT_CODE" | python3 -c 'import json,sys; print(json.dumps({"code":sys.stdin.read(),"platform":"linux","capabilities":["project_entry","data_runtime","collector"]}))' | curl "${ca_options[@]}" --silent --show-error --connect-timeout 5 --max-time 15 -H 'Content-Type: application/json' --data-binary @- "$SERVER_URL/api/v1/ops/agent/enrollments/validate") || { echo "无法完成接入预检查，尚未安装。" >&2; exit 2; }
    progress_stop
    PROGRESS_LABEL=""
    if printf '%s' "$result" | python3 -c 'import json,sys; d=json.load(sys.stdin); ok=d.get("code")==0 and d.get("data",{}).get("valid") is True; print("接入预检查通过。" if ok else d.get("msg","中心响应无效")); sys.exit(0 if ok else 1)'; then break; fi
    read -r -s -p "请重新输入接入码（Ctrl+C 退出）：" ENROLLMENT_CODE || exit 1
    echo ""
  done
  printf '\n中心地址：%s\n运行数据目录：%s\n将安装节点程序和采集能力。\n' "$SERVER_URL" "$HOST_DATA_DIR"
  read -r -p "确认开始安装？[y/N]：" CONFIRM || exit 1
  case "$CONFIRM" in y|Y|yes|YES) ;; *) echo "已取消安装。"; exit 0;; esac
fi
progress_start "准备安装目录与节点用户"
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
progress_done
progress_start "安装节点程序、运行组件和离线资源"
# 能力和离线资产是版本化制品，不保留上一包的未知文件；身份、运行数据、
# Release 与日志目录单独保留，升级不会误删用户数据。
rm -rf -- "$PREFIX/capabilities/$ARCH" "$PREFIX/k3s/$ARCH"
install -d -m 0755 "$PREFIX/bin" "$PREFIX/capabilities/$ARCH" "$PREFIX/k3s/$ARCH" "$PREFIX/data" "$PREFIX/runtime" "$PREFIX/releases" "$PREFIX/logs" "$CONFIG_DIR"
install -m 0755 "$SCRIPT_DIR/bin/node-agent-linux-$ARCH" "$PREFIX/bin/node-agent"
install -m 0755 "$SCRIPT_DIR/bin/node-hostd-linux-$ARCH" "$PREFIX/bin/node-hostd"
install -m 0755 "$SCRIPT_DIR/bin/node-hostctl-linux-$ARCH" "$PREFIX/bin/node-hostctl"
cp -R "$SCRIPT_DIR/capabilities/$ARCH/." "$PREFIX/capabilities/$ARCH/"
cp "$SCRIPT_DIR/k3s/$ARCH/k3s" "$SCRIPT_DIR/k3s/$ARCH/k3s-airgap-images-$ARCH.tar.zst" "$SCRIPT_DIR/k3s/$ARCH/SHA256SUMS" "$PREFIX/k3s/$ARCH/"
find "$PREFIX/capabilities/$ARCH" -type f -exec chmod 0755 {} +
chmod 0755 "$PREFIX/k3s/$ARCH/k3s"
chmod 0600 "$PREFIX/k3s/$ARCH/k3s-airgap-images-$ARCH.tar.zst" "$PREFIX/k3s/$ARCH/SHA256SUMS"
progress_done
progress_start "写入节点配置"
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
# 发行信任根允许运维人员保留多个 key。升级本包时只替换本包 keyId 的全部
# 旧条目；不能用 sed 只重写 trustKeys 行，否则遗留列表项会造成重复 keyId，
# 使 Agent 在加载配置时 fail-closed，进而停止心跳。
upsert_release_signing_key() {
  local temporary
  if ! grep -q '^[[:space:]]*trustKeys:' "$CONFIG_DIR/config.yaml"; then
    insert_after_yaml_key dataDir "trustKeys:" "  - keyId: '$SAFE_RELEASE_SIGNING_KEY_ID'" "    publicKey: '$SAFE_RELEASE_SIGNING_PUBLIC_KEY'"
    return
  fi
  temporary="$(mktemp "$CONFIG_DIR/config.yaml.XXXXXX")"
  if ! awk -v target_key_id="$RELEASE_SIGNING_KEY_ID" -v target_public_key="$RELEASE_SIGNING_PUBLIC_KEY" '
    function clear_entry(  entry_index) {
      entry_started=0; entry_key=""
      for (entry_index in entry_lines) delete entry_lines[entry_index]
      entry_count=0
    }
    function flush_entry(  entry_index) {
      if (entry_started && entry_key != target_key_id) {
        for (entry_index=1; entry_index<=entry_count; entry_index++) print entry_lines[entry_index]
      }
      clear_entry()
    }
    function print_target() {
      print trust_indent "  - keyId: '\''" target_key_id "'\''"
      print trust_indent "    publicKey: '\''" target_public_key "'\''"
    }
    {
      if (!inside) {
        if ($0 ~ /^[[:space:]]*trustKeys:[[:space:]]*(\[\])?[[:space:]]*$/ && !found) {
          match($0, /^[[:space:]]*/); trust_indent=substr($0, RSTART, RLENGTH)
          print trust_indent "trustKeys:"
          print_target()
          inside=1; found=1
          next
        }
        print
        next
      }

      if ($0 !~ /^[[:space:]]*$/) {
        match($0, /^[[:space:]]*/); current_indent=substr($0, RSTART, RLENGTH)
        if (length(current_indent) <= length(trust_indent)) {
          flush_entry(); inside=0
          print
          next
        }
      }
      if ($0 ~ /^[[:space:]]*-[[:space:]]+keyId:[[:space:]]*/) {
        flush_entry()
        entry_started=1; entry_count=1; entry_lines[entry_count]=$0
        entry_key=$0
        sub(/^[[:space:]]*-[[:space:]]+keyId:[[:space:]]*/, "", entry_key)
        sub(/[[:space:]]+#.*$/, "", entry_key)
        sub(/^['\'']/, "", entry_key); sub(/['\'']$/, "", entry_key)
        sub(/^"/, "", entry_key); sub(/"$/, "", entry_key)
        next
      }
      if (entry_started) {
        entry_count++; entry_lines[entry_count]=$0
      } else {
        print
      }
    }
    END {
      if (inside) flush_entry()
      if (!found) exit 42
    }
  ' "$CONFIG_DIR/config.yaml" > "$temporary"; then
    rm -f "$temporary"
    echo "cannot update release signing key in existing config" >&2
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
upsert_release_signing_key
if [ "$ENABLE_COLLECTOR" = true ]; then
  sed -i.bak '/group: collector/,/enabled: false/ s/installed: false/installed: true/' "$CONFIG_DIR/config.yaml" && rm -f "$CONFIG_DIR/config.yaml.bak"
fi
progress_done
progress_start "检查并安装系统依赖"
if [ "$(id -u)" -eq 0 ]; then
	if [ ! -f /etc/os-release ]; then echo "cannot identify Linux distribution for offline Chrony installation" >&2; exit 1; fi
	# shellcheck disable=SC1091
	. /etc/os-release
	if [ "${ID:-}" != "ubuntu" ] || [ "${VERSION_ID:-}" != "24.04" ]; then echo "this NodeAgent package currently supports offline Chrony installation on Ubuntu 24.04 only" >&2; exit 1; fi
	(cd "$SCRIPT_DIR/time-sync/ubuntu-24.04/$ARCH" && shasum -a 256 -c SHA256SUMS >/dev/null)
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
progress_done
progress_start "配置中心证书与节点服务"
# 私有中心证书仅供节点进程使用，不修改操作系统全局信任库。
if [ "$NO_SERVICE" != true ]; then
if [ -f "$SCRIPT_DIR/center-ca.crt" ]; then
  cat /etc/ssl/certs/ca-certificates.crt "$SCRIPT_DIR/center-ca.crt" > "$CONFIG_DIR/center-ca-bundle.crt"
else
  cp /etc/ssl/certs/ca-certificates.crt "$CONFIG_DIR/center-ca-bundle.crt"
fi
chmod 0644 "$CONFIG_DIR/center-ca-bundle.crt"
fi
CONNECTION_STARTED_AT=$(date +%s)
START_SERVICE=false
if [ -n "$SERVER_URL" ] && { [ -n "$ENROLLMENT_CODE" ] || [ -f "$PREFIX/data/ops-agent-identity.json" ]; }; then START_SERVICE=true; fi
if [ "$NO_SERVICE" != true ] && command -v systemctl >/dev/null 2>&1; then
  install -d -m 0700 -o "$RUN_USER" -g "$RUN_GROUP" /var/lib/induforge/image-cache
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
ReadWritePaths=/var/lib/induforge/image-cache $PREFIX/data/hostd $RUNTIME_DATA_DIR $HOST_DATA_DIR /etc/rancher/induforge-k3s /etc/chrony /etc/systemd/system /usr/local/bin /run/induforge
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
Environment=SSL_CERT_FILE=$CONFIG_DIR/center-ca-bundle.crt
ExecStart=$PREFIX/bin/node-agent --daemon
Restart=always
RestartSec=3
[Install]
WantedBy=multi-user.target
EOF
  progress_done
  progress_start "启动节点服务"
  systemctl daemon-reload
  systemctl enable --now "$HOSTD_UNIT_NAME.service"
  if systemctl is-active --quiet "$HOSTD_UNIT_NAME.service"; then systemctl restart "$HOSTD_UNIT_NAME.service"; fi
  if [ "$START_SERVICE" = true ]; then
    systemctl enable "$UNIT_NAME.service"
    if systemctl is-active --quiet "$UNIT_NAME.service"; then systemctl restart "$UNIT_NAME.service"; else systemctl start "$UNIT_NAME.service"; fi
  else echo "Service installed but not started: pass --server-url and --enrollment-code to connect."; fi
fi
progress_done
if [ "$NO_SERVICE" != true ] && [ "$START_SERVICE" = true ]; then
  wait_for_center
  exit $?
fi
echo "节点文件已安装到 ${PREFIX}，尚未验证中心接入。"
