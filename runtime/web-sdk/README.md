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
