# 发布流水线

本文档描述工程的发布编译流程，从校验到生成可部署的工程制品（.ifp）。

> **重要**：发布操作在 **dev_ide** 的工程卡片上触发，不是在 Designer 内部。
> 发布内容包括 **Designer Schema** 和 **DataCenter 配置**。

## 1. 流程概览

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Publish Pipeline                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  1. validate          校验 Schema、绑定、资源引用、数据点状态               │
│         ↓                                                                   │
│  2. compileManifest   生成 manifest.json（dataRequirements、capabilities）  │
│         ↓                                                                   │
│  3. assetNormalize    资源分层处理、路径重写                                │
│         ↓                                                                   │
│  4. bundleIFP         打包 zip（project.json + datacenter.json + assets）   │
│         ↓                                                                   │
│  5. createSnapshot    创建工程级快照，上传存储                              │
│         ↓                                                                   │
│  6. sign              签名（可选）                                          │
│         ↓                                                                   │
│  7. upload            上传 registry + 写版本记录                            │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## 2. validate（校验）

### 2.1 校验规则

```typescript
async function validate(doc: DocumentModel): Promise<ValidationResult> {
  const errors: ValidationError[] = [];
  const warnings: ValidationWarning[] = [];

  // 1. Schema 版本
  if (doc.schemaVersion !== CURRENT_VERSION) {
    warnings.push({ code: "SCHEMA_OUTDATED", message: "..." });
  }

  // 2. 入口页面
  if (!doc.pagesById[doc.entry.homePageId]) {
    errors.push({ code: "ENTRY_NOT_FOUND", message: "首页不存在" });
  }
  if (!doc.pagesById[doc.entry.loginPageId]) {
    errors.push({ code: "LOGIN_NOT_FOUND", message: "登录页不存在" });
  }

  // 3. 角色引用
  for (const [nodeId, node] of Object.entries(doc.nodesById)) {
    validatePermissionRoles(node.permissions, doc.securityDecl.roles, errors);
  }

  // 4. 绑定校验（含数据点状态）
  await validateBindings(doc, errors, warnings);

  // 5. 资源引用
  validateAssets(doc, errors);

  return { valid: errors.length === 0, errors, warnings };
}
```

### 2.2 数据点状态校验

```typescript
async function validateBindings(
  doc: DocumentModel,
  errors: ValidationError[],
  warnings: ValidationWarning[]
): Promise<void> {
  // 收集所有绑定
  const bindings = collectAllBindings(doc);

  // 批量查询状态
  const paths = bindings
    .filter((b) => b.binding.kind === "datapoint")
    .map((b) => b.binding.path);
  const statuses = await fetchDatapointStatuses(doc.project.projectId, paths);

  // 提取 dataRequirements
  const requirements = extractDataRequirements(doc);

  // 校验
  for (const dp of requirements.datapoints) {
    const status = statuses.get(dp.path);

    if (dp.required && status === "invalid") {
      errors.push({
        code: "DATAPOINT_INVALID",
        message: `必需数据点 ${dp.path} 已失效`,
        path: dp.path,
      });
    }

    if (dp.required && status === "unknown") {
      errors.push({
        code: "DATAPOINT_UNKNOWN",
        message: `必需数据点 ${dp.path} 不存在`,
        path: dp.path,
      });
    }

    if (!dp.required && status !== "active") {
      warnings.push({
        code: "DATAPOINT_INACTIVE",
        message: `可选数据点 ${dp.path} 状态异常: ${status}`,
        path: dp.path,
      });
    }
  }
}
```

## 3. compileManifest（编译清单）

### 3.1 manifest.json 结构

```json
{
  "name": "产线A看板",
  "version": "1.0.0",
  "schemaVersion": 2,
  "buildTime": "2026-01-06T10:30:00Z",

  "entry": {
    "loginPageId": "page_login",
    "homePageId": "page_home"
  },

  "uiTargets": ["pc", "bigscreen"],

  "security": {
    "mode": "nodeLocalAuth",
    "roles": ["admin", "operator", "viewer"]
  },

  "dataRequirements": {
    "providers": ["dc_main"],
    "datapoints": [
      {
        "path": "mqtt.EMQX.温度组.temperature",
        "sourceType": "mqtt.tag",
        "required": true
      },
      {
        "path": "db.生产库.设备统计",
        "sourceType": "db.query",
        "required": true
      }
    ],
    "capabilities": ["mqtt", "db"]
  },

  "resourcePacks": [
    {
      "id": "pack_video",
      "name": "视频资源包",
      "size": 524288000,
      "required": false
    }
  ],

  "pages": [
    { "id": "page_home", "path": "/home", "target": "pc" },
    { "id": "page_home_bs", "path": "/home", "target": "bigscreen" }
  ]
}
```

### 3.2 提取 dataRequirements

```typescript
function extractDataRequirements(doc: DocumentModel): DataRequirements {
  const providers = new Set<string>();
  const datapoints = new Map<string, DatapointRequirement>();
  const capabilities = new Set<string>();

  // 遍历所有节点的绑定
  for (const node of Object.values(doc.nodesById)) {
    for (const binding of Object.values(node.bindings)) {
      if (binding.kind === "datapoint") {
        providers.add(binding.provider);

        if (!datapoints.has(binding.path)) {
          datapoints.set(binding.path, {
            path: binding.path,
            sourceType: inferSourceType(binding.path),
            required: true, // 默认必需
          });
        }

        // 推断 capability
        const cap = inferCapability(binding.path);
        if (cap) capabilities.add(cap);
      }
    }

    // 遍历事件中的动作
    for (const actions of Object.values(node.events)) {
      for (const action of actions) {
        if (action.type === "writeTag") {
          capabilities.add("mqtt");
        }
        if (action.type === "callApi") {
          capabilities.add("http");
        }
      }
    }
  }

  return {
    providers: Array.from(providers),
    datapoints: Array.from(datapoints.values()),
    capabilities: Array.from(capabilities),
  };
}
```

## 4. assetNormalize（资源归一化）

### 4.1 资源分层策略

| 存储模式     | 条件                 | 行为          |
| ------------ | -------------------- | ------------- |
| packaged     | size < 5MB           | 打包进 ifp    |
| external     | size >= 5MB 且有 CDN | 保留 URL 引用 |
| resourcePack | size >= 5MB 且需离线 | 单独资源包    |

### 4.2 实现

```typescript
async function assetNormalize(
  doc: DocumentModel,
  assets: Map<string, AssetInfo>
): Promise<NormalizeResult> {
  const packaged: AssetEntry[] = [];
  const external: AssetEntry[] = [];
  const resourcePacks: ResourcePack[] = [];
  const assetMapping: Record<string, string> = {};

  // 收集 schema 中引用的资源
  const usedAssetIds = collectUsedAssets(doc);

  for (const assetId of usedAssetIds) {
    const asset = assets.get(assetId);
    if (!asset) continue;

    // 决策存储模式
    let mode = asset.storageMode;
    if (!mode) {
      mode = asset.size < 5 * 1024 * 1024 ? "packaged" : "external";
    }

    switch (mode) {
      case "packaged":
        // 复制到输出目录
        const outputPath = `assets/${asset.contentHash}${getExtension(
          asset.name
        )}`;
        packaged.push({ assetId, outputPath, asset });
        assetMapping[assetId] = outputPath;
        break;

      case "external":
        // 验证 URL 可达性
        if (asset.externalUrl) {
          external.push({ assetId, url: asset.externalUrl, asset });
          assetMapping[assetId] = asset.externalUrl;
        }
        break;

      case "resourcePack":
        // 加入资源包
        addToResourcePack(resourcePacks, asset);
        assetMapping[assetId] = `resourcepack://${asset.packId}/${asset.name}`;
        break;
    }
  }

  // 重写 schema 中的资源引用
  const normalizedDoc = rewriteAssetRefs(doc, assetMapping);

  return {
    normalizedDoc,
    packaged,
    external,
    resourcePacks,
    assetMapping,
  };
}
```

### 4.3 重写资源引用

```typescript
function rewriteAssetRefs(
  doc: DocumentModel,
  mapping: Record<string, string>
): DocumentModel {
  const result = JSON.parse(JSON.stringify(doc));

  // 遍历所有节点
  for (const node of Object.values(result.nodesById)) {
    // 重写 props 中的资源引用
    rewritePropsAssets(node.props, mapping);

    // 重写 style 中的资源引用
    rewriteStyleAssets(node.style, mapping);
  }

  return result;
}

function rewritePropsAssets(props: any, mapping: Record<string, string>): void {
  for (const [key, value] of Object.entries(props)) {
    if (typeof value === "object" && value?.kind === "asset") {
      const newPath = mapping[value.assetId];
      if (newPath) {
        props[key] = { ...value, resolvedUri: newPath };
      }
    }
  }
}
```

## 5. bundleIFP（打包）

### 5.1 IFP 结构

```
my-project-v1.0.0.ifp (zip)
├── manifest.json           # 清单文件
├── project.json            # 设计器 Schema（页面、组件、绑定）
├── datacenter.json         # 数据中心配置（连接、查询、数据点定义）
├── assets/                 # 打包的资源
│   ├── abc123.png
│   └── def456.svg
└── assets-mapping.json     # 资源映射表
```

> **datacenter.json** 包含运行时建立数据连接所需的配置：
>
> - 数据连接定义（type、name、capabilities）
> - SQL 查询定义（不含实际连接参数）
> - MQTT 订阅/变量组定义
> - 数据点元信息（path、dataType、sourceType）
>
> **注意**：实际的连接参数（host、port、credentials）由部署时的 **ConnectionProfile** 提供，不打包进 IFP。

### 5.2 实现

```typescript
async function bundleIFP(
  manifest: Manifest,
  doc: DocumentModel,
  datacenterConfig: DataCenterConfig,
  packaged: AssetEntry[],
  assetMapping: Record<string, string>
): Promise<Blob> {
  const zip = new JSZip();

  // 添加 manifest
  zip.file("manifest.json", JSON.stringify(manifest, null, 2));

  // 添加 project.json（设计器 Schema）
  zip.file("project.json", JSON.stringify(doc, null, 2));

  // 添加 datacenter.json（数据中心配置）
  zip.file("datacenter.json", JSON.stringify(datacenterConfig, null, 2));

  // 添加 assets-mapping
  zip.file("assets-mapping.json", JSON.stringify(assetMapping, null, 2));

  // 添加打包的资源
  for (const entry of packaged) {
    const content = await fetchAssetContent(entry.asset.url);
    zip.file(entry.outputPath, content);
  }

  return zip.generateAsync({ type: "blob" });
}

// DataCenter 配置结构
interface DataCenterConfig {
  // 数据连接定义（不含实际连接参数）
  connections: Array<{
    id: string;
    name: string;
    type: "relational" | "mqtt";
    capabilities: string[];
  }>;

  // SQL 查询定义
  queries: Array<{
    id: string;
    connectionId: string;
    name: string;
    sql: string;
    params?: QueryParam[];
    refreshInterval?: number;
  }>;

  // MQTT 订阅定义
  mqttSubscriptions: Array<{
    id: string;
    connectionId: string;
    topic: string;
    parseMode: string;
  }>;

  // 数据点元信息
  datapoints: Array<{
    id: string;
    path: string;
    sourceType: string;
    dataType: string;
    connectionId: string;
  }>;
}
```

## 6. createSnapshot（创建快照）

### 6.1 快照内容

```typescript
interface ProjectSnapshot {
  // 工程 Schema
  project: DocumentModel;

  // 全局配置
  settings: DesignProjectSettings;

  // 运行时角色
  roles: DesignRole[];

  // 自定义组件
  customComponents: CustomComponent[];

  // 资源映射
  assetMapping: Record<string, string>;

  // 清单
  manifest: Manifest;
}
```

### 6.2 实现

```typescript
async function createSnapshot(
  deploymentId: string,
  snapshot: ProjectSnapshot
): Promise<SnapshotInfo> {
  // 序列化并压缩
  const content = JSON.stringify(snapshot);
  const compressed = await compress(content);

  // 计算哈希
  const hash = await sha256(compressed);

  // 上传到对象存储
  const url = await uploadToStorage(
    `snapshots/${snapshot.project.project.projectId}/${deploymentId}/snapshot.tar.gz`,
    compressed
  );

  // 写入数据库
  await db.deploymentSnapshots.create({
    id: generateId(),
    deploymentId,
    snapshotUrl: url,
    snapshotHash: hash,
    snapshotSize: compressed.byteLength,
    snapshotType: "full",
    createdAt: new Date(),
  });

  return { url, hash, size: compressed.byteLength };
}
```

## 7. 完整流水线

```typescript
class PublishPipeline {
  async publish(
    projectId: string,
    options: PublishOptions
  ): Promise<PublishResult> {
    const { version, type = "development" } = options;

    // 1. 加载工程（Designer Schema + DataCenter 配置）
    const doc = await this.loadProject(projectId);
    const assets = await this.loadAssets(projectId);
    const datacenterConfig = await this.loadDataCenterConfig(projectId);

    // 2. 校验（包括数据点有效性）
    const validation = await validate(doc, datacenterConfig);
    if (!validation.valid) {
      return { success: false, errors: validation.errors };
    }

    // 3. 编译清单
    const manifest = compileManifest(doc, datacenterConfig, version);

    // 4. 资源归一化
    const normalized = await assetNormalize(doc, assets);

    // 5. 打包（包含 datacenter.json）
    const bundle = await bundleIFP(
      manifest,
      normalized.normalizedDoc,
      datacenterConfig,
      normalized.packaged,
      normalized.assetMapping
    );

    // 6. 计算哈希
    const hash = await sha256(bundle);

    // 7. 上传
    const artifactUrl = await this.uploadBundle(projectId, version, bundle);

    // 8. 创建版本记录
    const deployment = await db.deployments.create({
      id: generateId(),
      projectId,
      version,
      type,
      status: "success",
      artifactUrl,
      artifactHash: hash,
      pageCount: Object.keys(doc.pagesById).length,
      componentCount: Object.keys(doc.nodesById).length,
      deployedBy: this.userId,
      startedAt: this.startTime,
      completedAt: new Date(),
    });

    // 9. 创建快照
    await createSnapshot(deployment.id, {
      project: normalized.normalizedDoc,
      settings: await this.loadSettings(projectId),
      roles: await this.loadRoles(projectId),
      customComponents: await this.loadCustomComponents(projectId),
      assetMapping: normalized.assetMapping,
      manifest,
    });

    return {
      success: true,
      deployment,
      warnings: validation.warnings,
      resourcePacks: normalized.resourcePacks,
    };
  }
}
```

## 8. UI 集成（dev_ide）

> **注意**：发布操作在 dev_ide 的工程卡片上触发，不是在 Designer 内部。

### 8.1 发布对话框（dev_ide 工程卡片）

```
┌─ 发布工程 ──────────────────────────────────────────┐
│                                                      │
│  版本号: [ 1.0.0 ]                                  │
│  部署类型: [● 开发] [ 预发] [ 生产]                 │
│                                                      │
│  ─────────────────────────────────────────────────  │
│                                                      │
│  包含内容:                                           │
│  ✅ Designer Schema (5 页面, 42 组件)               │
│  ✅ DataCenter 配置 (2 连接, 8 数据点)              │
│  ✅ 资源文件 (12 个, 共 2.3MB)                      │
│                                                      │
│  ─────────────────────────────────────────────────  │
│                                                      │
│  校验结果:                                           │
│  ✅ Schema 版本正常                                  │
│  ✅ 入口页面存在                                     │
│  ✅ 角色引用正确                                     │
│  ⚠️ 1 个可选数据点状态异常                          │
│                                                      │
│  资源统计:                                           │
│  • 打包资源: 12 个 (2.3 MB)                         │
│  • 外部资源: 3 个                                    │
│  • 资源包: 1 个 (需单独下载)                        │
│                                                      │
│                        [取消]    [发布]             │
└──────────────────────────────────────────────────────┘
```

### 8.2 发布进度

```
┌─ 发布进度 ──────────────────────────────────────────┐
│                                                      │
│  [============================      ] 75%           │
│                                                      │
│  ✅ 校验完成                                         │
│  ✅ 编译清单完成                                     │
│  ✅ 资源处理完成                                     │
│  ⏳ 打包中...                                        │
│  ○ 上传制品                                          │
│  ○ 创建快照                                          │
│                                                      │
└──────────────────────────────────────────────────────┘
```

## 9. 部署管理（dev_ide 运维界面）

发布产生 IFP 包后，需要在运维管理界面部署到节点。

### 9.1 节点管理

```
┌─ 节点管理 ──────────────────────────────────────────────────────────────┐
│                                                                          │
│  [+ 注册节点]    [刷新]                                                 │
│                                                                          │
│  节点列表:                                                               │
│  ┌────────────────┬──────────┬──────────────┬────────────┬──────────┐   │
│  │ 节点名称       │ 状态     │ 当前工程     │ 版本       │ 操作     │   │
│  ├────────────────┼──────────┼──────────────┼────────────┼──────────┤   │
│  │ 车间大屏-01    │ 🟢 在线  │ 产线看板     │ v1.2.0     │ [详情]   │   │
│  │ 车间大屏-02    │ 🟢 在线  │ 产线看板     │ v1.2.0     │ [详情]   │   │
│  │ 办公室看板     │ 🟡 离线  │ -            │ -          │ [详情]   │   │
│  │ 测试节点       │ 🟢 在线  │ 测试工程     │ v0.1.0     │ [详情]   │   │
│  └────────────────┴──────────┴──────────────┴────────────┴──────────┘   │
└──────────────────────────────────────────────────────────────────────────┘
```

### 9.2 部署工程到节点

```
┌─ 部署工程 ─────────────────────────────────────────────────────────────┐
│                                                                         │
│  工程: 产线A看板                                                        │
│  版本: [ v1.2.0 ▼ ]                                                     │
│                                                                         │
│  目标节点:                                                              │
│  ☑ 车间大屏-01 (当前: v1.1.0)                                          │
│  ☑ 车间大屏-02 (当前: v1.1.0)                                          │
│  ☐ 办公室看板 (离线)                                                   │
│                                                                         │
│  ─────────────────────────────────────────────────────────────────────  │
│                                                                         │
│  部署配置:                                                              │
│                                                                         │
│  运行端口: [ 8080 ]                                                     │
│  目标终端: [ bigscreen ▼ ]                                              │
│  默认语言: [ 简体中文 ▼ ]                                               │
│  默认主题: [ 浅色主题 ▼ ]                                               │
│                                                                         │
│                                     [取消]    [部署]                   │
└─────────────────────────────────────────────────────────────────────────┘
```

> **注意**：数据连接配置在工程设置中预先配置，不在部署时设置。

### 9.3 部署约束

- **同一工程可部署到多个节点**（如上图选择多个节点）
- **同一节点只能运行一个工程**（部署新工程会替换旧工程）
- **数据连接配置**：在工程设置中预先配置，随 IFP 包发布

### 9.4 部署流程

```typescript
async function deployToNode(
  deploymentId: string,
  nodeId: string,
  config: DeployConfig
): Promise<DeployResult> {
  // 1. 获取 IFP 包信息
  const deployment = await db.deployments.findById(deploymentId);

  // 2. 检查节点状态
  const node = await nodeService.getNode(nodeId);
  if (node.status !== "online") {
    return { success: false, error: "节点离线" };
  }

  // 3. 检查是否已有运行中工程（同一节点只能运行一个）
  if (node.currentDeploymentId && node.currentDeploymentId !== deploymentId) {
    // 提示用户将替换现有工程
  }

  // 4. 下发部署指令（数据连接配置已在 IFP 中）
  await nodeService.sendCommand(nodeId, {
    type: "deploy",
    payload: {
      artifactUrl: deployment.artifactUrl,
      artifactHash: deployment.artifactHash,
      runtimeConfig: {
        port: config.port,
        uiTarget: config.uiTarget,
        defaultLocale: config.defaultLocale,
        defaultTheme: config.defaultTheme,
      },
    },
  });

  // 5. 记录部署关系
  await db.nodeDeployments.upsert({
    nodeId,
    deploymentId,
    runtimeConfig: config.runtimeConfig,
    deployedAt: new Date(),
    deployedBy: this.userId,
  });

  return { success: true };
}
```

## 10. 实现步骤

1. **实现校验模块** - validator.ts, bindingValidator.ts
2. **实现清单编译** - manifestCompiler.ts
3. **实现资源归一化** - assetNormalizer.ts
4. **实现打包** - bundler.ts（含 datacenter.json）
5. **实现快照** - snapshotService.ts
6. **集成流水线** - publishPipeline.ts
7. **实现发布 UI** - dev_ide/PublishDialog.vue
8. **实现部署 UI** - dev_ide/DeployDialog.vue
9. **实现节点管理** - dev_ide/NodeManager.vue

## 11. 运行架构设计

### 11.1 两种运行模式

系统设计支持两种运行模式，共享同一套核心代码：

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          运行模式架构                                        │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────┐  ┌─────────────────────────────────┐  │
│  │      Server 模式 (联网)          │  │   Standalone 模式 (单机桌面)    │  │
│  ├─────────────────────────────────┤  ├─────────────────────────────────┤  │
│  │                                 │  │                                 │  │
│  │  NodeAgent (Electron)           │  │  Standalone App (Electron)      │  │
│  │  ├─ 监测工程状态                │  │  ├─ 内嵌 RuntimeEngine          │  │
│  │  ├─ 接收部署指令                │  │  ├─ 内嵌 DataService            │  │
│  │  └─ 上报健康状态                │  │  ├─ BrowserWindow 渲染          │  │
│  │           │                     │  │  └─ 本地数据连接                │  │
│  │           ▼                     │  │                                 │  │
│  │  RuntimeEngine (后台服务)        │  │  无需浏览器                      │  │
│  │  ├─ HTTP Server                 │  │  无需网络（可选）               │  │
│  │  ├─ WebSocket Server            │  │                                 │  │
│  │  └─ 数据连接                    │  │                                 │  │
│  │           │                     │  │                                 │  │
│  │           ▼                     │  │                                 │  │
│  │  浏览器访问 (http://node:8080)  │  │                                 │  │
│  │                                 │  │                                 │  │
│  └─────────────────────────────────┘  └─────────────────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 11.2 核心代码复用

```typescript
// 抽象渲染宿主接口
interface RuntimeHost {
  // 获取项目配置
  getProjectConfig(): ProjectConfig;

  // 资源加载
  loadAsset(uri: string): Promise<string | Blob>;

  // 数据服务（统一接口）
  getDataService(): DataService;

  // 导航
  navigate(path: string): void;

  // 日志
  log(level: LogLevel, message: string, meta?: object): void;
}

// Server 模式宿主实现
class BrowserHost implements RuntimeHost {
  getDataService(): DataService {
    // 通过 HTTP/WebSocket 连接后台服务
    return new RemoteDataService(this.config.apiBase);
  }

  loadAsset(uri: string): Promise<string> {
    return fetch(`${this.config.apiBase}/assets/${uri}`).then((r) => r.text());
  }
}

// Standalone 模式宿主实现
class ElectronHost implements RuntimeHost {
  getDataService(): DataService {
    // 直接使用本地 DataService（进程内）
    return new LocalDataService(this.config);
  }

  loadAsset(uri: string): Promise<string> {
    // 从本地文件系统加载
    return fs.readFile(path.join(this.appPath, "assets", uri), "utf-8");
  }
}

// RuntimeEngine 不关心宿主类型
class RuntimeEngine {
  constructor(private host: RuntimeHost) {}

  async start(): Promise<void> {
    const config = this.host.getProjectConfig();
    const dataService = this.host.getDataService();
    // 统一的启动逻辑...
  }
}
```

### 11.3 Server 模式架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           节点 (Server 模式)                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                      NodeAgent (Electron)                            │   │
│  │                                                                      │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐               │   │
│  │  │ 状态监测      │  │ 指令接收     │  │ 健康上报      │               │   │
│  │  │              │  │              │  │              │               │   │
│  │  │ - 进程存活   │  │ - 部署       │  │ - CPU/内存   │               │   │
│  │  │ - 端口监听   │  │ - 启动/停止  │  │ - 工程状态   │               │   │
│  │  │ - 连接状态   │  │ - 回滚       │  │ - 数据连接   │               │   │
│  │  └──────────────┘  └──────────────┘  └──────────────┘               │   │
│  │                           │                                          │   │
│  └───────────────────────────┼──────────────────────────────────────────┘   │
│                              │ 管理                                         │
│                              ▼                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                  RuntimeEngine Service (Node.js)                     │   │
│  │                                                                      │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐               │   │
│  │  │ HTTP Server  │  │ WebSocket    │  │ DataService  │               │   │
│  │  │ :8080        │  │ /socket.io   │  │              │               │   │
│  │  │              │  │              │  │ - MQTT       │               │   │
│  │  │ - 静态资源   │  │ - 数据推送   │  │ - MySQL      │               │   │
│  │  │ - API 代理   │  │ - 状态同步   │  │ - HTTP       │               │   │
│  │  └──────────────┘  └──────────────┘  └──────────────┘               │   │
│  │                                                                      │   │
│  │  工程目录: /opt/induforge/projects/proj_xxx/                        │   │
│  │  ├── manifest.json                                                  │   │
│  │  ├── project.json                                                   │   │
│  │  ├── datacenter.json                                                │   │
│  │  └── assets/                                                        │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                              │                                              │
│                              ▼ 浏览器访问                                   │
│                    http://192.168.1.100:8080                               │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 11.4 Standalone 模式架构（低优先级，预留设计）

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    Standalone App (单机 Electron 桌面应用)                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                      Main Process (主进程)                           │   │
│  │                                                                      │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐               │   │
│  │  │ 应用管理     │  │ DataService  │  │ IPC Bridge   │               │   │
│  │  │              │  │ (进程内)     │  │              │               │   │
│  │  │ - 启动/退出  │  │              │  │ - 数据推送   │               │   │
│  │  │ - 全屏/窗口  │  │ - MQTT 客户端│  │ - 动作执行   │               │   │
│  │  │ - 托盘图标   │  │ - DB 连接    │  │ - 资源加载   │               │   │
│  │  └──────────────┘  └──────────────┘  └──────────────┘               │   │
│  │                                                                      │   │
│  └───────────────────────────────────┬──────────────────────────────────┘   │
│                                      │ IPC                                  │
│  ┌───────────────────────────────────┴──────────────────────────────────┐   │
│  │                    Renderer Process (渲染进程)                        │   │
│  │                                                                      │   │
│  │  ┌─────────────────────────────────────────────────────────────┐    │   │
│  │  │                    BrowserWindow                             │    │   │
│  │  │                                                              │    │   │
│  │  │  ┌────────────────────────────────────────────────────────┐ │    │   │
│  │  │  │              RuntimeEngine (Vue App)                    │ │    │   │
│  │  │  │                                                         │ │    │   │
│  │  │  │  - 页面渲染 (与 Server 模式相同)                        │ │    │   │
│  │  │  │  - Canvas 图形                                          │ │    │   │
│  │  │  │  - 数据绑定（通过 IPC 获取数据）                        │ │    │   │
│  │  │  │  - 动作执行（通过 IPC 调用主进程）                      │ │    │   │
│  │  │  │                                                         │ │    │   │
│  │  │  └────────────────────────────────────────────────────────┘ │    │   │
│  │  │                                                              │    │   │
│  │  └─────────────────────────────────────────────────────────────┘    │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  打包产物: InduForge-产线看板-Setup.exe (含 IFP + Electron Runtime)        │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 11.5 Standalone 模式导出

```typescript
// 导出配置
interface StandaloneExportConfig {
  projectId: string;
  version: string;

  // 目标平台
  platform: "win32" | "darwin" | "linux";
  arch: "x64" | "arm64";

  // 应用配置
  appName: string;
  icon: string;

  // 窗口配置
  window: {
    width: number;
    height: number;
    fullscreen: boolean;
    kiosk: boolean; // Kiosk 模式（全屏+禁止退出）
    frame: boolean; // 是否显示窗口边框
  };

  // 数据连接（嵌入配置，可覆盖）
  datacenter: DatacenterConfig;

  // 功能开关
  features: {
    devTools: boolean; // 是否允许打开开发者工具
    autoUpdate: boolean; // 是否自动更新
  };
}

// 导出流程
async function exportStandalone(
  config: StandaloneExportConfig
): Promise<string> {
  // 1. 获取 IFP 包
  const ifpPath = await downloadIFP(config.projectId, config.version);

  // 2. 准备 Electron 模板
  const templatePath = await prepareElectronTemplate(
    config.platform,
    config.arch
  );

  // 3. 合并 IFP 到模板
  await mergeIFPToTemplate(ifpPath, templatePath, config);

  // 4. 生成主进程代码
  await generateMainProcess(templatePath, config);

  // 5. 打包
  const outputPath = await packageElectron(templatePath, config);

  return outputPath; // 如: InduForge-产线看板-win32-x64-Setup.exe
}
```

### 11.6 IPC 数据桥接（Standalone 模式）

```typescript
// preload.js - 暴露安全的 API 给渲染进程
const { contextBridge, ipcRenderer } = require("electron");

contextBridge.exposeInMainWorld("induforge", {
  // 数据服务
  dataService: {
    subscribe: (path: string, callback: (value: any) => void) => {
      const channel = `data:${path}`;
      ipcRenderer.on(channel, (_, value) => callback(value));
      ipcRenderer.send("data:subscribe", path);

      return () => {
        ipcRenderer.removeAllListeners(channel);
        ipcRenderer.send("data:unsubscribe", path);
      };
    },

    getValue: (path: string) => ipcRenderer.invoke("data:getValue", path),
    setValue: (path: string, value: any) =>
      ipcRenderer.invoke("data:setValue", path, value),
  },

  // 动作执行
  action: {
    execute: (action: Action) => ipcRenderer.invoke("action:execute", action),
  },

  // 资源加载
  asset: {
    load: (uri: string) => ipcRenderer.invoke("asset:load", uri),
  },

  // 应用控制
  app: {
    minimize: () => ipcRenderer.send("app:minimize"),
    maximize: () => ipcRenderer.send("app:maximize"),
    close: () => ipcRenderer.send("app:close"),
    setFullscreen: (flag: boolean) => ipcRenderer.send("app:fullscreen", flag),
  },
});

// main.js - 主进程处理
class StandaloneMain {
  private dataService: LocalDataService;
  private mainWindow: BrowserWindow;

  async start(): Promise<void> {
    // 1. 初始化数据服务
    this.dataService = new LocalDataService(this.loadDatacenterConfig());
    await this.dataService.connect();

    // 2. 创建窗口
    this.mainWindow = new BrowserWindow({
      ...this.config.window,
      webPreferences: {
        preload: path.join(__dirname, "preload.js"),
        contextIsolation: true,
        nodeIntegration: false,
      },
    });

    // 3. 加载应用
    this.mainWindow.loadFile("renderer/index.html");

    // 4. 设置 IPC 处理
    this.setupIPC();
  }

  private setupIPC(): void {
    // 数据订阅
    ipcMain.on("data:subscribe", (event, path) => {
      this.dataService.subscribe(path, (value) => {
        event.sender.send(`data:${path}`, value);
      });
    });

    // 数据获取
    ipcMain.handle("data:getValue", (_, path) => {
      return this.dataService.getValue(path);
    });

    // 数据写入
    ipcMain.handle("data:setValue", (_, path, value) => {
      return this.dataService.setValue(path, value);
    });

    // 动作执行
    ipcMain.handle("action:execute", (_, action) => {
      return this.actionExecutor.execute(action);
    });
  }
}
```

## 12. NodeAgent 详细设计

### 12.1 职责边界

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              NodeAgent 职责                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ✅ 做什么：                           ❌ 不做什么：                        │
│  ├─ 监测 RuntimeEngine 进程状态        ├─ 不运行工程（工程在独立服务）      │
│  ├─ 接收 dev_ide 的部署指令            ├─ 不处理业务逻辑                    │
│  ├─ 启动/停止/重启 RuntimeEngine       ├─ 不管理数据连接                    │
│  ├─ 上报节点健康状态                   ├─ 不渲染页面                        │
│  ├─ 管理本地工程文件                   └─ 不处理用户交互                    │
│  ├─ 提供本地管理界面（托盘）                                                │
│  └─ 版本回滚                                                                │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 12.2 架构

```typescript
// NodeAgent 主类
class NodeAgent {
  private config: NodeAgentConfig;
  private websocket: WebSocket; // 与 dev_ide 通信
  private engineManager: EngineManager; // 管理 RuntimeEngine 进程
  private tray: Tray; // 系统托盘

  async start(): Promise<void> {
    // 1. 加载配置
    this.config = await this.loadConfig();

    // 2. 连接 dev_ide
    await this.connectToIDE();

    // 3. 启动健康检查
    this.startHealthCheck();

    // 4. 初始化托盘
    this.initTray();

    // 5. 恢复上次运行的工程
    await this.engineManager.restoreLastState();
  }
}

// 引擎管理器
class EngineManager {
  private engines: Map<string, EngineProcess> = new Map();

  // 部署工程
  async deploy(ifpPath: string, config: DeployConfig): Promise<void> {
    // 1. 解压 IFP 到工程目录
    const projectPath = await this.extractIFP(ifpPath);

    // 2. 停止旧版本（如果存在）
    const existingEngine = this.engines.get(config.projectId);
    if (existingEngine) {
      await existingEngine.stop();
    }

    // 3. 启动新版本
    const engine = new EngineProcess(projectPath, config);
    await engine.start();

    this.engines.set(config.projectId, engine);
  }

  // 回滚
  async rollback(projectId: string, version: string): Promise<void> {
    // 从本地备份恢复指定版本
    const backupPath = this.getBackupPath(projectId, version);
    await this.deploy(backupPath, this.getConfig(projectId));
  }
}

// 引擎进程封装
class EngineProcess {
  private process: ChildProcess | null = null;
  private port: number;

  async start(): Promise<void> {
    this.process = spawn("node", [
      path.join(this.projectPath, "runtime-server.js"),
      "--port",
      String(this.port),
      "--config",
      path.join(this.projectPath, "runtime-config.json"),
    ]);

    // 监听进程事件
    this.process.on("exit", (code) => this.handleExit(code));
    this.process.on("error", (err) => this.handleError(err));

    // 等待端口就绪
    await this.waitForReady();
  }

  async stop(): Promise<void> {
    if (this.process) {
      this.process.kill("SIGTERM");
      await this.waitForExit();
    }
  }

  getStatus(): EngineStatus {
    return {
      running: this.process !== null && this.process.exitCode === null,
      port: this.port,
      uptime: this.getUptime(),
      lastError: this.lastError,
    };
  }
}
```

### 12.3 与 dev_ide 通信协议

```typescript
// 消息类型
type NodeMessage =
  | {
      type: "register";
      nodeId: string;
      nodeName: string;
      capabilities: string[];
    }
  | { type: "heartbeat"; status: NodeStatus }
  | {
      type: "deployResult";
      deploymentId: string;
      success: boolean;
      error?: string;
    }
  | { type: "engineStatus"; projectId: string; status: EngineStatus }
  | { type: "log"; level: LogLevel; message: string; meta?: object };

type IDEMessage =
  | {
      type: "deploy";
      deploymentId: string;
      ifpUrl: string;
      config: DeployConfig;
    }
  | { type: "stop"; projectId: string }
  | { type: "restart"; projectId: string }
  | { type: "rollback"; projectId: string; version: string }
  | { type: "getStatus" }
  | { type: "getLogs"; projectId: string; lines: number };

// 节点状态
interface NodeStatus {
  nodeId: string;
  online: boolean;
  uptime: number;
  system: {
    cpu: number;
    memory: number;
    disk: number;
  };
  engines: Record<string, EngineStatus>;
}
```

### 12.4 托盘界面

```
┌─ NodeAgent 托盘菜单 ─────────────────┐
│                                      │
│  节点: 车间大屏-01                   │
│  状态: 🟢 在线                       │
│                                      │
│  ─────────────────────────────────   │
│                                      │
│  运行中工程:                         │
│  ├─ 📊 产线看板 v1.2.0              │
│  │   └─ http://localhost:8080       │
│  │       [打开] [重启] [停止]       │
│  │                                   │
│  └─ (无其他工程)                     │
│                                      │
│  ─────────────────────────────────   │
│                                      │
│  [查看日志]                          │
│  [设置]                              │
│  [退出]                              │
│                                      │
└──────────────────────────────────────┘
```

## 13. 节点注册与健康检查

### 13.1 节点注册流程

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            节点注册流程                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  1. 管理员在 dev_ide 生成节点注册码                                         │
│     ┌──────────────────────────────────────────────────────────────────┐   │
│     │  注册码: ABC123-XYZ789 (有效期 24 小时)                          │   │
│     └──────────────────────────────────────────────────────────────────┘   │
│                                        │                                    │
│                                        ▼                                    │
│  2. 在目标节点上安装并启动 NodeAgent                                        │
│     $ ./nodeagent --register ABC123-XYZ789 --name "车间大屏-01"            │
│                                        │                                    │
│                                        ▼                                    │
│  3. NodeAgent 向 dev_ide 验证注册码                                        │
│     POST /api/v1/nodes/register                                            │
│     { code: "ABC123-XYZ789", name: "车间大屏-01", ... }                    │
│                                        │                                    │
│                                        ▼                                    │
│  4. dev_ide 返回节点凭证                                                   │
│     { nodeId: "node_xxx", token: "jwt...", wsUrl: "wss://..." }           │
│                                        │                                    │
│                                        ▼                                    │
│  5. NodeAgent 保存凭证，建立 WebSocket 长连接                              │
│                                        │                                    │
│                                        ▼                                    │
│  6. 节点出现在 dev_ide 节点列表中                                          │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 13.2 健康检查机制

```typescript
// 健康检查配置
interface HealthCheckConfig {
  heartbeatInterval: number; // 心跳间隔，默认 30s
  timeoutThreshold: number; // 超时判定，默认 90s (3 次心跳)
  reconnectInterval: number; // 断线重连间隔，默认 5s
  maxReconnectAttempts: number; // 最大重连次数，默认 10
}

// 健康状态
interface HealthStatus {
  timestamp: number;
  system: {
    cpu: number; // CPU 使用率 (%)
    memory: number; // 内存使用率 (%)
    disk: number; // 磁盘使用率 (%)
  };
  engines: {
    [projectId: string]: {
      running: boolean;
      port: number;
      uptime: number; // 运行时长 (s)
      connections: number; // 当前连接数
      datapoints: {
        total: number;
        connected: number;
        error: number;
      };
      lastError?: string;
    };
  };
}

// dev_ide 侧的节点状态判定
class NodeStatusManager {
  private lastHeartbeat: Map<string, number> = new Map();

  updateHeartbeat(nodeId: string, status: HealthStatus): void {
    this.lastHeartbeat.set(nodeId, Date.now());
    // 更新数据库
    db.nodes.update(nodeId, { lastSeen: new Date(), status });
  }

  checkNodeStatus(nodeId: string): "online" | "offline" | "unknown" {
    const lastSeen = this.lastHeartbeat.get(nodeId);
    if (!lastSeen) return "unknown";

    const elapsed = Date.now() - lastSeen;
    return elapsed < this.config.timeoutThreshold * 1000 ? "online" : "offline";
  }
}
```

## 14. 版本回滚

### 14.1 回滚策略

```typescript
// 版本管理
interface VersionManager {
  // 获取可回滚版本列表
  listVersions(projectId: string): Promise<DeploymentVersion[]>;

  // 执行回滚
  rollback(nodeId: string, projectId: string, version: string): Promise<void>;

  // 自动回滚条件
  checkAutoRollback(nodeId: string, projectId: string): Promise<boolean>;
}

interface DeploymentVersion {
  version: string;
  deployedAt: Date;
  status: "active" | "previous" | "archived";
  canRollback: boolean;
  backupPath?: string; // 本地备份路径
  ifpUrl?: string; // 远程备份 URL
}

// 自动回滚条件
interface AutoRollbackConfig {
  enabled: boolean;
  conditions: {
    startupFailure: boolean; // 启动失败自动回滚
    healthCheckFailure: boolean; // 健康检查连续失败
    failureThreshold: number; // 连续失败次数阈值
    rollbackWindow: number; // 回滚时间窗口 (ms)，部署后多久内可自动回滚
  };
}
```

### 14.2 回滚 UI

```
┌─ 版本回滚 ─────────────────────────────────────────────────────────────────┐
│                                                                            │
│  工程: 产线看板                                                            │
│  当前版本: v1.2.0 (2026-01-07 14:30 部署)                                  │
│                                                                            │
│  ─────────────────────────────────────────────────────────────────────────│
│                                                                            │
│  可回滚版本:                                                               │
│  ┌─────────┬──────────────────┬─────────────────┬────────────┐            │
│  │ 版本    │ 部署时间         │ 状态            │ 操作       │            │
│  ├─────────┼──────────────────┼─────────────────┼────────────┤            │
│  │ v1.1.0  │ 2026-01-05 10:00 │ 📦 本地备份     │ [回滚]     │            │
│  │ v1.0.0  │ 2026-01-01 09:00 │ 📦 本地备份     │ [回滚]     │            │
│  │ v0.9.0  │ 2025-12-20 15:00 │ ☁️ 远程存储     │ [下载回滚] │            │
│  └─────────┴──────────────────┴─────────────────┴────────────┘            │
│                                                                            │
│  ⚠️ 回滚将停止当前版本并启动选定版本，可能导致短暂服务中断                  │
│                                                                            │
│                                                     [取消]  [确认回滚]     │
└────────────────────────────────────────────────────────────────────────────┘
```

## 15. 日志收集

### 15.1 日志分类

| 日志类型 | 来源          | 存储位置         | 保留时间 |
| -------- | ------------- | ---------------- | -------- |
| 系统日志 | NodeAgent     | 本地 + 远程上报  | 7 天     |
| 运行日志 | RuntimeEngine | 本地 + 远程上报  | 7 天     |
| 操作日志 | 部署/回滚操作 | 数据库           | 永久     |
| 数据日志 | 数据点变化    | 本地（按需开启） | 1 天     |
| 错误日志 | 异常/错误     | 本地 + 远程告警  | 30 天    |

### 15.2 日志格式

```typescript
interface LogEntry {
  timestamp: string;          // ISO 8601
  level: 'debug' | 'info' | 'warn' | 'error';
  source: 'nodeagent' | 'runtime' | 'dataservice';
  nodeId: string;
  projectId?: string;
  message: string;
  meta?: {
    traceId?: string;
    userId?: string;
    action?: string;
    duration?: number;
    error?: {
      name: string;
      message: string;
      stack?: string;
    };
    [key: string]: any;
  };
}

// 示例
{
  "timestamp": "2026-01-07T14:30:25.123Z",
  "level": "error",
  "source": "dataservice",
  "nodeId": "node_xxx",
  "projectId": "proj_yyy",
  "message": "MQTT connection failed",
  "meta": {
    "traceId": "abc123",
    "error": {
      "name": "ConnectionError",
      "message": "Connection refused",
      "stack": "..."
    },
    "broker": "mqtt://10.0.0.20:1883",
    "retryCount": 3
  }
}
```

### 15.3 日志查看

```
┌─ 节点日志 ─ 车间大屏-01 ───────────────────────────────────────────────────┐
│                                                                            │
│  [全部 ▼] [今天 ▼] [🔍 搜索...]                      [实时] [下载]        │
│                                                                            │
│  ─────────────────────────────────────────────────────────────────────────│
│                                                                            │
│  14:30:25 INFO   [runtime]  工程启动完成 port=8080                        │
│  14:30:24 INFO   [runtime]  数据服务连接成功 mqtt=EMQX mysql=生产库       │
│  14:30:20 INFO   [runtime]  加载工程配置 project=产线看板 version=1.2.0   │
│  14:30:18 INFO   [nodeagent] 部署完成 deployment=dep_xxx                   │
│  14:30:15 INFO   [nodeagent] 解压 IFP 包                                   │
│  14:30:10 INFO   [nodeagent] 收到部署指令 project=产线看板 version=1.2.0  │
│  14:29:00 WARN   [runtime]  数据点超时 path=mqtt.EMQX.温度组.temp          │
│  14:28:30 ERROR  [dataservice] MQTT 重连失败 broker=mqtt://10.0.0.20:1883 │
│  ...                                                                       │
│                                                                            │
│  [加载更多]                                                                │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

## 16. 测试要点

### 发布流水线

- [ ] 各校验规则正确性
- [ ] 数据点状态校验
- [ ] dataRequirements 提取正确性
- [ ] 资源分层决策正确性
- [ ] 资源引用重写正确性
- [ ] IFP 结构完整性（含 datacenter.json）
- [ ] 快照内容完整性
- [ ] 哈希计算一致性

### 部署管理

- [ ] 节点注册与状态监控
- [ ] 部署到单节点
- [ ] 批量部署到多节点
- [ ] 同一节点替换工程
- [ ] 部署回滚
- [ ] **NodeAgent 进程管理正确**
- [ ] **健康检查上报正常**
- [ ] **断线重连正常**
- [ ] **日志收集正常**

### Standalone 模式（低优先级）

- [ ] IFP 打包成桌面应用
- [ ] IPC 数据桥接正常
- [ ] 窗口控制（全屏/最小化）
- [ ] 本地数据连接正常

---

**相关文档**：

- [运行时引擎](./runtime-engine.md)
- [数据绑定 v2](./data-binding-v2.md)
- [高层设计 - 发布与部署](../../../高层设计.md#5-发布与部署流程)
