#!/bin/sh
set -eu

image="${INDUFORGE_WORKSPACE_IMAGE:-induforge/designer-code-server:workspace-templates-source}"
container="induforge-designer-workspace-dev"
workspace_volume="induforge-designer-workspace-dev-workspace-v2"

# 开发环境固定容器名、端口和持久化卷；替换镜像时只重建容器，不删除工程数据。
docker rm --force "$container" >/dev/null 2>&1 || true
docker run --detach \
  --name "$container" \
  --restart unless-stopped \
  --publish 127.0.0.1:36200:3000 \
  --publish 127.0.0.1:36241:30141 \
  --publish 127.0.0.1:38173:5173 \
  --publish 127.0.0.1:38174:5174 \
  --volume "$workspace_volume":/workspace \
  --volume induforge-designer-workspace-dev-cache:/cache \
  --volume induforge-designer-workspace-dev-pi-agent:/home/coder/.pi/agent \
  --volume induforge-designer-workspace-dev-code-data:/home/coder/.local/share/code-server \
  --volume induforge-designer-workspace-dev-code-config:/home/coder/.config/code-server \
  "$image"

printf '已启动 %s，镜像 %s。\n' "$container" "$image"
