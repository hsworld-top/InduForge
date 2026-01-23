# NodeAgent 文档索引

## 快速导航

### 我要了解 NodeAgent 系统整体架构

📖 **[系统架构文档](./README_nodeagent.md)** - 系统整体架构和组件说明

### 我要开发或部署 NodeAgent 后端

📖 **[NodeAgent 后端文档](./node_agent/README.md)** - Go 后端服务详细功能
- 首次启动与初始化
- 运行模式（在线/离线）
- 执行器系统（进程、Docker）
- 状态机编排
- 健康检查
- 心跳上报
- API 接口
- 配置管理
- 日志系统

📖 **[初始化流程详解](./node_agent/初始化流程.md)** - 节点注册与审批流程
- 在线模式注册认证流程
- 离线模式配置流程
- 审批流程说明
- API 接口详解
- 安全考虑
- 常见问题

### 我要使用 NodeAgent 前端管理界面

📖 **[NodeAgent Front 文档](./node_agent_front/README.md)** - Vue 前端管理界面详解
- Dashboard（节点状态总览）
- Projects（项目管理）
- Profile（连接配置）
- Logs（日志查看）
- 技术栈说明
- 开发指南

## 目录结构

```
docs/
├── README.md                           # 总体文档入口
├── README_nodeagent.md                 # ⭐ NodeAgent 系统架构文档
├── node_agent/                         # NodeAgent 后端文档
│   └── README.md                      # Go 后端详细功能文档
└── node_agent_front/                   # NodeAgent 前端文档
    └── README.md                      # Vue 前端详细功能文档
```

## 相关代码位置

### 后端代码
```
runtime/node_agent/
├── cmd/main.go                    # 主程序入口
├── configs/config.yaml           # 配置文件
├── internal/
│   ├── agent/                    # 核心 Agent 逻辑
│   │   ├── orchestrator/        # 状态机编排器
│   │   ├── executor/            # 执行器（进程、Docker）
│   │   ├── store/              # 本地存储
│   │   ├── health/             # 健康检查
│   │   └── network/            # 心跳上报
│   ├── pkg/                     # 公共包
│   │   ├── types/              # 类型定义
│   │   ├── logger/             # 日志系统
│   │   └── utils/              # 工具函数
│   └── web/handler/             # HTTP API
├── Dockerfile                    # Docker 构建
├── Makefile                     # 构建工具
└── README.md                    # 后端文档
```

### 前端代码
```
runtime/node_agent_front/
├── src/
│   ├── views/                   # 页面组件
│   │   ├── Dashboard.vue      # 节点状态总览
│   │   ├── Projects.vue        # 项目管理
│   │   ├── Profile.vue         # 连接配置
│   │   └── Logs.vue           # 日志查看
│   ├── store/                 # Pinia 状态管理
│   ├── api/                   # API 调用封装
│   └── router/                # 路由配置
├── package.json                # 依赖配置
├── vite.config.js            # Vite 构建配置
└── README.md                  # 前端文档
```

## 快速开始

### 1. 启动 NodeAgent 后端

```bash
cd runtime/node_agent
make dev
```

### 2. 启动前端开发服务器

```bash
cd runtime/node_agent_front
pnpm install
pnpm dev
```

### 3. 访问地址

- 前端: http://localhost:3000
- 后端 API: http://localhost:8080

## 常见问题

### Q: 如何配置执行器类型？
A: 编辑 `runtime/node_agent/configs/config.yaml` 文件，设置 `executor.type` 为 `process` 或 `docker`。

### Q: 如何查看 NodeAgent 日志？
A:
- NodeAgent 日志: `/var/log/node_agent/agent.log`
- 项目日志: `/var/log/node_agent/runtime/{projectID}.log`

### Q: 如何添加新的执行器？
A:
1. 在 `runtime/node_agent/internal/agent/executor/` 目录下创建新的执行器文件
2. 实现 `Executor` 接口
3. 在 `cmd/main.go` 中注册

### Q: 前端如何调用后端 API？
A: 前端通过 `src/api/nodeApi.js` 中的封装方法调用后端 API，开发模式下 Vite 会自动代理到 `http://localhost:8080`。

## API 快速参考

### 项目操作
```bash
# 部署项目
curl -X POST http://127.0.0.1:8080/api/v1/projects/demo/deploy \
  -H "Content-Type: application/json" \
  -d '{"version":"1.0.0","autoStart":true}'

# 启动项目
curl -X POST http://127.0.0.1:8080/api/v1/projects/demo/start

# 停止项目
curl -X POST http://127.0.0.1:8080/api/v1/projects/demo/stop

# 查看项目状态
curl http://127.0.0.1:8080/api/v1/projects/demo
```

### 健康检查
```bash
# 基础健康检查
curl http://127.0.0.1:8080/health

# 详细状态
curl http://127.0.0.1:8080/status
```

## 贡献指南

欢迎为 NodeAgent 项目贡献代码或文档！

### 贡献代码
1. Fork 项目
2. 创建特性分支
3. 提交代码
4. 创建 Pull Request

### 贡献文档
1. 在对应文档目录修改或添加文档
2. 更新索引文件
3. 提交 Pull Request

## 许可证

MIT License

## 联系支持

如有问题或建议，请提交 Issue 或联系开发团队。
