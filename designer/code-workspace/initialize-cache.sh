#!/bin/sh
set -eu

source_store="/opt/induforge/pnpm-store"
source_version="/opt/induforge/pnpm-store.version"
target_store="${PNPM_CONFIG_STORE_DIR:-/cache/pnpm-store}"
target_version="$target_store/.induforge-seed.version"

# pnpm 11 会在 store 中维护 SQLite 索引。缓存种子来自只读镜像层，复制后必须
# 恢复当前容器用户的写权限，且已有缓存卷也要在版本命中前完成权限修复。
mkdir -p "$target_store"
chmod -R u+rwX "$target_store"

if cmp -s "$source_version" "$target_version"; then
  exit 0
fi

# 工程缓存卷首次挂载时为空；只预热固定模板依赖，不删除或重置客户已产生的缓存。
cp -a "$source_store/." "$target_store/"
cp "$source_version" "$target_version"
chmod -R u+rwX "$target_store"
