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
  'name: compute-sandbox' \
  'image: induforge/compute-sandbox:1.0.0' \
  'value: http://compute-sandbox:18103' \
  'key: DATA_SERVICE_COMPUTE_SANDBOX_TOKEN' \
  'name: induforge-center-control-node-labeler' \
  'resources: ["nodes"]' \
  'verbs: ["get", "list", "patch"]' \
  'value: http://center-data:18102' \
	'value: induforge/designer-code-server:4.131.0-node24.19.0-pnpm11.21.0-a8eafe26-arm64' \
	'DATA_SERVICE_INTERNAL_TOKEN' \
  'value: /contracts/collector-protocols' \
  'hostPort: 18080' \
  'supplementalGroups: [998]' \
  'name: wait-center-infrastructure' \
  'name: initialize-dev-timeseries' \
  'CREATE EXTENSION IF NOT EXISTS timescaledb;' \
  'key: IF_META_STORE_DEV_DATA_DB' \
  'name: wait-center-control' \
  'name: release-signing-key' \
  'RELEASE_BUILDER_ENABLED' \
	'AUTHORING_SNAPSHOT_CURRENT_KEY_ID' \
	'AUTHORING_SNAPSHOT_KEYRING_JSON' \
  'value: induforge/release-builder:1.0.0-node24-pnpm11.21.0' \
  'value: induforge-center-workspaces' \
  'defaultMode: 0400' \
  'publishNotReadyAddresses: true' \
  'path: /var/lib/induforge/center/meta-store'; do
  grep -Fq "$expected" "$temp_dir/rendered.yaml"
done
if ! grep -A8 -F 'name: center-control' "$temp_dir/rendered.yaml" | grep -Fq 'replicas: 0'; then
  echo "center-control must remain stopped until database bootstrap succeeds" >&2
  exit 1
fi

IF_CENTER_NODE_NAME=if-center-01 \
IF_CENTER_CONTROL_IMAGE=induforge/control:1.0.0 \
IF_CENTER_DATA_ROOT=/var/lib/induforge/center \
IF_CENTER_DOCKER_GID=998 \
  sed \
    -e 's|__CONTROL_IMAGE__|induforge/control:1.0.0|g' \
    -e 's|__DATA_ROOT__|/var/lib/induforge/center|g' \
    -e 's|__DOCKER_GID__|998|g' \
    "$SCRIPT_DIR/center-bootstrap-job.yaml.template" > "$temp_dir/bootstrap.yaml"
for expected in \
  'name: center-control-bootstrap' \
  'image: induforge/control:1.0.0' \
  'args: ["init"]' \
  'name: initialize-control-database' \
  'invalid IF_META_STORE_CORE_DB' \
  'key: IF_META_STORE_ADMIN_DATABASE' \
  'key: IF_META_STORE_CORE_DB' \
  'value: http://center-data:18102' \
  'key: DATA_SERVICE_INTERNAL_TOKEN' \
  'path: /var/lib/induforge/center/workspaces' \
  'supplementalGroups: [998]'; do
  grep -Fq "$expected" "$temp_dir/bootstrap.yaml"
done
if [ "$(grep -Fc 'allowPrivilegeEscalation: false' "$temp_dir/bootstrap.yaml")" -ne 2 ] || [ "$(grep -Fc 'drop: ["ALL"]' "$temp_dir/bootstrap.yaml")" -ne 2 ]; then
  echo "bootstrap containers must use the minimal center container security context" >&2
  exit 1
fi
init_block=$(sed -n '/name: initialize-control-database/,/containers:/p' "$temp_dir/bootstrap.yaml")
if printf '%s\n' "$init_block" | grep -Fq 'envFrom:'; then
  echo "database bootstrap init container must not receive the complete center secret" >&2
  exit 1
fi
for key in IF_META_STORE_USER IF_META_STORE_PASSWORD IF_META_STORE_ADMIN_DATABASE IF_META_STORE_CORE_DB; do
  printf '%s\n' "$init_block" | grep -Fq "key: $key"
done
bootstrap_line=$(grep -n 'run_control_bootstrap "$temp_dir/center-bootstrap-job.yaml"' "$SCRIPT_DIR/centerctl" | cut -d: -f1)
control_line=$(grep -n 'for workload in deployment/center-control deployment/center-edge' "$SCRIPT_DIR/centerctl" | cut -d: -f1)
release_line=$(grep -n 'scale deployment/center-control --replicas=1' "$SCRIPT_DIR/centerctl" | cut -d: -f1)
stopped_line=$(grep -n 'rollout status deployment/center-control --timeout=180s' "$SCRIPT_DIR/centerctl" | head -n 1 | cut -d: -f1)
if [ -z "$stopped_line" ] || [ -z "$bootstrap_line" ] || [ -z "$release_line" ] || [ -z "$control_line" ] || [ "$bootstrap_line" -le "$stopped_line" ] || [ "$release_line" -le "$bootstrap_line" ] || [ "$control_line" -le "$release_line" ]; then
  echo "control bootstrap must complete before waiting for center-control" >&2
  exit 1
fi
if ! grep -Fq 'shared_preload_libraries=timescaledb' "$temp_dir/rendered.yaml"; then
  echo "center meta store must preload TimescaleDB" >&2
  exit 1
fi
if [ "$(grep -Fc 'induforge.io/center-node: "true"' "$temp_dir/rendered.yaml")" -ne 7 ]; then
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
if ! grep -Fq 'preview-control)-[0-9a-f-]{36}' "$SCRIPT_DIR/nginx.conf" || ! grep -Fq 'proxy_pass http://center-control:18101;' "$SCRIPT_DIR/nginx.conf"; then
  echo "workspace hosts are not routed to the authenticated center gateway" >&2
  exit 1
fi
if [ "$(grep -Fc 'location /api/v1/data' "$SCRIPT_DIR/nginx.conf")" -ne 2 ] || [ "$(grep -Fc 'proxy_pass http://center-data:18102;' "$SCRIPT_DIR/nginx.conf")" -ne 2 ]; then
  echo "center data API is not exposed on both edge listeners" >&2
  exit 1
fi
if ! grep -Fq 'COPY contracts/runtime /contracts/runtime' "$SCRIPT_DIR/data-prebuilt.Dockerfile"; then
  echo "data image must include runtime artifact schemas" >&2
  exit 1
fi
if ! grep -Fq 'COPY contracts/runtime /contracts/runtime' "$SCRIPT_DIR/../../../data_service/Dockerfile"; then
  echo "source-built data image must include runtime artifact schemas" >&2
  exit 1
fi
if ! awk '/^[[:space:]]*doctor$/ { doctor_line=NR } /^[[:space:]]*persist_config$/ { persist_line=NR } END { exit !(doctor_line > 0 && persist_line == doctor_line + 2) }' "$SCRIPT_DIR/centerctl"; then
  echo "successful center apply must persist its resolved configuration" >&2
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
