# Designer 布局与按钮测试用例

## 用例说明
- 测试对象: 设计中心画布编辑器（布局容器 + Button）
- 目标范围: 6 布局（Horizontal/Vertical/Collapse/Tabs/FormLayout/ElContainer）+ Button
- 资产拖拽补充范围: Image / Video / DownloadLink（隐藏组件，仅资源拖拽入口）

## 用例矩阵

| CaseID | 场景 | 步骤 | 预期 |
|---|---|---|---|
| C01 | 物料可见性-布局 | 打开物料区布局分组 | 仅显示 6 个布局容器，顺序固定 |
| C02 | 物料可见性-UI | 打开物料区 UI 分组 | 仅显示 Button |
| C03 | 禁止旧组件曝光 | 搜索旧组件关键字（如 EChart、Table） | 搜索不到旧组件 |
| C04 | Button 放入 HorizontalLayout | 拖入布局后再拖入 Button | Button 可正常插入，定位正确 |
| C05 | Button 放入 VerticalLayout | 同上 | Button 可正常插入，定位正确 |
| C06 | Button 放入 Collapse | 新建 Collapse 后插入 Button | Button 可插入活动面板 |
| C07 | Button 放入 Tabs | 新建 Tabs 后插入 Button | Button 可插入当前 Tab 页 |
| C08 | Button 放入 FormLayout | 新建 FormLayout 后插入 Button | Button 以流式布局插入 |
| C09 | Button 放入 ElContainer 区域 | 新建 ElContainer 后分别拖入 Header/Aside/Main/Footer | 各区域均可插入且区域结构稳定 |
| C10 | 自由布局重叠 | 在页面根自由布局中放置两个 Button 并重叠 | 可重叠显示，无异常抖动 |
| C11 | 层级上移下移 | 对重叠 Button 执行上移/下移 | 视觉层级与 children 顺序一致 |
| C12 | 跨容器移动 | Button 在不同布局容器间拖拽 | 目标容器接收正确，源容器 children 更新正确 |
| C13 | 撤销重做 | 执行插入/移动后撤销重做 | 画布与树结构一致回放 |
| C14 | 预览一致性 | 设计态与页面预览对比 | 结构与视觉一致 |
| C15 | 资产拖拽-图片 | 从资源区拖拽图片到画布 | 落为 Image 组件，`src/alt/fit` 正确 |
| C16 | 资产拖拽-视频 | 从资源区拖拽视频到画布 | 落为 Video 组件，`src/controls` 等属性正确 |
| C17 | 资产拖拽-文件 | 从资源区拖拽非图/视频文件 | 落为 DownloadLink 组件，下载属性正确 |

