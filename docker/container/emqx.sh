#!/bin/bash

# EMQX 容器启动脚本
# 用于 InduForge 项目的 MQTT 消息服务（用于实时数据订阅）

CONTAINER_NAME="induforge-emqx"
EMQX_VERSION="5.6"
MQTT_PORT=1883
MQTT_WS_PORT=8083
EMQX_DASHBOARD_PORT=18083
EMQX_ADMIN_USER="admin"
EMQX_ADMIN_PASSWORD="public"

echo "=========================================="
echo "启动 EMQX 容器"
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
        -p $MQTT_PORT:1883 \
        -p $MQTT_WS_PORT:8083 \
        -p $EMQX_DASHBOARD_PORT:18083 \
        -e EMQX_NAME=induforge \
        -e EMQX_HOST=127.0.0.1 \
        -v induforge-emqx-data:/opt/emqx/data \
        -v induforge-emqx-log:/opt/emqx/log \
        --restart unless-stopped \
        emqx/emqx:$EMQX_VERSION
    
    echo "等待 EMQX 启动..."
    sleep 10
fi

echo ""
echo "=========================================="
echo "EMQX 容器信息"
echo "=========================================="
echo "容器名称: $CONTAINER_NAME"
echo "MQTT 端口: $MQTT_PORT"
echo "WebSocket 端口: $MQTT_WS_PORT"
echo "Dashboard 端口: $EMQX_DASHBOARD_PORT"
echo "管理员账号: $EMQX_ADMIN_USER"
echo "管理员密码: $EMQX_ADMIN_PASSWORD"
echo "数据卷: induforge-emqx-data, induforge-emqx-log"
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
