#!/bin/sh
set -eu

# 工作区镜像会同时由中心 K3s、离线包和 Docker 生产路径消费。若这些引用漂移，
# centerctl 可能通过镜像预检却让控制面创建不存在或旧版本的工作区镜像。
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/../../.." && pwd)
workspace_image='induforge/designer-code-server:4.131.0-node24.19.0-pnpm11.21.0-a8eafe26-arm64'
legacy_image='induforge/designer-code-server:4.131.0-node22-pnpm10.19.0'

for source_file in \
  "$REPO_ROOT/designer/code-workspace/verify-image.sh" \
  "$SCRIPT_DIR/centerctl" \
  "$SCRIPT_DIR/center-system.yaml.template" \
  "$REPO_ROOT/scripts/offline/build-package.sh" \
  "$REPO_ROOT/scripts/offline/build-package.ps1" \
  "$REPO_ROOT/scripts/release/build-offline-package-linux.sh" \
  "$REPO_ROOT/scripts/docker/docker-compose.prod.yml" \
  "$REPO_ROOT/scripts/docker/docker-compose.offline.yml"; do
  if ! grep -Fq "$workspace_image" "$source_file"; then
    echo "正式工作区镜像引用缺失: $source_file" >&2
    exit 1
  fi
  if grep -Fq "$legacy_image" "$source_file"; then
    echo "正式工作区镜像仍引用旧版本: $source_file" >&2
    exit 1
  fi
done

echo "正式工作区镜像引用一致性测试通过"
