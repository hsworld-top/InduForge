# 数据点与报警 SDK 用户教程

本教程面向两类用户：在数据中心编写计算脚本的工程师，以及在 AI 开发中心编写 Vue + JavaScript 页面的开发者。两处使用同一数据点语义，方法返回格式也相同。

## 1. 先认识返回结果

SDK 方法不会直接把失败伪装成空值：

```js
const result = await point.read()
if (result.code !== 0) {
  console.error(result.msg)
  return
}
console.log(result.data)
```

只有 `code === 0` 表示成功。`data` 是业务结果，失败时为 `null`。

常用读取方法：

- `get()`：获取业务值，按需数据点可能执行一次来源。
- `read()`：获取 `value/quality/timestamp` 完整样本。
- `peek()`：只看已存储当前值，不主动执行来源。

常用操作方法：

- `set(value)`：写入可写数据点。
- `subscribe(handler)`：订阅后续变化。
- `history(query)`：读取历史样本。
- `refresh()`：刷新支持主动读取的来源。
- `run(input)`：运行计算类数据点。
- `execute(input)`：执行按需查询或命令类数据点。
- `publish(payload)`：发布消息。

每个数据点始终拥有这些方法。不支持时返回非零结果，可先查看 `point.capabilities` 控制界面按钮。

## 2. 在计算工作台使用数据点

### 2.1 绑定变量

1. 打开数据中心的“计算”模块，新建或打开计算单元。
2. 在底部切换到“变量”。
3. 点击“插入数据点变量”，选择数据点。
4. 为数据点设置脚本变量名，例如 `temperature`。
5. 保存计算单元，再进行检查或试运行。

变量名只使用英文字母、数字和下划线，不能以数字开头。`ctx`、`dp`、`argv`、`console` 和 `require` 是平台保留名。

绑定后，`temperature` 已经是数据点对象，不是温度数值：

```js
const result = temperature.read()
if (result.code !== 0) return result
return result.data.value
```

也可以通过集合访问同一对象：

```js
temperature === dp.temperature
temperature === ctx.points.temperature
```

通常直接使用变量名最清楚。

### 2.2 读取数值并计算

假设变量面板绑定了：

| 变量名 | 数据点 |
| --- | --- |
| `temperature` | 设备温度 |
| `pressure` | 设备压力 |

```js
const temperatureResult = temperature.read()
const pressureResult = pressure.read()
if (temperatureResult.code !== 0) return temperatureResult
if (pressureResult.code !== 0) return pressureResult

if (temperatureResult.data.quality !== 'good') {
  return { valid: false, reason: '温度质量不可用' }
}

return {
  temperature: temperatureResult.data.value,
  pressure: pressureResult.data.value,
  score: temperatureResult.data.value * pressureResult.data.value,
  observedAt: temperatureResult.data.observedAt,
}
```

### 2.3 预览写入和执行动作

开发态计算沙箱不会直接写设备或数据库。调用受支持的操作会返回“已记录”，并在调试结果的 `sideEffects` 中展示意图：

```js
const result = setpoint.set(1200)
if (result.code !== 0) return result
return { accepted: true, effect: result.data }
```

节点发布运行后，节点侧才会重新执行脚本、验证权限并真实写入。不要把开发态受理当成现场写入成功。

完整 JavaScript 场景脚本见 [计算数据点场景.js](./examples/计算数据点场景.js)，Python 版本见 [计算数据点场景.py](./examples/计算数据点场景.py)。

### 2.4 数据点变化不要在脚本里订阅

计算脚本是一次性执行。不要在脚本里调用 `subscribe()` 等待后续事件；应在“触发”面板选择“数据点变化”或“条件触发”。节点收到事件后再启动一次脚本。

## 3. 在 AI 开发页面使用数据点

Vue + JavaScript 模板已经依赖 `@induforge/runtime-sdk`。页面无需管理 token，也不要请求数据中心内部 URL：

```vue
<script setup>
import { onMounted, ref } from 'vue'
import { alarms, points } from '@induforge/runtime-sdk'

const temperature = ref(null)
const errorMessage = ref('')

async function loadData() {
  const result = await points.factory.line1.temperature.read()
  if (result.code !== 0) {
    errorMessage.value = result.msg
    return
  }
  temperature.value = result.data
}

onMounted(loadData)
</script>
```

如果路径来自配置而不是固定代码，使用：

```js
const point = points.byPath(config.temperaturePath)
const result = await point.read()
```

当宿主注入数据点契约后，可以读取稳定属性：

```js
const point = points.factory.line1.temperature
console.log(point.name, point.dataType, point.unit, point.source, point.capabilities)
```

### 3.1 订阅变化

```js
let unsubscribe

const result = await points.factory.line1.temperature.subscribe((sample) => {
  console.log('温度变化', sample)
})
if (result.code === 0) unsubscribe = result.data

// Vue 组件卸载时
if (typeof unsubscribe === 'function') unsubscribe()
```

### 3.2 查询和处理报警

“当前报警”是节点存储的当前状态，不等于订阅：

```js
const current = await alarms.current.list({ status: 'active', page: 1, pageSize: 20 })
if (current.code === 0) console.log(current.data)

const ack = await alarms.actions.acknowledge('alarm-state-id', {
  comment: '现场已检查',
})
```

实时变化单独订阅：

```js
const subscription = await alarms.changes.subscribe((event) => {
  console.log('报警状态变化', event)
})
```

页面独立运行而宿主未提供报警节点适配器时，会返回 `50031`。这表示当前预览环境不支持，不代表返回了“没有报警”。

## 4. Vue + JavaScript 完整 Demo

仓库的 [Vue + JavaScript 模板](../../../contracts/project-templates/vite-vue-js/src/App.vue) 已包含完整示例，覆盖：

- 当前值与质量展示。
- 历史样本加载。
- 数据点变化订阅与释放。
- 写入设定值。
- 当前报警列表和确认。
- 计算单元调用。
- 加载中、空数据和错误状态。

复制模板后只需替换 `POINT_PATHS` 和 `COMPUTE_REF`：

```js
const POINT_PATHS = {
  temperature: 'factory.line1.temperature',
  pressure: 'factory.line1.pressure',
  setpoint: 'factory.line1.setpoint',
}
const COMPUTE_REF = 'temperatureConvert'
```

## 5. 开发与发布的差异

| 场景 | 开发态 | 发布到节点后 |
| --- | --- | --- |
| `read/get/peek` | 当前值、默认值或模拟值 | 节点存储或真实来源 |
| `set/publish/execute/run` | 计算沙箱记录副作用；页面由预览宿主策略决定 | 节点鉴权后真实执行 |
| 数据点订阅 | 计算脚本不支持；页面由预览宿主提供 | 节点事件总线 |
| 当前/历史报警 | 宿主未提供时明确返回不支持 | 节点报警存储 |

上线前至少验证成功、无数据、质量异常、能力不支持、超时和权限拒绝六条路径。

## 6. 常见错误

### 把变量当成普通值

错误：

```js
return temperature * 1.8 + 32
```

正确：

```js
const result = temperature.get()
if (result.code !== 0) return result
return result.data * 1.8 + 32
```

### 忽略 code

错误：

```js
const value = (await point.read()).data.value
```

正确：

```js
const result = await point.read()
if (result.code !== 0 || !result.data) throw new Error(result.msg)
const value = result.data.value
```

### 用 get 轮询可能产生副作用的数据点

数据库写查询、命令和按需执行数据点的 `get()` 可能执行来源。需要只看已存储值时使用 `peek()`，需要变化通知时使用 `subscribe()`，不要定时轮询有副作用的 `get()`。
