#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
temp_dir=$(mktemp -d)
docker_test_network=""
docker_test_upstream=""
docker_test_edge=""
cleanup() {
  if [ -n "$docker_test_edge" ]; then docker rm -f "$docker_test_edge" >/dev/null 2>&1 || true; fi
  if [ -n "$docker_test_upstream" ]; then docker rm -f "$docker_test_upstream" >/dev/null 2>&1 || true; fi
  if [ -n "$docker_test_network" ]; then docker network rm "$docker_test_network" >/dev/null 2>&1 || true; fi
  rm -rf "$temp_dir"
}
trap cleanup EXIT INT TERM

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
	'value: induforge/designer-code-server:4.131.0-node24.19.0-pnpm11.21.0-8147b161-arm64' \
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
if [ "$(grep -Fc 'listen 80 default_server;' "$SCRIPT_DIR/nginx.conf")" -ne 1 ] || \
   [ "$(grep -Fc 'listen 443 ssl default_server;' "$SCRIPT_DIR/nginx.conf")" -ne 1 ]; then
  echo "center HTTP and HTTPS listeners must remain the only default servers" >&2
  exit 1
fi
edge_block=$(sed -n '/name: center-edge/,/volumes:/p' "$temp_dir/rendered.yaml")
if [ "$(printf '%s\n' "$edge_block" | grep -Fc 'path: /health')" -ne 2 ]; then
  echo "center edge readiness and liveness probes must use /health" >&2
  exit 1
fi
if ! grep -Fq 'preview-control)-[0-9a-f-]{36}' "$SCRIPT_DIR/nginx.conf" || ! grep -Fq 'proxy_pass http://center-control:18101;' "$SCRIPT_DIR/nginx.conf"; then
  echo "workspace hosts are not routed to the authenticated center gateway" >&2
  exit 1
fi
if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
  mkdir -p "$temp_dir/nginx-tls"
  openssl req -x509 -newkey rsa:2048 -nodes -days 1 \
    -subj '/CN=center.induforge.test' \
    -keyout "$temp_dir/nginx-tls/center.key" \
    -out "$temp_dir/nginx-tls/center.crt" >/dev/null 2>&1
  docker run --rm \
    --add-host center-control:127.0.0.1 \
    --add-host center-data:127.0.0.1 \
    -v "$SCRIPT_DIR/nginx.conf:/etc/nginx/nginx.conf:ro" \
    -v "$temp_dir/nginx-tls:/etc/nginx/tls:ro" \
    nginx:1.28-alpine nginx -t >/dev/null

  docker_test_network="induforge-center-nginx-test-$$"
  docker_test_upstream="induforge-center-upstream-test-$$"
  docker_test_edge="induforge-center-edge-test-$$"
  cat > "$temp_dir/upstream-nginx.conf" <<'EOF'
events {}
http {
  server { listen 18101; location / { return 200 'workspace-gateway'; } }
  server { listen 18102; location / { return 200 'data-service'; } }
}
EOF
  docker network create "$docker_test_network" >/dev/null
  docker run -d --name "$docker_test_upstream" \
    --network "$docker_test_network" \
    --network-alias center-control \
    --network-alias center-data \
    -v "$temp_dir/upstream-nginx.conf:/etc/nginx/nginx.conf:ro" \
    nginx:1.28-alpine >/dev/null
  docker run -d --name "$docker_test_edge" \
    --network "$docker_test_network" \
    --network-alias center-edge \
    -v "$SCRIPT_DIR/nginx.conf:/etc/nginx/nginx.conf:ro" \
    -v "$temp_dir/nginx-tls:/etc/nginx/tls:ro" \
    nginx:1.28-alpine >/dev/null

  docker run --rm --network "$docker_test_network" nginx:1.28-alpine \
    wget -qO- --header='Host: kubelet-probe.invalid' http://center-edge/health >/dev/null
  workspace_headers=$(docker run --rm --network "$docker_test_network" nginx:1.28-alpine \
    sh -c "wget -S -O /dev/null --header='Host: code-01234567-89ab-cdef-0123-456789abcdef.workspace.induforge.test' http://center-edge/ 2>&1 || true")
  if ! printf '%s\n' "$workspace_headers" | grep -Fq 'HTTP/1.1 308 Permanent Redirect'; then
    echo "workspace HTTP host must redirect to HTTPS" >&2
    exit 1
  fi
  for center_host in center.induforge.test kubelet-probe.invalid; do
    center_body=$(docker run --rm --network "$docker_test_network" nginx:1.28-alpine \
      wget -qO- --no-check-certificate --header="Host: $center_host" https://center-edge/)
    if ! printf '%s\n' "$center_body" | grep -Fq 'Welcome to nginx!'; then
      echo "center HTTPS host $center_host did not reach the static center entry" >&2
      exit 1
    fi
  done
  workspace_body=$(docker run --rm --network "$docker_test_network" nginx:1.28-alpine \
    wget -qO- --no-check-certificate \
      --header='Host: code-01234567-89ab-cdef-0123-456789abcdef.workspace.induforge.test' \
      https://center-edge/)
  if [ "$workspace_body" != 'workspace-gateway' ]; then
    echo "workspace HTTPS host did not reach the authenticated center gateway" >&2
    exit 1
  fi
else
  echo "skip nginx syntax test: Docker is unavailable" >&2
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

fake_bin="$temp_dir/fake-bin"
fake_root="$temp_dir/fake-center"
mkdir -p "$fake_bin" "$fake_root/tls"
printf 'test certificate\n' > "$fake_root/tls/center.crt"
printf 'test key\n' > "$fake_root/tls/center.key"
cat > "$temp_dir/center.env" <<'EOF'
CENTER_PUBLIC_ORIGIN=https://center.induforge.test:18443
WORKSPACE_PUBLIC_ORIGIN_TEMPLATE=https://{service}-{projectId}.workspace.induforge.test:18443
CODE_WORKSPACE_ALLOWED_ORIGINS=https://center.induforge.test:18443
DATA_SERVICE_COMPUTE_SANDBOX_TOKEN=0123456789abcdef0123456789abcdef
EOF
cat > "$fake_bin/id" <<'EOF'
#!/bin/sh
if [ "${1:-}" = -u ]; then echo 0; else exec /usr/bin/id "$@"; fi
EOF
cat > "$fake_bin/docker" <<'EOF'
#!/bin/sh
case "$1 $2" in
  'volume inspect')
    case " $* " in *' -f '*) printf 'none|bind|%s/workspaces\n' "$IF_CENTER_DATA_ROOT";; esac
    ;;
  'volume create'|'image inspect') exit 0 ;;
esac
EOF
cat > "$fake_bin/openssl" <<'EOF'
#!/bin/sh
case "${1:-}" in
  rand)
    case " ${2:-} " in
      ' -base64 ') printf 'MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=\n' ;;
      *) printf '0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef\n' ;;
    esac
    ;;
  genpkey)
    while [ "$#" -gt 0 ]; do
      if [ "$1" = -out ]; then shift; printf 'fake-ed25519-key\n' > "$1"; exit 0; fi
      shift
    done
    exit 1
    ;;
  pkey) exit 0 ;;
  *) exit 1 ;;
esac
EOF
cat > "$fake_bin/stat" <<'EOF'
#!/bin/sh
case " $* " in
  *" -c %a "*) printf '600\n' ;;
  *" -c %g "*) printf '998\n' ;;
  *) exec /usr/bin/stat "$@" ;;
esac
EOF
cat > "$fake_bin/k3s" <<'EOF'
#!/bin/sh
printf '%s\n' "$*" >> "$FAKE_K3S_LOG"
if [ "${1:-}" = ctr ]; then
  cat <<'IMAGES'
docker.io/induforge/control:1.0.0
docker.io/induforge/data:1.0.0
docker.io/induforge/edge:1.0.0
docker.io/induforge/compute-sandbox:1.0.0
docker.io/induforge/designer-code-server:4.131.0-node24.19.0-pnpm11.21.0-8147b161-arm64
docker.io/timescale/timescaledb:2.26.4-pg16
docker.io/library/redis:7.2-alpine
docker.io/chrislusf/seaweedfs:3.85
docker.io/rancher/mirrored-library-busybox:1.37.0
IMAGES
  exit 0
fi
shift
case " $* " in
  *' get deployment center-control -o jsonpath='*)
    case "$FAKE_LEGACY_CONTROL_ENV" in
      __default__) printf '%s\n' 'CODE_WORKSPACE_ALLOWED_ORIGINS
CENTER_PUBLIC_ORIGIN
WORKSPACE_PUBLIC_ORIGIN_TEMPLATE' ;;
      *) printf '%s\n' "$FAKE_LEGACY_CONTROL_ENV" ;;
    esac
    ;;
  *' get job center-control-bootstrap -o name --ignore-not-found=true '*)
    case "${FAKE_BOOTSTRAP_JOB_STATE:-missing}" in
      missing) : ;;
      *) printf 'job.batch/center-control-bootstrap\n' ;;
    esac
    ;;
  *' get job center-control-bootstrap -o jsonpath='*)
    case "${FAKE_BOOTSTRAP_JOB_STATE:-missing}" in
      failed) printf 'Failed=True\n' ;;
      complete) printf 'Complete=True\n' ;;
      active) : ;;
      *) exit 1 ;;
    esac
    ;;
  *' get job center-control-bootstrap '*)
    case "${FAKE_BOOTSTRAP_JOB_STATE:-missing}" in missing) exit 1 ;; *) : ;; esac
    ;;
  *' set env deployment/center-control '*)
    if [ "$FAKE_SET_ENV_FAILURE" = 1 ]; then
      exit 1
    fi
    ;;
  *' get deployment center-control '*) [ "${FAKE_DEPLOYMENT_EXISTS:-1}" = 1 ] ;;
  *' get secret center-env '*) exit 1 ;;
  *' create secret generic '*|*' create configmap '*) printf 'apiVersion: v1\nkind: Secret\n' ;;
  *' apply -f - '*) cat >/dev/null; printf 'configured\n' ;;
  *' get node '*'jsonpath='*) printf 'True' ;;
  *' get deployment,statefulset '*'jsonpath='*) : ;;
  *' get endpointslice '*) printf 'true' ;;
  *) : ;;
esac
EOF
chmod +x "$fake_bin/id" "$fake_bin/docker" "$fake_bin/openssl" "$fake_bin/stat" "$fake_bin/k3s"

run_fake_apply() {
  deployment_exists=$1
  log_file=$2
  bootstrap_job_state=${3:-missing}
  set_env_failure=${4:-0}
  legacy_control_env=${5-__default__}
  : > "$log_file"
  PATH="$fake_bin:$PATH" \
  FAKE_K3S_LOG="$log_file" \
  FAKE_DEPLOYMENT_EXISTS="$deployment_exists" \
  FAKE_BOOTSTRAP_JOB_STATE="$bootstrap_job_state" \
  FAKE_LEGACY_CONTROL_ENV="$legacy_control_env" \
  FAKE_SET_ENV_FAILURE="$set_env_failure" \
  IF_CENTER_CONFIG_FILE="$temp_dir/center-k3s.conf" \
  IF_CENTER_K3S_BIN="$fake_bin/k3s" \
  IF_CENTER_NODE_NAME=if-center-01 \
  IF_CENTER_CONTROL_IMAGE=induforge/control:1.0.0 \
  IF_CENTER_DATA_IMAGE=induforge/data:1.0.0 \
  IF_CENTER_EDGE_IMAGE=induforge/edge:1.0.0 \
  IF_CENTER_DATA_ROOT="$fake_root" \
  IF_CENTER_DOCKER_GID=998 \
    "$SCRIPT_DIR/centerctl" apply "$temp_dir/center.env" >/dev/null
}

upgrade_log="$temp_dir/upgrade.log"
run_fake_apply 1 "$upgrade_log"
stop_line=$(grep -n 'scale deployment/center-control --replicas=0' "$upgrade_log" | cut -d: -f1)
stop_wait_line=$(grep -n 'rollout status deployment/center-control --timeout=180s' "$upgrade_log" | head -n 1 | cut -d: -f1)
clear_line=$(grep -n 'set env deployment/center-control --containers=control CODE_WORKSPACE_ALLOWED_ORIGINS- CENTER_PUBLIC_ORIGIN- WORKSPACE_PUBLIC_ORIGIN_TEMPLATE-' "$upgrade_log" | cut -d: -f1)
apply_line=$(grep -n 'apply -f .*/center-system.yaml' "$upgrade_log" | cut -d: -f1)
bootstrap_line=$(grep -n 'apply -f .*/center-bootstrap-job.yaml' "$upgrade_log" | cut -d: -f1)
start_line=$(grep -n 'scale deployment/center-control --replicas=1' "$upgrade_log" | cut -d: -f1)
if [ -z "$stop_line" ] || [ -z "$stop_wait_line" ] || [ -z "$clear_line" ] || [ -z "$apply_line" ] || [ -z "$bootstrap_line" ] || [ -z "$start_line" ] || \
  [ "$stop_wait_line" -le "$stop_line" ] || [ "$clear_line" -le "$stop_wait_line" ] || [ "$apply_line" -le "$clear_line" ] || \
  [ "$bootstrap_line" -le "$apply_line" ] || [ "$start_line" -le "$bootstrap_line" ]; then
  echo "center upgrade order must be stop, clear legacy env, apply, bootstrap, start" >&2
  exit 1
fi

fresh_log="$temp_dir/fresh.log"
run_fake_apply 0 "$fresh_log"
if grep -Eq 'scale deployment/center-control --replicas=0|set env deployment/center-control' "$fresh_log"; then
  echo "fresh center install must skip the legacy deployment preparation" >&2
  exit 1
fi

# 同名终态 bootstrap Job 必须在新建前精确删除并确认消失；失败 Job 的日志要先保留。
for terminal_state in failed complete; do
  retry_log="$temp_dir/bootstrap-${terminal_state}.log"
  run_fake_apply 1 "$retry_log" "$terminal_state"
  delete_line=$(grep -n 'delete job center-control-bootstrap --wait=true' "$retry_log" | cut -d: -f1)
  delete_wait_line=$(grep -n 'wait --for=delete job/center-control-bootstrap --timeout=60s' "$retry_log" | cut -d: -f1)
  bootstrap_apply_line=$(grep -n 'apply -f .*/center-bootstrap-job.yaml' "$retry_log" | cut -d: -f1)
  if [ -z "$delete_line" ] || [ -z "$delete_wait_line" ] || [ -z "$bootstrap_apply_line" ] || [ "$delete_wait_line" -le "$delete_line" ] || [ "$bootstrap_apply_line" -le "$delete_wait_line" ]; then
    echo "terminal bootstrap job was not deleted and awaited before recreation" >&2
    exit 1
  fi
  if [ "$terminal_state" = failed ]; then
    failed_logs_line=$(grep -n 'logs job/center-control-bootstrap --all-containers=true' "$retry_log" | cut -d: -f1)
    failed_describe_line=$(grep -n 'describe job/center-control-bootstrap' "$retry_log" | cut -d: -f1)
    if [ -z "$failed_logs_line" ] || [ -z "$failed_describe_line" ] || [ "$failed_logs_line" -ge "$delete_line" ] || [ "$failed_describe_line" -ge "$delete_line" ]; then
      echo "failed bootstrap evidence was not retained before deletion" >&2
      exit 1
    fi
  fi
done

active_bootstrap_log="$temp_dir/bootstrap-active.log"
if run_fake_apply 1 "$active_bootstrap_log" active >/dev/null 2>&1; then
  echo "active bootstrap job was incorrectly replaced" >&2
  exit 1
fi
if grep -Eq 'delete job center-control-bootstrap|apply -f .*/center-bootstrap-job.yaml' "$active_bootstrap_log"; then
  echo "active bootstrap job must not be deleted or recreated" >&2
  exit 1
fi

# 已清理旧环境变量的重复 apply 必须继续到 bootstrap；但 kubectl 的真实失败不能吞掉。
empty_legacy_log="$temp_dir/empty-legacy.log"
run_fake_apply 1 "$empty_legacy_log" missing 0 ''
if grep -Fq 'set env deployment/center-control' "$empty_legacy_log"; then
  echo "missing legacy control env must not be removed again" >&2
  exit 1
fi
set_env_failure_log="$temp_dir/set-env-failure.log"
if run_fake_apply 1 "$set_env_failure_log" missing 1 >/dev/null 2>&1; then
  echo "control env removal failure was incorrectly ignored" >&2
  exit 1
fi
if grep -Eq 'apply -f .*/center-system.yaml' "$set_env_failure_log"; then
  echo "apply continued after control env removal failed" >&2
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
