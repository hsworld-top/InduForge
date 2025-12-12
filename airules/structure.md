# 项目结构

## 根目录布局

```
InduForge/
├── dev_core/          # 后端 API 服务
├── dev_ide/           # 主 IDE 应用
├── datacenter/        # 数据中心应用
├── designer/          # 可视化设计器应用
├── docs/              # 项目文档
├── nginx/             # Nginx 配置
└── .env               # 环境变量
```

## 后端结构 (dev_core/)

```
dev_core/
├── config/            # 配置文件 (config.json)
├── database/          # 数据库初始化脚本
├── scripts/           # 工具脚本 (init-database.js)
└── src/
    ├── app.js         # Express 应用设置
    ├── index.js       # 入口文件
    ├── config/        # 应用配置模块
    ├── constants/     # 错误码和常量
    ├── controllers/   # 路由控制器
    ├── dsl/           # DSL 验证和类型
    ├── locales/       # 国际化翻译 (en.json, zh-CN.json)
    ├── middlewares/   # Express 中间件 (auth, locale, validate)
    ├── models/        # Sequelize 模型
    ├── routes/        # API 路由定义
    ├── services/      # 业务逻辑层
    │   ├── drivers/   # 数据库驱动 (MySQL, PostgreSQL, SQL Server)
    │   └── protocols/ # 协议实现
    └── utils/         # 工具函数 (AppError, logger, response, token)
```

## 前端结构 (dev_ide/, datacenter/, designer/)

所有前端应用共享相似的结构：

```
<app>/
├── public/            # 静态资源
├── src/
│   ├── api/           # API 客户端模块
│   ├── assets/        # 图片、样式
│   ├── components/    # Vue 组件
│   ├── constants/     # 常量和枚举
│   ├── router/        # Vue Router 配置
│   ├── store/         # Pinia 状态管理
│   ├── utils/         # 工具函数
│   ├── views/         # 页面组件
│   ├── App.vue        # 根组件
│   └── main.js        # 入口文件
├── index.html         # HTML 模板
├── vite.config.js     # Vite 配置
├── tailwind.config.js # Tailwind 配置
└── package.json       # 依赖配置
```

## Designer 特有结构

设计器应用有额外的目录：

```
designer/src/
├── composables/       # Vue 组合式函数 (useCanvas, useDragDrop, useHistory)
├── engine/            # 核心引擎模块
│   ├── animation/     # 动画系统
│   ├── binding/       # 数据绑定引擎
│   ├── canvas/        # Canvas 工具 (标尺、选择框、辅助线)
│   └── datasource/    # 数据源管理
└── registry/          # 组件注册表
    ├── basic/         # 基础图形 (Rectangle, Circle, Text, Image)
    ├── charts/        # 图表组件 (Bar, Line, Pie, Gauge)
    ├── layout/        # 布局容器
    └── ui/            # UI 组件 (Button, Input, Table, Form)
```

## 关键约定

### 文件命名
- 组件：PascalCase（如 `ComponentTree.vue`）
- 工具函数：camelCase（如 `dropZoneCalculator.js`）
- API 模块：kebab-case，带 `.api.js` 后缀（如 `design.api.js`）
- 测试：`__tests__/` 目录，带 `.test.js` 或 `.property.test.js` 后缀

### 代码组织
- API 调用集中在 `src/api/` 目录
- 业务逻辑在 services（后端）或 composables（前端）
- 共享工具函数在 `src/utils/`
- 常量和枚举在 `src/constants/`

### DSL 和 Schema
- 页面 Schema 遵循 DSL 2.0.0 规范（参见 `docs/dsl-design.md`）
- 组件定义在 `designer/src/registry/`
- DSL 验证在 `dev_core/src/dsl/validators.js`

### 状态管理
- 使用 Pinia stores 管理应用状态
- Store 文件按领域命名（如 `design.js`）
- Actions 处理异步操作，getters 处理计算状态

### 错误处理
- 后端使用 `AppError` 类，错误码来自 `constants/errorCodes.js`
- 前端通过 Element Plus 消息组件显示错误
- API 拦截器中集中处理错误

### 测试
- 单元测试放在 `__tests__/` 目录
- 属性测试使用 fast-check 库
- 测试文件与源代码同级放置
