# HTTP 接入源工作台设计

## 背景与定位

HTTP 接入源不采用 MQTT、Kafka、OPC UA、Modbus、S7 的“变量管理”心智。HTTP 的核心对象是一次可保存、可发送、可复用的接口请求，因此工作台应复刻 Postman 的接口调试体验：一个 HTTP 接入源下管理多个接口，用户在开发态配置请求、发送测试、查看响应，并将接口整体响应作为一个对象型数据点写入平台。

HTTP 工作台仍是开发态能力，不承担长期轮询、定时采集或后台运行。后续如需定时执行 HTTP 请求，应由独立运行态编排能力承接。

## 核心心智

- 一个 HTTP 接入源等价于一个接口集合。
- 左侧树用于管理接口集合，包含分组和接口请求，体验类似 Postman Collection。
- 一个接口请求就是一个数据点，数据点类型固定为 `object`。
- 保存接口请求时自动创建或同步 `http.request` 数据点。
- 点击 `Send` 成功返回后，将本次完整响应写入对应数据点的最后值。
- 不做字段级变量，不做 `http.field`，不提供变量表格和变量分组。

## 页面布局

### 左侧集合树

左侧用于管理当前 HTTP 接入源下的请求集合：

- 顶部显示接入源名称和搜索框。
- 支持新建分组、新建接口、刷新。
- 分组可展开/收起，接口项显示 Method 和接口名称。
- 支持接口重命名、移动分组、删除。
- 支持分组重命名、移动、删除；删除分组时，组内接口移动到未分组或根集合。

接口项示例：

```text
InduForge HTTP
├─ 设备接口
│  ├─ GET  查询设备状态
│  └─ POST 写入设备参数
└─ 告警接口
   └─ GET  查询告警列表
```

### 主工作区

主区域复刻 Postman 请求发送界面，由上到下分为四层：

1. 多接口标签页
2. 面包屑与保存操作区
3. 请求编辑区
4. 响应查看区

#### 多接口标签页

面包屑上方提供接口标签页，用于同时开发多个接口：

- 从左侧点击接口时，如果标签页已打开则激活；否则新增标签页。
- 标签显示接口名称和未保存状态。
- 标签可关闭，最后一个标签关闭后显示空状态。
- 切换标签时保留当前编辑内容。
- 未保存标签关闭或切换离开时，应提示保存、放弃或取消。

#### 面包屑与保存

标签页下方展示当前接口位置：

```text
HTTP 接入源 / 设备接口 / 查询设备状态
```

右侧提供：

- `Save`：保存当前请求配置并同步数据点。
- 更多操作：复制接口、移动接口、删除接口。

保存成功后：

- 如果是新接口，创建接口记录。
- 如果数据点不存在，创建 `http.request` 数据点。
- 如果数据点已存在，同步名称、路径、启停状态和 source config。

#### 请求编辑区

请求行：

- Method 下拉：`GET`、`POST`、`PUT`、`PATCH`、`DELETE`。
- URL 输入框：支持完整 URL，也支持基于接入源 `baseUrl` 的相对路径。
- `Send` 按钮：执行一次开发态请求。

请求配置 Tabs：

- `Params`：Query 参数表格，列为启用、Key、Value、Description。
- `Authorization`：第一版支持 `No Auth`、`Bearer Token`、`Basic Auth`。
- `Headers`：Header 表格，列为启用、Key、Value、Description。
- `Body`：支持 `none`、`json`、`form-data`、`x-www-form-urlencoded`、`raw`。
- `Settings`：超时时间、跟随重定向、TLS 校验、响应解码方式。

第一版不实现 Postman Scripts、Tests、环境变量、Cookie 管理和 Mock Server。

#### 响应查看区

响应区位于页面下半部分，支持展开/收起和拖拽调整高度。

未发送时显示空状态：`Click Send to get a response`。

发送后展示：

- 状态码，例如 `200 OK`。
- 耗时，例如 `123 ms`。
- 响应大小。
- 响应时间。
- Tabs：
  - `Body`：JSON 美化、Raw 文本切换。
  - `Headers`：响应 Header 表格。
  - `History`：当前接口最近发送记录。

响应失败时展示错误摘要和诊断信息，不写入数据点最后值，数据点质量可更新为 `bad`。

## 数据模型与数据点

### 接口请求

新增 HTTP 请求记录，作用域为当前 HTTP 接入源：

- `project_id`
- `connection_id`
- `group_id`
- `name`
- `method`
- `url`
- `params`
- `headers`
- `auth`
- `body_type`
- `body`
- `settings`
- `sort_order`
- `created_by`
- `updated_by`
- `created_at`
- `updated_at`

### 请求分组

新增 HTTP 请求分组，作用域为当前 HTTP 接入源：

- `project_id`
- `connection_id`
- `parent_id`
- `name`
- `sort_order`
- `created_by`
- `updated_by`
- `created_at`
- `updated_at`

### 数据点规则

每个接口请求对应一个数据点：

```text
sourceType = http.request
dataType = object
path = http.<接入源名称>.<接口名称>
```

数据点 `sourceConfig` 至少包含：

```json
{
  "connectionId": "HTTP 接入源 ID",
  "requestId": "接口请求 ID",
  "method": "GET",
  "url": "/device/status"
}
```

保存接口时同步数据点：

- 新建接口：创建数据点。
- 编辑接口名称或 URL：同步数据点名称、路径和 source config。
- 删除接口：对应数据点标记为 `invalid`。
- 停用接口：对应数据点状态同步为不可用状态。

### Send 写回

点击 `Send` 成功返回后，将完整响应对象写入对应数据点最后值：

```json
{
  "status": 200,
  "statusText": "OK",
  "headers": {
    "content-type": "application/json"
  },
  "body": {
    "temperature": 23.5
  },
  "durationMs": 123,
  "sizeBytes": 512,
  "receivedAt": "2026-05-27 01:20:00"
}
```

写回规则：

- 请求成功且响应可解析：质量写为 `good`。
- 请求失败、超时或响应解析失败：质量写为 `bad`，保留错误信息。
- 未发送过：质量为 `unknown`。
- 最后值只来自工作台短时发送，不代表长期运行态采集。

## API 能力

新增 HTTP 工作台接口：

- `GET /api/v1/data/projects/{projectId}/http/sources/{connectionId}/request-groups`
- `POST /api/v1/data/projects/{projectId}/http/sources/{connectionId}/request-groups`
- `PUT /api/v1/data/projects/{projectId}/http/request-groups/{groupId}`
- `DELETE /api/v1/data/projects/{projectId}/http/request-groups/{groupId}`
- `GET /api/v1/data/projects/{projectId}/http/sources/{connectionId}/requests`
- `POST /api/v1/data/projects/{projectId}/http/sources/{connectionId}/requests`
- `GET /api/v1/data/projects/{projectId}/http/requests/{requestId}`
- `PUT /api/v1/data/projects/{projectId}/http/requests/{requestId}`
- `DELETE /api/v1/data/projects/{projectId}/http/requests/{requestId}`
- `POST /api/v1/data/projects/{projectId}/http/requests/{requestId}/send`

列表接口支持：

- `groupId`
- `q`
- `page`
- `pageSize`

`send` 接口执行一次开发态请求，并返回响应对象；成功或失败都会记录诊断，成功响应写回数据点最后值。

## 交互边界

- 不做字段级变量管理。
- 不做 `http.field` 数据点。
- 不做后台定时轮询。
- 不做脚本、测试断言、环境变量和 Cookie 管理。
- 不做接口响应字段自动拆点。
- 不做运行态订阅或采集任务。

## 验收标准

- HTTP 接入源进入工作台后展示 Postman 式左侧集合树和请求发送界面。
- 左侧可创建分组和接口，接口可在多个标签页中同时打开。
- 编辑接口配置后点击 `Save` 可保存，并自动生成 `http.request` 数据点。
- 点击 `Send` 可执行接口请求，响应区展示状态、耗时、Headers、Body。
- `Send` 成功后，对应数据点最后值写入完整响应对象，数据点类型为 `object`。
- 删除接口后，对应数据点标记为失效。
- HTTP 工作台不出现变量表格、变量管理、字段路径建模等 Kafka/MQTT 类界面。
