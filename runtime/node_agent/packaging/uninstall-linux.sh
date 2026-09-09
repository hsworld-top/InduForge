#!/usr/bin/env bash
# 默认先校验备份，再清理节点数据；--keep-data 仅用于明确保留身份和日志。
set -euo pipefail
# 在任何停服或删除动作前取得权限，避免普通用户跳过服务清理后直接删文件。
if [ "$(id -u)" -ne 0 ]; then
  command -v sudo >/dev/null 2>&1 || { echo "卸载需要管理员权限，请使用 root 运行。" >&2; exit 1; }
  echo "正在获取管理员权限，系统可能要求输入当前 Linux 用户的密码。"
  exec sudo -- bash "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/$(basename "${BASH_SOURCE[0]}")" "$@"
fi
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
    printf '未完成：%s。已停止后续清理；如已停服，服务保持停止。\n' "$PROGRESS_LABEL" >&2
  fi
  return "$result"
}
trap progress_exit EXIT
trap 'exit 130' INT
trap 'exit 143' TERM


SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PREFIX="${PREFIX:-/opt/induforge/node-agent}"
CONFIG_DIR="${CONFIG_DIR:-/etc/induforge/node-agent}"
RUNTIME_DATA_DIR="${RUNTIME_DATA_DIR:-/var/lib/induforge/node-agent}"
BACKUP_DIR="$(dirname "$SCRIPT_DIR")"
PURGE=true
YES=false
while [ "$#" -gt 0 ]; do
  case "$1" in
    --prefix) PREFIX="$2"; shift 2;;
    --config-dir) CONFIG_DIR="$2"; shift 2;;
    --backup-dir) BACKUP_DIR="$2"; shift 2;;
    --purge) PURGE=true; shift;;
    --keep-data) PURGE=false; shift;;
    --yes) YES=true; shift;;
    *) echo "不支持的参数：$1" >&2; exit 2;;
  esac
done
printf '\nInduFrame 节点卸载\n\n'
command -v python3 >/dev/null || { echo "需要 Python 3 完成备份校验，尚未卸载。" >&2; exit 1; }
command -v systemctl >/dev/null || { echo "未检测到 systemd，尚未卸载。" >&2; exit 1; }
if [ "$YES" != true ]; then
  [ -t 0 ] || { echo "请在终端运行卸载向导。" >&2; exit 2; }
  echo "仅备份节点安装配置和身份信息；工程数据、镜像及日志不备份，卸载时会清理运行数据。"
  read -r -p "备份保存目录 [$BACKUP_DIR]：" answer
  BACKUP_DIR="${answer:-$BACKUP_DIR}"
  read -r -p "确认备份并卸载？[y/N]：" answer
  case "$answer" in y|Y) ;; *) echo "已取消。"; exit 0;; esac
fi
# 所有路径在停服前检查。备份不得位于任何将被清理的目录内。
BACKUP_DIR="$(python3 - "$BACKUP_DIR" "$SCRIPT_DIR" "$PREFIX" "$CONFIG_DIR" "$RUNTIME_DATA_DIR" <<'CHECK'
import pathlib,sys
out=pathlib.Path(sys.argv[1]).expanduser().resolve()
for value in sys.argv[2:]:
 p=pathlib.Path(value)
 if not p.is_absolute() or p.is_symlink() or p.resolve() in [pathlib.Path(x) for x in ('/','/home','/var','/etc','/opt','/usr')]:
  raise SystemExit('目录不安全，已停止：'+str(p))
 if out==p.resolve() or p.resolve() in out.parents: raise SystemExit('备份目录不能位于安装包或待清理目录内')
out.mkdir(parents=True,exist_ok=True)
print(out)
CHECK
)"
STATE_FILE="$PREFIX/data/hostd/cluster-state.json"
HOST_DATA_DIR="$(python3 - "$STATE_FILE" "$BACKUP_DIR" <<'STATE'
import json,pathlib,sys
p=pathlib.Path(sys.argv[1]); data=json.loads(p.read_text()) if p.exists() else {}
value=data.get('dataDir','')
if value:
 root=pathlib.Path(value)
 if not root.is_absolute() or root.is_symlink() or len(root.parts)<3: raise SystemExit('运行数据目录不安全，已停止')
 out=pathlib.Path(sys.argv[2]);root=root.resolve()
 if root==out or root in out.parents: raise SystemExit('备份不能存入运行数据目录')
print(value)
STATE
)"
unit_exists() { [ "$(systemctl show "$1" -p LoadState --value)" != 'not-found' ]; }
INSTALLED=false
if unit_exists induforge-node-hostd.service; then
  INSTALLED=true
  [ -x "$PREFIX/bin/node-hostctl" ] || { echo "卸载工具缺失，已停止。" >&2; exit 1; }
else
  for unit in induforge-k3s-agent.service induforge-k3s-server.service; do
    if unit_exists "$unit"; then echo "节点管理服务缺失但运行组件仍存在，已停止自动清理。" >&2; exit 1; fi
  done
  echo "节点服务已卸载，将检查并备份剩余文件。"
fi
progress_start "停止节点和运行数据写入"
if unit_exists induforge-node-agent.service; then systemctl stop induforge-node-agent.service; fi
# 临时禁止 ExecStopPost 清理，并停止整个控制组，先保存数据再由 Hostd 正式卸载。
for unit in induforge-k3s-agent.service induforge-k3s-server.service; do
  if unit_exists "$unit"; then
    dropin="/run/systemd/system/$unit.d/90-induframe-backup.conf"
    mkdir -p "$(dirname "$dropin")"
    printf '[Service]\nExecStopPost=\nKillMode=control-group\n' > "$dropin"
    systemctl daemon-reload
    systemctl stop "$unit"
    rm -f "$dropin"
    systemctl daemon-reload
  fi
done
progress_done
progress_start "压缩并校验备份"
BACKUP_FILE="$(python3 - "$BACKUP_DIR" "$PREFIX" "$CONFIG_DIR" "$RUNTIME_DATA_DIR" "$HOST_DATA_DIR" <<'BACKUP'
import pathlib,sys,tarfile,tempfile,os,datetime,hashlib
out=pathlib.Path(sys.argv[1]); prefix=pathlib.Path(sys.argv[2])
# 明确列出安装信息，禁止递归归档整个运行目录（包含工程和镜像缓存）。
runtime=pathlib.Path(sys.argv[4])
roots=[pathlib.Path(sys.argv[3]), prefix/'BUILD_VERSION',
       prefix/'data'/'ops-agent-identity.json', runtime/'ops-agent-identity.json',
       prefix/'data'/'hostd'/'cluster-state.json']
roots=[p for p in roots if p.exists()]
roots=[p for p in roots if not any(q!=p and q in p.parents for q in roots)]
if not roots: print('');sys.exit(0)
fd,name=tempfile.mkstemp(prefix='InduFrame-node-backup-'+datetime.datetime.now().strftime('%Y%m%d-%H%M%S')+'-',suffix='.tar.gz.partial',dir=out)
os.close(fd)
try:
 with tarfile.open(name,'w:gz',dereference=False,compresslevel=1) as archive:
  for p in roots: archive.add(str(p),arcname=str(p).lstrip('/'))
 # 完整读取归档内容，确保压缩流可解压；不把“文件创建成功”当作备份成功。
 with tarfile.open(name,'r:gz') as archive:
  for member in archive:
   if member.isfile():
    f=archive.extractfile(member)
    while f.read(1024*1024): pass
 final=name.removesuffix('.partial');os.rename(name,final)
 os.chmod(final,0o600)
 if os.environ.get('SUDO_UID'): os.chown(final,int(os.environ['SUDO_UID']),int(os.environ['SUDO_GID']))
 print(final)
except BaseException:
 if os.path.exists(name): os.unlink(name)
 raise
BACKUP
)"
progress_done
if [ -n "$BACKUP_FILE" ]; then echo "备份已保存：$BACKUP_FILE"; else echo "没有需要备份的剩余数据。"; fi
if [ "$INSTALLED" = true ]; then
  progress_start "清理运行组件"
  systemctl start induforge-node-hostd.service
  "$PREFIX/bin/node-hostctl" uninstall-current --purge-data >/dev/null
  progress_done
  progress_start "同步中心卸载状态"
  if NODE_AGENT_CONFIG="$CONFIG_DIR/config.yaml" SSL_CERT_FILE="$CONFIG_DIR/center-ca-bundle.crt" "$PREFIX/bin/node-agent" --notify-uninstalled; then
    progress_done
  else
    progress_stop; PROGRESS_LABEL=""
    echo "本机清理已完成，但中心状态未同步。" >&2
  fi
  systemctl disable --now induforge-node-hostd.service >/dev/null
fi
progress_start "清理节点程序文件"
if unit_exists induforge-node-agent.service; then systemctl disable induforge-node-agent.service >/dev/null; fi
rm -f /etc/systemd/system/induforge-node-agent.service /etc/systemd/system/induforge-node-hostd.service
systemctl daemon-reload
if [ "$PURGE" = true ]; then
  # 清理只接受已检查的节点目录；有挂载残留时绝不递归删除。
  python3 - "$PREFIX" "$CONFIG_DIR" "$RUNTIME_DATA_DIR" <<'PURGE'
import pathlib,sys,shutil
mounts=[line.split()[4].replace('\\040',' ') for line in pathlib.Path('/proc/self/mountinfo').read_text().splitlines()]
for value in sys.argv[1:]:
 p=pathlib.Path(value)
 if any(m==str(p) or m.startswith(str(p)+'/') for m in mounts): raise SystemExit('仍有挂载，保留目录：'+str(p))
for value in sys.argv[1:]:
 p=pathlib.Path(value)
 if p.exists(): shutil.rmtree(p)
PURGE
else
  rm -f "$PREFIX/bin/node-agent" "$PREFIX/bin/node-hostd" "$PREFIX/bin/node-hostctl"
fi
progress_done
printf '\n卸载完成。\n'
if [ -n "$BACKUP_FILE" ]; then echo "备份：$BACKUP_FILE"; fi
if [ "$YES" != true ] && [ -t 0 ] && [ -f "$SCRIPT_DIR/BUILD_VERSION" ] && [ -d "$SCRIPT_DIR/bin" ] && [ "$SCRIPT_DIR" != "$PREFIX" ]; then
  read -r -p "删除解压安装包目录？[y/N]：" answer
  case "$answer" in y|Y) cd "$BACKUP_DIR"; rm -rf -- "$SCRIPT_DIR"; echo "解压安装包已删除。";; esac
fi
