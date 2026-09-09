#!/usr/bin/env bash
# 从已构建的离线 Docker 归档生成中心镜像资源，构建过程不拉取镜像。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ARCH="${1:?usage: build-center-image-resources.sh arm64|amd64 docker-save.tar output-directory}"
ARCHIVE="${2:?offline docker save archive is required}"
OUTPUT="${3:?output directory is required}"
case "$ARCH" in arm64|amd64) ;; *) echo "unsupported architecture" >&2; exit 2;; esac
mkdir -p "$OUTPUT/bin"
OUTPUT="$(cd "$OUTPUT" && pwd)"
python3 "$ROOT/scripts/release/export-node-image-resources.py" --archive "$ARCHIVE" --output "$OUTPUT/resources" --architecture "$ARCH"
(cd "$ROOT/dev_core" && CGO_ENABLED=0 GOOS=linux GOARCH="$ARCH" go build -trimpath -ldflags='-s -w' -o "$OUTPUT/bin/image-import" ./cmd/image-import)
(cd "$ROOT/runtime/node_agent" && CGO_ENABLED=0 GOOS=linux GOARCH="$ARCH" go build -trimpath -ldflags='-s -w' -o "$OUTPUT/bin/node-hostd" ./cmd/hostd)
cp "$ROOT/scripts/k3s/center/install-image-service.sh" "$ROOT/scripts/k3s/center/import-image-resources.py" "$OUTPUT/"
(cd "$OUTPUT" && find bin resources -type f -exec shasum -a 256 {} \; > SHA256SUMS)
echo "中心镜像资源及本机导入服务已生成：$OUTPUT"
