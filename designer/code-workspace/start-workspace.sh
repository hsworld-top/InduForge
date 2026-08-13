#!/bin/sh
set -eu

pi_web_pid=''
code_server_pid=''
preview_control_pid=''

stop_children() {
  if [ -n "$pi_web_pid" ]; then
    kill "$pi_web_pid" 2>/dev/null || true
  fi
  if [ -n "$code_server_pid" ]; then
    kill "$code_server_pid" 2>/dev/null || true
  fi
  if [ -n "$preview_control_pid" ]; then
    kill "$preview_control_pid" 2>/dev/null || true
  fi
  [ -z "$pi_web_pid" ] || wait "$pi_web_pid" 2>/dev/null || true
  [ -z "$code_server_pid" ] || wait "$code_server_pid" 2>/dev/null || true
  [ -z "$preview_control_pid" ] || wait "$preview_control_pid" 2>/dev/null || true
}

trap stop_children EXIT INT TERM

# 仅在工程尚未建立时复制公共模板，保留工作区中已有的任意文件。
if [ ! -f /workspace/package.json ]; then
  cp -a -n /opt/induforge/template/. /workspace/
fi

if [ ! -f /workspace/package.json ] || [ ! -f /workspace/pnpm-lock.yaml ]; then
  echo "工作区缺少 package.json 或 pnpm-lock.yaml，无法启动 Vite。" >&2
  exit 1
fi

cd /workspace
install_version="$(sha256sum package.json pnpm-lock.yaml | sha256sum | cut -d ' ' -f 1)"
install_marker="node_modules/.induforge-install.version"
if [ ! -d node_modules ] || [ ! -f "$install_marker" ] || [ "$(cat "$install_marker")" != "$install_version" ]; then
  CI=true pnpm install --offline --frozen-lockfile --store-dir "${npm_config_store_dir:-/cache/pnpm-store}"
  mkdir -p node_modules
  printf '%s\n' "$install_version" > "$install_marker"
fi

# 三个服务共享工作目录和 coder 用户，AI 修改会直接触发 Vite HMR 并反映到编辑器。
pi-web \
  --hostname "${PI_WEB_HOSTNAME:-0.0.0.0}" \
  --port "${PI_WEB_PORT:-30141}" \
  --no-open &
pi_web_pid=$!

code-server \
  --bind-addr 0.0.0.0:3000 \
  --auth none \
  --disable-telemetry \
  --disable-update-check \
  --disable-workspace-trust \
  --disable-getting-started-override \
  /workspace &
code_server_pid=$!

node /opt/induforge/preview-control.mjs &
preview_control_pid=$!

while kill -0 "$pi_web_pid" 2>/dev/null && kill -0 "$code_server_pid" 2>/dev/null && kill -0 "$preview_control_pid" 2>/dev/null; do
  sleep 1
done

if ! kill -0 "$pi_web_pid" 2>/dev/null; then
  echo "Pi Web 进程已退出，停止工程工作区。" >&2
elif ! kill -0 "$code_server_pid" 2>/dev/null; then
  echo "code-server 进程已退出，停止工程工作区。" >&2
else
  echo "Preview Control 进程已退出，停止工程工作区。" >&2
fi

exit 1
