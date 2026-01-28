# storageService 使用说明

本文档描述 `dev_core/src/services/storageService.js` 的使用方式与配置约定。

## 1. 依赖

已在 `dev_core/package.json` 中引入：
- `minio`

## 2. 环境变量配置

在项目根目录 `.env` 配置：

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
- `MINIO_BUCKET_IFP`：运维制品桶（IFP）
- `MINIO_BUCKET_DESIGN`：设计素材桶（图片/视频等）

## 3. 初始化时机

`dev_core/src/index.js` 已在启动时初始化：
1) Redis
2) MinIO（自动创建上述两个 bucket）

## 4. API 说明

### 4.1 初始化

```js
const { initStorage } = require('./services/storageService');
await initStorage();
```

### 4.2 上传对象

```js
const { uploadObject } = require('./services/storageService');

// 运维制品
await uploadObject('ifp', objectKey, stream, size, meta);

// 设计资产
await uploadObject('design', objectKey, stream, size, meta);
```

参数：
- `bucketType`：`ifp` 或 `design`
- `objectKey`：对象路径（建议包含租户/工程/日期）
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

## 5. 约定建议

- **对象 Key 命名**建议：  
  `tenants/{tenantId}/projects/{projectId}/{type}/{yyyyMMdd}/{sha256}.{ext}`
- **权限控制**建议在业务层做，存储服务仅负责读写。

