#!/usr/bin/env bash

# InduForge Linux 开发环境初始化入口。
#
# 面向对象：
# - 开发人员在 Linux 机器上准备本地开发依赖。
#
# 输入：
# - 仓库根目录 `.env`。如果不存在，会从 `.env.development.example` 自动复制。
# - Docker 与 Docker Compose。脚本只检测，不安装 Docker。
#
# 输出：
# - 启动开发所需基础设施容器。
# - 创建平台数据库。
# - 启用开发态时序扩展。
# - 默认不安装 Node 依赖，也不启动业务项目。
#
# 边界：
# - 不启动 dev_core、data_service、前端项目等业务进程。
# - 默认不执行 pnpm install，避免 Windows + WSL 共用工作区时产生跨系统依赖问题。
# - 不删除已有容器和数据卷。
# - 如需重置环境，请先由开发人员明确执行 docker compose down --volumes。

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ENV_FILE="$REPO_ROOT/.env"
DEV_ENV_TEMPLATE="$REPO_ROOT/.env.development.example"
COMPOSE_FILE="$REPO_ROOT/scripts/docker/docker-compose.dev.yml"
META_CONTAINER="induforge-meta-store"
META_STORE_RETRY_COUNT=5
META_STORE_RETRY_DELAY_SEC=3
META_STORE_SETTLE_DELAY_SEC=3

usage() {
  cat <<'USAGE'
用法: ./scripts/dev/init-linux.sh

默认行为：
  启动 Docker 开发基础设施，创建平台数据库，并启用开发态时序扩展。
USAGE
}

parse_args() {
  for arg in "$@"; do
    case "$arg" in
      -h|--help)
        usage
        exit 0
        ;;
      *)
        echo "未知参数: $arg" >&2
        usage >&2
        exit 1
        ;;
    esac
  done
}

require_command() {
  local command_name="$1"
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "缺少命令: $command_name。请先安装或启用对应工具。" >&2
    exit 1
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

compose_command() {
  # 优先使用 Docker Compose v2 插件，兼容旧版 docker-compose。
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

ensure_env_file() {
  if [ -f "$ENV_FILE" ]; then
    return
  fi

  if [ ! -f "$DEV_ENV_TEMPLATE" ]; then
    echo "缺少开发环境模板: $DEV_ENV_TEMPLATE" >&2
    exit 1
  fi

  cp "$DEV_ENV_TEMPLATE" "$ENV_FILE"
  echo "已从 .env.development.example 生成 .env。"
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

meta_store_psql() {
  local user password admin_db database
  user="$(read_env IF_META_STORE_USER postgres)"
  password="$(read_env IF_META_STORE_PASSWORD postgres)"
  admin_db="$(read_env IF_META_STORE_ADMIN_DATABASE postgres)"
  database="$admin_db"

  if [ $# -gt 0 ] && [[ ! "$1" =~ ^- ]]; then
    database="$1"
    shift
  fi

  docker exec -e PGPASSWORD="$password" "$META_CONTAINER" \
    psql -U "$user" -d "$database" "$@"
}

with_meta_store_retry() {
  local label attempt
  label="${1:-元数据操作}"
  shift

  for attempt in $(seq 1 "$META_STORE_RETRY_COUNT"); do
    if "$@"; then
      return 0
    fi
    if [ "$attempt" -lt "$META_STORE_RETRY_COUNT" ]; then
      echo "warning: ${label}失败，${META_STORE_RETRY_DELAY_SEC} 秒后重试 (${attempt}/${META_STORE_RETRY_COUNT})..." >&2
      sleep "$META_STORE_RETRY_DELAY_SEC"
    fi
  done

  echo "error: ${label}在 ${META_STORE_RETRY_COUNT} 次尝试后仍然失败" >&2
  return 1
}

wait_for_meta_store() {
  echo "等待元数据能力就绪..."
  for _ in $(seq 1 60); do
    if meta_store_psql -tAc "SELECT 1" >/dev/null 2>&1 \
      && meta_store_psql -tAc "SELECT count(*) FROM pg_database" >/dev/null 2>&1; then
      echo "元数据能力已就绪，等待服务稳定..."
      sleep "$META_STORE_SETTLE_DELAY_SEC"
      return
    fi
    sleep 2
  done

  echo "元数据能力启动超时，请检查容器日志: docker logs $META_CONTAINER" >&2
  exit 1
}

start_infra() {
  local compose
  compose="$(compose_command)"

  echo "启动 InduForge 开发基础设施容器..."
  # shellcheck disable=SC2086
  $compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d
}

init_databases() {
  local core_db data_db dev_data_db

  echo "初始化平台数据库和时序扩展..."
  core_db="$(read_env IF_META_STORE_CORE_DB if_core)"
  data_db="$(read_env IF_META_STORE_DATA_DB if_data)"
  dev_data_db="$(read_env IF_META_STORE_DEV_DATA_DB if_dev_data)"

  create_database_if_needed "$core_db"
  create_database_if_needed "$data_db"
  create_database_if_needed "$dev_data_db"
  enable_timeseries_extension "$dev_data_db"
  check_optional_infra_ports
}

create_database_if_needed() {
  local database="$1"
  local exists quoted

  quoted="$(quote_sql_ident "$database")"

  with_meta_store_retry "检查或创建数据库 ${database}" create_database_once "$database" "$quoted"
}

create_database_once() {
  local database="$1"
  local quoted="$2"
  local exists

  exists="$(meta_store_psql -tAc "SELECT 1 FROM pg_database WHERE datname = '$database';" | tr -d '[:space:]')"

  if [ "$exists" = "1" ]; then
    echo "✓ 数据库已存在: $database"
    return 0
  fi

  meta_store_psql -c "CREATE DATABASE $quoted;" >/dev/null
  echo "✓ 数据库已创建: $database"
}

enable_timeseries_extension() {
  local database="$1"
  local extension_name

  extension_name="time""scaledb"
  with_meta_store_retry "启用 ${database} 时序扩展" enable_timeseries_extension_once "$database" "$extension_name"
}

enable_timeseries_extension_once() {
  local database="$1"
  local extension_name="$2"

  meta_store_psql "$database" -c "CREATE EXTENSION IF NOT EXISTS $extension_name;" >/dev/null
  echo "✓ ${database} 已启用扩展: $extension_name"
}

tcp_check() {
  local host="$1"
  local port="$2"
  local label="$3"

  if timeout 3 bash -c "cat < /dev/null > /dev/tcp/$host/$port" >/dev/null 2>&1; then
    echo "✓ ${label} 可连接 ${host}:${port}"
    return
  fi

  echo "! ${label} 不可连接 ${host}:${port}"
}

check_optional_infra_ports() {
  tcp_check "$(read_env IF_CACHE_STORE_HOST 127.0.0.1)" "$(read_env IF_CACHE_STORE_PORT 18379)" "缓存能力"
  tcp_check "$(read_env IF_MESSAGE_HUB_HOST 127.0.0.1)" "$(read_env IF_MESSAGE_HUB_MQTT_PORT 18883)" "消息接入能力"
  tcp_check "$(read_env IF_OBJECT_STORE_ENDPOINT 127.0.0.1)" "$(read_env IF_OBJECT_STORE_PORT 18500)" "对象存储能力"
}

main() {
  parse_args "$@"

  require_command docker
  docker info >/dev/null

  ensure_env_file
  start_infra
  wait_for_meta_store
  init_databases

  echo "开发环境基础设施初始化完成。"
  echo "dev_core 启动时会根据 DB_AUTO_SCHEMA_SYNC 自动同步 if_core 表结构和初始数据。"
  echo "业务项目不会由本脚本启动，请开发人员按模块自行启动 dev_core、data_service 和前端项目。"
}

main "$@"
