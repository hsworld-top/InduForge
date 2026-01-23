# NodeAgent Front 详细功能文档

## 概述

NodeAgent Front 是基于 Vue 3 开发的 NodeAgent Web 管理界面，提供现代化的用户交互体验。它采用前后端分离架构，通过 RESTful API 与 NodeAgent 后端通信。

## 技术栈

| 技术 | 版本 | 用途 |
|------|------|------|
| Vue.js | ^3.4.0 | 前端框架 |
| Vue Router | ^4.2.5 | 路由管理 |
| Pinia | ^2.1.7 | 状态管理 |
| Element Plus | ^2.5.0 | UI 组件库 |
| Axios | ^1.6.0 | HTTP 客户端 |
| Vite | ^5.0.0 | 构建工具 |
| Sass | ^1.69.0 | CSS 预处理器 |

## 项目结构

```
src/
├── api/              # API 调用封装
│   ├── nodeApi.js   # NodeAgent API 接口
│   └── registerApi.js # 注册相关 API
├── components/       # 公共组件（可扩展）
├── router/          # 路由配置
│   └── index.js    # 路由定义（含初始化守卫）
├── store/           # Pinia 状态管理
│   └── nodeStore.js # 节点状态 Store
├── utils/           # 工具函数
│   ├── initCheck.js    # 初始化状态检查
│   └── nameValidator.js # 节点名称验证
├── views/           # 页面组件
│   ├── InitWizard.vue # 初始化向导
│   ├── Dashboard.vue # 仪表板
│   ├── Projects.vue  # 项目管理
│   ├── Profile.vue   # 连接配置
│   └── Logs.vue     # 日志查看
├── App.vue         # 根组件
└── main.js         # 应用入口
```

## 核心功能详解

### 1. 初始化向导 (InitWizard)

**文件位置**: `src/views/InitWizard.vue`

**功能描述**: 首次启动时的配置向导，引导用户完成节点初始化。

#### 1.1 向导流程

**4步骤向导**:
1. 选择运行模式（在线/离线）
2. 填写配置信息
3. 等待审批（仅在线模式且非自动审批）
4. 完成初始化

#### 1.2 模式选择

**在线模式特性**:
- 卡片式展示，点击选择
- 显示功能特点：远程管控、自动部署、实时监控
- 适合生产环境

**离线模式特性**:
- 卡片式展示，点击选择
- 显示功能特点：独立运行、手动管理、隔离网络
- 适合开发测试

**选择提示**:
- 模式选择后无法修改的警告提示

#### 1.3 在线模式配置

**表单字段**:
- 运维中心地址（必填，URL格式验证）
  - 提供"测试连接"按钮
  - 实时验证地址格式
- 用户名（必填）
- 密码（必填，密码框）
- 节点名称（必填，实时验证）
  - 正则验证：`^[a-zA-Z0-9_-]{3,50}$`
  - 显示验证结果和规则提示
- 节点描述（选填）
- IP地址（选填，自动获取）
- 管理端口（默认8080）

**表单验证**:
- 实时验证节点名称格式
- 提交前完整表单验证
- 友好的错误提示

#### 1.4 离线模式配置

**表单字段**:
- 节点名称（必填，实时验证）
- 节点描述（选填）

**功能说明**:
- 显示离线模式的特点和限制
- 提示手动管理工程包和启停

#### 1.5 注册与审批

**提交注册**:
- 向运维中心发送注册请求（带用户凭证）
- 请求 API: `POST /api/v1/node-register/register-with-auth`

**自动审批**:
- 用户角色为 OPS_ADMIN 或 SYSTEM_ADMIN
- 立即返回 approved 状态和 registrationToken
- 保存配置后直接跳转到完成步骤

**人工审批**:
- 用户角色为其他角色
- 返回 pending 状态和 nodeId
- 进入等待审批界面

**轮询机制**:
- 每5秒自动查询审批状态
- 查询 API: `GET /api/v1/node-register/:nodeId/approval-status`
- 超时时间：30分钟（360次轮询）
- 提供手动刷新按钮（防抖处理）
- 显示最后刷新时间和轮询计数

**状态处理**:
- approved: 保存 token，跳转到完成步骤
- rejected: 显示拒绝提示，提供重新申请按钮
- pending: 继续等待

#### 1.6 配置持久化

**前端存储（localStorage）**:
- 节点模式（mode）
- 节点ID（nodeId）
- 节点名称（nodeName）
- 运维中心地址（centerUrl）
- 注册令牌（registrationToken）
- 审批状态（approvalStatus）
- IP地址和端口

**后端配置（config.yaml）**:
- 前端调用 NodeAgent 后端接口保存配置
- 保存 API: `POST /api/v1/config/save`
- 写入 config.yaml 文件
- 在线模式启动心跳服务

### 2. 仪表板 (Dashboard)

**文件位置**: `src/views/Dashboard.vue`

**功能描述**: 展示节点整体状态和运行项目的快速概览。

#### 1.1 节点信息展示

**显示内容**:
- 节点 ID
- 版本号
- 执行器类型
- 工作目录
- 创建和更新时间

**UI 组件**:
- `el-card` - 卡片容器
- `el-descriptions` - 描述列表（可扩展）

**数据来源**:
```javascript
// API 调用
GET /api/v1/node/info
```

**代码示例**:
```vue
<template>
  <el-card class="box-card">
    <template #header>
      <div class="card-header">
        <span>节点信息</span>
      </div>
    </template>
    <div class="node-info">
      <div class="info-item">
        <span class="label">节点ID:</span>
        <span class="value">{{ nodeStore.nodeInfo?.id || '-' }}</span>
      </div>
      <!-- 更多字段... -->
    </div>
  </el-card>
</template>
```

#### 1.2 运行项目列表

**显示内容**:
- 项目 ID
- 当前版本
- 运行状态
- PID（运行时）
- 快速操作按钮

**状态映射**:
```javascript
const getStatusType = (status) => {
  const map = {
    running: 'success',   // 绿色 - 运行中
    stopped: 'info',     // 灰色 - 已停止
    error: 'danger',     // 红色 - 错误
    deploying: 'warning', // 黄色 - 部署中
  }
  return map[status] || 'info'
}
```

**快速操作**:
- 启动/停止按钮（根据状态动态显示）
- 重启按钮
- 查看日志按钮

**数据来源**:
```javascript
// API 调用
GET /api/v1/projects
```

#### 1.3 实时刷新

**功能**:
- 手动刷新按钮
- 自动刷新（可选配置）
- 加载状态指示

**实现**:
```javascript
const refresh = async () => {
  await nodeStore.fetchNodeInfo()
  await nodeStore.fetchProjects()
  ElMessage.success('刷新成功')
}
```

### 2. 项目管理 (Projects)

**文件位置**: `src/views/Projects.vue`

**功能描述**: 提供完整的项目生命周期管理功能。

#### 2.1 项目列表

**表格列定义**:
- 项目 ID
- 当前版本
- 状态（带标签）
- 运行时状态
- PID
- 最后启动时间
- 操作（固定右侧列）

**表格特性**:
- 加载状态指示
- 空数据提示
- 响应式列宽
- 固定操作列

**代码示例**:
```vue
<el-table
  v-loading="nodeStore.loading"
  :data="nodeStore.projects"
  style="width: 100%"
>
  <el-table-column prop="id" label="项目ID" />
  <el-table-column label="状态" width="100">
    <template #default="{ row }">
      <el-tag :type="getStatusType(row.status)">
        {{ row.status }}
      </el-tag>
    </template>
  </el-table-column>
  <!-- 更多列... -->
</el-table>
```

#### 2.2 部署项目

**对话框表单**:
- 项目 ID（必填）
- 版本号（必填）
- IFP 包路径（必填）
- 自动启动开关

**表单验证**:
```javascript
const deployProject = async () => {
  if (!deployForm.projectId || !deployForm.version) {
    ElMessage.warning('请填写项目ID和版本')
    return
  }
  // 部署逻辑...
}
```

**请求示例**:
```javascript
// API 调用
POST /api/v1/projects/{id}/deploy
{
  "version": "1.0.0",
  "ifpPackage": "/path/to/demo.ifp",
  "autoStart": true
}
```

#### 2.3 项目操作

**操作类型**:

1. **启动**
   ```javascript
   const startProject = async (projectId) => {
     await nodeStore.startProject(projectId)
   }
   ```

2. **停止**
   ```javascript
   const stopProject = async (projectId) => {
     await nodeStore.stopProject(projectId)
   }
   ```

3. **重启**
   ```javascript
   const restartProject = async (projectId) => {
     await nodeStore.restartProject(projectId)
   }
   ```

4. **查看日志**
   ```javascript
   const viewLogs = (projectId) => {
     router.push(`/logs/${projectId}`)
   }
   ```

**确认对话框**:
```javascript
const stopProject = async (projectId) => {
  await ElMessageBox.confirm(
    `确定要停止项目 ${projectId} 吗？`,
    '警告',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    }
  )
  await nodeStore.stopProject(projectId)
}
```

#### 2.4 版本回滚

**回滚对话框**:
- 版本选择器（列出所有可用版本）
- 确认按钮

**版本数据来源**:
```javascript
const availableVersions = ref([])
const showRollbackDialog = (project) => {
  currentProject.value = project
  availableVersions.value = project.versions || []
  showRollbackDialog.value = true
}
```

**回滚请求**:
```javascript
const confirmRollback = async () => {
  await nodeStore.rollbackProject(
    currentProject.value.id,
    rollbackVersion.value
  )
}
```

#### 2.5 更多操作（下拉菜单）

**下拉菜单选项**:
- 回滚版本
- 连接配置
- 删除项目（带确认）

**实现**:
```vue
<el-dropdown>
  <el-button size="small">
    更多<el-icon class="el-icon--right"><ArrowDown /></el-icon>
  </el-button>
  <template #dropdown>
    <el-dropdown-menu>
      <el-dropdown-item @click="showRollbackDialog(row)">
        回滚版本
      </el-dropdown-item>
      <el-dropdown-item @click="showProfileDialog(row)">
        连接配置
      </el-dropdown-item>
      <el-dropdown-item divided @click="deleteProject(row)">
        删除项目
      </el-dropdown-item>
    </el-dropdown-menu>
  </template>
</el-dropdown>
```

### 3. 连接配置 (Profile)

**文件位置**: `src/views/Profile.vue`

**功能描述**: 查看和编辑项目的 ConnectionProfile，敏感信息自动脱敏。

#### 3.1 配置加载

**项目 ID 输入**:
- 支持 URL 参数预填
- 加载按钮

**加载逻辑**:
```javascript
const loadProfile = async () => {
  if (!projectId.value) {
    ElMessage.warning('请输入项目ID')
    return
  }
  loading.value = true
  try {
    const data = await nodeStore.getProfile(projectId.value)
    profileData.value = data
  } catch (error) {
    ElMessage.error('加载配置失败')
  } finally {
    loading.value = false
  }
}
```

#### 3.2 配置展示

**描述列表**:
- 名称
- 端点
- 认证类型（带标签）
- 元数据（JSON 格式化）

**代码示例**:
```vue
<el-descriptions :column="2" border>
  <el-descriptions-item label="名称">
    {{ profileData.name || '-' }}
  </el-descriptions-item>
  <el-descriptions-item label="端点">
    {{ profileData.endpoint || '-' }}
  </el-descriptions-item>
  <el-descriptions-item label="认证类型">
    <el-tag>{{ profileData.authType || '-' }}</el-tag>
  </el-descriptions-item>
  <el-descriptions-item label="元数据">
    <pre class="metadata">{{
      JSON.stringify(profileData.metadata, null, 2)
    }}</pre>
  </el-descriptions-item>
</el-descriptions>
```

#### 3.3 编辑模式

**切换按钮**:
- 进入编辑模式
- 取消编辑

**编辑表单**:
- 名称输入
- 端点输入
- 认证类型选择器
- 认证数据（JSON 文本域）

**表单处理**:
```vue
<el-form v-if="editMode" :model="editForm" label-width="120px">
  <el-form-item label="名称">
    <el-input v-model="editForm.name" />
  </el-form-item>
  <el-form-item label="认证类型">
    <el-select v-model="editForm.authType">
      <el-option label="Token" value="token" />
      <el-option label="Basic" value="basic" />
      <el-option label="OAuth" value="oauth" />
    </el-select>
  </el-form-item>
</el-form>
```

#### 3.4 保存配置

**数据验证**:
- JSON 格式验证
- 必填字段检查

**保存请求**:
```javascript
const saveProfile = async () => {
  try {
    const authData = JSON.parse(authDataStr.value)
    const success = await nodeStore.saveProfile(projectId.value, {
      ...editForm,
      authData,
    })
    if (success) {
      editMode.value = false
      await loadProfile()
    }
  } catch (e) {
    ElMessage.error('认证数据格式不正确')
  }
}
```

### 4. 日志查看 (Logs)

**文件位置**: `src/views/Logs.vue`

**功能描述**: 实时查看项目日志，支持过滤、搜索和下载。

#### 4.1 日志控制

**控制选项**:
- 日志级别选择器（全部、DEBUG、INFO、WARN、ERROR）
- 搜索框（关键词搜索）
- 自动刷新开关

**代码实现**:
```vue
<div class="log-controls">
  <el-select v-model="logLevel" placeholder="选择日志级别">
    <el-option label="全部" value="" />
    <el-option label="DEBUG" value="debug" />
    <el-option label="INFO" value="info" />
    <el-option label="WARN" value="warn" />
    <el-option label="ERROR" value="error" />
  </el-select>

  <el-input
    v-model="searchKeyword"
    placeholder="搜索日志内容"
    clearable
  >
    <template #prefix>
      <el-icon><Search /></el-icon>
    </template>
  </el-input>

  <el-switch
    v-model="autoRefresh"
    active-text="自动刷新"
  />
</div>
```

#### 4.2 日志展示

**样式设计**:
- 黑色背景（模拟终端）
- 彩色日志级别
- 等宽字体
- 自动滚动到底部

**日志项格式**:
```vue
<div
  v-for="(log, index) in filteredLogs"
  :key="index"
  class="log-item"
  :class="`log-${log.level?.toLowerCase()}`"
>
  <span class="log-time">{{ formatLogTime(log.time) }}</span>
  <span class="log-level">{{ log.level }}</span>
  <span class="log-message">{{ log.message }}</span>
</div>
```

**颜色方案**:
```css
.log-debug .log-level { color: #808080; }  /* 灰色 */
.log-info .log-level { color: #4fc3f7; }   /* 蓝色 */
.log-warn .log-level { color: #ffb74d; }   /* 橙色 */
.log-error .log-level { color: #f06292; }  /* 粉色 */
```

#### 4.3 过滤逻辑

**级别过滤**:
```javascript
const filteredLogs = computed(() => {
  let filtered = logs.value

  if (logLevel.value) {
    filtered = filtered.filter(log =>
      log.level?.toLowerCase() === logLevel.value
    )
  }

  if (searchKeyword.value) {
    filtered = filtered.filter(log =>
      log.message?.toLowerCase().includes(
        searchKeyword.value.toLowerCase()
      )
    )
  }

  return filtered
})
```

#### 4.4 自动刷新

**刷新机制**:
- 定时器设置（默认 5 秒）
- 组件销毁时清理
- 可手动开关

**实现**:
```javascript
watch(autoRefresh, (newVal) => {
  if (newVal) {
    refreshTimer = setInterval(loadLogs, 5000)
  } else {
    if (refreshTimer) {
      clearInterval(refreshTimer)
      refreshTimer = null
    }
  }
})

onUnmounted(() => {
  if (refreshTimer) {
    clearInterval(refreshTimer)
  }
})
```

#### 4.5 日志下载

**下载功能**:
- 按当前过滤条件导出
- 文件名包含项目 ID 和日期
- 纯文本格式

**实现**:
```javascript
const downloadLogs = () => {
  const content = filteredLogs.value
    .map(log => `[${log.time}] ${log.level}: ${log.message}`)
    .join('\n')

  const blob = new Blob([content], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${projectId.value}_logs_${
    new Date().toISOString().split('T')[0]
  }.txt`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}
```

### 5. 状态管理 (Pinia Store)

**文件位置**: `src/store/nodeStore.js`

**功能描述**: 集中管理应用状态，处理数据获取和业务逻辑。

#### 5.1 Store 结构

**状态定义**:
```javascript
export const useNodeStore = defineStore('node', () => {
  const nodeInfo = ref(null)      // 节点信息
  const projects = ref([])        // 项目列表
  const loading = ref(false)      // 加载状态
})
```

#### 5.2 异步操作

**操作方法**:
```javascript
// 获取项目列表
const fetchProjects = async () => {
  loading.value = true
  try {
    const data = await nodeApi.getProjects()
    projects.value = data
  } catch (error) {
    ElMessage.error('获取项目列表失败')
  } finally {
    loading.value = false
  }
}

// 部署项目
const deployProject = async (projectId, data) => {
  try {
    await nodeApi.deployProject(projectId, data)
    ElMessage.success('部署成功')
    await fetchProjects()
    return true
  } catch (error) {
    ElMessage.error('部署失败')
    return false
  }
}
```

#### 5.3 响应式数据

**计算属性**:
```javascript
const runningProjects = computed(() => {
  return projects.value.filter(p => p.status === 'running')
})

const failedProjects = computed(() => {
  return projects.value.filter(p => p.status === 'error')
})
```

### 6. API 调用封装

**文件位置**: `src/api/nodeApi.js`

**功能描述**: 封装所有 HTTP 请求，统一错误处理和响应格式。

#### 6.1 Axios 配置

**配置**:
```javascript
const api = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
})
```

#### 6.2 API 方法

**节点信息**:
```javascript
export const nodeApi = {
  getNodeInfo() {
    return api.get('/node/info').then(res => res.data)
  },

  getNodeStatus() {
    return api.get('/node/status').then(res => res.data)
  },
}
```

**项目管理**:
```javascript
getProjects() {
  return api.get('/projects').then(res => res.data)
},

deployProject(projectId, data) {
  return api.post(`/projects/${projectId}/deploy`, data)
    .then(res => res.data)
},

startProject(projectId) {
  return api.post(`/projects/${projectId}/start`)
    .then(res => res.data)
},
// 更多方法...
```

#### 6.3 错误处理

**全局拦截器**:
```javascript
api.interceptors.response.use(
  response => response.data,
  error => {
    const message = error.response?.data?.message || '请求失败'
    ElMessage.error(message)
    return Promise.reject(error)
  }
)
```

### 7. 路由配置

**文件位置**: `src/router/index.js`

**路由定义**:
```javascript
const routes = [
  {
    path: '/',
    name: 'Dashboard',
    component: Dashboard,
  },
  {
    path: '/projects',
    name: 'Projects',
    component: Projects,
  },
  {
    path: '/profile',
    name: 'Profile',
    component: Profile,
  },
  {
    path: '/logs/:projectId',
    name: 'Logs',
    component: Logs,
    props: true,  // 传递路由参数作为 props
  },
]
```

**导航守卫**（可扩展）:
```javascript
router.beforeEach((to, from, next) => {
  // 权限检查
  next()
})
```

### 8. UI/UX 设计

#### 8.1 响应式布局

**断点设置**:
- `xs`: < 768px (手机)
- `sm`: 768px - 992px (平板)
- `md`: 992px - 1200px (桌面)
- `lg`: > 1200px (大屏)

**自适应元素**:
```css
.projects-grid {
  display: grid;
  gap: 16px;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
}
```

#### 8.2 加载状态

**骨架屏**:
```vue
<el-skeleton :rows="5" animated v-if="loading" />
```

**加载指示器**:
```vue
<el-table v-loading="nodeStore.loading" :data="projects">
  <!-- 表格内容 -->
</el-table>
```

#### 8.3 消息提示

**消息类型**:
- `ElMessage.success()` - 成功提示
- `ElMessage.error()` - 错误提示
- `ElMessage.warning()` - 警告提示
- `ElMessage.info()` - 信息提示

**示例**:
```javascript
ElMessage.success('操作成功')
ElMessage.error('操作失败')
ElMessage.warning('请确认操作')
```

#### 8.4 确认对话框

**结构**:
```javascript
await ElMessageBox.confirm(
  '此操作将永久删除该项目，是否继续？',
  '警告',
  {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning',
  }
)
```

### 9. 开发指南

#### 9.1 开发环境搭建

**安装依赖**:
```bash
cd runtime/node_agent_front
pnpm install
```

**启动开发服务器**:
```bash
pnpm dev
```

**Vite 配置**:
```javascript
export default defineConfig({
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
```

#### 9.2 构建部署

**构建生产版本**:
```bash
pnpm build
```

**构建输出**:
- 目录: `dist/`
- 文件: HTML、CSS、JS、资源文件

**部署到服务器**:
```bash
# Nginx 配置
server {
    listen 80;
    server_name nodeagent.example.com;

    location / {
        root /var/www/node_agent_front/dist;
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://localhost:8080;
    }
}
```

#### 9.3 添加新页面

**步骤**:
1. 创建 Vue 组件
2. 在路由中注册
3. 在导航中添加链接

**示例**:
```vue
<!-- src/views/NewPage.vue -->
<template>
  <div class="new-page">
    <h1>新页面</h1>
  </div>
</template>
```

```javascript
// src/router/index.js
import NewPage from '@/views/NewPage.vue'

const routes = [
  // ... 其他路由
  {
    path: '/new',
    name: 'NewPage',
    component: NewPage,
  },
]
```

#### 9.4 自定义组件

**组件结构**:
```vue
<!-- src/components/CustomComponent.vue -->
<template>
  <div class="custom-component">
    <!-- 组件内容 -->
  </div>
</template>

<script setup>
// 组件逻辑
</script>

<style scoped>
.custom-component {
  /* 样式 */
}
</style>
```

**使用组件**:
```vue
<template>
  <div>
    <CustomComponent />
  </div>
</template>

<script setup>
import CustomComponent from '@/components/CustomComponent.vue'
</script>
```

### 10. 性能优化

#### 10.1 代码分割

**路由懒加载**:
```javascript
const Dashboard = () => import('@/views/Dashboard.vue')
const Projects = () => import('@/views/Projects.vue')
```

**组件懒加载**:
```javascript
const HeavyComponent = defineAsyncComponent(() =>
  import('@/components/HeavyComponent.vue')
)
```

#### 10.2 缓存策略

**Pinia 持久化**:
```javascript
import { createPinia } from 'pinia'
import piniaPluginPersistedstate from 'pinia-plugin-persistedstate'

const pinia = createPinia()
pinia.use(piniaPluginPersistedstate)
```

**HTTP 缓存**:
```javascript
const api = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
  headers: {
    'Cache-Control': 'no-cache',
  },
})
```

#### 10.3 虚拟滚动

**大数据列表**:
```vue
<el-table-v2
  :columns="columns"
  :data="largeData"
  :height="400"
  fixed
/>
```

## 总结

NodeAgent Front 提供了完整的前端管理能力，包括：

- ✅ 现代化的 Vue 3 + Element Plus UI
- ✅ 完整的项目生命周期管理
- ✅ 实时日志查看和过滤
- ✅ 连接配置可视化管理
- ✅ 响应式设计支持多端访问
- ✅ 完善的错误处理和用户提示
- ✅ 状态集中管理和 API 封装
- ✅ 可扩展的组件化架构

通过这些功能，用户可以通过直观的 Web 界面高效地管理 NodeAgent 和运行在其上的 RuntimeEngine 项目。
