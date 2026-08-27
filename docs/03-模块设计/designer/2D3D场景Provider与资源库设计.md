# 2D/3D 场景 Provider 与资源库设计

## 1. 边界

平台同一时间只启用一个 `SceneProvider`，首版为 HT。场景数据存在时禁止切换 Provider，工程前端
源码仍位于开发容器 `/workspace`，场景和 Provider 资源不进入工作区文件系统。Designer 负责场景
身份和公开契约；Provider 编辑器负责场景画布及工程级资源库。

一个 Designer 场景卡片只对应一个 Provider 根入口：2D 固定为 `displays/{sceneId}.json`，3D 固定为
`scenes/{sceneId}.json`。HT 不再提供同一场景内新建第二张根图纸或第二个根场景的能力；节点、图元、
图层和分组仍属于当前入口。若后续需要多页面，应由平台新增显式页面模型，不能复用 HT 隐式文件树。

## 2. Provider 能力

`SceneProvider` 必须声明引擎版本、支持的场景类型、资源分类与格式，并实现入口校验、依赖和测点
引用分析、资源导入分析、挂载路径、编辑器/资源编辑器入口及 Viewer 地址生成。Runtime SDK 只传
`sceneId` 和运行参数，由平台解析当前 Provider。

HT Provider 支持：

| 分类 | 场景 | 上传格式 | 可在线编辑 |
| --- | --- | --- | --- |
| 图片 | 2D/3D | PNG、JPEG、WebP、GIF | 否 |
| 字体 | 2D/3D | WOFF、WOFF2 | 否 |
| 模型 | 3D | OBJ/MTL/贴图 ZIP | 否 |
| 材质 | 3D | HT JSON/贴图 ZIP | 否 |
| Symbol | 2D | HT JSON/贴图 ZIP | 是 |
| Component | 2D | HT JSON/贴图 ZIP | 是 |

## 3. 资源与场景闭环

资源名称在同工程、Provider 和分类内大小写不敏感唯一。上传包先校验条目数、解压大小、路径、格式、
JSON 和 OBJ/MTL/贴图依赖闭包，再写入内容对象并在一个数据库事务中创建内部代次。普通用户不看到
逻辑路径、内容哈希或代次历史。

场景通过 `scene_asset_bindings` 固定资源内部代次。拖入资源只建立绑定并由 Provider 生成挂载路径，
不复制 MinIO 字节；场景草稿节点引用同一个内容对象。资源替换只生成新代次，既有绑定显示“资源有
更新”。`update` 仅更新当前场景绑定并受 `baseDraftVersion` 乐观锁保护。

Symbol/Component 编辑会话从当前代次建立工作副本。保存文件只修改工作副本草稿，显式“保存资源”
才发布新内部代次。归档资源从新选择列表隐藏，但已绑定草稿、revision 和 Release 继续可用。

## 4. Revision 与发布

场景 commit 同时固定入口、公开契约、数据点引用、依赖文件、资源代次和挂载路径。Viewer 默认只读
已提交 revision；开发 Viewer 必须明确标识草稿。发布前拒绝未提交草稿、缺失依赖、摘要不一致、
Provider 版本不兼容或资源代次不存在的工程。Release Manifest 记录 Provider、引擎版本、sceneId、
revision、资源 ID、内部代次、挂载路径和根摘要。

工程页面通过 `induforge-scene-2d` 和 `induforge-scene-3d` Web Components 加载场景，只传稳定
`sceneId`。开发态组件向 Designer 受控预览宿主申请 Redis 短期 Viewer 会话，会话固定当前已提交
revision，文件请求只读取 `scene_revision_files`。公开交互只包含结构化参数、事件和命令，运行时按
JSON Schema Draft 2020-12 严格校验；开发态数据点实时桥已由受控宿主实现，Release Loader 独立实施。

## 5. HT 功能收口与 OEM

正式编辑器左侧只提供资源库和只读高级依赖诊断；固定场景入口由平台在后台加载，不展示作品树、
逻辑目录或 Provider 文件名。诊断接口按页返回文件、来源场景或资源、
内容摘要与可用状态，不提供目录修改能力。不提供 `locate`、任意目录创建、复制粘贴、
物理路径定位或外部文件轮询，也不提供新建图纸、场景、模型或材质入口。服务端会拒绝向场景草稿
写入固定入口之外的同类根 JSON，ZIP 导入执行相同校验。数据点来自会话代理接口，不内置演示测点。生产构建只保留 2D/3D
编辑器、2D/3D/Symbol Viewer 五个正式入口，并自动剔除兼容页、customstyle、示例预览、高炉演示和
源路径元数据。浏览器统一使用 `/designer/scene-studio/`，运行模块使用通用文件名并关闭 Source Map；
原始 Provider 源码目录、模块名和版本只存在于服务端仓库与内部存储，不作为静态资源发布。

可见标题、favicon、帮助、支持信息和控制台横幅统一使用 InduForge。构建阶段移除第三方源码注释、
厂商网址、旧全局变量和可识别文件名；普通场景、资源、revision 和 Release 响应只返回公开产品标识，
不返回内部 Provider 名称或引擎版本。原始授权与版权材料保留在非公开源码和交付归档中。

## 6. 管道、交互与数据联动

管道使用可编辑 Shape 中心线，平台属性 `induforge.pipe.*` 统一映射静止、连续、分段和颗粒/箭头模式。
连续与颗粒模式使用 `path-flow`，分段模式使用 `dash-flow`；自动绕障只在编辑态调用
`route-planning`，结果保存为普通 Shape 点集，Viewer 不执行路由计算。

对象的数据联动和交互分别保存在显式 `induforge.bindings` 与 `induforge.interactions` 属性中。
Provider 保存入口 JSON 时在同一事务中更新只读 `provider_contract` 与 `datapoint_refs`；提交时把用户
契约和 Provider 契约合并，同名 Schema 不一致则拒绝。提交形成 revision 前还会向数据服务校验每个
显式绑定的数据点是否存在且为 `active`，并核对数据类型与 `get/sub/set` 能力；任一项不满足均拒绝提交。
运行态数据桥仅允许访问 revision 声明的数据点，
查询、订阅和按稳定 path 写入均由受控宿主代理，Viewer 和工程预览 iframe 不接收长期 Token。

图形模板保存时剥离数据联动与公开交互，业务组件允许保留工程内绑定。两类资源均支持上传，也支持
从当前画布选择直接保存。保存所选时后端从场景草稿解析依赖闭包，将引用改写为资源内部路径，并直接
复用已有内容对象，不经浏览器下载再上传；外部 URL、HTML/WebView、私有数据源配置和不受控脚本不得进入资源代次。
