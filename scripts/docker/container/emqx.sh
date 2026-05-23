#!/bin/bash

# InduForge 消息中心容器启动脚本
# 用于单独调试 InduForge 消息接入能力

set -e

CONTAINER_NAME="${CONTAINER_NAME:-induforge-message-hub}"
EMQX_VERSION="${EMQX_VERSION:-5.6.1}"
MQTT_PORT="${MQTT_PORT:-18883}"
MQTT_WS_PORT="${MQTT_WS_PORT:-18083}"
EMQX_DASHBOARD_PORT="${EMQX_DASHBOARD_PORT:-18084}"
EMQX_ADMIN_USER="${EMQX_ADMIN_USER:-admin}"
EMQX_ADMIN_PASSWORD="${EMQX_ADMIN_PASSWORD:-public}"
IMAGE_REPO="${EMQX_IMAGE_REPO:-emqx/emqx}"
IMAGE_NAME="${IMAGE_REPO}:${EMQX_VERSION}"
DOCKER_REGISTRY_PREFIX="${DOCKER_REGISTRY_PREFIX:-}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
IMAGE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)/images"
IMAGE_TAR="$IMAGE_DIR/emqx-${EMQX_VERSION}.tar"

if [ -n "$DOCKER_REGISTRY_PREFIX" ]; then
    PULL_IMAGE="${DOCKER_REGISTRY_PREFIX%/}/${IMAGE_NAME}"
else
    PULL_IMAGE="$IMAGE_NAME"
fi

echo "=========================================="
echo "启动 InduForge 消息中心容器"
echo "=========================================="

ensure_image() {
    mkdir -p "$IMAGE_DIR"

    if docker image inspect "$IMAGE_NAME" >/dev/null 2>&1; then
        echo "镜像已存在: $IMAGE_NAME"
    elif [ -f "$IMAGE_TAR" ]; then
        echo "从离线镜像加载: $IMAGE_TAR"
        docker load -i "$IMAGE_TAR"
    else
        echo "拉取镜像: $PULL_IMAGE"
        docker pull "$PULL_IMAGE"

        if [ "$PULL_IMAGE" != "$IMAGE_NAME" ]; then
            docker tag "$PULL_IMAGE" "$IMAGE_NAME"
        fi
    fi

    if [ ! -f "$IMAGE_TAR" ]; then
        echo "保存离线镜像: $IMAGE_TAR"
        docker save -o "$IMAGE_TAR" "$IMAGE_NAME"
    fi
}

ensure_image

# 检查容器是否已存在
if [ "$(docker ps -aq -f name="^/${CONTAINER_NAME}$")" ]; then
    echo "容器 $CONTAINER_NAME 已存在"
    
    # 检查容器是否正在运行
    if [ "$(docker ps -q -f name="^/${CONTAINER_NAME}$")" ]; then
        echo "容器 $CONTAINER_NAME 正在运行"
        read -p "是否重启容器? (y/n): " restart
        if [ "$restart" = "y" ]; then
            echo "重启容器..."
            docker restart "$CONTAINER_NAME"
        fi
    else
        echo "启动已存在的容器..."
        docker start "$CONTAINER_NAME"
    fi
else
    echo "创建并启动新容器..."
    docker run -d \
        --name "$CONTAINER_NAME" \
        -p "$MQTT_PORT":1883 \
        -p "$MQTT_WS_PORT":8083 \
        -p "$EMQX_DASHBOARD_PORT":18083 \
        -e EMQX_NAME=induforge \
        -e EMQX_HOST=127.0.0.1 \
        -v induforge-message-hub-data:/opt/emqx/data \
        -v induforge-message-hub-log:/opt/emqx/log \
        --restart unless-stopped \
        "$IMAGE_NAME"
    
    echo "等待 EMQX 启动..."
    sleep 10
fi

echo ""
echo "=========================================="
echo "InduForge 消息中心容器信息"
echo "=========================================="
echo "容器名称: $CONTAINER_NAME"
echo "镜像: $IMAGE_NAME"
echo "MQTT 端口: $MQTT_PORT"
echo "WebSocket 端口: $MQTT_WS_PORT"
echo "Dashboard 端口: $EMQX_DASHBOARD_PORT"
echo "管理员账号: $EMQX_ADMIN_USER"
echo "管理员密码: $EMQX_ADMIN_PASSWORD"
echo "数据卷: induforge-message-hub-data, induforge-message-hub-log"
echo "离线镜像: $IMAGE_TAR"
echo ""
echo "Dashboard 访问地址:"
echo "  http://localhost:$EMQX_DASHBOARD_PORT"
echo ""
echo "MQTT 连接信息:"
echo "  TCP: mqtt://localhost:$MQTT_PORT"
echo "  WebSocket: ws://localhost:$MQTT_WS_PORT/mqtt"
echo ""
echo "查看日志:"
echo "  docker logs -f $CONTAINER_NAME"
echo ""
echo "进入容器:"
echo "  docker exec -it $CONTAINER_NAME sh"
echo "=========================================="
