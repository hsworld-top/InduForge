# 设计器编辑器主题与国际化同步

## 范围

- 本能力只作用于 `designer` 编辑器壳层、工具栏、停靠面板、状态栏和常用编辑器提示。
- 画布内用户自行搭建的页面内容、组件文本和运行态样式不跟随本能力变化。
- 预览页继续与编辑器壳层主题/语言隔离。

## 宿主同步

- `dev_ide` 打开设计器时通过 URL 传入 `theme` 与 `locale`。
- `dev_ide` 运行中通过 `postMessage` 向嵌入的设计器发送：
  - `THEME_UPDATE`
  - `LOCALE_UPDATE`
- 语言广播只发送到 `designer` iframe，不影响 `datacenter`。

## 设计器行为

- `designer` 使用 `vue-i18n` 管理编辑器壳层文案。
- 编辑器主题与语言统一由编辑器 UI 状态层维护，并同步到：
  - `document.documentElement`
  - Element Plus locale
  - 编辑器壳层文案
- 设计器内部保留独立主题/语言切换能力。

## 本地存储

- 设计器独立持久化键：
  - `designer_theme`
  - `designer_language`
- 独立打开设计器时优先读取设计器专属键。
- 专属键缺失时允许只读回退历史通用键 `theme` / `language`，但写入只落到设计器专属键，不反向影响 IDE。

## 验证要求

- 从 IDE 打开设计器时，编辑器初始主题与语言与 IDE 一致。
- IDE 切换主题或语言时，设计器壳层同步变化，画布用户页面不变。
- 设计器独立打开时，本地切换刷新后仍保留。
- 设计器本地切换后，IDE 自身主题与语言不被反向改写。
