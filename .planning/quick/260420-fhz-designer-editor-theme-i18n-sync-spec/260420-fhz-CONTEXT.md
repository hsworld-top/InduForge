# Quick Task 260420-fhz: 设计中心编辑器主题与国际化同步规格 - Context

**Gathered:** 2026-04-20
**Status:** Ready for planning

## Task Boundary

为设计中心编辑器补充主题与国际化同步设计规格，明确 IDE 与 `designer` 的同步协议、作用边界、首轮范围、风险与验证方式。

## Implementation Decisions

### 状态模型

- 采用“编辑器独立状态层 + IDE 增量同步”方案
- 编辑器 UI 设置仅包含 `theme` 与 `locale`
- 设置来源优先级为：URL > `localStorage` > 默认值

### 同步边界

- 同步只作用于设计中心编辑器
- 画布内用户自己构建的页面部分不受影响
- 设计中心保留独立切换主题与语言的能力
- 设计中心本地切换不反向同步 IDE

### 首轮范围

- 先打通 URL 与 `postMessage` 双通道
- 先覆盖编辑器主链路高频文案
- 暂不处理预览页、运行态页面和发布态字段语义

## Specific Ideas

- `dev_ide` 负责补充 `locale` URL 参数与 `LOCALE_UPDATE` 广播
- `designer` 负责建立单一编辑器 UI 状态源
- `App.vue` 通过 `ElConfigProvider` 应用 Element Plus locale
- `DesignerView.vue` 中原有占位的主题与语言入口改为真实切换逻辑

## Canonical References

- `D:/SVNCode/indu-forge/designer/AGENTS.md`
- `D:/SVNCode/indu-forge/docs/designer/README.md`
- `D:/SVNCode/indu-forge/docs/contracts/designer-publish-schema.md`
