# NodeAgent Front Makefile

.PHONY: install dev build preview clean help

# 默认目标
.DEFAULT_GOAL := help

# 变量
NODE_VERSION := $(shell node --version 2>/dev/null || echo "not installed")
PNPM_VERSION := $(shell pnpm --version 2>/dev/null || echo "not installed")
DIST_DIR := dist

# 检查依赖
check-deps: ## 检查依赖是否安装
	@echo "检查依赖..."
	@echo "Node.js 版本: $(NODE_VERSION)"
	@echo "pnpm 版本: $(PNPM_VERSION)"
	@if ! command -v node >/dev/null 2>&1; then \
		echo "错误: 请先安装 Node.js"; \
		exit 1; \
	fi
	@if ! command -v pnpm >/dev/null 2>&1; then \
		echo "错误: 请先安装 pnpm"; \
		exit 1; \
	fi
	@echo "依赖检查完成"

# 安装依赖
install: check-deps ## 安装依赖
	@echo "安装依赖..."
	pnpm install
	@echo "依赖安装完成"

# 开发模式
dev: install ## 启动开发服务器
	@echo "启动开发服务器..."
	pnpm dev

# 构建
build: install ## 构建生产版本
	@echo "构建生产版本..."
	pnpm build
	@echo "构建完成，产物在 $(DIST_DIR)/ 目录"

# 预览
preview: build ## 预览构建结果
	@echo "预览构建结果..."
	pnpm preview

# 清理
clean: ## 清理构建文件
	@echo "清理构建文件..."
	rm -rf $(DIST_DIR)
	rm -rf node_modules
	@echo "清理完成"

# 代码格式化
fmt: ## 格式化代码
	@echo "格式化代码..."
	pnpm exec prettier --write "src/**/*.{js,vue,css,scss}"
	@echo "格式化完成"

# 代码检查
lint: install ## 代码质量检查
	@echo "代码质量检查..."
	pnpm exec eslint "src/**/*.{js,vue}"
	@echo "检查完成"

# 打包分析
analyze: install ## 分析打包结果
	@echo "分析打包结果..."
	pnpm build --mode analyze
	@echo "分析完成"

# 类型检查
type-check: install ## TypeScript 类型检查（如果有）
	@echo "TypeScript 类型检查..."
	pnpm exec vue-tsc --noEmit
	@echo "检查完成"

# 创建符号链接到全局
link: build ## 创建符号链接（可选）
	@echo "创建符号链接..."
	@if [ -d "$(DIST_DIR)" ]; then \
		ln -sf $(shell pwd)/$(DIST_DIR) /tmp/node_agent_front_dist; \
		echo "符号链接创建完成: /tmp/node_agent_front_dist"; \
	else \
		echo "错误: 请先运行 make build"; \
		exit 1; \
	fi

# 显示帮助信息
help: ## 显示帮助信息
	@echo "NodeAgent Front Makefile"
	@echo ""
	@echo "可用命令:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "示例:"
	@echo "  make install   # 安装依赖"
	@echo "  make dev      # 启动开发服务器"
	@echo "  make build    # 构建生产版本"
	@echo "  make clean    # 清理文件"
