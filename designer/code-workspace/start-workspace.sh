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

cd /workspace

# 三个常驻服务允许空工作区启动；Preview Control 在模板初始化完成后再启动 Vite。
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
