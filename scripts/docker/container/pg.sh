#!/bin/bash

# InduForge 元数据存储容器启动脚本
# 用于单独调试 InduForge 元数据与时序扩展能力

set -e

CONTAINER_NAME="${CONTAINER_NAME:-induforge-meta-store}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-postgres}"
POSTGRES_USER="${POSTGRES_USER:-postgres}"
POSTGRES_DB="${POSTGRES_DB:-postgres}"
POSTGRES_PORT="${POSTGRES_PORT:-18432}"
POSTGRES_VOLUME="${POSTGRES_VOLUME:-induforge-meta-store-data}"
TIMESCALEDB_DATABASES="${TIMESCALEDB_DATABASES:-postgres,if_dev_data}"
TIMESCALEDB_VERSION="${TIMESCALEDB_VERSION:-2.26.4-pg16}"
IMAGE_REPO="${TIMESCALEDB_IMAGE_REPO:-timescale/timescaledb}"
IMAGE_NAME="${IMAGE_REPO}:${TIMESCALEDB_VERSION}"
DOCKER_REGISTRY_PREFIX="${DOCKER_REGISTRY_PREFIX:-}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
IMAGE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)/images"
IMAGE_TAR="$IMAGE_DIR/timescaledb-${TIMESCALEDB_VERSION}.tar"

if [ -n "$DOCKER_REGISTRY_PREFIX" ]; then
    PULL_IMAGE="${DOCKER_REGISTRY_PREFIX%/}/${IMAGE_NAME}"
else
    PULL_IMAGE="$IMAGE_NAME"
fi

echo "=========================================="
echo "启动 InduForge 元数据存储容器"
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

database_exists() {
    local database="$1"
    local exists

    exists=$(docker exec "$CONTAINER_NAME" psql -U "$POSTGRES_USER" -d postgres -tAc \
        "SELECT 1 FROM pg_database WHERE datname = '$database';")

    [ "$exists" = "1" ]
}

ensure_extensions() {
    # 容器脚本只负责已有库的扩展启用；业务库建库由根目录初始化脚本统一处理。
    IFS="," read -ra databases <<< "$TIMESCALEDB_DATABASES"
    for database in "${databases[@]}"; do
        database="$(echo "$database" | xargs)"
        if [ -z "$database" ]; then
            continue
        fi

        if database_exists "$database"; then
            echo "启用 TimescaleDB 扩展: $database"
            docker exec "$CONTAINER_NAME" psql -U "$POSTGRES_USER" -d "$database" \
                -c "CREATE EXTENSION IF NOT EXISTS timescaledb;"
        else
            echo "跳过不存在的数据库: $database"
        fi
    done
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
        -p "$POSTGRES_PORT":5432 \
        -e POSTGRES_PASSWORD="$POSTGRES_PASSWORD" \
        -e POSTGRES_USER="$POSTGRES_USER" \
        -e POSTGRES_DB="$POSTGRES_DB" \
        -e PGDATA=/var/lib/postgresql/data/pgdata \
        -v "$POSTGRES_VOLUME":/var/lib/postgresql/data \
        --restart unless-stopped \
        "$IMAGE_NAME" \
        -c shared_preload_libraries=timescaledb
fi

echo "等待 TimescaleDB 启动..."
sleep 10
ensure_extensions

echo ""
echo "=========================================="
echo "InduForge 元数据存储容器信息"
echo "=========================================="
echo "容器名称: $CONTAINER_NAME"
echo "镜像: $IMAGE_NAME"
echo "端口映射: $POSTGRES_PORT:5432"
echo "默认数据库: $POSTGRES_DB"
echo "扩展目标库: $TIMESCALEDB_DATABASES"
echo "用户名: $POSTGRES_USER"
echo "密码: $POSTGRES_PASSWORD"
echo "数据卷: $POSTGRES_VOLUME"
echo "离线镜像: $IMAGE_TAR"
echo ""
echo "连接命令:"
echo "  docker exec -it $CONTAINER_NAME psql -U $POSTGRES_USER -d $POSTGRES_DB"
echo ""
echo "查看日志:"
echo "  docker logs -f $CONTAINER_NAME"
echo "=========================================="
