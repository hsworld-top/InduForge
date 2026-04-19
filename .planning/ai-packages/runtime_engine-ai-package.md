# runtime_engine AI 分发包

## 1. 模块定位
- `runtime/engine_*` 是低代码工程运行时服务与前端。
- 当前阶段先承接共享渲染内核，优先服务设计器开发态预览，后续再承接独立运行态宿主。

## 2. 当前已实现
- 当前目录为空，需要从 0 到 1 建设。

## 3. 当前缺口
- 无基础目录骨架
- 无制品加载器
- 无页面路由
- 无健康检查和运行状态接口

## 4. 本模块目标
- 实现共享渲染内核 MVP
- 支持基础组件渲染
- 提供最小数据绑定能力
- 为未来独立运行态预留宿主接口

## 5. 直接输入
- [产品定义](../../docs/产品定义.md)
- [开发态预览专项计划](../docs/process/platform/开发态预览专项计划.md)
- [发布态 Schema 契约](../../docs/contracts/designer-publish-schema.md)
- [IFP Manifest 契约](../../docs/contracts/ifp-manifest-contract.md)
- [NodeAgent 启动协议](../../docs/contracts/node-agent-runtime-protocol.md)
- [Runtime 健康状态协议](../../docs/contracts/runtime-health-status-contract.md)
- [runtime_engine.task](./tasks/runtime_engine.task.md)

## 6. 输出物
- `engine_core` 骨架
- `engine_front` 基础页面渲染能力
- `engine_data` 最小数据上下文
- 供预览宿主复用的接口

## 7. 关键能力
- 制品加载
- 页面路由
- 基础组件渲染
- 数据点/查询/全局变量绑定

## 8. 必须遵守的约束
- 先做 MVP，不扩动作系统和多视图
- 白名单组件优先
- 运行态字段严格以契约文档为准

## 9. 模块边界
### 本模块负责
- 解释发布态工程
- 提供共享渲染接口
- 输出宿主无关能力

### 本模块不负责
- 制品打包
- 节点下载和进程托管
- 设计器编辑逻辑

## 10. 验收标准
- 能读取发布态工程结构
- 能在预览宿主中访问入口页
- 支持基础组件
- 基础绑定可工作

## 11. 建议推进顺序
1. 初始化三层目录
2. 实现 Schema 读取和校验
3. 实现页面解析与路由
4. 接入基础组件和绑定
5. 抽离宿主接口

## 12. 非目标
- 不做复杂工业组件库
- 不做独立运行态宿主优先实现
- 不做完整脚本沙箱
