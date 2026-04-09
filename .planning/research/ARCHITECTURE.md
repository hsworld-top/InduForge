# ARCHITECTURE

## 目标架构（本轮）
保持现有模块拆分，围绕“契约统一 + 闭环可观测”做增量收敛。

## 组件边界
- `designer`：输出发布态 schema（页面、组件树、绑定信息）。
- `dev_core`：负责项目发布、部署编排、认证授权、统一 API 契约。
- `runtime/node_agent`：执行部署与运行生命周期，回传状态。
- `runtime/node_agent_front`：节点本地管理与运行态查看。
- `dev_ide`/`datacenter`：平台管理与数据中心消费统一 API。

## 核心数据流
1. 设计发布流：`designer` -> `dev_core`（发布请求）-> 生成 IFP 制品。
2. 部署执行流：`dev_core` -> `node_agent`（命令）-> Runtime 启停/回滚。
3. 状态回传流：Runtime `/health`/`/status` -> NodeAgent -> 平台可观测面板。
4. 实时数据流：设备/MQTT -> `dev_core` -> Socket 房间广播 -> 前端消费。

## 构建顺序建议
1. 先统一后端响应与错误语义（降低联调成本）。
2. 再固化发布产物与 manifest 契约（保证部署输入稳定）。
3. 再完善 NodeAgent/Runtime 状态协议（保证部署输出可观测）。
4. 最后收敛权限与重复逻辑，补齐关键回归测试。

## 接口边界约束
- API 前缀固定 `/api/v1`。
- 健康检查固定 `/health`。
- 业务接口返回统一 `ApiResponse`。
- 错误统一 `AppError` + `ErrorCodes`。

## 风险热点
- 旧路由直接 `res.json` 与新规范并存。
- 发布产物中 assets 与契约字段可能不完整。
- NodeAgent 执行日志覆盖导致排障链路断裂。

## 结论
本轮不改变宏观架构，重点是让跨模块契约从“可用”提升为“可验证、可运维、可持续演进”。
