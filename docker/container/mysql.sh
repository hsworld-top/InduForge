#!/bin/bash

# MySQL 容器启动脚本
# 用于 InduForge 项目的 MySQL 数据库服务

CONTAINER_NAME="induforge-mysql"
MYSQL_ROOT_PASSWORD="zhu0y1zh1nengF888"
MYSQL_DATABASE="tenant_management"
MYSQL_PORT=3306
MYSQL_VERSION="8.0"

echo "=========================================="
echo "启动 MySQL 容器"
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
        -p $MYSQL_PORT:3306 \
        -e MYSQL_ROOT_PASSWORD=$MYSQL_ROOT_PASSWORD \
        -e MYSQL_DATABASE=$MYSQL_DATABASE \
        -e MYSQL_ROOT_HOST=% \
        -v induforge-mysql-data:/var/lib/mysql \
        --restart unless-stopped \
        mysql:$MYSQL_VERSION \
        --character-set-server=utf8mb4 \
        --collation-server=utf8mb4_unicode_ci \
        --default-authentication-plugin=mysql_native_password
    
    echo "等待 MySQL 启动..."
    sleep 10
fi

echo ""
echo "=========================================="
echo "MySQL 容器信息"
echo "=========================================="
echo "容器名称: $CONTAINER_NAME"
echo "端口映射: $MYSQL_PORT:3306"
echo "数据库名: $MYSQL_DATABASE"
echo "Root 密码: $MYSQL_ROOT_PASSWORD"
echo "数据卷: induforge-mysql-data"
echo ""
echo "连接命令:"
echo "  docker exec -it $CONTAINER_NAME mysql -uroot -p$MYSQL_ROOT_PASSWORD"
echo ""
echo "查看日志:"
echo "  docker logs -f $CONTAINER_NAME"
echo "=========================================="
