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

## 11. 测试要点

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
- [ ] ConnectionProfile 配置正确性
- [ ] 部署回滚

---

**相关文档**：

- [运行时引擎](./runtime-engine.md)
- [数据绑定 v2](./data-binding-v2.md)
- [高层设计 - 发布与部署](../../../高层设计.md#5-发布与部署流程)
