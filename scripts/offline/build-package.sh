#!/usr/bin/env bash

# InduForge Linux/macOS 离线交付包构建脚本。
#
# 输入：
# - 本机 Docker 中已经存在的正式镜像。
# - 仓库内的生产环境模板、离线 compose 和安装脚本。
#
# 输出：
# - `dist/induforge-offline-package/` 目录。
# - `dist/induforge-offline-package.tar.gz` 压缩包。
#
# 重要约束：
# - 本脚本只打包镜像和部署文件，不负责构建业务镜像。
# - 如果镜像缺失会直接失败，避免生成一个目标机器无法启动的不完整安装包。

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
PACKAGE_NAME="${PACKAGE_NAME:-induforge-offline-package}"
DIST_DIR="$REPO_ROOT/dist"
PACKAGE_DIR="$DIST_DIR/$PACKAGE_NAME"
IMAGE_DIR="$PACKAGE_DIR/scripts/docker/images"

REQUIRED_IMAGES=(
  "induforge/edge:latest"
  "induforge/control:latest"
  "induforge/designer-code-server:4.131.0-node24.19.0-pnpm11.21.0-a8eafe26-arm64"
  "induforge/data:latest"
  "induforge/meta-store:latest"
  "induforge/cache-store:latest"
  "induforge/message-hub:latest"
  "induforge/object-store:latest"
  "induforge/project-gateway:1.0.2"
  "induforge/project-runtime-api:1.0.0"
  "induforge/runtime-engine:1.0.30"
  "induforge/compute-sandbox:1.0.8"
  "induforge/collector-engine:1.0.0"
)

PRODUCT_IMAGE_SOURCES=(
  "meta-store"
  "cache-store"
  "message-hub"
  "object-store"
)

image_tar_name() {
  # Docker 镜像名包含 `/` 和 `:`，不能直接作为跨平台文件名。
  # 这里统一替换成 `_`，安装脚本按目录批量 docker load，不依赖文件名反推镜像名。
  printf '%s.tar' "$1" | sed 's#[/:]#_#g'
}

require_command() {
  local command_name="$1"
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "缺少命令: $command_name" >&2
    exit 1
  fi
}

check_required_images() {
  local missing=()
  for image in "${REQUIRED_IMAGES[@]}"; do
    if ! docker image inspect "$image" >/dev/null 2>&1; then
      missing+=("$image")
    fi
  done

  if [ "${#missing[@]}" -gt 0 ]; then
    echo "缺少以下 Docker 镜像，无法生成正式离线包:" >&2
    printf '  - %s\n' "${missing[@]}" >&2
    echo "请先在构建机完成镜像构建或拉取，再重新执行本脚本。" >&2
    exit 1
  fi
}

ensure_product_image_tags() {
  local product

  # 正式交付包只保存 InduForge 产品体系镜像名。
  # 产品镜像通过轻量 wrapper 固化启动命令，避免离线 compose 暴露底层产品命令。
  for product in "${PRODUCT_IMAGE_SOURCES[@]}"; do
    echo "构建产品体系镜像: induforge/$product:latest"
    docker build \
      -t "induforge/$product:latest" \
      -f "$REPO_ROOT/scripts/docker/infra/$product/Dockerfile" \
      "$REPO_ROOT/scripts/docker/infra/$product"
  done
}

prepare_package_dir() {
  rm -rf "$PACKAGE_DIR"
  mkdir -p "$IMAGE_DIR" "$PACKAGE_DIR/scripts/docker" "$PACKAGE_DIR/scripts/offline" "$PACKAGE_DIR/scripts"
}

copy_deploy_files() {
  cp "$REPO_ROOT/.env.production.example" "$PACKAGE_DIR/.env.production.example"
  cp "$REPO_ROOT/scripts/docker/docker-compose.offline.yml" "$PACKAGE_DIR/scripts/docker/docker-compose.offline.yml"
  cp "$REPO_ROOT/scripts/offline/install.sh" "$PACKAGE_DIR/install.sh"
  cp "$REPO_ROOT/scripts/offline/install.ps1" "$PACKAGE_DIR/install.ps1"
  cp "$REPO_ROOT/scripts/offline/uninstall.sh" "$PACKAGE_DIR/uninstall.sh"
  cp "$REPO_ROOT/scripts/offline/uninstall.ps1" "$PACKAGE_DIR/uninstall.ps1"
  cp "$REPO_ROOT/scripts/offline/README.md" "$PACKAGE_DIR/README.md"
  chmod +x "$PACKAGE_DIR/install.sh"
  chmod +x "$PACKAGE_DIR/uninstall.sh"
}

save_images() {
  for image in "${REQUIRED_IMAGES[@]}"; do
    local tar_file="$IMAGE_DIR/$(image_tar_name "$image")"
    echo "保存镜像: $image -> $tar_file"
    docker save -o "$tar_file" "$image"
  done
}

create_archive() {
  rm -f "$DIST_DIR/$PACKAGE_NAME.tar.gz"
  tar -C "$DIST_DIR" -czf "$DIST_DIR/$PACKAGE_NAME.tar.gz" "$PACKAGE_NAME"
  echo "离线交付包已生成: $DIST_DIR/$PACKAGE_NAME.tar.gz"
}

main() {
  require_command docker
  require_command tar
  ensure_product_image_tags
  check_required_images
  prepare_package_dir
  copy_deploy_files
  save_images
  create_archive
}

main "$@"
