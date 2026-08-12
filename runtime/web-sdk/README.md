# @induforge/runtime-sdk

面向 Designer 页面工程的固定版本 JavaScript SDK。客户代码直接使用 JavaScript，`index.d.ts` 只为编辑器提供提示。

```js
import { access, points, scenes } from '@induforge/runtime-sdk'

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
