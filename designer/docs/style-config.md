# 样式配置使用说明（与按钮一致）

## 背景
- 所有 Element 组件现在都支持与按钮一致的 `样式配置` 能力。
- 样式配置入口在属性面板右侧的“配置”按钮，点击弹出 Monaco 编辑器填写样式。
- 采用统一解析规则：**CSS 代码块** 或 **key:value/JSON**。

## 写法
1) **CSS 代码块（推荐，有选择器）**
```css
#domId {
  background: red;
  padding: 8px;
}
```
- 通过 `domId` 选择器控制作用范围。
- 组件会自动把 `#domId` 替换为该组件真实 id（也支持 `[id="domId"]`）。

2) **key:value 行配置（无选择器，直接合并到组件 style）**
```text
background: red
padding: 8px
domId: card-1   # 可选，配合 CSS 块使用
```
- `domId` 行会被记录，但其余属性直接合并进组件样式。

3) **JSON / 对象字面量**
```json
{
  "background": "#f5f7fa",
  "borderRadius": "8px",
  "domId": "panel-1"
}
```

## 生效规则
- 保存时会：
  - 解析样式并合并到组件 `style`（key:value/JSON 模式）。
  - 若存在 CSS 代码块，注入 `<style>` 到 `document.head`，自动替换 `#domId` 为组件真实 id，避免冲突。
- 按钮组件沿用原有逻辑，其余 Element 组件使用同一入口和解析规则。

## 使用步骤
1. 在属性面板点击“配置”按钮。
2. 选择一种写法填写样式。
3. 如需作用于特定组件，请在样式中使用 `#domId { ... }`，或写 `domId: xxx`。
4. 点击“确定”保存，样式立即生效。

## 常见问题
- **写了 CSS 不生效？**
  - 确保选择器使用了 `#domId` 或 `[id="domId"]`，并在样式文本或属性中设置了 `domId`。
  - 代码块需包含 `{}`；行配置需包含冒号。
- **多个组件同一个 domId？**
  - 真实 DOM id 会自动替换成组件唯一 id，避免冲突；`domId` 仅作为选择器别名。
- **已有样式被覆盖？**
  - key:value 会直接合并到组件 style；如需更细粒度控制，请使用 CSS 代码块（权重更高）。
