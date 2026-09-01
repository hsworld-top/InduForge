#!/usr/bin/env bash

# 静态核对离线交付清单：每个交付镜像都必须有构建来源，ARM64 运行镜像必须显式构建。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# shellcheck source=build-offline-package-linux.sh
source "$SCRIPT_DIR/build-offline-package-linux.sh"

required_images=()
while IFS= read -r image; do
  required_images+=("$image")
done < <(awk '
  /^REQUIRED_IMAGES=\(/ { reading = 1; next }
  reading && /^\)/ { exit }
  reading && /^  "/ {
    sub(/^  "/, "")
    sub(/"$/, "")
    print
  }
' "$REPO_ROOT/scripts/offline/build-package.sh")

if [ "${#required_images[@]}" -ne "${#PACKAGE_IMAGES[@]}" ]; then
  echo "离线打包清单数量不一致" >&2
  exit 1
fi

for index in "${!required_images[@]}"; do
  if [ "${required_images[$index]}" != "${PACKAGE_IMAGES[$index]}" ]; then
    echo "离线打包清单顺序或镜像不一致: ${required_images[$index]} != ${PACKAGE_IMAGES[$index]}" >&2
    exit 1
  fi
done

buildable_images=("${BUSINESS_IMAGES[@]}" "${ARM64_RUNTIME_IMAGES[@]}")
for product in "${PRODUCT_INFRA_IMAGES[@]}"; do
  buildable_images+=("induforge/$product:latest")
done

for image in "${required_images[@]}"; do
  found=0
  for buildable in "${buildable_images[@]}"; do
    if [ "$image" = "$buildable" ]; then
      found=1
      break
    fi
  done
  if [ "$found" -ne 1 ]; then
    echo "交付镜像缺少构建函数或命令: $image" >&2
    exit 1
  fi
done

if [ "${#ARM64_RUNTIME_IMAGES[@]}" -ne "${#ARM64_RUNTIME_DOCKERFILES[@]}" ] || [ "${#ARM64_RUNTIME_IMAGES[@]}" -ne "${#ARM64_RUNTIME_CONTEXTS[@]}" ]; then
  echo "ARM64 运行镜像构建清单不一致" >&2
  exit 1
fi

for index in "${!ARM64_RUNTIME_IMAGES[@]}"; do
  image="${ARM64_RUNTIME_IMAGES[$index]}"
  dockerfile="$REPO_ROOT/${ARM64_RUNTIME_DOCKERFILES[$index]}"
  context="$REPO_ROOT/${ARM64_RUNTIME_CONTEXTS[$index]}"
  [ -f "$dockerfile" ] || { echo "ARM64 Dockerfile 不存在: $dockerfile" >&2; exit 1; }
  [ -d "$context" ] || { echo "ARM64 构建上下文不存在: $context" >&2; exit 1; }
  grep -Fq "ARG TARGETOS=linux" "$dockerfile" || { echo "ARM64 Dockerfile 未声明 TARGETOS: $dockerfile" >&2; exit 1; }
  grep -Fq "ARG TARGETARCH=arm64" "$dockerfile" || { echo "ARM64 Dockerfile 未声明 TARGETARCH: $dockerfile" >&2; exit 1; }
  grep -Fq "CGO_ENABLED=0 GOOS=\$TARGETOS GOARCH=\$TARGETARCH go build" "$dockerfile" || { echo "ARM64 Dockerfile 未显式交叉编译: $dockerfile" >&2; exit 1; }
  printf '已验证 ARM64 构建映射: %s\n' "$image"
done

echo "离线构建镜像清单测试通过"
