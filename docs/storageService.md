# storageService 使用说明

本文档说明 `dev_core/src/services/storageService.js` 与设计器资源上传的使用方式。

## 1. 依赖

已在 `dev_core/package.json` 中引入：
- `minio`

## 2. 环境变量配置

在项目根目录 `.env` 中配置：

```
MINIO_ENDPOINT=127.0.0.1
MINIO_PORT=9000
MINIO_USE_SSL=false
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_BUCKET_IFP=ifp-artifacts
MINIO_BUCKET_DESIGN=design-assets
MINIO_REGION=us-east-1
```

说明：
- `MINIO_BUCKET_IFP`：运维产物桶（IFP）
- `MINIO_BUCKET_DESIGN`：设计资源桶（图片、PDF 等）

## 3. 初始化时机

`dev_core/src/index.js` 启动时初始化：
1) Redis
2) MinIO（自动创建上述两个 bucket）

## 4. storageService API

### 4.1 初始化

```js
const { initStorage } = require('./services/storageService');
await initStorage();
```

### 4.2 上传对象

```js
const { uploadObject } = require('./services/storageService');

// 运维产物
await uploadObject('ifp', objectKey, stream, size, meta);

// 设计资源
await uploadObject('design', objectKey, stream, size, meta);
```

参数：
- `bucketType`：`ifp` 或 `design`
- `objectKey`：对象路径（建议包含租户/工程/日期等信息）
- `stream`：可读流
- `size`：文件大小
- `meta`：可选元信息

### 4.3 获取对象流

```js
const { getObjectStream } = require('./services/storageService');

const stream = await getObjectStream('design', objectKey);
```

### 4.4 获取对象元信息

```js
const { statObject } = require('./services/storageService');

const stat = await statObject('ifp', objectKey);
```

### 4.5 删除对象

```js
const { removeObject } = require('./services/storageService');

await removeObject('design', objectKey);
```

## 5. 设计资源上传（前端 + 后端）

### 5.1 资源数据模型

- 资源分组表：`design_asset_folders`
- 资源表：`design_assets`
- 资源支持分组（父子结构），资源必须归属某个分组或根目录。

### 5.2 资源 API 路由（dev_core）

路由文件：`dev_core/src/routes/v1/designAssets.js`

- 公开访问资源文件：
  - `GET /api/v1/design/projects/:projectId/assets/:assetId/file`

- 资源分组：
  - `GET /api/v1/design/projects/:projectId/asset-folders`
  - `POST /api/v1/design/projects/:projectId/asset-folders`
  - `PATCH /api/v1/design/projects/:projectId/asset-folders/:folderId`
  - `DELETE /api/v1/design/projects/:projectId/asset-folders/:folderId`

- 资源文件：
  - `GET /api/v1/design/projects/:projectId/assets`
  - `POST /api/v1/design/projects/:projectId/assets`
  - `PATCH /api/v1/design/projects/:projectId/assets/:assetId`
  - `POST /api/v1/design/projects/:projectId/assets/:assetId/copy`
  - `DELETE /api/v1/design/projects/:projectId/assets/:assetId`

### 5.3 上传参数（前端表单）

上传接口：`POST /api/v1/design/projects/:projectId/assets`

- `files`：文件列表（multipart/form-data）
- `folderId`：目标文件夹（可选）
- `conflictStrategy`：冲突策略（`replace` 或 `rename`）

前端调用封装：`designer/src/services/assetApi.js`。

### 5.4 设计器资源面板行为

- 拖拽文件到指定分组或上传区域完成上传。
- 同名冲突提示“替换/重命名”。
- 右键菜单支持：预览、查看详情、重命名、移动、复制、剪切、粘贴、删除。
- 预览支持图片与 PDF（通过公开文件接口）。

### 5.5 样式配置资源引用

样式配置弹窗右侧“资源库”会枚举当前工程资源，支持分组结构。双击资源会把资源的 `src` 自动插入到编辑器光标处。

## 6. 命名与权限建议

- **对象 Key 命名建议**：
  `tenants/{tenantId}/projects/{projectId}/{type}/{yyyyMMdd}/{sha256}.{ext}`
- **权限控制**：建议在业务层处理；存储服务只负责读写。
