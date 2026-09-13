# @induforge/runtime-sdk

面向 AI 页面工程和 Designer 运行态的固定版本 JavaScript SDK。页面代码只使用平台数据点、报警、计算、权限和场景抽象，不直接拼接数据服务 URL。

```js
import { access, alarms, computes, points, scenes } from '@induforge/runtime-sdk'
import '@induforge/runtime-sdk/scene-elements'

const temperature = points.factory.line1.temperature
const current = await temperature.read()
if (current.code === 0) console.log(current.data.value, current.data.quality)

const history = await temperature.history({ from: '-1h', limit: 100 })
const subscription = await temperature.subscribe((nextValue) => console.log(nextValue))
await points.factory.line1.setpoint.set(80)
await points.factory.events.publish({ type: 'refresh' })

const activeAlarms = await alarms.current.list({ status: 'active' })
await alarms.actions.acknowledge('alarm-state-id', { comment: '现场已确认' })
await computes.temperatureConvert.run({ value: 26.5 })

if (access.hasAnyRole(['admin', 'operator'])) {
  scenes.open3D('main-factory')
}
```

页面只使用平台生成的稳定场景 ID 加载已提交 revision：

```html
<induforge-scene-2d scene-id="scene-id"></induforge-scene-2d>
<induforge-scene-3d scene-id="scene-id"></induforge-scene-3d>
```

```js
const scene = document.querySelector('induforge-scene-2d')
await scene.setParams({ deviceId: 'A01' })
scene.addEventListener('scene-event', ({ detail }) => console.log(detail.name, detail.payload))
scene.addEventListener('scene-error', ({ detail }) => console.error(detail.code, detail.msg))
const result = await scene.invoke('focusDevice', { id: 'A01' })
```

参数、事件和命令按场景已提交公开契约进行 JSON Schema Draft 2020-12 校验。开发态由 Designer
受控预览宿主创建短期 Viewer 会话；后续 Release Loader 复用相同解析协议：

```js
window.__INDUFORGE_RUNTIME__.sceneResolver = {
  async resolve({ sceneId, kind }) {
    return { url, revision, contract, expiresAt }
  },
}
```

所有数据点与报警方法统一返回 `{ code, msg, data, reqId? }`，只有 `code === 0` 表示成功。运行时默认读取 `window.__INDUFORGE_RUNTIME__`：

```js
window.__INDUFORGE_RUNTIME__ = {
  roles: ['operator'],
  pointContracts: {
    'factory.line1.temperature': {
      id: 'point-id',
      name: '一号线温度',
      dataType: 'float64',
      unit: '℃',
      capabilities: { get: true, read: true, subscribe: true, history: true },
    },
  },
  adapter: {
    get(path, params) {},
    read(path, params) {},
    peek(path) {},
    set(path, value) {},
    subscribe(path, handler) {},
    history(path, query) {},
    refresh(path, options) {},
    run(path, input) {},
    execute(path, input) {},
    publish(path, payload) {},
  },
  alarmAdapter: {
    listItems(query) {},
    getItem(id) {},
    getSettings() {},
    updateSettings(patch) {},
    listCurrent(query) {},
    getCurrent(id) {},
    subscribeChanges(handler, options) {},
    acknowledge(id, input) {},
    unacknowledge(id, input) {},
    forceClear(id, input) {},
    shelve(id, input) {},
    unshelve(id, input) {},
    listHistory(query) {},
    getHistory(id) {},
  },
  computeAdapter: {
    run(ref, input) {},
    describe(ref) {},
  },
  navigation: {
    open2D(sceneId, options) {},
    open3D(sceneId, options) {},
  },
}
```

也可以使用 `configureRuntime(runtime)` 配置顶层导出，或用 `createRuntimeClient(runtime)` 创建相互隔离的客户端。数据点路径可以通过属性链形成，也可以使用 `points.byPath('完整.路径')`；静态属性来自宿主注入的 `pointContracts`。`sub/pub` 保留为 `subscribe/publish` 的简写，新代码优先使用完整方法名。

## 发布工程：Runtime API HTTP/WebSocket 适配器

发布后的工程可使用 `createHttpRuntime()` 接入同节点 Project Gateway 代理的 Runtime API，不必依赖
Designer 预览桥接或 `window.__INDUFORGE_RUNTIME__` 注入。默认请求同源
`/api/v1/runtime`，并连接同源 `/ws/v1/points`；Gateway 负责注入 deployment/project 身份头。

```js
import { configureRuntime, createHttpRuntime } from '@induforge/runtime-sdk'

const runtime = createHttpRuntime({
  // 默认同源；只有直连 Runtime API 或测试时才传 baseUrl、identity。
  timeoutMs: 10_000,
})

// Bearer token 只用于创建 HttpOnly Runtime 会话，后续 HTTP/WS 均使用同源 Cookie。
const session = await runtime.session.establish(runtimeAccessToken)
if (session.code !== 0) throw new Error(session.msg)

configureRuntime(runtime)
const current = await points.factory.line1.temperature.read()
const history = await points.factory.line1.temperature.history({
  from: '2026-08-31T09:00:00Z',
  to: '2026-08-31T10:00:00Z',
  limit: 100,
})
const activeAlarms = await alarms.current.list({ limit: 100 })

// name 和来源路径由 Runtime API 从当前部署的发布工件补全；组合报警使用 sourceDatapoints。
for (const alarm of activeAlarms.data?.items ?? []) {
  console.log(
    alarm.name,
    alarm.datapointPath ?? alarm.sourceDatapoints?.map((source) => source.path),
  )
}
const compute = await computes.byRef('compute-unit-uuid').describe()

// 对象库只按 assetId 访问；内容响应支持 Range/ETag，内部对象存储地址不会返回给工程。
const assets = await runtime.assets.list({ query: 'intro', page: 1, pageSize: 20 })
const asset = assets.data?.items?.[0]
if (asset) {
  const metadata = await runtime.assets.get(asset.assetId)
  const content = await runtime.assets.open(asset.assetId)
  console.log(metadata.data?.contentType, content.data?.url)
}

const live = await points.factory.line1.temperature.subscribe(
  (sample) => console.log(sample.value),
  {
    onError: (error) => console.error('实时连接异常', error),
    onClose: ({ code, reason }) => console.log('实时连接关闭', code, reason),
  },
)
// live.data 是取消订阅函数。调用后连接以正常关闭语义结束。
live.data?.()

await runtime.session.exit()
```

`runtime.catalog.get()` 返回 Runtime Artifact 的点位、计算和报警目录，并将点位元数据缓存到
`runtime.pointContracts`。会话查询使用 `runtime.session.query()`。每次 HTTP 调用都接受
`{ signal, timeoutMs }`；取消和超时会返回 `code: 50031` 的统一失败包络。WebSocket 在订阅成功后不自动
重连：异常会交给 `onError`，关闭始终交给 `onClose`，便于工程按自身生命周期决定是否重连。

Runtime API V1 目前只读。因此 `points.*.set()` 和 `computes.*.run()` 会将服务端的
`501 / code: 50031` 原样返回；其他尚未定义的写入、发布和报警动作同样返回失败，绝不会伪成功。

## 会话与权限

当前运行态会话入口使用发布网关注入的 Runtime Bearer Token：调用 `runtime.session.establish(token)` 建立 HttpOnly 会话，之后可用 `query()`、`exit()` 管理会话。`access.currentUser()` 返回当前会话身份，`access.hasRole()` 与 `access.hasCapability()` 只读取服务端会话快照。用户名密码登录需要工程用户快照和密码验证链路完成后再开放，SDK 不伪造本地登录结果。
