#!/usr/bin/env bash

# InduForge Linux 测试打包入口。
#
# 面向对象：
# - 测试人员在 Linux 构建机上生成正式离线安装包。
#
# 输入：
# - Docker 与 Docker Compose。脚本只检测，不安装 Docker。
# - 项目源码。
# - 可选的离线镜像缓存：scripts/docker/images/*.tar。
#
# 输出：
# - 构建或加载所需 Docker 镜像。
# - 将所需镜像保存到 scripts/docker/images。
# - 生成 dist/induforge-offline-package.tar.gz。
#
# 规则：
# - 有镜像 tar 时优先 docker load。
# - 本地已有镜像时不重复拉取。
# - 基础镜像缺失时自动 docker pull。
# - 业务镜像和产品体系基础设施镜像由本机源码构建。
# - 安装包内只保存 induforge/* 产品体系镜像名。

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
IMAGE_CACHE_DIR="${IMAGE_CACHE_DIR:-$REPO_ROOT/scripts/docker/images}"

UPSTREAM_IMAGES=(
  "timescale/timescaledb:2.26.4-pg16"
  "redis:7.2-alpine"
  "emqx/emqx:5.6.1"
  "chrislusf/seaweedfs:3.85"
)

PRODUCT_INFRA_IMAGES=(
  "meta-store"
  "cache-store"
  "message-hub"
  "object-store"
)

PACKAGE_IMAGES=(
  "induforge/edge:latest"
  "induforge/control:latest"
  "induforge/designer-code-server:4.131.0-node22-pnpm10.19.0"
  "induforge/data:latest"
  "induforge/meta-store:latest"
  "induforge/cache-store:latest"
  "induforge/message-hub:latest"
  "induforge/object-store:latest"
)

image_tar_name() {
  # 镜像名包含 `/` 和 `:`，保存为跨平台安全文件名。
  printf '%s.tar' "$1" | sed 's#[/:]#_#g'
}

require_command() {
  local command_name="$1"
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "缺少命令: $command_name。请先准备构建环境。" >&2
    exit 1
  fi
}

load_cached_image_if_present() {
  local image="$1"
  local tar_file="$IMAGE_CACHE_DIR/$(image_tar_name "$image")"

  if docker image inspect "$image" >/dev/null 2>&1; then
    return
  fi

  if [ -f "$tar_file" ]; then
    echo "从缓存加载镜像: $tar_file"
    docker load -i "$tar_file"
  fi
}

ensure_upstream_images() {
  local image
  mkdir -p "$IMAGE_CACHE_DIR"

  for image in "${UPSTREAM_IMAGES[@]}"; do
    load_cached_image_if_present "$image"

    if ! docker image inspect "$image" >/dev/null 2>&1; then
      echo "拉取基础镜像: $image"
      docker pull "$image"
    fi

    save_cached_image "$image"
  done
}

save_cached_image() {
  local image="$1"
  local tar_file="$IMAGE_CACHE_DIR/$(image_tar_name "$image")"

  if [ -f "$tar_file" ]; then
    echo "镜像缓存已存在: $tar_file"
    return
  fi

  echo "保存镜像缓存: $image -> $tar_file"
  docker save -o "$tar_file" "$image"
}

build_business_images() {
  echo "构建控制面镜像..."
  docker build -t induforge/control:latest -f "$REPO_ROOT/dev_core/Dockerfile" "$REPO_ROOT"

  docker build \
    -t induforge/designer-code-server:4.131.0-node22-pnpm10.19.0 \
    -f "$REPO_ROOT/designer/code-workspace/Dockerfile" \
    "$REPO_ROOT"

  echo "构建数据服务镜像..."
  docker build -t induforge/data:latest -f "$REPO_ROOT/data_service/Dockerfile" "$REPO_ROOT/data_service"

  echo "构建边缘入口镜像..."
  docker build -t induforge/edge:latest -f "$REPO_ROOT/scripts/docker/edge/Dockerfile" "$REPO_ROOT"
}

build_product_infra_images() {
  local product

  for product in "${PRODUCT_INFRA_IMAGES[@]}"; do
    echo "构建产品体系基础设施镜像: induforge/$product:latest"
    docker build \
      -t "induforge/$product:latest" \
      -f "$REPO_ROOT/scripts/docker/infra/$product/Dockerfile" \
      "$REPO_ROOT/scripts/docker/infra/$product"
  done
}

save_package_image_cache() {
  local image

  for image in "${PACKAGE_IMAGES[@]}"; do
    save_cached_image "$image"
  done
}

main() {
  require_command docker
  require_command tar
  docker info >/dev/null

  ensure_upstream_images
  build_business_images
  build_product_infra_images
  save_package_image_cache

  "$REPO_ROOT/scripts/offline/build-package.sh"

  echo "测试安装包构建完成: $REPO_ROOT/dist/induforge-offline-package.tar.gz"
}

main "$@"
