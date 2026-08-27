# @induforge/runtime-sdk

面向 Designer 页面工程的固定版本 JavaScript SDK。客户代码直接使用 JavaScript，`index.d.ts` 只为编辑器提供提示。

```js
import { access, points, scenes } from '@induforge/runtime-sdk'
import '@induforge/runtime-sdk/scene-elements'

const value = await points.factory.line1.temperature.get({ range: '1h' })
await points.factory.line1.temperature.set(80)
const unsubscribe = points.factory.line1.temperature.sub((nextValue) => {
  console.log(nextValue)
})
await points.factory.events.pub({ type: 'refresh' })

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

运行时默认读取 `window.__INDUFORGE_RUNTIME__`：

```js
window.__INDUFORGE_RUNTIME__ = {
  roles: ['operator'],
  adapter: {
    get(path, params) {},
    set(path, value) {},
    subscribe(path, handler) {},
    publish(path, payload) {},
  },
  navigation: {
    open2D(sceneId, options) {},
    open3D(sceneId, options) {},
  },
}
```

也可以使用 `configureRuntime(runtime)` 配置顶层导出，或用 `createRuntimeClient(runtime)` 创建相互隔离的客户端。SDK 不包含数据点清单，数据点路径在访问 `points.<path>` 时按需形成。
