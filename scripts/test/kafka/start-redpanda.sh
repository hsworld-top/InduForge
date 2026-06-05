#!/usr/bin/env bash
set -euo pipefail

CONTAINER_NAME="${CONTAINER_NAME:-induforge-test-redpanda}"
IMAGE="${IMAGE:-docker.redpanda.com/redpandadata/redpanda:v24.3.6}"
HOST_KAFKA_PORT="${HOST_KAFKA_PORT:-19092}"
HOST_ADMIN_PORT="${HOST_ADMIN_PORT:-19644}"

if ! command -v docker >/dev/null 2>&1; then
  echo "未找到 docker，请先在 WSL Ubuntu 中安装 Docker 或启用 Docker Desktop WSL 集成。" >&2
  exit 1
fi

if docker ps --format '{{.Names}}' | grep -qx "${CONTAINER_NAME}"; then
  echo "Kafka 测试容器已运行：${CONTAINER_NAME}"
else
  if docker ps -a --format '{{.Names}}' | grep -qx "${CONTAINER_NAME}"; then
    echo "启动已有 Kafka 测试容器：${CONTAINER_NAME}"
    docker start "${CONTAINER_NAME}" >/dev/null
  else
    echo "创建 Kafka 测试容器：${CONTAINER_NAME}"
    docker run -d \
      --name "${CONTAINER_NAME}" \
      -p "${HOST_KAFKA_PORT}:19092" \
      -p "${HOST_ADMIN_PORT}:9644" \
      "${IMAGE}" \
      redpanda start \
        --overprovisioned \
        --smp 1 \
        --memory 512M \
        --reserve-memory 0M \
        --node-id 0 \
        --check=false \
        --kafka-addr internal://0.0.0.0:9092,external://0.0.0.0:19092 \
        --advertise-kafka-addr internal://127.0.0.1:9092,external://127.0.0.1:${HOST_KAFKA_PORT} >/dev/null
  fi
fi

echo "等待 Kafka 就绪..."
for _ in $(seq 1 60); do
  if docker exec "${CONTAINER_NAME}" rpk cluster info --brokers 127.0.0.1:9092 >/dev/null 2>&1; then
    echo "Kafka 已就绪。"
    echo "数据中心 Kafka 接入源服务器地址：127.0.0.1:${HOST_KAFKA_PORT}"
    echo "容器内脚本 broker：127.0.0.1:9092"
    exit 0
  fi
  sleep 1
done

echo "Kafka 容器启动超时，请执行 docker logs ${CONTAINER_NAME} 查看原因。" >&2
exit 1
