# 数据点与报警开发上下文 SDK 设计

## 1. 目标与范围

本设计统一计算脚本、AI 页面工程和后续节点运行时对数据点、报警及计算单元的使用方式。用户面向平台抽象编程，不感知关系库、时序库、Redis、MQTT、采集器或计算输出等来源差异。

本阶段交付开发态 SDK：

- 计算工作台在独立沙箱中注入数据点对象，支持语法检查、快照读取和受控副作用预览。
- AI 页面工程通过 `@induforge/runtime-sdk` 使用同一数据点模型和独立报警领域 API。
- 开发态使用数据中心保存的当前值、默认值或模拟值；发布后的真实读取、写入、订阅和报警动作由节点侧适配器执行。
- 数据中心不承担节点常驻订阅、定时调度、活动报警状态机和现场设备写入。

### 1.1 内置教程工程

系统首次提供该功能时创建一个稳定 ID 的普通教程工程，并为其配置 IF 关系库、IF 时序库、IF 实时库、IF 消息库、查询与订阅数据点、计算单元与报警项。设计器识别该 ID，在首次进入 AI 开发中心时自动初始化 Vue + JavaScript 示例模板。

教程工程不设置不可删除标记，继承普通工程的编辑、发布和删除语义。启动逻辑会同时查找活跃和已删除记录；删除后保留的墓碑会阻止再次创建。数据服务以幂等方式补齐教程资源，但不覆盖用户已编辑的数据和配置。

## 2. 领域划分

开发上下文包含三个并列领域：

```text
ctx
├── points       数据点：统一值、元数据和来源能力
├── alarms       报警：配置、当前状态、变化订阅、动作和历史
└── computes     计算单元：按身份运行或查询描述
```

报警不是数据点。数据点描述可读写的数据对象，报警描述规则、活动状态和处置过程。计算单元可以产生数据点，但运行计算属于计算领域方法。

## 3. 统一返回格式

所有可能失败的异步方法统一返回：

```ts
interface SDKResult<T> {
  code: number
  msg: string
  data: T | null
  reqId?: string
}
```

- `code = 0` 表示成功。
- 不支持的能力不抛出“方法不存在”，返回稳定的非零 `code`、明确 `msg` 和 `data: null`。
- 参数格式错误可以在调用前抛出 `TypeError`；远端业务错误仍使用 `SDKResult`。
- 计算沙箱中的对象与 AI 页面 SDK 使用相同语义。

首版 SDK 预留以下本地错误码：

| code    | 含义                                 |
| ------- | ------------------------------------ |
| `40031` | 当前数据点或当前运行环境不支持该能力 |
| `40032` | 数据点未声明、路径为空或参数无效     |
| `50031` | 页面宿主或节点适配器未提供对应能力   |

正式 HTTP API 仍遵守平台统一错误码表；这些值只用于 SDK 适配层的本地失败结果，后续如转为服务端错误码须同步统一错误码文档。

## 4. 数据点对象

### 4.1 稳定属性

所有来源的数据点都暴露同一组属性。不适用的可选属性返回 `null`，集合返回空集合，不因来源不同删除字段。

```ts
interface DataPoint<T = unknown> {
  id: string | null
  ref: string
  path: string
  name: string | null
  displayName: string | null
  dataType: string | null
  schema: Record<string, unknown> | null
  source: { type: string | null; id: string | null }
  status: string | null
  unit: string | null
  precision: number | null
  min: number | null
  max: number | null
  defaultValue: T | null
  tags: unknown[]
  attributes: Record<string, string>
  capabilities: DataPointCapabilities
}
```

`ref` 与 `path` 首版均使用数据点完整路径。`displayName` 未单独配置时回退到 `name`。

### 4.2 稳定方法

所有数据点对象始终具有以下方法：

| 方法                           | 语义                                         | 开发态                                      |
| ------------------------------ | -------------------------------------------- | ------------------------------------------- |
| `get(options?)`                | 获取业务值；按需型数据点可能触发一次来源执行 | 读取已预取快照                              |
| `read(options?)`               | 获取带质量与时间戳的完整样本                 | 读取已预取快照                              |
| `peek()`                       | 只读当前已物化值，不触发来源执行             | 读取已预取快照                              |
| `set(value, options?)`         | 写入可写数据点                               | 只生成受控副作用记录                        |
| `subscribe(handler, options?)` | 订阅后续变化                                 | 计算单次执行不支持；AI 页面由宿主适配器提供 |
| `history(query)`               | 查询历史样本                                 | 由页面/节点适配器提供                       |
| `refresh(options?)`            | 主动刷新可刷新来源                           | 计算开发态只生成副作用记录                  |
| `run(input?)`                  | 运行计算类数据点对应的计算单元               | 计算开发态只生成副作用记录                  |
| `execute(input?)`              | 执行按需查询或命令型来源                     | 计算开发态只生成副作用记录                  |
| `publish(payload, options?)`   | 向消息型数据点发布消息                       | 计算开发态只生成副作用记录                  |

`get()` 与 `peek()` 必须区分副作用。写 SQL 查询等按需型数据点允许 `get()` 时执行一次，这是数据点自身契约；订阅实现不得通过轮询此类 `get()` 模拟，否则会重复产生写入。`peek()` 永远不得触发来源执行。

完整样本格式：

```ts
interface DataPointSample<T> {
  path: string
  value: T
  quality: string
  timestamp: string | null
  observedAt: string | null
  sourceTimestamp: string | null
  status: string | null
}
```

### 4.3 能力说明

`capabilities` 至少包含与稳定方法同名的布尔值。方法始终存在，能力只说明能否成功执行。页面可据此禁用按钮，但脚本仍应检查 `SDKResult.code`，避免能力在运行期间变化。

首版计算开发态只真实支持快照类方法；其他操作生成 `sideEffects`，不修改开发数据库和现场设备。节点侧必须根据发布契约、角色和数据点运行权限再次鉴权。

## 5. 计算脚本使用方式

### 5.1 变量绑定

用户在“变量”面板选择数据点并填写脚本变量名，例如：

```text
变量名 temperature
数据点 db.IF时序库.device_temperature.value
```

沙箱把 `temperature` 直接注入为 `DataPoint` 对象。用户不需要编写 `ctx.points`：

```js
const current = await temperature.read()
if (current.code !== 0) return current
return current.data.value * 1.8 + 32
```

Python 同样直接使用：

```py
def main(argv, dp, ctx):
    current = temperature.read()
    if current.code != 0:
        return current
    return current.data["value"] * 1.8 + 32
```

`ctx.points` 保留为完整命名空间，主要用于生成代码、公共函数和动态选择；`dp` 是按变量名索引的数据点对象集合。三种入口引用同一个对象：

```js
temperature === dp.temperature
temperature === ctx.points.temperature
```

变量名只允许 JavaScript 与 Python 通用标识符：`^[A-Za-z_][A-Za-z0-9_]*$`，并拒绝关键字、双下划线前缀及 `ctx/dp/argv/console/require` 等平台保留名。

### 5.2 计算上下文

```ts
interface ComputeContext {
  points: Record<string, DataPoint>
  alarms: AlarmSDK
  computes: ComputeSDK
  sql: { query(key: string): unknown }
  args: Record<string, unknown>
  trigger: Record<string, unknown>
  runtime: Record<string, unknown>
  logger: Console
}
```

- `args` 是本次调试/运行输入。
- `trigger` 描述手动、定时、数据点变化或条件触发上下文。
- `runtime` 提供工程、计算单元和开发态标识。
- `sql.query(key)` 只读取配置阶段已声明并由服务端预执行的查询结果。
- 单次计算脚本不允许建立长生命周期订阅；数据点变化和条件执行应在“触发”面板声明。

### 5.3 开发态副作用

`set/refresh/run/execute/publish` 在首版开发沙箱中不直接操作目标，而是返回成功受理结果并追加：

```json
{
  "domain": "point",
  "operation": "set",
  "path": "realtime.IF实时库.device.speed",
  "payload": 1200
}
```

调试结果中的 `sideEffects` 用于用户确认脚本意图。节点运行时消费同一发布配置后，才根据能力和权限执行真实操作。

## 6. AI 页面开发 SDK

Vue/React/原生 JavaScript 页面统一安装 `@induforge/runtime-sdk`，宿主通过 `window.__INDUFORGE_RUNTIME__` 注入适配器、数据点契约、角色和导航能力。

```js
import { points, alarms } from '@induforge/runtime-sdk'

const sample = await points.factory.line1.temperature.read()
if (sample.code === 0) {
  console.log(sample.data.value, sample.data.quality)
}

const active = await alarms.current.list({ status: 'active' })
```

页面 SDK 不保存 token，不直接拼接数据服务 URL。开发预览和发布运行由宿主提供不同适配器，但页面代码不变。

## 7. 报警 SDK

报警使用独立命名空间：

```text
alarms
├── items       报警项定义：list/get
├── settings    工程报警配置：get/update
├── current     已物化的当前报警状态：list/get
├── changes     当前报警变化流：subscribe
├── actions     ack/unack/forceClear/shelve/unshelve
└── history     历史报警：list/get
```

“当前报警”是节点报警存储中的当前状态查询；“实时变化”专指 `changes.subscribe()`。两者不得混用。

开发态首版提供稳定对象和适配器转发：宿主未实现的方法返回 `50031`，不伪造活动报警。节点侧后续实现当前状态、变化订阅、确认、取消确认、强制清除、搁置和历史查询。

## 8. 开发态、预览态与节点运行态

| 能力           | 计算开发态            | AI 页面预览    | 节点运行态           |
| -------------- | --------------------- | -------------- | -------------------- |
| 数据点当前值   | 服务端预取快照/默认值 | 宿主开发适配器 | 节点当前值存储或来源 |
| 写入/发布/执行 | 记录副作用            | 由预览策略决定 | 真实执行并鉴权       |
| 数据点订阅     | 不支持，使用触发配置  | 宿主预览订阅   | 节点事件总线         |
| 报警配置       | 可读取开发配置        | 宿主 API       | 发布配置             |
| 当前/历史报警  | 不伪造                | 宿主能力决定   | 节点报警存储         |

发布 Artifact 必须保留数据点稳定 ID、路径、来源能力、变量别名和计算触发配置。节点不得信任开发态 `sideEffects`，必须重新执行脚本并验证权限。

## 9. 首版验收标准

1. 计算变量绑定后，JavaScript 与 Python 都能直接以变量名调用 `get/read/peek`。
2. `dp.<alias>` 与 `ctx.points.<alias>` 可访问同一数据点对象。
3. 不支持的方法返回统一失败结果；开发态可预览允许操作的 `sideEffects`。
4. Web SDK 暴露完整数据点方法、稳定属性及报警六个子域。
5. 现有场景、访问控制和导航 API 不回退。
6. 提供面向用户的计算 JavaScript/Python 示例和 Vue+JavaScript 页面示例。

## 10. 后续工作

- 节点侧实现真实数据点适配器、权限校验、订阅和副作用执行。
- 建立报警当前状态、变化流、动作和历史适配器。
- AI 开发中心根据数据点与报警契约生成类型提示、上下文说明和可运行代码片段。
- 将 SDK 本地错误码纳入统一错误码表，补充端到端发布 Artifact 契约测试。
