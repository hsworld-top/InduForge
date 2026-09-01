#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT INT TERM

IF_CENTER_NODE_NAME=if-center-01 \
IF_CENTER_CONTROL_IMAGE=induforge/control:1.0.0 \
IF_CENTER_DATA_IMAGE=induforge/data:1.0.0 \
IF_CENTER_EDGE_IMAGE=induforge/edge:1.0.0 \
IF_CENTER_DATA_ROOT=/var/lib/induforge/center \
IF_EDGE_HOST_PORT=18080 \
IF_EDGE_TLS_HOST_PORT=18443 \
IF_CENTER_DOCKER_GID=998 \
  "$SCRIPT_DIR/centerctl" render > "$temp_dir/rendered.yaml"

ln -s "$SCRIPT_DIR/centerctl" "$temp_dir/centerctl"
IF_CENTER_NODE_NAME=if-center-01 \
IF_CENTER_CONTROL_IMAGE=induforge/control:1.0.0 \
IF_CENTER_DATA_IMAGE=induforge/data:1.0.0 \
IF_CENTER_EDGE_IMAGE=induforge/edge:1.0.0 \
IF_CENTER_DATA_ROOT=/var/lib/induforge/center \
IF_CENTER_DOCKER_GID=998 \
  "$temp_dir/centerctl" render > "$temp_dir/rendered-from-symlink.yaml"
cmp "$temp_dir/rendered.yaml" "$temp_dir/rendered-from-symlink.yaml"

if grep -q '__[A-Z_]*__' "$temp_dir/rendered.yaml"; then
  echo "render left unresolved placeholders" >&2
  exit 1
fi
for expected in \
  'namespace: induforge-system' \
  'induforge.io/center-node: "true"' \
  'image: induforge/control:1.0.0' \
  'image: induforge/data:1.0.0' \
  'image: induforge/edge:1.0.0' \
  'name: center-data' \
  'value: http://center-data:18102' \
	'DATA_SERVICE_INTERNAL_TOKEN' \
  'value: /contracts/collector-protocols' \
  'hostPort: 18080' \
  'supplementalGroups: [998]' \
  'name: wait-center-infrastructure' \
  'name: wait-center-control' \
  'name: release-signing-key' \
  'RELEASE_BUILDER_ENABLED' \
  'value: induforge/release-builder:1.0.0-node24-pnpm11.21.0' \
  'value: induforge-center-workspaces' \
  'defaultMode: 0400' \
  'publishNotReadyAddresses: true' \
  'path: /var/lib/induforge/center/meta-store'; do
  grep -Fq "$expected" "$temp_dir/rendered.yaml"
done
if [ "$(grep -Fc 'induforge.io/center-node: "true"' "$temp_dir/rendered.yaml")" -ne 6 ]; then
  echo "not every center workload is pinned to the center node" >&2
  exit 1
fi
if [ "$(grep -Fc 'path: /var/run/docker.sock' "$temp_dir/rendered.yaml")" -ne 1 ]; then
  echo "Docker compatibility socket escaped the single control workload boundary" >&2
  exit 1
fi
if [ "$(grep -Fc 'location = /health' "$SCRIPT_DIR/nginx.conf")" -ne 2 ]; then
  echo "center health endpoint is not exposed on both edge listeners" >&2
  exit 1
fi

if IF_CENTER_NODE_NAME=if-center-01 IF_CENTER_DATA_ROOT=/ IF_CENTER_DOCKER_GID=998 "$SCRIPT_DIR/centerctl" render >/dev/null 2>&1; then
  echo "unsafe data root was accepted" >&2
  exit 1
fi
if IF_CENTER_NODE_NAME=if-center-01 IF_EDGE_HOST_PORT=18080 IF_EDGE_TLS_HOST_PORT=18080 IF_CENTER_DOCKER_GID=998 "$SCRIPT_DIR/centerctl" render >/dev/null 2>&1; then
  echo "duplicate edge ports were accepted" >&2
  exit 1
fi
if IF_CENTER_NODE_NAME=if-center-01 IF_CENTER_DOCKER_GID=998 IF_RELEASE_WORKSPACE_VOLUME=bad/name "$SCRIPT_DIR/centerctl" render >/dev/null 2>&1; then
  echo "unsafe release workspace volume was accepted" >&2
  exit 1
fi

echo "centerctl render tests passed"
