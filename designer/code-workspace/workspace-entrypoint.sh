#!/bin/sh
set -eu

# 保留 code-server 基础镜像的动态 UID 修正与启动初始化钩子。
eval "$(fixuid -q)"

# Docker named volume 可能由 root 预创建，启动前只修正专用缓存根目录权限。
sudo mkdir -p /cache
sudo chown "$(id -u):$(id -g)" /cache

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
