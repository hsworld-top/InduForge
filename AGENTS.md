# Codex 开发规范（InduForge）

本文件面向内部开发，作为 Codex 在本项目内的通用开发引导与模块约束说明。

## 公用设置

- **工作目录**：`/mnt/d/SVNCode/indu-forge`
- **主要模块**：`dev_core`（后端）、`dev_ide`（IDE 前端）、`datacenter`（数据中心前端）、`designer`（设计器前端）
- **文档目录**：`docs/`（除根目录 `README.md` 外）
- **文档放置约束**：
  - 项目文档与过程文档一律放在 `docs/` 下
  - 各模块代码目录下不允许新增或保存开发过程的 `.md` 文档
  - 根目录仅保留 `README.md` 与 `AGENTS.md`
- **忽略范围**：`docker/`、`nginx/`
- **包管理器**：`pnpm`
- **环境变量**：统一从项目根目录 `.env` 加载
- **接口前缀**：后端 API 统一 `/api/v1`，健康检查 `/health`
- **文档规则**：修改/新增文档需先与需求方确认

## 通用开发约束

- **接口响应**：后端统一使用 `ApiResponse` 格式（`success`、`errorCode`、`message`、`requestId`、`data`）
- **错误处理**：业务错误使用 `AppError` 与 `ErrorCodes`
- **数据库初始化**：使用 `dev_core/scripts/init-database.js`，不使用 `sequelize.sync()`
- **端口默认值**：`dev_core` 9099，`dev_ide` 9091，`datacenter` 9092，`designer` 9093
- **时间处理**：`dev_core`、`dev_ide`、`datacenter`、`designer` 统一使用 `dayjs`，字符串时间展示格式 `YYYY-MM-DD HH:mm:ss`
- **提交规范**：git 提交时 `commit` 使用中文
- **注释与命名（全局）**：
  - 类/函数：必须添加文档注释，说明功能描述、参数含义、返回值类型及异常情况
  - 注释语言：统一使用中文，语法清晰、简洁，避免冗余描述
  - 复杂逻辑：关键业务逻辑代码上方添加单行注释，说明逻辑目的
  - 数据库表/字段：采用下划线命名法（如：`user_info`、`order_id`）
  - 类/组件：采用大驼峰命名法（如：`UserForm`、`OrderList`）
  - 变量/函数：采用小驼峰命名法（如：`userName`、`getUserInfo`）
- **安全约束（全局）**：
  - 数据库操作需使用参数化查询，避免 SQL 注入攻击
- **性能约束（全局）**：
  - 避免冗余计算，重复使用的结果需缓存（如：使用局部变量、缓存组件）
  - 前端代码需考虑懒加载（如：图片、组件懒加载），减少初始加载时间
- **交付要求（全局）**：
  - 若需求描述不清晰，优先生成基础版本代码，并在输出结果中列出需要进一步确认的问题
  - 生成代码后需自行校验语法正确性、逻辑完整性，避免明显错误
  - 对于复杂功能（如：分布式事务、高并发处理），需在输出中说明实现方案的优缺点及适用场景
- 本规范未尽事宜，请在 `docs/` 下补充对应的开发规范文档并在此处引用；如后续存在冲突，以补充文档为准

## 模块约束

### dev_core（后端）

- **技术栈**：Node.js + Express 5 + Sequelize + Redis + Socket.IO
- **路由结构**：`/api/v1` 为主，`/api/v2` 仅含测试接口
- **鉴权**：JWT（访问令牌+刷新令牌），支持验证码登录
- **多租户**：租户级鉴权与工程访问控制
- **核心域**：
  - 设计页面管理（DesignPage）
  - 数据连接/查询（DataConnection、DataQuery）
  - MQTT 连接/订阅/Tag（DataMqttConfig、DataMqttSubscription、DataMqttTag）
  - WebSocket 推送（`SocketService`）
- **命名规范**：
  - 文件/目录：`camelCase` 或 `kebab-case`，保持与现有结构一致
  - 路由：`/api/v1/<resource>`，动作用动词子路径
  - 变量/函数：`camelCase`，类名 `PascalCase`
- **代码格式**：
  - 遵循现有 ESLint/Prettier 约束
  - 避免无意义的多层嵌套，优先早返回
- **注释要求**：
  - 关键业务流程与边界条件添加简要注释
  - 公共服务方法需简要说明输入与输出
- **安全约束**：
  - 参数校验统一走 `validate` 中间件或服务层校验
  - 认证与权限检查优先在路由层完成
  - 禁止在日志中输出密码等敏感字段
- **性能优化**：
  - 避免 N+1 查询，必要时使用关联预加载
  - 长耗时任务避免阻塞主线程
  - Socket.IO 推送避免广播全局，优先房间粒度

### datacenter（数据中心前端）

- **技术栈**：Vue 3 + Vite + Pinia + Element Plus + Monaco
- **图标规范**：统一使用 `unplugin-icons`
- **主入口**：`DataCenterNew.vue`（统一标签页系统）
- **连接模型**：`type=relational|mqtt`，具体 DB 类型由 `config.dbType` 指定
- **MQTT 能力**：连接、订阅、消息查看、变量组/变量管理、实时推送（Socket.IO）
- **通信**：API 基于 `/api/v1`，WebSocket 走同域 Socket.IO
- **命名规范**：
  - 组件：`PascalCase`
  - 文件：`kebab-case`
  - Composable：`useXxx`
  - Store：按领域命名
- **代码格式**：
  - Vue SFC 结构保持 template/script/style 顺序
  - UI 逻辑优先拆分到 composables
- **注释要求**：
  - 复杂交互逻辑添加简短说明
  - 组件对外事件/props 增加说明
- **安全约束**：
  - API 调用必须走统一 `request` 封装
  - Token 与租户信息仅从 Storage 读取
- **性能优化**：
  - 列表渲染避免不必要的深度 watch
  - WebSocket 订阅需及时解绑

### designer（设计器前端）

- **技术栈**：Vue 3 + Pinia + Element Plus + Konva + ECharts + GSAP
- **图标规范**：统一使用 `unplugin-icons`
- **渲染架构**：DOM 组件渲染 + Canvas 辅助渲染（对齐线/选择框等）
- **数据绑定**：
  - **API 模式**（默认）：直接调用后端
  - **Bridge 模式**：通过 `DataCenterBridge` 与 DataCenter iframe 通信
- **表达式引擎**：`ExpressionEngine`（`{{ }}` 语法）
- **命名规范**：
  - 组件：`PascalCase`
  - 引擎/工具类：`PascalCase`
  - composables：`useXxx`
- **代码格式**：
  - 复杂状态变更统一走 store action
  - 画布相关逻辑优先拆分到 `engine/` 与 `composables/`
- **注释要求**：
  - 关键算法与坐标/布局转换需简要注释
- **安全约束**：
  - 表达式与脚本执行仅在可控上下文内
  - Bridge 模式通信需处理未就绪降级
- **性能优化**：
  - 避免频繁触发全量重渲染
  - 画布辅助图形与 DOM 渲染分层保持

### dev_ide（IDE 前端）

- **技术栈**：Vue 3 + Vite + Pinia + Element Plus
- **图标规范**：统一使用 `unplugin-icons`
- **角色访问**：路由基于角色限制（SUPER_ADMIN、SYSTEM_ADMIN、PROJECT_ADMIN、OPS_ADMIN、USER_ADMIN）
- **登录跳转**：未登录统一跳转 `/login`；超级管理员固定进入 `/admin`
- **注意**：路由内仍存在 `TENANT_ADMIN` 字符串，需与后端角色枚举保持一致
- **命名规范**：
  - 组件：`PascalCase`
  - 文件：`kebab-case`
  - API：`*.api.js`
- **代码格式**：
  - 统一使用 ESLint/Prettier
  - 视图层逻辑优先抽离为 composable 或 util
- **注释要求**：
  - 路由与权限控制逻辑需简要说明
- **安全约束**：
  - 认证与权限只信任后端 Token
  - 重要操作需前后端双重校验
- **性能优化**：
  - 列表类视图注意分页与懒加载
  - 避免无意义的全量 store 更新
