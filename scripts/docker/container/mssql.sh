#!/bin/bash

# SQL Server 容器启动脚本
# 用于 InduForge 项目的 SQL Server 数据库服务

CONTAINER_NAME="induforge-mssql"
MSSQL_SA_PASSWORD="Sqlserver!"
MSSQL_PORT=1433
MSSQL_VERSION="2022-latest"

echo "=========================================="
echo "启动 SQL Server 容器"
echo "=========================================="

# 检查密码强度（SQL Server 要求强密码）
echo "注意: SQL Server 要求强密码（至少8位，包含大小写字母、数字和特殊字符）"

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
        -p $MSSQL_PORT:1433 \
        -e "ACCEPT_EULA=Y" \
        -e "MSSQL_SA_PASSWORD=$MSSQL_SA_PASSWORD" \
        -e "MSSQL_PID=Developer" \
        -v induforge-mssql-data:/var/opt/mssql \
        --restart unless-stopped \
        mcr.microsoft.com/mssql/server:$MSSQL_VERSION
    
    echo "等待 SQL Server 启动..."
    sleep 15
    
    # 创建数据库
    echo "创建数据库 tenant_management..."
    docker exec $CONTAINER_NAME /opt/mssql-tools18/bin/sqlcmd \
        -S localhost -U sa -P "$MSSQL_SA_PASSWORD" -C \
        -Q "CREATE DATABASE tenant_management;"
    
    if [ $? -eq 0 ]; then
        echo "数据库 tenant_management 创建成功"
    else
        echo "警告: 数据库创建失败，可能已存在或命令执行出错"
    fi
fi

echo ""
echo "=========================================="
echo "SQL Server 容器信息"
echo "=========================================="
echo "容器名称: $CONTAINER_NAME"
echo "端口映射: $MSSQL_PORT:1433"
echo "SA 密码: $MSSQL_SA_PASSWORD"
echo "数据卷: induforge-mssql-data"
echo ""
echo "连接命令:"
echo "  docker exec -it $CONTAINER_NAME /opt/mssql-tools18/bin/sqlcmd -S localhost -U sa -P '$MSSQL_SA_PASSWORD' -C"
echo ""
echo "查看日志:"
echo "  docker logs -f $CONTAINER_NAME"
echo ""
echo "连接字符串示例:"
echo "  Server=localhost,$MSSQL_PORT;Database=tenant_management;User Id=sa;Password=$MSSQL_SA_PASSWORD;TrustServerCertificate=true"
echo "=========================================="
