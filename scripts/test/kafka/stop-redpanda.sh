#!/usr/bin/env bash
set -euo pipefail

CONTAINER_NAME="${CONTAINER_NAME:-induforge-test-redpanda}"

if ! command -v docker >/dev/null 2>&1; then
  echo "未找到 docker。" >&2
  exit 1
fi

if docker ps --format '{{.Names}}' | grep -qx "${CONTAINER_NAME}"; then
  docker stop "${CONTAINER_NAME}" >/dev/null
  echo "已停止 Kafka 测试容器：${CONTAINER_NAME}"
else
  echo "Kafka 测试容器未运行：${CONTAINER_NAME}"
fi
