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
K3S_VERSION="${K3S_VERSION:-v1.36.4+k3s1}"
if [ "$K3S_VERSION" != "v1.36.4+k3s1" ]; then
  echo "K3S_VERSION is pinned to v1.36.4+k3s1 for this release line" >&2
  exit 2
fi
K3S_CACHE="${K3S_ASSET_CACHE:-$REPO_ROOT/.data/release-cache/k3s/$K3S_VERSION}"
CHRONY_VERSION="${CHRONY_VERSION:-4.5-1ubuntu4.2}"
if (( ${#VERSION} == 0 || ${#VERSION} > 128 )) || [[ ! "$VERSION" =~ ^[0-9A-Za-z][0-9A-Za-z._+-]*$ ]]; then
  echo "NODE_AGENT_VERSION must be 1-128 ASCII letters, digits, dot, underscore, plus, or hyphen; it must start with a letter or digit" >&2
  exit 2
fi
STAGE_DIR="$(mktemp -d)"
trap 'rm -rf "$STAGE_DIR"' EXIT

build_linux() {
  local arch="$1" package="$2" output="$3"
  (cd "$MODULE_DIR" && CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -trimpath -ldflags "-s -w" -o "$output" "$package")
}

prepare_k3s_assets() {
  local arch="$1" destination="$2"
  local upstream_binary="k3s"
  if [ "$arch" = "arm64" ]; then upstream_binary="k3s-arm64"; fi
  local image_archive="k3s-airgap-images-$arch.tar.zst"
  local cache_dir="$K3S_CACHE/$arch"
  local encoded_version="${K3S_VERSION/+/%2B}"
  local release_url="https://github.com/k3s-io/k3s/releases/download/$encoded_version"
  mkdir -p "$cache_dir" "$destination"
  for asset in "sha256sum-$arch.txt" "$upstream_binary" "$image_archive"; do
    if [ ! -s "$cache_dir/$asset" ]; then
      curl --fail --location --retry 5 --retry-all-errors --connect-timeout 20 \
        --output "$cache_dir/$asset.tmp" "$release_url/$asset"
      mv "$cache_dir/$asset.tmp" "$cache_dir/$asset"
    fi
  done
  for asset in "$upstream_binary" "$image_archive"; do
    (cd "$cache_dir" && grep -E "  ${asset}$" "sha256sum-$arch.txt" | shasum -a 256 -c -)
  done
  cp "$cache_dir/$upstream_binary" "$destination/k3s"
  cp "$cache_dir/$image_archive" "$destination/$image_archive"
  chmod 0755 "$destination/k3s"
  chmod 0600 "$destination/$image_archive"
  (
    cd "$destination"
    shasum -a 256 k3s "$image_archive" > SHA256SUMS
  )
}

prepare_foundation_images() {
  local arch="$1" destination="$2"
  local archive="$destination/induforge-foundation-images-$arch.tar.gz"
  local images=(
    "timescale/timescaledb:2.26.4-pg16"
    "redis:7.2-alpine"
    "emqx/emqx:5.6.1"
    "nats:2.12.8-alpine"
    "chrislusf/seaweedfs:3.85"
    "nginx:1.28-alpine"
  )
  command -v docker >/dev/null 2>&1 || { echo "docker is required to bundle foundation images" >&2; exit 1; }
  for image in "${images[@]}"; do docker pull --platform "linux/$arch" "$image"; done
  docker image save "${images[@]}" | gzip -9 > "$archive.tmp"
  mv "$archive.tmp" "$archive"
  chmod 0600 "$archive"
  (cd "$destination" && shasum -a 256 "$(basename "$archive")" >> SHA256SUMS)
}

prepare_chrony_asset() {
  local arch="$1" destination="$2"
  local cache_dir="${CHRONY_ASSET_CACHE:-$REPO_ROOT/.data/release-cache/chrony/$CHRONY_VERSION}/ubuntu-24.04/$arch"
  mkdir -p "$cache_dir" "$destination"
  if ! compgen -G "$cache_dir/chrony_*.deb" >/dev/null; then
    command -v docker >/dev/null 2>&1 || { echo "docker is required to bundle Chrony" >&2; exit 1; }
    rm -f "$cache_dir"/*.deb
    docker run --rm --platform "linux/$arch" -v "$cache_dir:/out" ubuntu:24.04 sh -ec \
      "apt-get clean; apt-get update >/dev/null; apt-get install -y --download-only chrony=$CHRONY_VERSION >/dev/null; cp /var/cache/apt/archives/*.deb /out/"
  fi
  cp "$cache_dir"/*.deb "$destination/"
  (cd "$destination" && shasum -a 256 -- *.deb > SHA256SUMS)
  chmod 0600 "$destination"/*.deb "$destination/SHA256SUMS"
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
  find "$output_dir" -type f -name '*.pdb' -delete
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
  local arch="$1"
  local package_dir="$STAGE_DIR/induforge-node-agent-linux-$arch-$VERSION"
  mkdir -p "$package_dir/bin"
  build_linux "$arch" ./cmd "$package_dir/bin/node-agent-linux-$arch"
  build_linux "$arch" ./cmd/hostd "$package_dir/bin/node-hostd-linux-$arch"
  build_linux "$arch" ./cmd/hostctl "$package_dir/bin/node-hostctl-linux-$arch"
  cp "$MODULE_DIR/packaging/install-linux.sh" "$package_dir/install.sh"
  cp "$MODULE_DIR/packaging/uninstall-linux.sh" "$package_dir/uninstall.sh"
  cp "$MODULE_DIR/packaging/README-linux.md" "$package_dir/README.md"
  cp "$MODULE_DIR/packaging/config-linux.yaml" "$package_dir/config.yaml"
  printf '%s\n' "$VERSION" > "$package_dir/BUILD_VERSION"

  command -v dotnet >/dev/null 2>&1 || { echo "dotnet SDK is required to build collector NodeAgent package" >&2; exit 1; }
  mkdir -p "$package_dir/capabilities/$arch" "$package_dir/k3s/$arch" "$package_dir/time-sync/ubuntu-24.04/$arch"
  build_linux_capability "$REPO_ROOT/runtime/project_gateway" ./cmd/project-gateway "$arch" "$package_dir/capabilities/$arch/project-gateway"
  build_linux_capability "$REPO_ROOT/runtime/runtime_api" ./cmd/runtime-api "$arch" "$package_dir/capabilities/$arch/runtime-api"
  build_linux_capability "$REPO_ROOT/runtime/runtime_engine" ./cmd/runtime-engine "$arch" "$package_dir/capabilities/$arch/runtime-engine"
  build_collector_linux "$arch" "$package_dir/capabilities/$arch/collector"
  prepare_k3s_assets "$arch" "$package_dir/k3s/$arch"
  prepare_foundation_images "$arch" "$package_dir/k3s/$arch"
  prepare_chrony_asset "$arch" "$package_dir/time-sync/ubuntu-24.04/$arch"
  chmod +x "$package_dir/capabilities/$arch/project-gateway" "$package_dir/capabilities/$arch/runtime-api" "$package_dir/capabilities/$arch/runtime-engine" "$package_dir/capabilities/$arch/collector/industrial_collector"
  chmod +x "$package_dir/install.sh" "$package_dir/uninstall.sh" "$package_dir/bin/"*
  # macOS bsdtar 默认会写入宿主机扩展属性，Linux 解包会产生噪声且污染制品。
  COPYFILE_DISABLE=1 tar --no-xattrs -C "$STAGE_DIR" -czf "$PACKAGE_OUT/induforge-node-agent-linux-$arch.tar.gz" "$(basename "$package_dir")"
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
rm -f "$PACKAGE_OUT/node-agent-version.txt" "$PACKAGE_OUT/induforge-node-agent-linux.tar.gz"
LINUX_ARCHES="${NODE_PACKAGE_LINUX_ARCHES:-amd64 arm64}"
for arch in $LINUX_ARCHES; do
  case "$arch" in amd64|arm64) make_linux_package "$arch";; *) echo "unsupported NODE_PACKAGE_LINUX_ARCHES entry: $arch" >&2; exit 2;; esac
done
if [ "${NODE_PACKAGE_INCLUDE_WINDOWS:-true}" = "true" ]; then make_windows_package; fi
VERSION_SIDECAR="$PACKAGE_OUT/node-agent-version.txt"
printf '%s\n' "$VERSION" > "$VERSION_SIDECAR.tmp"
mv "$VERSION_SIDECAR.tmp" "$VERSION_SIDECAR"
printf 'NodeAgent packages written to %s\n' "$PACKAGE_OUT"
