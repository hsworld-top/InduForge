# 组件开发指南

## 1. 文档定位

- 本文档定义 `designer` 组件开发的正式约定，用于约束组件声明、属性结构、事件暴露和与发布态结构的对齐方式。
- 本文档聚焦组件开发规则，不替代 [设计器概览](./README.md) 和 [Designer 发布态 Schema 契约](../contracts/designer-publish-schema.md)。

## 2. 组件声明原则

- 组件必须具有稳定的 `type` 标识。
- 组件属性必须能够被明确声明，而不是依赖隐式约定。
- 组件事件必须显式暴露，避免运行态再反向猜测。
- 组件布局能力必须明确，避免容器关系不清。

## 3. 推荐声明结构

```typescript
interface ComponentManifest {
  type: string
  title: string
  category: string
  icon?: string
  propsSchema: Record<string, unknown>
  events: Array<{
    name: string
    title: string
    allowedActions?: string[]
  }>
  layoutCaps?: {
    canContain?: boolean
    allowedParentLayouts?: string[]
  }
}
```

## 4. 属性与绑定约定

- 组件属性需要声明默认值、类型和可配置范围。
- 绑定必须使用正式数据源或正式变量语义，不应依赖编辑器内部临时状态。
- 对外暴露的属性结构需要能稳定映射到发布态 Schema。

## 5. 事件约定

- 事件名称应表达明确业务语义。
- 事件暴露应与组件能力一致，不应为了运行时兼容临时扩展匿名事件。
- 事件允许的动作范围需要清楚，避免运行时产生不受控行为。

## 6. 布局与结构约定

- 组件必须明确是否可容纳子节点。
- 组件需要明确允许的父级布局类型。
- 组件开发应遵守页面层级、放置和尺寸约定。

## 7. 国际化与主题约定

- 文本类属性应支持国际化资源映射。
- 主题相关属性应能稳定进入发布态结构，不依赖编辑器临时状态。

## 8. 关联文档

- [设计器概览](./README.md)
- [Designer 发布态 Schema 契约](../contracts/designer-publish-schema.md)
- [层级约定](./layer-order-convention.md)
- [放置与堆叠](./placement-and-stacking.md)
- [尺寸约定](./size-convention.md)
- [详细设计](../详细设计.md)
