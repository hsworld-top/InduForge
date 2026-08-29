#!/usr/bin/env bash
# 构建三类 NodeAgent 角色安装包。默认输出到被 git 忽略的 .data/node-packages。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
MODULE_DIR="$REPO_ROOT/runtime/node_agent"
PACKAGE_OUT="${NODE_PACKAGE_OUTPUT:-$REPO_ROOT/.data/node-packages}"
VERSION="${NODE_AGENT_VERSION:-dev}"
STAGE_DIR="$(mktemp -d)"
trap 'rm -rf "$STAGE_DIR"' EXIT

build_linux() {
  local arch="$1" output="$2"
  (cd "$MODULE_DIR" && CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -trimpath -ldflags "-s -w" -o "$output" ./cmd)
}

make_linux_package() {
  local role="$1"
  local package_dir="$STAGE_DIR/induforge-node-agent-$role-$VERSION"
  mkdir -p "$package_dir/bin"
  build_linux amd64 "$package_dir/bin/node-agent-linux-amd64"
  build_linux arm64 "$package_dir/bin/node-agent-linux-arm64"
  cp "$MODULE_DIR/packaging/install-linux.sh" "$package_dir/install.sh"
  cp "$MODULE_DIR/packaging/uninstall-linux.sh" "$package_dir/uninstall.sh"
  cp "$MODULE_DIR/packaging/README-linux.md" "$package_dir/README.md"
  cp "$MODULE_DIR/packaging/config-$role.yaml" "$package_dir/config.yaml"
  chmod +x "$package_dir/install.sh" "$package_dir/uninstall.sh" "$package_dir/bin/"*
  local output_name="induforge-node-runtime-linux.tar.gz"
  if [ "$role" = "collector_linux" ]; then output_name="induforge-node-collector-linux.tar.gz"; fi
  tar -C "$STAGE_DIR" -czf "$PACKAGE_OUT/$output_name" "$(basename "$package_dir")"
}

make_windows_package() {
  local package_dir="$STAGE_DIR/induforge-node-agent-collector_windows-$VERSION"
  mkdir -p "$package_dir/bin"
  (cd "$MODULE_DIR" && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o "$package_dir/bin/node-agent.exe" ./cmd)
  cp "$MODULE_DIR/packaging/install-windows.ps1" "$package_dir/install.ps1"
  cp "$MODULE_DIR/packaging/uninstall-windows.ps1" "$package_dir/uninstall.ps1"
  cp "$MODULE_DIR/packaging/README-windows.md" "$package_dir/README.md"
  cp "$MODULE_DIR/packaging/config-collector_windows.yaml" "$package_dir/config.yaml"
  (cd "$STAGE_DIR" && zip -qr "$PACKAGE_OUT/induforge-node-collector-windows.zip" "$(basename "$package_dir")")
}

mkdir -p "$PACKAGE_OUT"
make_linux_package runtime_linux
make_linux_package collector_linux
make_windows_package
printf 'NodeAgent packages written to %s\n' "$PACKAGE_OUT"
