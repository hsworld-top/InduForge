# NodeAgent Makefile

.PHONY: build build-windows build-linux run dev clean test help

# 默认目标
.DEFAULT_GOAL := help

# 变量
BINARY_NAME := node_agent
BUILD_DIR := build
MAIN_FILE := cmd/main.go
CONFIG_FILE := configs/config.yaml

# Go 变量
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOMOD := $(GOCMD) mod
GOFMT := gofmt

# 版本号
VERSION := 1.0.0

# Windows 构建（带控制台，用于所有场景）
build-windows: ## 构建 Windows 控制台版本
	@echo "构建 Windows 版本..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME).exe $(MAIN_FILE)
	@echo "构建完成: $(BUILD_DIR)/$(BINARY_NAME).exe"

# Linux 构建
build-linux: ## 构建 Linux 版本
	@echo "构建 Linux 版本..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GOBUILD) -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)_linux $(MAIN_FILE)
	@echo "构建完成: $(BUILD_DIR)/$(BINARY_NAME)_linux"

# 构建项目（默认 Windows）
build: build-windows ## 构建项目（默认 Windows）

# 运行项目
run: build ## 构建并运行
	@echo "启动 NodeAgent..."
	@mkdir -p ./data
	@mkdir -p ./runtime
	@mkdir -p ./logs
	./$(BUILD_DIR)/$(BINARY_NAME).exe

# 直接运行（不构建）
dev: ## 直接运行（不构建）
	@echo "启动 NodeAgent..."
	@mkdir -p ./data
	@mkdir -p ./runtime
	@mkdir -p ./logs
	$(GOCMD) run $(MAIN_FILE)
	@mkdir -p ./data
	@mkdir -p ./runtime
	@mkdir -p ./logs
	$(GOCMD) run $(MAIN_FILE)

# 清理构建文件
clean: ## 清理构建文件
	@echo "清理构建文件..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@echo "清理完成"

# 下载依赖
deps: ## 下载依赖
	@echo "下载依赖..."
	$(GOMOD) tidy
	@echo "依赖下载完成"

# 运行测试
test: ## 运行测试
	@echo "运行测试..."
	$(GOTEST) -v ./...
	@echo "测试完成"

# 运行测试并生成覆盖率报告
test-coverage: ## 运行测试并生成覆盖率报告
	@echo "运行测试并生成覆盖率报告..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	@echo "覆盖率报告已生成: coverage.out"
	@if command -v go tool >/dev/null 2>&1 && command -v cover >/dev/null 2>&1; then \
		go tool cover -func=coverage.out; \
		echo ""; \
		echo "HTML 报告: go tool cover -html=coverage.out -o coverage.html"; \
	else \
		echo "使用 go tool cover 查看详细报告"; \
	fi

# 运行特定包的测试
test-pkg: ## 运行特定包的测试 (用法: make test-pkg PKG=./internal/pkg/config)
	@echo "运行 $(PKG) 测试..."
	$(GOTEST) -v $(PKG)
	@echo "测试完成"

# 运行基准测试
test-bench: ## 运行基准测试
	@echo "运行基准测试..."
	$(GOTEST) -bench=. -benchmem ./...
	@echo "基准测试完成"

# 运行测试并检查竞争条件
test-race: ## 运行测试并检查竞争条件
	@echo "运行测试（竞争检测）..."
	$(GOTEST) -race -v ./...
	@echo "测试完成"

# 格式化代码
fmt: ## 格式化代码
	@echo "格式化代码..."
	$(GOFMT) -s -w .
	@echo "格式化完成"

# 检查代码
vet: ## 检查代码
	@echo "检查代码..."
	$(GOCMD) vet ./...
	@echo "检查完成"

# 构建 Docker 镜像
docker-build: ## 构建 Docker 镜像
	@echo "构建 Docker 镜像..."
	docker build -t node-agent:latest .
	@echo "Docker 镜像构建完成"

# 格式化检查
lint: ## 代码质量检查
	@echo "代码质量检查..."
	@command -v golangci-lint >/dev/null 2>&1 || (echo "请先安装 golangci-lint" && exit 1)
	golangci-lint run
	@echo "检查完成"

# 生成依赖图
deps-graph: ## 生成依赖图
	@echo "生成依赖图..."
	@command -v go-modiff >/dev/null 2>&1 || (echo "请先安装 go-modiff" && exit 1)
	go-modiff modules
	@echo "依赖图生成完成"

# 安装到系统
install: build ## 安装到系统
	@echo "安装 NodeAgent..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@sudo mkdir -p /etc/node_agent
	@sudo cp $(CONFIG_FILE) /etc/node_agent/config.yaml
	@echo "安装完成"

# 卸载
uninstall: ## 从系统卸载
	@echo "卸载 NodeAgent..."
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@sudo rm -rf /etc/node_agent
	@echo "卸载完成"

# 创建符号链接
symlink: ## 创建符号链接
	@echo "创建符号链接..."
	@sudo ln -sf $(shell pwd)/$(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/node_agent
	@echo "符号链接创建完成"

# 检查依赖
check-deps: ## 检查依赖是否安装
	@echo "检查依赖..."
	@command -v $(GOCMD) >/dev/null 2>&1 || (echo "Go 未安装" && exit 1)
	@echo "Go 版本: $(shell go version)"
	@echo "依赖检查完成"

# 显示帮助信息
help: ## 显示帮助信息
	@echo "NodeAgent Makefile"
	@echo ""
	@echo "可用命令:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "构建命令:"
	@echo "  make build              # 构建 Windows GUI 版本（生产）"
	@echo "  make build-windows      # 构建 Windows GUI 版本"
	@echo "  make build-windows-console  # 构建 Windows 控制台版本（开发）"
	@echo "  make build-linux        # 构建 Linux 版本"
	@echo ""
	@echo "运行命令:"
	@echo "  make run                # 构建并运行（开发模式）"
	@echo "  make dev                # 直接运行（开发模式）"
	@echo ""
	@echo "示例:"
	@echo "  make build              # 生产构建"
	@echo "  make run                # 开发运行（带控制台）"
	@echo "  make dev                # 直接运行（不构建）"
