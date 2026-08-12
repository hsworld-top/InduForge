#!/bin/sh
set -eu

source_store="/opt/induforge/pnpm-store"
source_version="/opt/induforge/pnpm-store.version"
target_store="${npm_config_store_dir:-/cache/pnpm-store}"
target_version="$target_store/.induforge-seed.version"

if cmp -s "$source_version" "$target_version"; then
  exit 0
fi

# 工程缓存卷首次挂载时为空；只预热固定模板依赖，不删除或重置客户已产生的缓存。
mkdir -p "$target_store"
cp -a "$source_store/." "$target_store/"
cp "$source_version" "$target_version"
