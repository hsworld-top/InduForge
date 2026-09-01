#!/bin/sh
set -eu

if [ "$(id -u)" -ne 0 ]; then
  echo "migrate-docker-center.sh must run as root" >&2
  exit 1
fi

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
DATA_ROOT=${IF_CENTER_DATA_ROOT:-/var/lib/induforge/center}
CENTER_SOURCE=${IF_CENTER_SOURCE:-/opt/induforge/center}
CENTER_ENV_FILE=${IF_CENTER_ENV_FILE:-/etc/induforge/center.env}
BACKUP_PARENT=${IF_CENTER_BACKUP_ROOT:-/var/backups/induforge}
CENTER_NODE_NAME=${IF_CENTER_NODE_NAME:-}
CONTROL_IMAGE=${IF_CENTER_CONTROL_IMAGE:-induforge/control:latest}
EDGE_IMAGE=${IF_CENTER_EDGE_IMAGE:-induforge/edge:latest}
AGENT_SERVER_URL=${IF_CENTER_AGENT_SERVER_URL:-}
NODE_AGENT_CONFIG=${IF_CENTER_NODE_AGENT_CONFIG:-/etc/induforge/node-agent/config.yaml}
K3S_BIN=${IF_CENTER_K3S_BIN:-/usr/local/bin/k3s}
KUBECONFIG_PATH=${IF_CENTER_KUBECONFIG:-/var/lib/induforge/k3s/server/cred/admin.kubeconfig}
CENTER_CONFIG_FILE=${IF_CENTER_CONFIG_FILE:-/etc/induforge/center-k3s.conf}
HTTP_PORT=${IF_EDGE_HOST_PORT:-18080}
HTTPS_PORT=${IF_EDGE_TLS_HOST_PORT:-18443}
DOCKER_GID=${IF_CENTER_DOCKER_GID:-}

case "$DATA_ROOT" in
  /|/var|/var/lib|*[!a-zA-Z0-9._/-]*) echo "IF_CENTER_DATA_ROOT is unsafe" >&2; exit 1 ;;
  /*) ;;
  *) echo "IF_CENTER_DATA_ROOT must be absolute" >&2; exit 1 ;;
esac
case "$CENTER_NODE_NAME" in
  ''|*[!a-zA-Z0-9._-]*) echo "IF_CENTER_NODE_NAME is missing or invalid" >&2; exit 1 ;;
esac
case "$AGENT_SERVER_URL" in
  https://*:*|https://* ) ;;
  *) echo "IF_CENTER_AGENT_SERVER_URL must be an HTTPS Center URL" >&2; exit 1 ;;
esac
for port in "$HTTP_PORT" "$HTTPS_PORT"; do
  case "$port" in ''|*[!0-9]*) echo "center edge port is invalid" >&2; exit 1 ;; esac
done
if [ -z "$DOCKER_GID" ] && [ -S /var/run/docker.sock ]; then
  DOCKER_GID=$(stat -c '%g' /var/run/docker.sock)
fi
case "$DOCKER_GID" in ''|*[!0-9]*) echo "IF_CENTER_DOCKER_GID is missing or invalid" >&2; exit 1 ;; esac
if [ -e "$DATA_ROOT/.k3s-center-installed" ]; then
  echo "center data was already migrated: $DATA_ROOT" >&2
  exit 1
fi
for required in "$CENTER_SOURCE/dev_core" "$CENTER_SOURCE/ide/index.html" "$CENTER_SOURCE/tls/center.crt" "$CENTER_SOURCE/tls/center.key" "$CENTER_ENV_FILE"; do
  if [ ! -f "$required" ]; then
    echo "required center asset is missing: $required" >&2
    exit 1
  fi
done
for container in induforge-center-meta-store induforge-center-cache-store induforge-center-object-store induforge-center-edge; do
  docker inspect "$container" >/dev/null
done

kubectl_cmd() {
  "$K3S_BIN" kubectl --kubeconfig "$KUBECONFIG_PATH" "$@"
}

volume_source() {
  volume_name=$1
  source_path=$(docker volume inspect -f '{{.Mountpoint}}' "$volume_name")
  case "$source_path" in
    /var/lib/docker/volumes/*/_data) ;;
    *) echo "refusing unexpected Docker volume path: $source_path" >&2; exit 1 ;;
  esac
  if [ -L "$source_path" ] || [ ! -d "$source_path" ]; then
    echo "Docker volume source is not a safe directory: $source_path" >&2
    exit 1
  fi
  printf '%s\n' "$source_path"
}

copy_tree() {
  source_path=$1
  target_path=$2
  install -d -m 0750 "$target_path"
  if [ -n "$(find "$target_path" -mindepth 1 -maxdepth 1 -print -quit)" ]; then
    echo "migration target is not empty: $target_path" >&2
    exit 1
  fi
  cp -a "$source_path/." "$target_path/"
}

rollback_legacy() {
  exit_code=$?
  trap - EXIT INT TERM
  if [ "$exit_code" -ne 0 ]; then
    echo "center migration failed; restoring legacy services" >&2
    kubectl_cmd -n induforge-system scale deployment/center-edge deployment/center-control --replicas=0 >/dev/null 2>&1 || true
    kubectl_cmd -n induforge-system scale statefulset/center-meta-store statefulset/center-cache-store statefulset/center-object-store --replicas=0 >/dev/null 2>&1 || true
    docker start induforge-center-meta-store induforge-center-cache-store induforge-center-object-store induforge-center-edge >/dev/null 2>&1 || true
    systemctl start induforge-center.service >/dev/null 2>&1 || true
  fi
  exit "$exit_code"
}
trap rollback_legacy EXIT INT TERM

timestamp=$(date -u +%Y%m%dT%H%M%SZ)
backup_dir="$BACKUP_PARENT/center-docker-$timestamp"
install -d -m 0700 "$backup_dir"
docker exec induforge-center-meta-store pg_dumpall -U postgres > "$backup_dir/postgres-all.sql"
cp -a "$CENTER_ENV_FILE" "$backup_dir/center.env"
tar -C "$CENTER_SOURCE" -czf "$backup_dir/center-assets.tar.gz" ide tls workspaces nginx.conf

meta_source=$(volume_source induforge-center-meta-store-data)
cache_source=$(volume_source induforge-center-cache-store-data)
object_source=$(volume_source induforge-center-object-store-data)

systemctl stop induforge-center.service
docker stop induforge-center-edge induforge-center-object-store induforge-center-cache-store induforge-center-meta-store >/dev/null

install -d -m 0755 "$DATA_ROOT"
copy_tree "$meta_source" "$DATA_ROOT/meta-store"
copy_tree "$cache_source" "$DATA_ROOT/cache-store"
copy_tree "$object_source" "$DATA_ROOT/object-store"
copy_tree "$CENTER_SOURCE/ide" "$DATA_ROOT/ide"
copy_tree "$CENTER_SOURCE/tls" "$DATA_ROOT/tls"
copy_tree "$CENTER_SOURCE/workspaces" "$DATA_ROOT/workspaces"
chown -R 1000:1000 "$DATA_ROOT/workspaces"
chmod 0755 "$DATA_ROOT" "$DATA_ROOT/ide"
find "$DATA_ROOT/ide" -type d -exec chmod 0755 {} +
find "$DATA_ROOT/ide" -type f -exec chmod 0644 {} +

install -d -m 0755 /opt/induforge/center-k3s
cp -a "$SCRIPT_DIR/." /opt/induforge/center-k3s/
chmod 0755 /opt/induforge/center-k3s/centerctl
chmod 0755 /opt/induforge/center-k3s/induforge
install -m 0755 /opt/induforge/center-k3s/induforge /usr/local/bin/induforge

# 非密钥运行参数独立持久化，确保前端失效后 centerctl 仍可直接诊断和恢复。
install -d -m 0755 "$(dirname -- "$CENTER_CONFIG_FILE")"
cat > "$CENTER_CONFIG_FILE" <<EOF
CENTER_NODE_NAME=$CENTER_NODE_NAME
CONTROL_IMAGE=$CONTROL_IMAGE
EDGE_IMAGE=$EDGE_IMAGE
DATA_ROOT=$DATA_ROOT
HTTP_PORT=$HTTP_PORT
HTTPS_PORT=$HTTPS_PORT
DOCKER_GID=$DOCKER_GID
EOF
chmod 0600 "$CENTER_CONFIG_FILE"

IF_CENTER_NODE_NAME="$CENTER_NODE_NAME" \
IF_CENTER_CONTROL_IMAGE="$CONTROL_IMAGE" \
IF_CENTER_EDGE_IMAGE="$EDGE_IMAGE" \
IF_CENTER_DATA_ROOT="$DATA_ROOT" \
  /opt/induforge/center-k3s/centerctl apply "$CENTER_ENV_FILE"

IF_CENTER_NODE_NAME="$CENTER_NODE_NAME" IF_CENTER_CONTROL_IMAGE="$CONTROL_IMAGE" IF_CENTER_EDGE_IMAGE="$EDGE_IMAGE" IF_CENTER_DATA_ROOT="$DATA_ROOT" \
  /opt/induforge/center-k3s/centerctl doctor

# 中心内置节点也必须经正式 HTTPS 入口访问控制面，不能依赖已经退出的宿主机 18101 端口。
if [ ! -f "$NODE_AGENT_CONFIG" ]; then
  echo "NodeAgent config does not exist: $NODE_AGENT_CONFIG" >&2
  exit 1
fi
cp -a "$NODE_AGENT_CONFIG" "$backup_dir/node-agent-config.yaml"
install -m 0644 "$DATA_ROOT/tls/center.crt" /usr/local/share/ca-certificates/induforge-center.crt
update-ca-certificates >/dev/null
sed -i -E "s|^([[:space:]]*serverUrl:).*|\\1 $AGENT_SERVER_URL|" "$NODE_AGENT_CONFIG"
if ! grep -Fq "serverUrl: $AGENT_SERVER_URL" "$NODE_AGENT_CONFIG"; then
  echo "failed to update center NodeAgent serverUrl" >&2
  exit 1
fi
systemctl restart induforge-node-agent.service

systemctl disable induforge-center.service >/dev/null
docker update --restart=no induforge-center-meta-store induforge-center-cache-store induforge-center-object-store induforge-center-edge >/dev/null
printf '%s\n' "$timestamp" > "$DATA_ROOT/.k3s-center-installed"
chmod 0600 "$DATA_ROOT/.k3s-center-installed"

trap - EXIT INT TERM
echo "center migration completed; legacy Docker volumes and backup remain available at $backup_dir"
