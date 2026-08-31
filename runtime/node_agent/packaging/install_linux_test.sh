#!/usr/bin/env bash
# 无 root 安装器验收：只验证文件、架构选择和配置模板，绝不启动业务服务。
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TEMP_DIR"' EXIT

case "$(uname -m)" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) echo "unsupported test architecture: $(uname -m)" >&2; exit 1 ;;
esac

PACKAGE_DIR="$TEMP_DIR/package"
PREFIX="$TEMP_DIR/prefix"
CONFIG_DIR="$TEMP_DIR/config"
mkdir -p "$PACKAGE_DIR/bin" "$PACKAGE_DIR/capabilities/$ARCH/collector"
cp "$ROOT_DIR/packaging/install-linux.sh" "$PACKAGE_DIR/install.sh"
cp "$ROOT_DIR/packaging/config-linux.yaml" "$PACKAGE_DIR/config.yaml"
chmod +x "$PACKAGE_DIR/install.sh"
printf '1.2.3+test\n' > "$PACKAGE_DIR/BUILD_VERSION"
for binary in node-agent-linux-$ARCH project-gateway runtime-api runtime-engine; do
  printf '#!/usr/bin/env sh\nexit 0\n' > "$PACKAGE_DIR/bin/$binary"
  if [ "$binary" != "node-agent-linux-$ARCH" ]; then
    mv "$PACKAGE_DIR/bin/$binary" "$PACKAGE_DIR/capabilities/$ARCH/$binary"
  fi
done
printf '#!/usr/bin/env sh\nexit 0\n' > "$PACKAGE_DIR/capabilities/$ARCH/collector/industrial_collector"
chmod +x "$PACKAGE_DIR/bin/node-agent-linux-$ARCH" "$PACKAGE_DIR/capabilities/$ARCH/project-gateway" "$PACKAGE_DIR/capabilities/$ARCH/runtime-api" "$PACKAGE_DIR/capabilities/$ARCH/runtime-engine" "$PACKAGE_DIR/capabilities/$ARCH/collector/industrial_collector"

PREFIX="$PREFIX" CONFIG_DIR="$CONFIG_DIR" "$PACKAGE_DIR/install.sh" --no-service --enable-collector --server-url 'https://center.example.com' --enrollment-code 'one-time-code'

test -x "$PREFIX/bin/node-agent"
test -x "$PREFIX/capabilities/$ARCH/project-gateway"
test -x "$PREFIX/capabilities/$ARCH/runtime-api"
test -x "$PREFIX/capabilities/$ARCH/runtime-engine"
test -x "$PREFIX/capabilities/$ARCH/collector/industrial_collector"
grep -Fq "agentVersion: '1.2.3+test'" "$CONFIG_DIR/config.yaml"
grep -Fq 'serverUrl: "https://center.example.com"' "$CONFIG_DIR/config.yaml"
grep -Fq 'enrollmentCode: "one-time-code"' "$CONFIG_DIR/config.yaml"
awk '/group: collector/,/enabled: false/' "$CONFIG_DIR/config.yaml" | grep -Fq 'installed: true'
echo 'install-linux no-root template test passed'
