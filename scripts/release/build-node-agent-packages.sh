#!/usr/bin/env bash
# 构建 Linux 和 Windows 两个 NodeAgent 安装包。能力由本机 ServiceConfig 声明，
# 不再把节点绑定到互斥角色。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
MODULE_DIR="$REPO_ROOT/runtime/node_agent"
PACKAGE_OUT="${NODE_PACKAGE_OUTPUT:-$REPO_ROOT/.data/node-packages}"
# 正式包版本必须由发布流水线显式注入；该值会写入 NodeAgent 配置并在领取/心跳时
# 由中心展示，禁止再用 demo/dev 作为不可追溯版本。
: "${NODE_AGENT_VERSION:?NODE_AGENT_VERSION is required, for example 1.0.0+build.20260831}"
VERSION="$NODE_AGENT_VERSION"
if (( ${#VERSION} == 0 || ${#VERSION} > 128 )) || [[ ! "$VERSION" =~ ^[0-9A-Za-z][0-9A-Za-z._+-]*$ ]]; then
  echo "NODE_AGENT_VERSION must be 1-128 ASCII letters, digits, dot, underscore, plus, or hyphen; it must start with a letter or digit" >&2
  exit 2
fi
STAGE_DIR="$(mktemp -d)"
trap 'rm -rf "$STAGE_DIR"' EXIT

build_linux() {
  local arch="$1" output="$2"
  (cd "$MODULE_DIR" && CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -trimpath -ldflags "-s -w" -o "$output" ./cmd)
}

# NodeAgent 安装包不再假定运行时由 demo 子进程提供。能力二进制与 Agent
# 一同构建；缺少对应工具链或发布失败时脚本直接失败，绝不生成空壳安装包。
build_linux_capability() {
  local module_dir="$1" package="$2" arch="$3" output="$4"
  (cd "$module_dir" && CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -trimpath -ldflags "-s -w" -o "$output" "$package")
}

build_collector_linux() {
  local arch="$1" output_dir="$2"
  local rid="linux-$arch"
  dotnet publish "$REPO_ROOT/collector/src/InduForge.Collector.Runtime/InduForge.Collector.Runtime.csproj" \
    -c Release -r "$rid" --self-contained true -p:PublishSingleFile=true -p:IncludeNativeLibrariesForSelfExtract=true -o "$output_dir"
  if [ ! -f "$output_dir/industrial_collector" ]; then
    echo "collector publish did not produce industrial_collector for $rid" >&2
    exit 1
  fi
}

build_collector_windows() {
  local output_dir="$1"
  dotnet publish "$REPO_ROOT/collector/src/InduForge.Collector.Runtime/InduForge.Collector.Runtime.csproj" \
    -c Release -r win-x64 --self-contained true -p:PublishSingleFile=true -p:IncludeNativeLibrariesForSelfExtract=true -o "$output_dir"
  if [ ! -f "$output_dir/industrial_collector.exe" ]; then
    echo "collector publish did not produce industrial_collector.exe for win-x64" >&2
    exit 1
  fi
}

make_linux_package() {
  local package_dir="$STAGE_DIR/induforge-node-agent-linux-$VERSION"
  mkdir -p "$package_dir/bin"
  build_linux amd64 "$package_dir/bin/node-agent-linux-amd64"
  build_linux arm64 "$package_dir/bin/node-agent-linux-arm64"
  cp "$MODULE_DIR/packaging/install-linux.sh" "$package_dir/install.sh"
  cp "$MODULE_DIR/packaging/uninstall-linux.sh" "$package_dir/uninstall.sh"
  cp "$MODULE_DIR/packaging/README-linux.md" "$package_dir/README.md"
  cp "$MODULE_DIR/packaging/config-linux.yaml" "$package_dir/config.yaml"
  printf '%s\n' "$VERSION" > "$package_dir/BUILD_VERSION"

  command -v dotnet >/dev/null 2>&1 || { echo "dotnet SDK is required to build collector NodeAgent package" >&2; exit 1; }
  for arch in amd64 arm64; do
    mkdir -p "$package_dir/capabilities/$arch"
    build_linux_capability "$REPO_ROOT/runtime/project_gateway" ./cmd/project-gateway "$arch" "$package_dir/capabilities/$arch/project-gateway"
    build_linux_capability "$REPO_ROOT/runtime/runtime_api" ./cmd/runtime-api "$arch" "$package_dir/capabilities/$arch/runtime-api"
    build_linux_capability "$REPO_ROOT/runtime/runtime_engine" ./cmd/runtime-engine "$arch" "$package_dir/capabilities/$arch/runtime-engine"
    build_collector_linux "$arch" "$package_dir/capabilities/$arch/collector"
    chmod +x "$package_dir/capabilities/$arch/project-gateway" "$package_dir/capabilities/$arch/runtime-api" "$package_dir/capabilities/$arch/runtime-engine" "$package_dir/capabilities/$arch/collector/industrial_collector"
  done
  chmod +x "$package_dir/install.sh" "$package_dir/uninstall.sh" "$package_dir/bin/"*
  tar -C "$STAGE_DIR" -czf "$PACKAGE_OUT/induforge-node-agent-linux.tar.gz" "$(basename "$package_dir")"
}

make_windows_package() {
  local package_dir="$STAGE_DIR/induforge-node-agent-windows-$VERSION"
  mkdir -p "$package_dir/bin"
  (cd "$MODULE_DIR" && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o "$package_dir/bin/node-agent.exe" ./cmd)
  cp "$MODULE_DIR/packaging/install-windows.ps1" "$package_dir/install.ps1"
  cp "$MODULE_DIR/packaging/uninstall-windows.ps1" "$package_dir/uninstall.ps1"
  cp "$MODULE_DIR/packaging/README-windows.md" "$package_dir/README.md"
  cp "$MODULE_DIR/packaging/config-windows.yaml" "$package_dir/config.yaml"
  printf '%s\n' "$VERSION" > "$package_dir/BUILD_VERSION"
  command -v dotnet >/dev/null 2>&1 || { echo "dotnet SDK is required to build collector NodeAgent package" >&2; exit 1; }
  mkdir -p "$package_dir/capabilities/collector/win-x64"
  build_collector_windows "$package_dir/capabilities/collector/win-x64"
  (cd "$STAGE_DIR" && zip -qr "$PACKAGE_OUT/induforge-node-agent-windows.zip" "$(basename "$package_dir")")
}

mkdir -p "$PACKAGE_OUT"
# 先撤销上一批 sidecar。构建任一平台失败时，旧包也不能被中心误当作本次正式版本。
rm -f "$PACKAGE_OUT/node-agent-version.txt"
make_linux_package
make_windows_package
VERSION_SIDECAR="$PACKAGE_OUT/node-agent-version.txt"
printf '%s\n' "$VERSION" > "$VERSION_SIDECAR.tmp"
mv "$VERSION_SIDECAR.tmp" "$VERSION_SIDECAR"
printf 'NodeAgent packages written to %s\n' "$PACKAGE_OUT"
