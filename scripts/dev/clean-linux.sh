#!/usr/bin/env bash

# InduForge Linux 开发基础设施清理入口。
#
# 输入：
# - 仓库根目录 `.env`，用于确定 Docker Compose 变量。
# - scripts/docker/docker-compose.dev.yml，限定清理范围。
#
# 输出：
# - 停止并删除开发基础设施容器。
# - 删除开发基础设施命名卷和网络。
#
# 异常处理：
# - Docker 或 Compose 不可用时直接失败。
# - 默认需要交互确认；自动化场景可传入 --yes。

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ENV_FILE="$REPO_ROOT/.env"
COMPOSE_FILE="$REPO_ROOT/scripts/docker/docker-compose.dev.yml"
ASSUME_YES=false

usage() {
  cat <<'USAGE'
用法: ./scripts/dev/clean-linux.sh [--yes]

默认行为：
  停止并删除 InduForge 开发基础设施容器、网络和数据卷。

选项：
  --yes, -y   跳过确认，直接清理。
  --help, -h  显示帮助。
USAGE
}

parse_args() {
  for arg in "$@"; do
    case "$arg" in
      --yes|-y)
        ASSUME_YES=true
        ;;
      --help|-h)
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

compose_command() {
  # 与 init-linux.sh 保持一致，优先使用 Docker Compose v2 插件。
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

confirm_cleanup() {
  if [ "$ASSUME_YES" = true ]; then
    return
  fi

  cat <<'NOTICE'
即将删除 InduForge 开发基础设施数据：
- PostgreSQL/TimescaleDB 开发数据库数据卷
- Redis 开发缓存数据卷
- EMQX/MQTT 开发消息数据卷
- SeaweedFS 开发对象存储数据卷

不会删除源码、.env、node_modules 或其他非本 compose 管理的 Docker 资源。
NOTICE

  read -r -p "确认继续？输入 yes 执行清理: " answer
  if [ "$answer" != "yes" ]; then
    echo "已取消清理。"
    exit 0
  fi
}

clean_infra() {
  local compose
  compose="$(compose_command)"

  echo "停止并删除 InduForge 开发基础设施容器、网络和数据卷..."
  if [ -f "$ENV_FILE" ]; then
    # shellcheck disable=SC2086
    $compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" down --volumes --remove-orphans
  else
    # shellcheck disable=SC2086
    $compose -f "$COMPOSE_FILE" down --volumes --remove-orphans
  fi
  echo "开发基础设施清理完成。"
}

main() {
  parse_args "$@"
  require_command docker
  docker info >/dev/null

  if [ ! -f "$COMPOSE_FILE" ]; then
    echo "缺少开发基础设施 compose 文件: $COMPOSE_FILE" >&2
    exit 1
  fi

  confirm_cleanup
  clean_infra
}

main "$@"
