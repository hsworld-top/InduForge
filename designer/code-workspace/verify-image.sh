#!/bin/sh
set -eu

# 离线 K3s 中心磁盘有限。docker image size 只反映层总量，还必须约束容器
# 合并根文件系统，避免压缩 tar 很小但 containerd 解压后耗尽节点根盘。
image=${1:-induforge/designer-code-server:4.131.0-node24.19.0-pnpm11.21.0-8147b161-arm64}
max_layer_bytes=$((3 * 1024 * 1024 * 1024))
max_rootfs_bytes=$((5 * 1024 * 1024 * 1024 / 2))
layer_size=$(docker image inspect --format '{{.Size}}' "$image")
if [ "$layer_size" -gt "$max_layer_bytes" ]; then
  echo "代码工作区镜像层总量超过 3GiB: $layer_size" >&2
  exit 1
fi

rootfs_size=$(docker run --rm --user 1000:1000 --entrypoint sh "$image" -ec '
  test -x /usr/bin/code-server
  test -x /usr/local/bin/induforge-workspace-entrypoint
  test -f /opt/induforge/pnpm-store.version
  test -f /opt/induforge/templates/catalog.json
  test ! -e /opt/induforge/templates/vite-vue-js/node_modules
  test ! -w /opt/induforge/pnpm-store
  du -sx -B1 / 2>/dev/null | cut -f1
')
if [ "$rootfs_size" -gt "$max_rootfs_bytes" ]; then
  echo "代码工作区镜像合并根文件系统超过 2.5GiB: $rootfs_size" >&2
  exit 1
fi

printf '代码工作区镜像校验通过：层总量=%s bytes，合并根文件系统=%s bytes\n' \
  "$layer_size" "$rootfs_size"
