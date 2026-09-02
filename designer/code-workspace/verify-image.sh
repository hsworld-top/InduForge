#!/bin/sh
set -eu

# 离线 K3s 中心磁盘有限；防止 OverlayFS 复制离线 pnpm store 后镜像回退到多 GB。
image=${1:-induforge/designer-code-server:4.131.0-node22-pnpm10.19.0}
max_bytes=$((3 * 1024 * 1024 * 1024))
size=$(docker image inspect --format '{{.Size}}' "$image")
if [ "$size" -gt "$max_bytes" ]; then
  echo "代码工作区镜像超过 3GiB: $size" >&2
  exit 1
fi

docker run --rm --user 1000:1000 --entrypoint sh "$image" -ec '
  test -x /usr/bin/code-server
  test -x /usr/local/bin/induforge-workspace-entrypoint
  test -f /opt/induforge/pnpm-store.version
  test -f /opt/induforge/templates/catalog.json
  test ! -e /opt/induforge/templates/vite-vue-js/node_modules
  test ! -w /opt/induforge/pnpm-store
'
printf "代码工作区镜像校验通过：%s bytes\n" "$size"
