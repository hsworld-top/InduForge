# InduForge AI 页面 Vue + JavaScript 模板

模板使用 Vue 3、Vite 和 `@induforge/runtime-sdk`，示例页面展示数据点当前值、历史、订阅、写入、计算调用、当前报警和报警确认。

在 `src/App.vue` 中替换工程路径与计算单元引用：

```js
const POINT_PATHS = {
  temperature: 'factory.line1.temperature',
  pressure: 'factory.line1.pressure',
  setpoint: 'factory.line1.setpoint',
}
const COMPUTE_REF = 'temperatureConvert'
```

页面代码不管理 token，也不拼接平台内部接口。AI 开发中心预览宿主和发布后的节点宿主会向 SDK 注入适配器。每次调用都检查 `result.code === 0`。

本地命令：

```bash
pnpm dev
pnpm build
```

脱离 InduForge 宿主独立打开时，数据与报警操作会明确显示“适配器未提供对应能力”，这是预期行为。
