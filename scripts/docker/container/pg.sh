#!/bin/bash

# PostgreSQL 容器启动脚本
# 用于 InduForge 项目的 PostgreSQL 数据库服务

CONTAINER_NAME="induforge-postgres"
POSTGRES_PASSWORD="postgres"
POSTGRES_USER="postgres"
POSTGRES_DB="tenant_management"
POSTGRES_PORT=5432
POSTGRES_VERSION="16"

echo "=========================================="
echo "启动 PostgreSQL 容器"
echo "=========================================="

# 检查容器是否已存在
if [ "$(docker ps -aq -f name=$CONTAINER_NAME)" ]; then
    echo "容器 $CONTAINER_NAME 已存在"
    
    # 检查容器是否正在运行
    if [ "$(docker ps -q -f name=$CONTAINER_NAME)" ]; then
        echo "容器 $CONTAINER_NAME 正在运行"
        read -p "是否重启容器? (y/n): " restart
        if [ "$restart" = "y" ]; then
            echo "重启容器..."
            docker restart $CONTAINER_NAME
        fi
    else
        echo "启动已存在的容器..."
        docker start $CONTAINER_NAME
    fi
else
    echo "创建并启动新容器..."
    docker run -d \
        --name $CONTAINER_NAME \
        -p $POSTGRES_PORT:5432 \
        -e POSTGRES_PASSWORD=$POSTGRES_PASSWORD \
        -e POSTGRES_USER=$POSTGRES_USER \
        -e POSTGRES_DB=$POSTGRES_DB \
        -e PGDATA=/var/lib/postgresql/data/pgdata \
        -v induforge-postgres-data:/var/lib/postgresql/data \
        --restart unless-stopped \
        postgres:$POSTGRES_VERSION
    
    echo "等待 PostgreSQL 启动..."
    sleep 10
fi

echo ""
echo "=========================================="
echo "PostgreSQL 容器信息"
echo "=========================================="
echo "容器名称: $CONTAINER_NAME"
echo "端口映射: $POSTGRES_PORT:5432"
echo "数据库名: $POSTGRES_DB"
echo "用户名: $POSTGRES_USER"
echo "密码: $POSTGRES_PASSWORD"
echo "数据卷: induforge-postgres-data"
echo ""
echo "连接命令:"
echo "  docker exec -it $CONTAINER_NAME psql -U $POSTGRES_USER -d $POSTGRES_DB"
echo ""
echo "查看日志:"
echo "  docker logs -f $CONTAINER_NAME"
echo "=========================================="
