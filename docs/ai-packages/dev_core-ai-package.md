# dev_core AI 分发包

## 1. 模块定位
- `dev_core` 是平台统一控制面后端。
- 当前阶段的核心职责是：为设计器开发态预览提供稳定的工程、资源和数据聚合能力，并保证后续可平滑过渡到发布与运行态。

## 2. 当前已实现
- 项目、页面、数据连接、查询、数据点、发布、部署、节点管理主流程已存在。
- 已有 `publishService`、`deploymentService`、`nodeService`。
- `/api/v1` 路由体系已成型。

## 3. 当前缺口
- 预览可消费的工程聚合结构还未明确。
- 发布与预览共享结构尚未完全收敛。
- 预览主链路测试不足。

## 4. 本模块目标
- 输出预览可消费、且后续可复用到运行态的统一工程结构。
- 提供稳定的数据、资源和页面聚合接口。
- 保持与发布态 Schema 一致，避免两套结构。

## 5. 直接输入
- [产品决策确认版](../产品决策确认版.md)
- [开发态预览专项计划](../开发态预览专项计划.md)
- [发布态 Schema 契约](../contracts/designer-publish-schema.md)
- [IFP Manifest 契约](../contracts/ifp-manifest-contract.md)
- [dev_core.task](./tasks/dev_core.task.md)

## 6. 输出物
- 预览可消费的工程聚合结果
- 统一数据/资源访问能力
- 后续发布可复用的结构约束

## 7. 关键对象
- 表：`projects`、`design_pages`、`data_connections`、`data_queries`、`data_points`、`deployments`、`nodes`、`node_deployments`、`node_commands`
- 服务：`publishService`、`deploymentService`、`nodeService`

## 8. 必须遵守的约束
- 接口统一 `/api/v1`
- 响应统一 `ApiResponse`
- 不使用 `sequelize.sync()`
- 优先做契约收敛，不扩非主线后台功能

## 9. 模块边界
### 本模块负责
- 工程聚合
- 页面/资源/数据对象聚合
- 预览和发布结构一致性

### 本模块不负责
- 页面运行渲染
- NodeAgent 本地进程托管
- Designer 编辑器内部模型

## 10. 验收标准
- 设计器预览能拿到稳定聚合结构
- 数据和资源访问规则稳定
- 后续发布不需要推翻预览输入结构
- 至少有预览主链路测试

## 11. 建议推进顺序
1. 冻结预览可消费的工程聚合结构
2. 保证和发布态 Schema 一致
3. 增加数据/资源接口适配
4. 补测试

## 12. 非目标
- 不优先投入完整部署与节点联调
- 不实现 Runtime 页面逻辑
