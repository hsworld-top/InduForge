#!/usr/bin/env bash

# InduForge Linux 离线安装入口。
#
# 输入：
# - 当前目录下的 `.env` 或 `.env.production.example`。
# - `scripts/docker/images/*.tar` 中的离线镜像。
# - 已安装并可用的 Docker Engine 与 Docker Compose。
#
# 输出：
# - 导入离线镜像。
# - 使用 Docker Compose 启动 InduForge 生产拓扑。
# - 通过容器内命令执行基础设施初始化。
#
# 重要边界：
# - 本脚本只检测 Docker 环境，不安装 Docker。
# - 若目标机器没有 Docker 或 Compose，会直接退出并提示用户先安装。
# - 初始化只依赖 Docker，不要求目标机器额外安装 Node、pnpm 或 Go。

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PACKAGE_ROOT="$SCRIPT_DIR"
ENV_FILE="$PACKAGE_ROOT/.env"
COMPOSE_FILE="$PACKAGE_ROOT/scripts/docker/docker-compose.offline.yml"
IMAGE_DIR="$PACKAGE_ROOT/scripts/docker/images"
META_CONTAINER="induforge-meta-store"
CONTROL_CONTAINER="induforge-control"

require_command() {
  local command_name="$1"
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "缺少命令: $command_name。请先在目标机器安装 Docker 环境。" >&2
    exit 1
  fi
}

compose_command() {
  # 优先使用 Docker Compose v2 插件；如果目标机仍使用旧版 docker-compose，则自动兼容。
  if docker compose version >/dev/null 2>&1; then
    echo "docker compose"
    return
  fi

  if command -v docker-compose >/dev/null 2>&1; then
    echo "docker-compose"
    return
  fi

  echo "缺少 Docker Compose。请先安装 Docker Compose v2 插件或 docker-compose。" >&2
  exit 1
}

prepare_env_file() {
  if [ -f "$ENV_FILE" ]; then
    return
  fi

  if [ ! -f "$PACKAGE_ROOT/.env.production.example" ]; then
    echo "缺少 .env.production.example，无法生成 .env。" >&2
    exit 1
  fi

  cp "$PACKAGE_ROOT/.env.production.example" "$ENV_FILE"
  echo "已生成 .env，请按生产环境要求修改 change_me_* 密钥后重新执行安装。"
  exit 1
}

read_env() {
  local name="$1"
  local fallback="${2:-}"
  local value

  value="$(grep -E "^${name}=" "$ENV_FILE" | tail -n 1 | cut -d= -f2- || true)"
  if [ -n "$value" ]; then
    printf '%s' "$value"
  else
    printf '%s' "$fallback"
  fi
}

quote_sql_ident() {
  local value="$1"
  if [[ ! "$value" =~ ^[a-zA-Z_][a-zA-Z0-9_]*$ ]]; then
    echo "非法数据库标识符: $value" >&2
    exit 1
  fi

  printf '"%s"' "$value"
}

load_images() {
  if ! compgen -G "$IMAGE_DIR/*.tar" >/dev/null; then
    echo "未找到离线镜像: $IMAGE_DIR/*.tar" >&2
    exit 1
  fi

  for image_tar in "$IMAGE_DIR"/*.tar; do
    echo "导入镜像: $image_tar"
    docker load -i "$image_tar"
  done
}

start_services() {
  local compose
  compose="$(compose_command)"

  # shellcheck disable=SC2086
  $compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d
}

wait_for_meta_store() {
  local user password admin_db port
  user="$(read_env IF_META_STORE_USER postgres)"
  password="$(read_env IF_META_STORE_PASSWORD postgres)"
  admin_db="$(read_env IF_META_STORE_ADMIN_DATABASE postgres)"
  port="$(read_env IF_META_STORE_PORT 18432)"

  echo "等待元数据能力就绪..."
  for _ in $(seq 1 60); do
    if docker exec -e PGPASSWORD="$password" "$META_CONTAINER" \
      psql -p "$port" -U "$user" -d "$admin_db" -tAc "SELECT 1" >/dev/null 2>&1; then
      return
    fi
    sleep 2
  done

  echo "元数据能力启动超时，请检查容器日志: docker logs $META_CONTAINER" >&2
  exit 1
}

create_database_if_needed() {
  local database="$1"
  local user password admin_db port exists quoted

  user="$(read_env IF_META_STORE_USER postgres)"
  password="$(read_env IF_META_STORE_PASSWORD postgres)"
  admin_db="$(read_env IF_META_STORE_ADMIN_DATABASE postgres)"
  port="$(read_env IF_META_STORE_PORT 18432)"
  quoted="$(quote_sql_ident "$database")"

  exists="$(docker exec -e PGPASSWORD="$password" "$META_CONTAINER" \
    psql -p "$port" -U "$user" -d "$admin_db" -tAc "SELECT 1 FROM pg_database WHERE datname = '$database';" | tr -d '[:space:]')"

  if [ "$exists" = "1" ]; then
    echo "数据库已存在: $database"
    return
  fi

  docker exec -e PGPASSWORD="$password" "$META_CONTAINER" \
    psql -p "$port" -U "$user" -d "$admin_db" -c "CREATE DATABASE $quoted;"
  echo "数据库已创建: $database"
}

enable_timeseries_extension() {
  local database="$1"
  local user password port
  local extension_name

  user="$(read_env IF_META_STORE_USER postgres)"
  password="$(read_env IF_META_STORE_PASSWORD postgres)"
  port="$(read_env IF_META_STORE_PORT 18432)"
  extension_name="time""scaledb"

  docker exec -e PGPASSWORD="$password" "$META_CONTAINER" \
    psql -p "$port" -U "$user" -d "$database" -c "CREATE EXTENSION IF NOT EXISTS $extension_name;"
  echo "时序扩展已启用: $database"
}

init_control_schema() {
  # 控制面镜像内已经包含 dev_core 的数据库初始化脚本和生产依赖。
  # 这里在数据库可用后显式执行一次，确保正式安装后核心表和默认管理员数据存在。
  echo "初始化控制面数据库结构..."
  docker exec "$CONTROL_CONTAINER" node scripts/bootstrap/init-core-database.js init
}

init_infra() {
  local core_db data_db dev_data_db

  wait_for_meta_store

  core_db="$(read_env IF_META_STORE_CORE_DB if_core)"
  data_db="$(read_env IF_META_STORE_DATA_DB if_data)"
  dev_data_db="$(read_env IF_META_STORE_DEV_DATA_DB if_dev_data)"

  create_database_if_needed "$core_db"
  create_database_if_needed "$data_db"
  create_database_if_needed "$dev_data_db"
  enable_timeseries_extension "$dev_data_db"
  init_control_schema
}

main() {
  require_command docker
  docker info >/dev/null

  prepare_env_file
  load_images
  start_services
  init_infra

  echo "InduForge 离线安装流程已执行完成。"
  echo "访问入口: http://localhost:$(read_env IF_EDGE_HOST_PORT 18080)"
}

main "$@"
