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
# - 执行控制面数据库结构和默认账号初始化。
#
# 边界：
# - 不启动 dev_core、data_service、前端项目等业务进程。
# - 不删除已有容器和数据卷。
# - 如需重置环境，请先由开发人员明确执行 docker compose down --volumes。

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ENV_FILE="$REPO_ROOT/.env"
DEV_ENV_TEMPLATE="$REPO_ROOT/.env.development.example"
COMPOSE_FILE="$REPO_ROOT/scripts/docker/docker-compose.dev.yml"
META_CONTAINER="induforge-meta-store"

require_command() {
  local command_name="$1"
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "缺少命令: $command_name。请先安装或启用对应工具。" >&2
    exit 1
  fi
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

wait_for_meta_store() {
  local user password admin_db
  user="$(read_env IF_META_STORE_USER postgres)"
  password="$(read_env IF_META_STORE_PASSWORD postgres)"
  admin_db="$(read_env IF_META_STORE_ADMIN_DATABASE postgres)"

  echo "等待元数据能力就绪..."
  for _ in $(seq 1 60); do
    if docker exec -e PGPASSWORD="$password" "$META_CONTAINER" \
      psql -U "$user" -d "$admin_db" -tAc "SELECT 1" >/dev/null 2>&1; then
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
  echo "初始化平台数据库和时序扩展..."
  node "$REPO_ROOT/scripts/dev/init-infra.mjs"
}

init_control_schema() {
  echo "初始化控制面数据库结构和默认账号..."
  pnpm --dir "$REPO_ROOT/dev_core" install --frozen-lockfile
  pnpm --dir "$REPO_ROOT/dev_core" db:init
}

main() {
  require_command docker
  require_command node
  require_command pnpm
  docker info >/dev/null

  ensure_env_file
  start_infra
  wait_for_meta_store
  init_databases
  init_control_schema

  echo "开发环境基础设施初始化完成。"
  echo "业务项目不会由本脚本启动，请开发人员按模块自行启动 dev_core、data_service 和前端项目。"
}

main "$@"
