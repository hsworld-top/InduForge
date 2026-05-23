#!/usr/bin/env bash

# InduForge Linux 离线卸载入口。
#
# 输入：
# - 当前目录下的 `.env` 和 `scripts/docker/docker-compose.offline.yml`。
# - 可选参数 `--volumes`，表示同时删除 Docker 数据卷。
#
# 输出：
# - 默认停止并删除 InduForge 容器和网络，保留数据卷。
# - 传入 `--volumes` 时额外删除数据库、缓存、消息和对象存储数据卷。
#
# 安全边界：
# - 本脚本不删除 Docker 镜像 tar，不删除安装包目录。
# - 默认不删除数据卷，避免误删正式业务数据。

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PACKAGE_ROOT="$SCRIPT_DIR"
ENV_FILE="$PACKAGE_ROOT/.env"
COMPOSE_FILE="$PACKAGE_ROOT/scripts/docker/docker-compose.offline.yml"
REMOVE_VOLUMES=false

for arg in "$@"; do
  case "$arg" in
    --volumes)
      REMOVE_VOLUMES=true
      ;;
    *)
      echo "未知参数: $arg" >&2
      echo "用法: ./uninstall.sh [--volumes]" >&2
      exit 1
      ;;
  esac
done

require_command() {
  local command_name="$1"
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "缺少命令: $command_name。请确认 Docker 环境仍可用后再卸载。" >&2
    exit 1
  fi
}

compose_command() {
  # 和安装脚本保持一致，优先使用 Compose v2 插件。
  if docker compose version >/dev/null 2>&1; then
    echo "docker compose"
    return
  fi

  if command -v docker-compose >/dev/null 2>&1; then
    echo "docker-compose"
    return
  fi

  echo "缺少 Docker Compose。请先恢复 Docker Compose 后再卸载。" >&2
  exit 1
}

main() {
  local compose down_args

  require_command docker
  docker info >/dev/null

  if [ ! -f "$ENV_FILE" ]; then
    echo "缺少 .env，卸载将使用 compose 默认值继续执行。"
  fi

  compose="$(compose_command)"
  down_args=(--remove-orphans)

  if [ "$REMOVE_VOLUMES" = true ]; then
    echo "即将删除 InduForge 容器、网络和数据卷。"
    down_args+=(--volumes)
  else
    echo "即将删除 InduForge 容器和网络，数据卷会保留。"
  fi

  if [ -f "$ENV_FILE" ]; then
    # shellcheck disable=SC2086
    $compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" down "${down_args[@]}"
  else
    # shellcheck disable=SC2086
    $compose -f "$COMPOSE_FILE" down "${down_args[@]}"
  fi

  echo "InduForge 卸载流程已完成。"
}

main "$@"
