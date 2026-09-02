#!/bin/sh
set -eu

# Docker named volume 可能由 root 预创建，保留基础镜像的动态 UID 修正。
# K3s 则由受限 initContainer 预先初始化工程目录；主容器不再需要 fixuid/sudo，
# 因而可以维持 no-new-privileges 和最小能力集。
if [ "${INDUFORGE_KUBERNETES_WORKSPACE:-}" != "true" ]; then
  eval "$(fixuid -q)"
  for writable_dir in \
    /workspace \
    /cache \
    /home/coder/.pi/agent \
    /home/coder/.local/share/code-server \
    /home/coder/.config/code-server; do
    sudo mkdir -p "$writable_dir"
    sudo chown "$(id -u):$(id -g)" "$writable_dir"
  done
fi

# 初始化产品默认设置；用户后续产生的设置文件不在启动时覆盖。
user_settings_dir=/home/coder/.local/share/code-server/User
user_settings_file="$user_settings_dir/settings.json"
if [ ! -f "$user_settings_file" ]; then
  mkdir -p "$user_settings_dir"
  cp /opt/induforge/code-server-settings.json "$user_settings_file"
fi

if [ -n "${DOCKER_USER-}" ]; then
  USER="$DOCKER_USER"
  if [ -z "$(id -u "$DOCKER_USER" 2>/dev/null)" ]; then
    echo "$DOCKER_USER ALL=(ALL) NOPASSWD:ALL" | sudo tee -a /etc/sudoers.d/nopasswd >/dev/null
    sudo usermod --login "$DOCKER_USER" coder
    sudo groupmod -n "$DOCKER_USER" coder
    sudo sed -i "/coder/d" /etc/sudoers.d/nopasswd
  fi
fi

for entrypoint_dir in "${ENTRYPOINTD:-/entrypoint.d}" /home/coder/entrypoint.d; do
  if [ -d "$entrypoint_dir" ]; then
    find "$entrypoint_dir" -type f -executable -print -exec {} \;
  fi
done

exec dumb-init /usr/local/bin/start-induforge-workspace
