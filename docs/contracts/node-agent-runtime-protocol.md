# NodeAgent 与 RuntimeEngine 启动协议

## 1. 文档定位
- 本文档定义 `runtime/node_agent` 如何托管 `runtime_engine`。
- 本协议只覆盖启动、停止、探活、目录约定和回滚切换，不覆盖页面业务逻辑。

## 2. 当前已实现
### 2.1 已有基础
- NodeAgent 已具备部署执行、目录管理、状态回传和本地控制能力。

### 2.2 当前缺口
- RuntimeEngine 尚未实现，因此启动命令、工作目录和探活协议尚未冻结。

## 3. 启动流程
1. NodeAgent 获取部署命令。
2. 下载 `.ifp` 到本地版本目录。
3. 解压并校验 `manifest.json`。
4. 生成 Runtime 启动参数。
5. 启动 Runtime 进程。
6. 轮询 `/health`。
7. 成功后标记部署为 `RUNNING`。

## 4. 工作目录约定
```text
runtime-data/
  projects/
    <projectId>/
      releases/
        <version>/
          package/
          runtime.log
      current/
```

## 5. 启动参数契约
### 建议参数
| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `--project-dir` | 是 | 当前版本解压目录 |
| `--port` | 是 | Runtime 服务端口 |
| `--host` | 否 | 监听地址 |
| `--locale` | 否 | 默认语言 |
| `--theme` | 否 | 默认主题 |

### 环境变量建议
| 变量 | 说明 |
| --- | --- |
| `INDUFORGE_RUNTIME_PORT` | Runtime 端口 |
| `INDUFORGE_PROJECT_DIR` | 项目目录 |
| `INDUFORGE_RELEASE_VERSION` | 当前版本号 |

## 6. 停止协议
- 优先发送优雅停止信号。
- 等待超时后强制停止。
- 停止完成后回传状态。

## 7. 探活协议
- 健康检查地址：`GET /health`
- 状态检查地址：`GET /status`
- 探活失败重试：建议 3 次
- 连续失败后状态：`FAILED`

## 8. 回滚约定
- 回滚只允许切换到本地已有可用版本。
- 切换后必须重新执行启动和探活流程。
- 回滚失败必须记录最后错误并上报平台。

## 9. 异常处理
### 下载失败
- 状态改为 `FAILED`
- 保存下载错误信息

### 解压或校验失败
- 拒绝启动 Runtime
- 上报制品错误

### 启动失败
- 记录进程退出码、标准错误输出
- 可按策略重试

### 探活失败
- 停止当前进程
- 标记版本启动失败

## 10. 当前阶段非目标
- 不实现多工程同时运行。
- 不实现复杂资源配额控制。
- 不实现跨节点热迁移。

## 11. 关联文档
- [运行时闭环专项计划](../运行时闭环专项计划.md)
- [RuntimeEngine](../designer/refactor/runtime-engine.md)
- [runtime_node_agent.task](../../runtime_node_agent.task.md)
- [runtime_engine.task](../../runtime_engine.task.md)

## 12. 待确认事项
- 首版是否明确“一节点只运行一个工程”。
- 端口由平台指定还是节点自动分配。
