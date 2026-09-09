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
mkdir -p "$PACKAGE_DIR/bin" "$PACKAGE_DIR/capabilities/$ARCH/collector" "$PACKAGE_DIR/k3s/$ARCH" "$PACKAGE_DIR/time-sync/ubuntu-24.04/$ARCH"
cp "$ROOT_DIR/packaging/install-linux.sh" "$PACKAGE_DIR/install.sh"
cp "$ROOT_DIR/packaging/config-linux.yaml" "$PACKAGE_DIR/config.yaml"
chmod +x "$PACKAGE_DIR/install.sh"
printf '1.2.3+test\n' > "$PACKAGE_DIR/BUILD_VERSION"
for binary in node-agent-linux-$ARCH node-hostd-linux-$ARCH node-hostctl-linux-$ARCH project-gateway runtime-api runtime-engine; do
  printf '#!/usr/bin/env sh\nexit 0\n' > "$PACKAGE_DIR/bin/$binary"
  if [ "$binary" != "node-agent-linux-$ARCH" ] && [ "$binary" != "node-hostd-linux-$ARCH" ] && [ "$binary" != "node-hostctl-linux-$ARCH" ]; then
    mv "$PACKAGE_DIR/bin/$binary" "$PACKAGE_DIR/capabilities/$ARCH/$binary"
  fi
done
printf 'fake k3s\n' > "$PACKAGE_DIR/k3s/$ARCH/k3s"
printf 'fake images\n' > "$PACKAGE_DIR/k3s/$ARCH/k3s-airgap-images-$ARCH.tar.zst"
K3S_SUM="$(shasum -a 256 "$PACKAGE_DIR/k3s/$ARCH/k3s" | awk '{print $1}')"
IMAGES_SUM="$(shasum -a 256 "$PACKAGE_DIR/k3s/$ARCH/k3s-airgap-images-$ARCH.tar.zst" | awk '{print $1}')"
printf '%s  k3s\n%s  k3s-airgap-images-%s.tar.zst\n' "$K3S_SUM" "$IMAGES_SUM" "$ARCH" > "$PACKAGE_DIR/k3s/$ARCH/SHA256SUMS"
printf 'fake chrony package\n' > "$PACKAGE_DIR/time-sync/ubuntu-24.04/$ARCH/chrony_4.5-1ubuntu4.2_$ARCH.deb"
printf 'fake chrony dependency\n' > "$PACKAGE_DIR/time-sync/ubuntu-24.04/$ARCH/tzdata-legacy_2026c_all.deb"
(cd "$PACKAGE_DIR/time-sync/ubuntu-24.04/$ARCH" && shasum -a 256 -- *.deb > SHA256SUMS)
printf '#!/usr/bin/env sh\nexit 0\n' > "$PACKAGE_DIR/capabilities/$ARCH/collector/industrial_collector"
chmod +x "$PACKAGE_DIR/bin/node-agent-linux-$ARCH" "$PACKAGE_DIR/bin/node-hostd-linux-$ARCH" "$PACKAGE_DIR/bin/node-hostctl-linux-$ARCH" "$PACKAGE_DIR/k3s/$ARCH/k3s" "$PACKAGE_DIR/capabilities/$ARCH/project-gateway" "$PACKAGE_DIR/capabilities/$ARCH/runtime-api" "$PACKAGE_DIR/capabilities/$ARCH/runtime-engine" "$PACKAGE_DIR/capabilities/$ARCH/collector/industrial_collector"

PUBLIC_KEY='AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA='
PREFIX="$PREFIX" CONFIG_DIR="$CONFIG_DIR" "$PACKAGE_DIR/install.sh" --no-service --enable-collector --node-data-dir "$TEMP_DIR/node-data/k3s" --server-url 'https://center.example.com' --enrollment-code 'one-time-code' --release-signing-key-id release-signing-key-v1 --release-signing-public-key "$PUBLIC_KEY"

test -x "$PREFIX/bin/node-agent"
test -x "$PREFIX/bin/node-hostd"
test -x "$PREFIX/bin/node-hostctl"
test -x "$PREFIX/k3s/$ARCH/k3s"
test -f "$PREFIX/k3s/$ARCH/k3s-airgap-images-$ARCH.tar.zst"
test -x "$PREFIX/capabilities/$ARCH/project-gateway"
test -x "$PREFIX/capabilities/$ARCH/runtime-api"
test -x "$PREFIX/capabilities/$ARCH/runtime-engine"
test -x "$PREFIX/capabilities/$ARCH/collector/industrial_collector"
grep -Fq "agentVersion: '1.2.3+test'" "$CONFIG_DIR/config.yaml"
grep -Fq "runtimeVersion: '1.2.3+test'" "$CONFIG_DIR/config.yaml"
grep -Fq 'serverUrl: "https://center.example.com"' "$CONFIG_DIR/config.yaml"
grep -Fq 'enrollmentCode: "one-time-code"' "$CONFIG_DIR/config.yaml"
grep -Fq "hostDataDir: '$TEMP_DIR/node-data/k3s'" "$CONFIG_DIR/config.yaml"
grep -Fq "keyId: 'release-signing-key-v1'" "$CONFIG_DIR/config.yaml"
grep -Fq "publicKey: '$PUBLIC_KEY'" "$CONFIG_DIR/config.yaml"
test "$(grep -Fc "keyId: 'release-signing-key-v1'" "$CONFIG_DIR/config.yaml")" -eq 1
grep -Fq 'Environment=NODE_AGENT_DATA_DIR=$RUNTIME_DATA_DIR' "$PACKAGE_DIR/install.sh"
awk '/group: collector/,/enabled: false/' "$CONFIG_DIR/config.yaml" | grep -Fq 'installed: true'
mkdir -p "$PREFIX/data"
printf '{"nodeId":"existing-node"}\n' > "$PREFIX/data/ops-agent-identity.json"

# 覆盖真实旧版本升级路径：旧配置由 YAML 序列化器输出为 8 空格层级。安装器
# 必须沿用原缩进插入 Hostd 字段，且不能破坏已领取的节点配置。
cat > "$CONFIG_DIR/config.yaml" <<'EOF'
agent:
    listen:
        host: 127.0.0.1
        port: 17601
    executor:
        type: process
        process:
            workDir: ./runtime
            logDir: ./logs
            binary: runtime_engine
    ops:
        enabled: true
        serverUrl: "https://center.example.com"
        enrollmentCode: ""
        agentVersion: "old"
        heartbeatEvery: 10s
        dataDir: ./data
        trustKeys:
          - keyId: 'release-signing-key-v1'
            publicKey: 'old-release-key-value'
          - keyId: 'other-release-key'
            publicKey: 'other-release-key-value'
          - keyId: 'release-signing-key-v1'
            publicKey: 'duplicate-release-key-value'
        services: []
logging:
    file: ./logs/agent.log
    level: info
EOF
printf 'stale capability\n' > "$PREFIX/capabilities/$ARCH/stale-from-old-package"
PREFIX="$PREFIX" CONFIG_DIR="$CONFIG_DIR" "$PACKAGE_DIR/install.sh" --no-service --node-data-dir "$TEMP_DIR/node-data/k3s" --node-ip 10.20.30.40 --server-url 'https://center.example.com' --release-signing-key-id release-signing-key-v1 --release-signing-public-key "$PUBLIC_KEY"
test ! -e "$PREFIX/capabilities/$ARCH/stale-from-old-package"
grep -Fq "        hostdSocket: /run/induforge/hostd.sock" "$CONFIG_DIR/config.yaml"
grep -Fq "        hostDataDir: '$TEMP_DIR/node-data/k3s'" "$CONFIG_DIR/config.yaml"
grep -Fq "        nodeIp: '10.20.30.40'" "$CONFIG_DIR/config.yaml"
grep -Fq "        trustKeys:" "$CONFIG_DIR/config.yaml"
grep -Fq "          - keyId: 'release-signing-key-v1'" "$CONFIG_DIR/config.yaml"
grep -Fq "            publicKey: '$PUBLIC_KEY'" "$CONFIG_DIR/config.yaml"
grep -Fq "          - keyId: 'other-release-key'" "$CONFIG_DIR/config.yaml"
grep -Fq "            publicKey: 'other-release-key-value'" "$CONFIG_DIR/config.yaml"
test "$(grep -Fc "keyId: 'release-signing-key-v1'" "$CONFIG_DIR/config.yaml")" -eq 1
grep -Fq '{"nodeId":"existing-node"}' "$PREFIX/data/ops-agent-identity.json"
# 第二次相同升级必须保持单条同 ID key，而不是再次追加旧列表项。
PREFIX="$PREFIX" CONFIG_DIR="$CONFIG_DIR" "$PACKAGE_DIR/install.sh" --no-service --node-data-dir "$TEMP_DIR/node-data/k3s" --release-signing-key-id release-signing-key-v1 --release-signing-public-key "$PUBLIC_KEY"
test "$(grep -Fc "keyId: 'release-signing-key-v1'" "$CONFIG_DIR/config.yaml")" -eq 1
grep -Fq "          - keyId: 'other-release-key'" "$CONFIG_DIR/config.yaml"
if PREFIX="$PREFIX" CONFIG_DIR="$CONFIG_DIR" "$PACKAGE_DIR/install.sh" --no-service --node-data-dir "/var/lib/induforge/bad'path" --release-signing-key-id release-signing-key-v1 --release-signing-public-key "$PUBLIC_KEY" >/dev/null 2>&1; then
  echo 'unsafe node data path unexpectedly accepted' >&2
  exit 1
fi
if PREFIX=/ CONFIG_DIR="$CONFIG_DIR" "$PACKAGE_DIR/install.sh" --no-service --release-signing-key-id release-signing-key-v1 --release-signing-public-key "$PUBLIC_KEY" >/dev/null 2>&1; then
  echo 'unsafe install prefix unexpectedly accepted' >&2
  exit 1
fi
echo 'install-linux no-root template test passed'
