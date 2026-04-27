/**
 * Schema v2 类型定义
 * 编辑器内核的核心类型系统
 *
 * 设计原则：
 * 1. 规范化存储：采用 pagesById + nodesById + graphicsById 扁平结构
 * 2. ID 引用：父子关系通过 ID 数组表达
 * 3. 配置分离：布局、绑定、权限、事件各自独立
 */

// ==================== 基础类型 ====================

const UUID_DASH_REGEX = /-/g;
const PAGE_PATH_SPACE_REGEX = /\s+/g;
const PAGE_PATH_FORBIDDEN_REGEX = /[/?#\\]+/g;

/**
 * 生成唯一 ID
 * @param {string} [prefix] - ID 前缀
 * @returns {string} 唯一 ID
 */
export function generateId(prefix = ""): string {
  return prefix + crypto.randomUUID().replace(UUID_DASH_REGEX, "").substring(0, 12);
}

/**
 * Schema 版本号
 * @type {number}
 */
export const CURRENT_SCHEMA_VERSION = 2;

// ==================== 工程元信息 ====================

/**
 * @typedef {'pc' | 'bigscreen' | 'mobile'} UITarget
 * 目标端类型
 */

/**
 * @typedef {object} ProjectMeta
 * 工程元信息
 * @property {string} projectId - 工程唯一 ID
 * @property {string} name - 工程名称
 * @property {UITarget[]} uiTargets - 支持的目标端
 * @property {number} createdAt - 创建时间戳
 * @property {number} updatedAt - 更新时间戳
 */

// ==================== 安全声明 ====================

/**
 * @typedef {'nodeLocalAuth' | 'centralAuth'} SecurityMode
 * 安全模式
 */

/**
 * @typedef {object} SecurityDecl
 * 安全声明
 * @property {string[]} roles - 角色列表
 * @property {SecurityMode} mode - 安全模式
 */

// ==================== 入口配置 ====================

/**
 * @typedef {object} EntryConfig
 * 入口配置
 * @property {string} [loginPageId] - 登录页面 ID
 * @property {string} homePageId - 首页 ID
 */

// ==================== 数据提供者 ====================

/**
 * @typedef {'dataCenter' | 'api' | 'mock'} DataProviderType
 * 数据提供者类型
 */

/**
 * @typedef {'mqtt' | 'db' | 'calc' | 'api'} DataCapability
 * 数据能力
 */

/**
 * @typedef {object} DataProvider
 * 数据提供者配置
 * @property {DataProviderType} type - 类型
 * @property {string} [description] - 描述
 * @property {DataCapability[]} [capabilities] - 能力列表
 */

// ==================== 变量系统 ====================

/**
 * @typedef {'string' | 'number' | 'boolean' | 'object' | 'array'} VarType
 * 变量类型
 */

/**
 * @typedef {object} VarDef
 * 变量定义
 * @property {VarType} type - 变量类型
 * @property {*} default - 默认值
 * @property {string} [description] - 描述
 */

/**
 * @typedef {object} VarsConfig
 * 变量配置
 * @property {Record<string, VarDef>} global - 全局变量
 * @property {Record<string, Record<string, VarDef>>} pages - 页面变量（pageId -> varName -> VarDef）
 */

// ==================== 资源引用 ====================

/**
 * @typedef {'image' | 'video' | 'audio' | 'font' | 'json' | 'other'} AssetType
 * 资源类型
 */

/**
 * @typedef {'packaged' | 'external' | 'inline'} StorageMode
 * 存储模式
 */

/**
 * @typedef {object} AssetRef
 * 资源引用
 * @property {AssetType} type - 资源类型
 * @property {string} uri - 资源 URI
 * @property {StorageMode} storageMode - 存储模式
 * @property {string} [contentHash] - 内容哈希
 */

// ==================== 页面配置 ====================

/**
 * @typedef {'contain' | 'cover' | 'fill' | 'none'} FitMode
 * 适配模式
 */

/**
 * @typedef {object} BackgroundConfig
 * 背景配置
 * @property {'color' | 'image' | 'gradient'} kind - 背景类型
 * @property {string} value - 背景值
 */

/**
 * @typedef {object} PageConfig
 * 页面配置
 * @property {number} width - 页面宽度
 * @property {number} height - 页面高度
 * @property {FitMode} [fitMode] - 适配模式
 * @property {boolean} [showGrid] - 显示网格
 * @property {boolean} [enableSnap] - 启用吸附
 * @property {boolean} [autoFit] - 预览自适应
 * @property {BackgroundConfig} [background] - 背景配置
 */

/**
 * @typedef {object} LifecycleConfig
 * 生命周期配置
 * @property {Action[]} [onMounted] - 挂载时
 * @property {Action[]} [onUnmounted] - 卸载时
 * @property {Action[]} [onActivated] - 激活时
 * @property {Action[]} [onDeactivated] - 停用时
 */

/**
 * @typedef {object} PageNode
 * 页面节点
 * @property {string} id - 页面唯一 ID
 * @property {string} name - 页面名称
 * @property {string} path - 路由路径
 * @property {UITarget} target - 目标端
 * @property {string} logicalId - 逻辑页面 ID（同路由共享）
 * @property {boolean} isDefaultTarget - 是否默认视图
 * @property {string} rootNodeId - 根节点 ID
 * @property {string[]} [graphicsIds] - 图形 ID 列表
 * @property {PageConfig} config - 页面配置
 * @property {LifecycleConfig} [lifecycle] - 生命周期配置
 */

// ==================== 布局配置 ====================

/**
 * @typedef {object} FlexLayoutItem
 * Flex 布局项
 * @property {number} [grow] - flex-grow
 * @property {number} [shrink] - flex-shrink
 * @property {string} [basis] - flex-basis
 * @property {string} [alignSelf] - align-self
 */

/**
 * @typedef {object} AbsolutePosition
 * 绝对定位
 * @property {number} x - X 坐标
 * @property {number} y - Y 坐标
 * @property {number} w - 宽度
 * @property {number} h - 高度
 * @property {number} [z] - 层级
 */

/**
 * @typedef {object} ConstraintsPosition
 * 约束定位
 * @property {number} [top] - 上边距
 * @property {number} [right] - 右边距
 * @property {number} [bottom] - 下边距
 * @property {number} [left] - 左边距
 * @property {number} [width] - 宽度
 * @property {number} [height] - 高度
 * @property {boolean} [keepAspect] - 保持宽高比
 */

/**
 * @typedef {object} FreeLayoutItem
 * 自由布局项
 * @property {'abs' | 'constraints'} mode - 定位模式
 * @property {AbsolutePosition} [abs] - 绝对定位
 * @property {ConstraintsPosition} [constraints] - 约束定位
 * @property {number} [z] - 层级
 */

/**
 * @typedef {object} GridLayoutItem
 * Grid 布局项
 * @property {number} row - 行
 * @property {number} col - 列
 * @property {number} [rowSpan] - 跨行数
 * @property {number} [colSpan] - 跨列数
 */

/**
 * @typedef {object} LayoutItem
 * 布局项（联合类型）
 * @property {FlexLayoutItem} [flex] - Flex 布局
 * @property {FreeLayoutItem} [free] - 自由布局
 * @property {GridLayoutItem} [grid] - Grid 布局
 */

// ==================== 数据绑定 ====================

/**
 * @typedef {object} TransformOp
 * 转换操作
 * @property {string} op - 操作名称
 * @property {*[]} [args] - 参数
 */

/**
 * @typedef {object} DatapointBinding
 * 数据点绑定
 * @property {'datapoint'} kind - 绑定类型
 * @property {string} provider - 数据提供者 ID
 * @property {string} datapointId - 数据点 ID
 * @property {string} path - 数据点路径
 * @property {TransformOp[]} [transform] - 转换操作
 * @property {*} [fallback] - 降级值
 * @property {*} [designMock] - 设计态 Mock 值
 */

/**
 * @typedef {object} VarBinding
 * 变量绑定
 * @property {'var'} kind - 绑定类型
 * @property {'page' | 'global'} scope - 变量作用域
 * @property {string} name - 变量名
 * @property {TransformOp[]} [transform] - 转换操作
 * @property {*} [fallback] - 降级值
 */

/**
 * @typedef {object} ExprBinding
 * 表达式绑定
 * @property {'expr'} kind - 绑定类型
 * @property {string} expr - 表达式
 * @property {*} [fallback] - 降级值
 */

/**
 * @typedef {DatapointBinding | VarBinding | ExprBinding} Binding
 * 绑定（联合类型）
 */

/**
 * @typedef {object} PermissionConfig
 * 权限配置
 * @property {ComponentRuntimeAccessConfig} [runtimeAccess] - 组件运行态权限方案引用
 */

// ==================== 动作系统 ====================

/**
 * @typedef {'navigate' | 'login' | 'logout' | 'setVar' | 'callApi' | 'writeTag' | 'notify' | 'openUrl' | 'openDialog' | 'closeDialog' | 'refresh' | 'condition' | 'loop' | 'parallel' | 'delay'} ActionType
 * 动作类型
 */

/**
 * @typedef {object} Action
 * 动作定义
 * @property {ActionType} type - 动作类型
 * @property {object} config - 动作配置
 */

// ==================== 动画系统 ====================

/**
 * @typedef {'dataChange' | 'hover' | 'click' | 'mount' | 'always'} AnimationTrigger
 * 动画触发类型
 */

/**
 * @typedef {object} Animation
 * 动画定义
 * @property {string} id - 动画 ID
 * @property {AnimationTrigger} trigger - 触发类型
 * @property {string} [condition] - 触发条件表达式
 * @property {string} type - 动画类型
 * @property {object} config - 动画配置
 */

// ==================== 条件渲染 ====================

/**
 * @typedef {object} ConditionsConfig
 * 条件渲染配置
 * @property {string | boolean} [visible] - 是否渲染
 * @property {string | boolean} [enabled] - 是否可用
 */

// ==================== 循环渲染 ====================

/**
 * @typedef {object} LoopConfig
 * 循环渲染配置
 * @property {string} source - 数据源表达式
 * @property {string} [itemVar] - 当前项变量名
 * @property {string} [indexVar] - 索引变量名
 * @property {string} [keyField] - 唯一键字段
 */

// ==================== 组件节点 ====================

/**
 * @typedef {object} ComponentNode
 * 组件节点
 * @property {string} id - 节点唯一 ID
 * @property {string} type - 组件类型
 * @property {string} [label] - 显示标签
 * @property {Record<string, *>} props - 组件属性
 * @property {Record<string, *>} style - 样式定义
 * @property {string} [styleConfig] - 样式配置（内联样式字符串）
 * @property {string} [detailConfig] - 详细配置（高级配置脚本）
 * @property {LayoutItem | null} layoutItem - 布局配置（兼容旧版）
 * @property {'absolute' | 'flow'} [positioning] - 定位模式（新架构）
 * @property {AbsolutePosition} [absolutePos] - 绝对定位数据（新架构）
 * @property {FlexLayoutItem | GridLayoutItem} [flowLayout] - 流式布局数据（新架构）
 * @property {Record<string, Binding>} bindings - 数据绑定
 * @property {PermissionConfig} permissions - 权限配置
 * @property {Record<string, Action[]>} events - 事件处理
 * @property {Animation[]} [animations] - 动画配置
 * @property {ConditionsConfig} [conditions] - 条件渲染
 * @property {LoopConfig} [loop] - 循环渲染
 * @property {Record<string, string[]>} [slots] - 插槽内容
 * @property {string} [refId] - 自定义组件引用 ID
 * @property {Record<string, object>} [overrides] - 自定义组件属性覆盖
 * @property {string[]} children - 子节点 ID 数组
 * @property {boolean} [locked] - 锁定状态
 * @property {boolean} [hidden] - 隐藏状态（新增）
 */

// ==================== 绘图组件（新架构） ====================

/**
 * @typedef {'line' | 'rect' | 'circle' | 'text' | 'image' | 'path'} ShapeType
 * 图元类型
 */

/**
 * @typedef {object} ShapeStyle
 * 图元样式
 * @property {string} [fill] - 填充颜色
 * @property {string} [stroke] - 边框颜色
 * @property {number} [strokeWidth] - 边框宽度
 * @property {string} [fontFamily] - 字体族
 * @property {number} [fontSize] - 字体大小
 * @property {string} [fontWeight] - 字体粗细
 * @property {string} [textAlign] - 文本对齐
 * @property {number} [opacity] - 不透明度
 */

/**
 * @typedef {object} LineData
 * 线段数据
 * @property {number} x1 - 起点 X
 * @property {number} y1 - 起点 Y
 * @property {number} x2 - 终点 X
 * @property {number} y2 - 终点 Y
 */

/**
 * @typedef {object} RectData
 * 矩形数据
 * @property {number} x - X 坐标
 * @property {number} y - Y 坐标
 * @property {number} width - 宽度
 * @property {number} height - 高度
 * @property {number} [rx] - 圆角 X 半径
 * @property {number} [ry] - 圆角 Y 半径
 */

/**
 * @typedef {object} CircleData
 * 圆形数据
 * @property {number} cx - 圆心 X
 * @property {number} cy - 圆心 Y
 * @property {number} radius - 半径
 */

/**
 * @typedef {object} TextData
 * 文本数据
 * @property {number} x - X 坐标
 * @property {number} y - Y 坐标
 * @property {string} text - 文本内容
 */

/**
 * @typedef {object} ImageData
 * 图片数据
 * @property {number} x - X 坐标
 * @property {number} y - Y 坐标
 * @property {number} width - 宽度
 * @property {number} height - 高度
 * @property {string} src - 图片源
 */

/**
 * @typedef {object} PathData
 * 路径数据
 * @property {string} d - SVG 路径数据
 */

/**
 * @typedef {object} Shape
 * 图元定义
 * @property {string} id - 图元唯一 ID
 * @property {ShapeType} type - 图元类型
 * @property {number} x - X 坐标（通用）
 * @property {number} y - Y 坐标（通用）
 * @property {ShapeStyle} style - 图元样式
 * @property {LineData | RectData | CircleData | TextData | ImageData | PathData} [data] - 类型特定数据
 * @property {number} [rotation] - 旋转角度
 * @property {number} [zIndex] - 层级
 * @property {boolean} [locked] - 锁定状态
 * @property {boolean} [hidden] - 隐藏状态
 */

/**
 * @typedef {object} DiagramData
 * 绘图数据（独立存储）
 * @property {string} diagramId - 绘图唯一 ID
 * @property {Shape[]} shapes - 图元列表
 * @property {number} version - 版本号
 * @property {number} [createdAt] - 创建时间戳
 * @property {number} [updatedAt] - 更新时间戳
 */

/**
 * @typedef {object} DiagramProps
 * 绘图组件属性
 * @property {boolean} [showGrid] - 显示网格
 * @property {number} [gridSize] - 网格大小
 * @property {string} [background] - 背景色
 * @property {boolean} [snapToGrid] - 吸附到网格
 */

// ==================== Canvas 图形节点 ====================

/**
 * @typedef {'Canvas.Line' | 'Canvas.Rect' | 'Canvas.Circle' | 'Canvas.Ellipse' | 'Canvas.Polygon' | 'Canvas.Path' | 'Canvas.Pipe' | 'Canvas.Text' | 'Canvas.Symbol' | 'Canvas.Group'} GraphicType
 * 图形类型
 */

/**
 * @typedef {object} GraphicProps
 * 图形属性（基础）
 * @property {number} [x] - X 坐标
 * @property {number} [y] - Y 坐标
 * @property {number} [cx] - 圆心 X 坐标
 * @property {number} [cy] - 圆心 Y 坐标
 * @property {number} [width] - 宽度
 * @property {number} [height] - 高度
 * @property {number} [radius] - 半径
 * @property {number} [rx] - X 半径
 * @property {number} [ry] - Y 半径
 * @property {Array<[number, number]>} [points] - 点数组
 * @property {string} [d] - 路径数据
 * @property {string} [fill] - 填充颜色
 * @property {string} [stroke] - 边框颜色
 * @property {number} [strokeWidth] - 边框宽度
 * @property {string} [text] - 文本内容
 * @property {number} [fontSize] - 字体大小
 * @property {string} [fontWeight] - 字体粗细
 * @property {string} [symbolId] - 符号 ID
 * @property {number} [scale] - 缩放比例
 * @property {number} [rotation] - 旋转角度
 * @property {string[]} [children] - 子图形 ID（用于 Group）
 */

/**
 * @typedef {object} PipeProps
 * 管道属性
 * @property {Array<[number, number]>} points - 路径点
 * @property {number} width - 管道宽度
 * @property {string} [strokeColor] - 边框颜色
 * @property {string} [fillColor] - 填充颜色
 * @property {'forward' | 'backward' | 'none'} [flowDirection] - 流向
 * @property {number} [flowSpeed] - 流动速度
 * @property {string} [flowColor] - 流动指示颜色
 * @property {number[]} [flowDash] - 流动虚线样式
 * @property {number} [cornerRadius] - 拐角圆角
 * @property {'flat' | 'round' | 'arrow'} [startCap] - 起点样式
 * @property {'flat' | 'round' | 'arrow'} [endCap] - 终点样式
 */

/**
 * @typedef {object} GraphicNode
 * 图形节点
 * @property {string} id - 图形唯一 ID
 * @property {GraphicType} type - 图形类型
 * @property {GraphicProps | PipeProps} props - 图形属性
 * @property {Record<string, Binding>} bindings - 数据绑定
 * @property {Record<string, Action[]>} events - 事件处理
 * @property {Animation[]} [animations] - 动画配置
 * @property {number} z - 图层顺序
 * @property {boolean} [locked] - 锁定状态
 * @property {boolean} [visible] - 可见性
 */

// ==================== 符号库 ====================

/**
 * @typedef {object} GraphicPrimitive
 * 基础图形（符号组成部分）
 * @property {'rect' | 'circle' | 'line' | 'polygon' | 'path' | 'text'} type - 图形类型
 * @property {number} [x] - X 坐标
 * @property {number} [y] - Y 坐标
 * @property {number} [cx] - 圆心 X
 * @property {number} [cy] - 圆心 Y
 * @property {number} [width] - 宽度
 * @property {number} [height] - 高度
 * @property {number} [radius] - 半径
 * @property {Array<[number, number]>} [points] - 点数组
 * @property {string} [d] - 路径数据
 * @property {string} [fill] - 填充颜色
 * @property {string} [stroke] - 边框颜色
 * @property {number} [strokeWidth] - 边框宽度
 * @property {string} [text] - 文本内容
 */

/**
 * @typedef {object} Anchor
 * 锚点定义
 * @property {string} name - 锚点名称
 * @property {number} x - X 偏移
 * @property {number} y - Y 偏移
 * @property {'up' | 'down' | 'left' | 'right'} [direction] - 连接方向
 */

/**
 * @typedef {object} SymbolDef
 * 符号定义
 * @property {string} id - 符号 ID
 * @property {string} name - 符号名称
 * @property {string} category - 符号分类
 * @property {GraphicPrimitive[]} graphics - 组成图形
 * @property {Anchor[]} anchors - 连接锚点
 * @property {{width: number, height: number}} defaultSize - 默认尺寸
 * @property {boolean} [isBuiltin] - 是否内置符号
 */

// ==================== 工程 Schema ====================

/**
 * @typedef {object} ProjectSchema
 * 工程 Schema
 * @property {number} schemaVersion - Schema 版本号
 * @property {ProjectMeta} project - 工程元信息
 * @property {SecurityDecl} securityDecl - 安全声明
 * @property {EntryConfig} entry - 入口配置
 * @property {Record<string, DataProvider>} dataProviders - 数据提供者
 * @property {VarsConfig} vars - 变量配置
 * @property {Record<string, AssetRef>} assetsById - 资源引用
 * @property {Record<string, PageNode>} pagesById - 页面节点
 * @property {Record<string, ComponentNode>} nodesById - 组件节点
 * @property {Record<string, GraphicNode>} graphicsById - 图形节点
 * @property {Record<string, SymbolDef>} symbolsById - 符号库
 * @property {Record<string, DiagramData>} [diagramsById] - 绘图数据（新架构）
 */

// ==================== 选中元素 ====================

/**
 * @typedef {object} SelectableNodeElement
 * 可选中的节点元素
 * @property {'node'} kind - 元素类型
 * @property {string} id - 节点 ID
 */

/**
 * @typedef {object} SelectableGraphicElement
 * 可选中的图形元素
 * @property {'graphic'} kind - 元素类型
 * @property {string} id - 图形 ID
 */

/**
 * @typedef {SelectableNodeElement | SelectableGraphicElement} SelectableElement
 * 可选中元素
 */

/**
 * @typedef {'select' | 'marquee' | 'pan' | 'line' | 'rect' | 'circle' | 'ellipse' | 'polygon' | 'pipe' | 'text'} DrawingTool
 * 绘图工具类型
 */

/**
 * @typedef {object} SelectionState
 * 选中状态
 * @property {SelectableElement[]} selectedElements - 选中的元素列表
 * @property {SelectableElement | null} hoveredElement - 当前 hover 的元素
 * @property {string | null} dropTargetId - 当前拖拽目标
 * @property {SelectableElement | null} anchorElement - 选中锚点
 * @property {DrawingTool | null} activeTool - 当前绘图工具
 */

// ==================== 变更类型 ====================

/**
 * @typedef {'insert' | 'remove' | 'update' | 'move'} ChangeType
 * 变更类型
 */

/**
 * @typedef {object} Change
 * 变更记录
 * @property {ChangeType} type - 变更类型
 * @property {'node' | 'graphic' | 'page' | 'symbol'} target - 变更目标类型
 * @property {string} id - 目标 ID
 * @property {string} [parentId] - 父节点 ID
 * @property {number} [index] - 索引位置
 * @property {object} [oldValue] - 旧值
 * @property {object} [newValue] - 新值
 */

// ==================== 校验结果 ====================

/**
 * @typedef {object} ValidationError
 * 校验错误
 * @property {string} code - 错误代码
 * @property {string} message - 错误信息
 * @property {string} path - 错误位置
 * @property {string} [nodeId] - 节点 ID
 */

/**
 * @typedef {object} ValidationWarning
 * 校验警告
 * @property {string} code - 警告代码
 * @property {string} message - 警告信息
 * @property {string} path - 警告位置
 * @property {string} [nodeId] - 节点 ID
 */

/**
 * @typedef {object} ValidationResult
 * 校验结果
 * @property {boolean} valid - 是否有效
 * @property {ValidationError[]} errors - 错误列表
 * @property {ValidationWarning[]} warnings - 警告列表
 */

// ==================== 页面锁 ====================

/**
 * @typedef {object} PageLockState
 * 页面锁状态
 * @property {string} pageId - 页面 ID
 * @property {boolean} locked - 是否锁定
 * @property {string} [lockedBy] - 锁定者用户 ID
 * @property {string} [lockedByName] - 锁定者用户名
 * @property {number} [lockedAt] - 锁定时间戳
 * @property {boolean} isOwner - 是否是自己持有的锁
 */

/**
 * @typedef {object} EditorReadonlyState
 * 编辑器只读状态
 * @property {boolean} readonly - 是否只读
 * @property {'no_permission' | 'page_locked' | 'viewer_role'} [reason] - 只读原因
 * @property {string} [lockedByName] - 锁定者用户名
 */

/**
 * @typedef {object} LockResult
 * 锁获取结果
 * @property {boolean} success - 是否成功
 * @property {'locked' | 'error'} [reason] - 失败原因
 * @property {string} [lockedByName] - 锁定者用户名
 * @property {*} [error] - 错误对象
 */

// ==================== Patch ====================

/**
 * @typedef {object} PatchAddOp
 * 添加操作
 * @property {'add'} op - 操作类型
 * @property {string} path - 路径
 * @property {*} value - 值
 */

/**
 * @typedef {object} PatchRemoveOp
 * 删除操作
 * @property {'remove'} op - 操作类型
 * @property {string} path - 路径
 */

/**
 * @typedef {object} PatchReplaceOp
 * 替换操作
 * @property {'replace'} op - 操作类型
 * @property {string} path - 路径
 * @property {*} value - 值
 */

/**
 * @typedef {PatchAddOp | PatchRemoveOp | PatchReplaceOp} PatchOp
 * Patch 操作
 */

/**
 * @typedef {object} Patch
 * 差量补丁
 * @property {PatchOp[]} ops - 操作列表
 * @property {number} timestamp - 时间戳
 * @property {string} [userId] - 用户 ID
 */

export type ChangeType = "insert" | "remove" | "update" | "move";
export type ChangeTarget = "node" | "graphic" | "page" | "symbol" | "entry";
export interface Change {
  type: ChangeType;
  target: ChangeTarget;
  id?: string | undefined;
  parentId?: string | undefined;
  index?: number | undefined;
  oldValue?: unknown;
  newValue?: unknown;
}

export type PatchOp =
  | { op: "add"; path: string; value: unknown }
  | { op: "remove"; path: string }
  | { op: "replace"; path: string; value: unknown };

export interface Patch {
  ops: PatchOp[];
  timestamp: number;
  userId?: string;
}

// ==================== TypeScript 类型（与上文 JSDoc 对齐） ====================

export type UITarget = "pc" | "bigscreen" | "mobile";
export type SecurityMode = "nodeLocalAuth" | "centralAuth";
export type DataProviderType = "dataCenter" | "api" | "mock";
export type DataCapability = "mqtt" | "db" | "calc" | "api";
export type VarType = "string" | "number" | "boolean" | "object" | "array";
export type AssetType = "image" | "video" | "audio" | "font" | "json" | "other";
export type StorageMode = "packaged" | "external" | "inline";
export type FitMode = "contain" | "cover" | "fill" | "none";
export type ActionType =
  | "navigate"
  | "login"
  | "logout"
  | "setVar"
  | "callApi"
  | "writeTag"
  | "notify"
  | "openUrl"
  | "openDialog"
  | "closeDialog"
  | "refresh"
  | "condition"
  | "loop"
  | "parallel"
  | "delay";

export interface ProjectMeta {
  projectId: string;
  name: string;
  uiTargets: UITarget[];
  createdAt: number;
  updatedAt: number;
}

export interface SecurityDecl {
  roles: string[];
  mode: SecurityMode;
}

export interface EntryConfig {
  loginPageId?: string;
  homePageId: string;
}

export interface DataProvider {
  type: DataProviderType;
  description?: string;
  capabilities?: DataCapability[];
}

export interface VarDef {
  type: VarType;
  default: unknown;
  description?: string;
}

export interface VarsConfig {
  global: Record<string, VarDef>;
  pages: Record<string, Record<string, VarDef>>;
}

export interface AssetRef {
  type: AssetType;
  uri: string;
  storageMode: StorageMode;
  contentHash?: string;
}

export interface BackgroundConfig {
  kind: "color" | "image" | "gradient";
  value: string;
  size?: "cover" | "contain" | "stretch" | "auto";
  position?: string;
  repeat?: "no-repeat" | "repeat" | "repeat-x" | "repeat-y";
}

export interface PageMetaConfig {
  title?: string;
  description?: string;
}

export interface PageRouteConfig {
  mode?: "auto" | "manual";
  path?: string;
  slug?: string;
}

export interface PageViewportConfig {
  preset?: "bigscreen" | "pc" | "tablet" | "phoneLandscape" | "phonePortrait" | "custom";
  width: number;
  height: number;
  autoFit?: boolean;
  lockAspectRatio?: boolean;
  minWidth?: number;
  minHeight?: number;
  overflowMode?: "auto" | "hidden" | "scroll";
}

export interface PageRuntimeConfig {
  openMode?: "replace" | "cover" | "popup";
  popup?: {
    width?: number;
    height?: number;
    center?: boolean;
    maskClosable?: boolean;
  };
  permission?: {
    summary?: string;
  };
  cacheMode?: "default" | "cache" | "no-cache";
  preloadMode?: "lazy" | "eager";
}

export interface RuntimeRoleRef {
  roleId: string;
  roleCode: string;
  roleName: string;
}

export interface PagePermissionScheme {
  id: string;
  name: string;
  roleRefs: RuntimeRoleRef[];
}

export interface PageRuntimeAccessConfig {
  enabled: boolean;
  allowedRoles: RuntimeRoleRef[];
  schemes: PagePermissionScheme[];
}

export interface PageConfig {
  meta?: PageMetaConfig;
  route?: PageRouteConfig;
  viewport?: PageViewportConfig;
  width: number;
  height: number;
  fitMode?: FitMode;
  showGrid?: boolean;
  enableSnap?: boolean;
  autoFit?: boolean;
  background?: BackgroundConfig;
  /** 页面级 CSS 样式配置，编辑态和预览态都会注入到页面画布中。 */
  styleConfig?: string;
  transition?: {
    type?: "none" | "fade" | "slide" | "zoom";
  };
  runtime?: PageRuntimeConfig;
  /**
   * 以下字段仍保留在编辑态模型中，确保旧页面与旧消费链在迁移期间继续可读。
   */
  description?: string;
  lockAspectRatio?: boolean;
  enableMinSize?: boolean;
  windowStyle?: "replace" | "cover" | "popup" | "normal";
  permissionDesc?: string;
  runtimeAccess?: PageRuntimeAccessConfig;
}

export interface ComponentRuntimeAccessConfig {
  visibleSchemeId?: string;
  operableSchemeId?: string;
}

export interface PermissionConfig {
  runtimeAccess?: ComponentRuntimeAccessConfig;
}

export interface Action {
  type: ActionType;
  config: Record<string, unknown>;
}

export interface LifecycleConfig {
  onMounted?: Action[];
  onUnmounted?: Action[];
  onActivated?: Action[];
  onDeactivated?: Action[];
}

export interface PageNode {
  id: string;
  name: string;
  path: string;
  target: UITarget;
  logicalId: string;
  isDefaultTarget: boolean;
  rootNodeId: string;
  graphicsIds?: string[];
  config: PageConfig;
  lifecycle?: LifecycleConfig;
}

export interface FlexLayoutItem {
  grow?: number;
  shrink?: number;
  basis?: string;
  alignSelf?: string;
}

export interface AbsolutePosition {
  x: number;
  y: number;
  w: number;
  h: number;
  z?: number;
}

export interface ConstraintsPosition {
  top?: number;
  right?: number;
  bottom?: number;
  left?: number;
  width?: number;
  height?: number;
  keepAspect?: boolean;
}

export interface FreeLayoutItem {
  mode: "abs" | "constraints";
  abs?: AbsolutePosition;
  constraints?: ConstraintsPosition;
  z?: number;
}

export interface GridLayoutItem {
  row: number;
  col: number;
  rowSpan?: number;
  colSpan?: number;
}

export interface LayoutItem {
  flex?: FlexLayoutItem;
  free?: FreeLayoutItem;
  grid?: GridLayoutItem;
}

export interface TransformOp {
  op: string;
  args?: unknown[];
}

export interface DatapointBinding {
  kind: "datapoint";
  provider: string;
  datapointId: string;
  path: string;
  transform?: TransformOp[];
  fallback?: unknown;
  designMock?: unknown;
}

export interface VarBinding {
  kind: "var";
  scope: "page" | "global";
  name: string;
  transform?: TransformOp[];
  fallback?: unknown;
}

export interface ExprBinding {
  kind: "expr";
  expr: string;
  fallback?: unknown;
}

export type Binding = DatapointBinding | VarBinding | ExprBinding;

export type GraphicType =
  | "Canvas.Line"
  | "Canvas.Rect"
  | "Canvas.Circle"
  | "Canvas.Ellipse"
  | "Canvas.Polygon"
  | "Canvas.Path"
  | "Canvas.Pipe"
  | "Canvas.Text"
  | "Canvas.Symbol"
  | "Canvas.Group";

export interface GraphicProps {
  x?: number;
  y?: number;
  cx?: number;
  cy?: number;
  width?: number;
  height?: number;
  radius?: number;
  rx?: number;
  ry?: number;
  points?: [number, number][];
  d?: string;
  fill?: string;
  stroke?: string;
  strokeWidth?: number;
  text?: string;
  fontSize?: number;
  fontWeight?: string;
  symbolId?: string;
  scale?: number;
  rotation?: number;
  children?: string[];
  [key: string]: unknown;
}

export interface PipeProps {
  points: [number, number][];
  width: number;
  strokeColor?: string;
  fillColor?: string;
  flowDirection?: "forward" | "backward" | "none";
  flowSpeed?: number;
  flowColor?: string;
  flowDash?: number[];
  cornerRadius?: number;
  startCap?: "flat" | "round" | "arrow";
  endCap?: "flat" | "round" | "arrow";
  [key: string]: unknown;
}

export interface ComponentNode {
  id: string;
  type: string;
  label?: string;
  props: Record<string, unknown>;
  style: Record<string, unknown>;
  styleConfig?: string;
  detailConfig?: string;
  layoutItem: LayoutItem | null;
  positioning?: "absolute" | "flow";
  absolutePos?: AbsolutePosition;
  flowLayout?: FlexLayoutItem | GridLayoutItem;
  bindings: Record<string, Binding>;
  permissions: PermissionConfig;
  events: Record<string, Action[]>;
  animations?: unknown[];
  conditions?: Record<string, unknown>;
  loop?: Record<string, unknown>;
  slots?: Record<string, string[]>;
  refId?: string;
  overrides?: Record<string, Record<string, unknown>>;
  children: string[];
  locked?: boolean;
  hidden?: boolean;
  visible?: boolean;
}

export interface GraphicNode {
  id: string;
  type: GraphicType;
  props: GraphicProps | PipeProps;
  bindings: Record<string, Binding>;
  events: Record<string, Action[]>;
  animations?: unknown[];
  z: number;
  locked?: boolean;
  visible?: boolean;
}

export interface SymbolDef {
  id: string;
  name: string;
  category: string;
  graphics: unknown[];
  anchors: unknown[];
  defaultSize: { width: number; height: number };
  isBuiltin?: boolean;
}

export interface DiagramData {
  diagramId: string;
  shapes: unknown[];
  version: number;
  createdAt?: number;
  updatedAt?: number;
}

export interface ProjectSchema {
  schemaVersion: number;
  project: ProjectMeta;
  securityDecl: SecurityDecl;
  entry: EntryConfig;
  dataProviders: Record<string, DataProvider>;
  vars: VarsConfig;
  assetsById: Record<string, AssetRef>;
  pagesById: Record<string, PageNode>;
  nodesById: Record<string, ComponentNode>;
  graphicsById: Record<string, GraphicNode>;
  symbolsById: Record<string, SymbolDef>;
  diagramsById?: Record<string, DiagramData>;
}

export type SelectableElement = { kind: "node"; id: string } | { kind: "graphic"; id: string };

export type DrawingTool =
  | "select"
  | "marquee"
  | "pan"
  | "line"
  | "rect"
  | "circle"
  | "ellipse"
  | "polygon"
  | "pipe"
  | "text";

export interface SelectionState {
  selectedElements: SelectableElement[];
  hoveredElement: SelectableElement | null;
  dropTargetId: string | null;
  anchorElement: SelectableElement | null;
  activeTool: DrawingTool | null;
}

export interface PageLockState {
  pageId: string;
  locked: boolean;
  lockedBy?: string | undefined;
  lockedByName?: string | undefined;
  lockedAt?: number | undefined;
  isOwner: boolean;
}

export interface EditorReadonlyState {
  readonly: boolean;
  reason?: "no_permission" | "page_locked" | "viewer_role" | undefined;
  lockedByName?: string | undefined;
}

export type LockResult =
  | { success: true }
  | {
      success: false;
      reason: "locked" | "error";
      lockedByName?: string | undefined;
      error?: unknown;
    };

// ==================== 工厂函数 ====================

/**
 * 创建空工程 Schema
 * @param meta - 工程元信息
 */
export function createEmptySchema(meta: Partial<ProjectMeta> = {}): ProjectSchema {
  const now = Date.now();
  return {
    schemaVersion: CURRENT_SCHEMA_VERSION,
    project: {
      projectId: meta.projectId || generateId("proj_"),
      name: meta.name || "新工程",
      uiTargets: meta.uiTargets || ["pc"],
      createdAt: meta.createdAt || now,
      updatedAt: meta.updatedAt || now,
    },
    securityDecl: {
      roles: ["admin", "operator", "viewer"],
      mode: "nodeLocalAuth",
    },
    entry: {
      homePageId: "",
    },
    dataProviders: {},
    vars: {
      global: {},
      pages: {},
    },
    assetsById: {},
    pagesById: {},
    nodesById: {},
    graphicsById: {},
    symbolsById: {},
  };
}

/**
 * 创建页面节点
 * @param options - 页面配置
 */
export function createPageNode(options: Partial<PageNode> = {}): PageNode {
  const id = options.id || generateId("page_");
  const rootNodeId = options.rootNodeId || generateId("node_");
  const defaultPath = buildPagePathFromName(options.name || "新页面");
  return {
    id,
    name: options.name || "新页面",
    path: options.path || defaultPath,
    target: options.target || "pc",
    logicalId: options.logicalId || generateId("logic_"),
    isDefaultTarget: options.isDefaultTarget !== false,
    rootNodeId,
    graphicsIds: options.graphicsIds || [],
    config: {
      meta: {
        title: "",
        description: "",
      },
      route: {
        mode: "auto",
        path: options.path || defaultPath,
        slug: defaultPath.replace(/^\//, ""),
      },
      viewport: {
        preset: "pc",
        width: 1920,
        height: 1080,
        autoFit: true,
        lockAspectRatio: false,
        minWidth: 0,
        minHeight: 0,
        overflowMode: "auto",
      },
      width: 1920,
      height: 1080,
      fitMode: "contain",
      showGrid: true,
      enableSnap: true,
      autoFit: true,
      background: {
        kind: "color",
        value: "#ffffff",
        size: "cover",
        position: "center",
        repeat: "no-repeat",
      },
      transition: {
        type: "none",
      },
      runtime: {
        openMode: "cover",
        popup: {
          width: 960,
          height: 540,
          center: true,
          maskClosable: true,
        },
        permission: {
          summary: "0item",
        },
        cacheMode: "default",
        preloadMode: "lazy",
      },
      description: "",
      lockAspectRatio: false,
      enableMinSize: false,
      windowStyle: "cover",
      permissionDesc: "0item",
      ...options.config,
    },
    lifecycle: options.lifecycle || {},
  };
}

/**
 * 根据页面名称生成路由路径
 */
export function buildPagePathFromName(name: string): string {
  const normalized = String(name || "")
    .trim()
    .replace(PAGE_PATH_SPACE_REGEX, "-");
  const sanitized = normalized.replace(PAGE_PATH_FORBIDDEN_REGEX, "-");
  return `/${sanitized || "page"}`;
}

/**
 * 创建组件节点
 */
export function createComponentNode(
  type: string,
  options: Partial<ComponentNode> = {},
): ComponentNode {
  return {
    id: options.id || generateId("node_"),
    type,
    label: options.label || type,
    props: options.props || {},
    style: options.style || {},
    styleConfig: options.styleConfig || "",
    detailConfig: options.detailConfig || "",
    layoutItem: options.layoutItem || null,
    bindings: options.bindings || {},
    permissions: options.permissions || {},
    events: options.events || {},
    animations: options.animations || [],
    conditions: options.conditions || {},
    children: options.children || [],
    locked: options.locked || false,
    visible: options.visible !== false,
  };
}

/**
 * 创建图形节点
 */
export function createGraphicNode(
  type: GraphicType,
  options: Partial<GraphicNode> = {},
): GraphicNode {
  return {
    id: options.id || generateId("gfx_"),
    type,
    props: (options.props ?? {}) as GraphicProps | PipeProps,
    bindings: options.bindings || {},
    events: options.events || {},
    animations: options.animations || [],
    z: options.z || 0,
    locked: options.locked || false,
    visible: options.visible !== false,
  };
}

/**
 * 创建可选中元素
 */
export function createSelectableElement(kind: "node" | "graphic", id: string): SelectableElement {
  return { kind, id };
}

export default {
  CURRENT_SCHEMA_VERSION,
  generateId,
  createEmptySchema,
  createPageNode,
  createComponentNode,
  createGraphicNode,
  createSelectableElement,
};
