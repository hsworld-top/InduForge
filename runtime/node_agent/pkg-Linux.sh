#!/bin/bash

# NodeAgent Linux 打包脚本

echo "============================================================"
echo "  NodeAgent 打包脚本"
echo "============================================================"
echo ""

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "工作目录: $SCRIPT_DIR"
echo ""

# 检查 Go 是否安装
if ! command -v go &> /dev/null; then
    echo "错误: Go 未安装或未在 PATH 中"
    echo "请先安装 Go: https://golang.org/dl/"
    echo ""
    read -p "按 Enter 退出..."
    exit 1
fi

echo "Go 版本:"
go version
echo ""

# 设置静态目录
STATIC_DIR="$SCRIPT_DIR/internal/web/static/dist"
BUILD_STATIC_DIR="$SCRIPT_DIR/build/internal/web/static/dist"

# 创建构建目录
mkdir -p build

# 如果存在前端构建产物，复制到 build 目录，方便随二进制一起分发
if [ -d "$STATIC_DIR" ]; then
    mkdir -p "$BUILD_STATIC_DIR"
    cp -r "$STATIC_DIR/." "$BUILD_STATIC_DIR/"
fi

echo "============================================================"
echo "  构建 Linux 版本"
echo "============================================================"
echo ""

# 清理旧的构建产物
rm -f build/node_agent_linux
rm -f build/node_agent_linux.pdb

# 构建
echo "正在构建 Linux 版本..."
CGO_ENABLED=0 go build -ldflags "-X main.version=1.0.0" -o build/node_agent_linux ./cmd

if [ $? -ne 0 ]; then
    echo ""
    echo "错误: 构建失败！"
    echo ""
    read -p "按 Enter 退出..."
    exit 1
fi

echo ""
echo "构建成功！"
echo ""

# 显示构建产物
if [ -f "build/node_agent_linux" ]; then
    echo "构建产物:"
    echo "  - node_agent_linux"
    SIZE=$(du -h "build/node_agent_linux" | cut -f1)
    echo "    大小: $SIZE"
    echo "    权限: $(stat -c %a build/node_agent_linux)"
fi

echo ""
echo "============================================================"
echo "  打包完成！"
echo "============================================================"
echo ""
echo "下一步:"
echo "  1. 复制 build/node_agent_linux 到目标机器"
echo "  2. 复制 configs/config.yaml 到同目录"
echo "  3. 添加执行权限: chmod +x node_agent_linux"
echo "  4. 运行: ./node_agent_linux"
echo ""
read -p "按 Enter 退出..."
