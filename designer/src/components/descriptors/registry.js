/**
 * 组件描述符注册中心
 *
 * 每个组件通过 registerDescriptor 注册一个描述符（descriptor），
 * 描述符包含渲染标签、容器样式、子项布局策略等元数据，
 * 替代 NodeRenderer / editor-store / factory 中大量硬编码的类型分支。
 *
 * @module components/descriptors/registry
 */

/**
 * 组件描述符类型定义
 * @typedef {Object} ComponentDescriptor
 * @property {string} [renderTag="div"] - 渲染时使用的 HTML 标签或 Vue 组件名
 * @property {boolean} [isContainer=false] - 是否为容器组件（影响 drop 目标判断）
 * @property {Object} [defaultStyle={}] - 组件插入时的默认样式
 * @property {function(node: Object): Object} [containerStyle] - 容器布局样式生成函数，仅容器组件需要。参数为 currentNode，返回样式对象
 * @property {'absolute'|'flow'|null} [childPositioning=null] - 子项默认定位模式；null 表示由调用方决定
 * @property {Object|null} [childFlowLayout=null] - 子项默认 flowLayout（当 childPositioning 为 'flow' 时生效）
 * @property {function(parentType: string): Object} [childStyle] - 子项默认样式生成函数
 * @property {boolean} [childResizable=true] - 子项是否允许通过 resize 手柄调整尺寸
 * @property {boolean} [isMovable=true] - 组件自身是否可自由拖拽移动
 * @property {boolean|string[]} [acceptChildren=false] - 允许的子组件类型；true=任意，false=不接受，string[]=限定类型列表
 * @property {number|null} [maxChildren=null] - 最大子项数量；null=无限，1=单子项容器（如 ElCol、ElHeader）
 * @property {boolean} [isRegion=false] - 是否为 El 容器布局区域（ElHeader/ElAside/ElMain/ElFooter）
 * @property {'row'|'column'|null} [flexDirection=null] - Flex 布局方向；row=水平，column=垂直，null=未指定（从 CSS 读取）
 * @property {function(node: Object, resolvedProps: Object): string|null} [displayContent] - 文本/占位内容生成；返回 null 表示无文本内容
 * @property {Object|null} [slots=null] - 复杂插槽定义（Select/Table/Tabs 等），暂未使用时可省略
 * @property {function(node: Object): string} [renderKey] - 渲染 key 生成函数，默认 node.id
 * @property {function(resolvedProps: Object): Object} [propsFilter] - 过滤/转换后传给 renderTag 的 props
 * @property {import('vue').Component|null} [customRenderer=null] - 复杂组件自定义渲染器（Vue 组件）
 * @property {'flex'|'grid'|'free'|'none'|null} [childLayout=null] - 子项布局类型；flex=Flex布局，grid=Grid布局，free=自由定位，none=非容器，null=未指定
 * @property {{ width: number, height: number }|null} [defaultSize=null] - 组件默认尺寸（插入时使用）
 * @property {function(node: Object, context?: Object): string} [renderKey] - 渲染 key；context 可含 tableRenderVersion、resolvedProps 等
 */

/** El 壳层内不可整体拖动的槽位类型（与 descriptor.isMovable:false 对齐） */
const FIXED_LAYOUT_SHELL_SLOT_TYPES = new Set([
  "ElHeader",
  "ElAside",
  "ElMain",
  "ElFooter",
  "ElCol",
  "ElLayoutRow",
]);

const LEGACY_FLEX_DIRECTION_TYPES = new Set(["FlexContainer", "ResponsiveLayout"]);

const REGION_DESIGNER_HINTS = {
  ElHeader: "Header区域",
  ElAside: "Aside区域",
  ElMain: "Main区域",
  ElFooter: "Footer区域",
};

const COMPONENT_WRAPPER_RENDER_TYPES = new Set(["ElLayoutRow", "ElCol"]);

/** @type {Map<string, ComponentDescriptor>} */
const _descriptors = new Map();

/**
 * 注册组件描述符
 * @param {string} type - 组件类型标识（需与 manifest.type 一致）
 * @param {ComponentDescriptor} descriptor - 组件描述符
 */
export function registerDescriptor(type, descriptor) {
  if (!type) {
    console.warn("[ComponentRegistry] registerDescriptor: type 不能为空");
    return;
  }
  _descriptors.set(type, { ...descriptor });
}

/**
 * 获取组件描述符
 * @param {string} type - 组件类型
 * @returns {ComponentDescriptor | null}
 */
export function getDescriptor(type) {
  return _descriptors.get(type) || null;
}

/**
 * 判断组件是否已注册描述符
 * @param {string} type - 组件类型
 * @returns {boolean}
 */
export function hasDescriptor(type) {
  return _descriptors.has(type);
}

/**
 * 获取组件的渲染标签，未注册时降级为 "div"
 * @param {string} type - 组件类型
 * @param {Object} [node] - 节点对象（当 renderTag 为函数时使用）
 * @param {Object} [resolvedProps] - 解析后的 props（当 renderTag 为函数时使用）
 * @returns {string|import('vue').Component}
 */
/** 由 NodeRenderer 单独解析为 Vue 组件，描述符层返回占位 */
const DEFERRED_RENDER_TAG_TYPES = new Set(["EChart"]);

export function getRenderTag(type, node, resolvedProps) {
  if (!type) return "div";
  if (DEFERRED_RENDER_TAG_TYPES.has(type)) return "div";
  const descriptor = _descriptors.get(type);
  if (!descriptor) {
    throw new Error(`[descriptors] 未注册组件类型: ${type}`);
  }
  const renderTag = descriptor.renderTag;
  if (!renderTag) {
    throw new Error(`[descriptors] 描述符缺少 renderTag: ${type}`);
  }
  if (typeof renderTag === "function") {
    return renderTag(node, resolvedProps) ?? "div";
  }
  return renderTag;
}

/**
 * 判断组件是否为容器
 * @param {string} type - 组件类型
 * @returns {boolean}
 */
export function isContainerType(type) {
  return _descriptors.get(type)?.isContainer ?? false;
}

/**
 * 获取子项在父容器内的定位模式
 * @param {string} parentType - 父容器类型
 * @returns {'absolute'|'flow'|null}
 */
export function getChildPositioning(parentType) {
  return _descriptors.get(parentType)?.childPositioning ?? null;
}

/**
 * 获取组件在该父容器内的默认 flowLayout
 * @param {string} parentType - 父容器类型
 * @returns {Object|null}
 */
export function getChildFlowLayout(parentType) {
  return _descriptors.get(parentType)?.childFlowLayout ?? null;
}

/**
 * 获取子项默认样式
 * @param {string} parentType - 父容器类型
 * @returns {Object}
 */
export function getChildStyle(parentType) {
  const descriptor = _descriptors.get(parentType);
  if (!descriptor?.childStyle) return {};
  return descriptor.childStyle(parentType);
}

/**
 * 判断子项在该容器内是否可 resize
 * @param {string} parentType - 父容器类型
 * @returns {boolean}
 */
export function isChildResizable(parentType) {
  const descriptor = _descriptors.get(parentType);
  if (!descriptor) return true;
  return descriptor.childResizable !== false;
}

/**
 * 计算容器布局样式
 * @param {string} type - 容器类型
 * @param {Object} node - 当前节点数据
 * @returns {Object|null} 返回 null 表示该类型没有注册容器样式，调用方走默认逻辑
 */
export function resolveDescriptorContainerStyle(type, node) {
  const descriptor = _descriptors.get(type);
  if (!descriptor?.containerStyle) return null;
  return descriptor.containerStyle(node);
}

/**
 * 获取组件显示内容（文本/占位），由 descriptor.displayContent 驱动
 * @param {string} type - 组件类型
 * @param {Object} node - 当前节点
 * @param {Object} resolvedProps - 已解析的 props
 * @returns {string|null} 显示文本或 null
 */
export function getDisplayContent(type, node, resolvedProps) {
  const descriptor = _descriptors.get(type);
  if (!descriptor?.displayContent) return null;
  return descriptor.displayContent(node, resolvedProps ?? {});
}

/**
 * 获取组件渲染 key
 * @param {string} type - 组件类型
 * @param {Object} node - 当前节点
 * @param {{ tableRenderVersion?: number, resolvedProps?: Object }} [context] - Table/Tabs 等需要的额外依赖
 * @returns {string}
 */
export function getRenderKey(type, node, context) {
  const descriptor = _descriptors.get(type);
  if (!descriptor?.renderKey) return node?.id ?? "";
  return descriptor.renderKey(node, context || {});
}

/**
 * Table / BigDataTable 等表格类组件
 * @param {string} type
 * @returns {boolean}
 */
export function isTableLikeType(type) {
  return type === "Table" || type === "BigDataTable";
}

/**
 * 设计画布内是否允许节点自由拖动（false 表示锁在布局壳上）
 * @param {string} type
 * @returns {boolean}
 */
export function isNodeDesignerMovable(type) {
  if (!type) return true;
  const descriptor = _descriptors.get(type);
  if (descriptor && descriptor.isMovable === false) return false;
  return !FIXED_LAYOUT_SHELL_SLOT_TYPES.has(type);
}

/**
 * 是否用真实组件根作外层包装（el-row / el-col）
 * @param {string} type
 * @returns {boolean}
 */
export function usesComponentWrapper(type) {
  return COMPONENT_WRAPPER_RENDER_TYPES.has(type);
}

/**
 * Flex 方向是否应从 props.direction 读取（旧版容器）
 * @param {string} type
 * @returns {boolean}
 */
export function usesLegacyFlexDirectionProps(type) {
  return LEGACY_FLEX_DIRECTION_TYPES.has(type);
}

/**
 * 区域组件在设计器中的提示文案
 * @param {string} type
 * @returns {string}
 */
export function getRegionDesignerHint(type) {
  return REGION_DESIGNER_HINTS[type] || "区域";
}

/**
 * 设计器节点上除通用类名外的布局相关 class（el-layout、tabs 等）
 * @param {string} type
 * @param {{ tabPosition?: string, parentGutter?: number }} [ctx]
 * @returns {string[]}
 */
export function getDesignerNodeLayoutClasses(type, ctx = {}) {
  if (!type) return [];
  const out = [];
  if (type === "ElLayout") out.push("el-layout");
  if (type === "ElLayoutRow") out.push("el-layout-row");
  if (type === "HorizontalLayout" || type === "VerticalLayout") {
    out.push("layout-container-visible");
  }
  if (type === "Tabs") {
    out.push("tabs-container");
    const position = ctx.tabPosition ?? "top";
    out.push(`tabs-pos-${position}`);
  }
  if (type === "ElCol") {
    out.push("el-col");
    if ((ctx.parentGutter ?? 0) > 0) out.push("is-guttered");
  }
  return out;
}

/**
 * 过滤/转换传给 renderTag 的 props
 * @param {string} type - 组件类型
 * @param {Object} resolvedProps - 已解析的 props
 * @returns {Object}
 */
export function getPropsFilter(type, resolvedProps) {
  const descriptor = _descriptors.get(type);
  if (!descriptor?.propsFilter) return resolvedProps ?? {};
  return descriptor.propsFilter(resolvedProps ?? {});
}

/**
 * 获取组件自定义渲染器（复杂组件用）
 * @param {string} type - 组件类型
 * @returns {import('vue').Component|null}
 */
export function getCustomRenderer(type) {
  return _descriptors.get(type)?.customRenderer ?? null;
}

/**
 * 获取组件默认尺寸
 * @param {string} type - 组件类型
 * @param {Object} [componentRegistry] - 组件注册表实例（可选，用于 manifest fallback）
 * @returns {{ width: number, height: number }|null}
 */
export function getDefaultSize(type, componentRegistry = null) {
  // 优先从 descriptor 读取（新架构组件）
  const descriptorSize = _descriptors.get(type)?.defaultSize;
  if (descriptorSize) {
    return descriptorSize;
  }
  // 向后兼容：未注册 descriptor 的组件，从 manifest 读取
  if (componentRegistry) {
    const manifest = componentRegistry.get(type);
    const defaultSize = manifest?.defaultSize;
    if (defaultSize && typeof defaultSize === "object") {
      return {
        width: defaultSize.width || 120,
        height: defaultSize.height || 40,
      };
    }
  }
  return null;
}

/**
 * 获取子项布局类型
 * @param {string} type - 组件类型
 * @returns {'flex'|'grid'|'free'|'none'|null}
 */
export function getChildLayout(type) {
  return _descriptors.get(type)?.childLayout ?? null;
}

/**
 * 判断是否为 Flex 容器（子项使用 Flex 布局）
 * @param {string} type - 组件类型
 * @returns {boolean}
 */
export function isFlexContainer(type) {
  return getChildLayout(type) === "flex";
}

/**
 * 判断是否为布局容器类型（用于 isLayoutContainerType 查询）
 * 包括 Flex、Grid、Free 容器，但不包括普通容器（如 Tabs）
 * @param {string} type - 组件类型
 * @returns {boolean}
 */
export function isLayoutContainerType(type) {
  const layout = getChildLayout(type);
  return layout === "flex" || layout === "grid" || layout === "free";
}

/**
 * 判断是否为布局节点类型（ElLayout/ElLayoutRow/ElCol 等）
 * @param {string} type - 组件类型
 * @returns {boolean}
 */
export function isLayoutType(type) {
  return (
    type === "ElLayout" ||
    type === "ElLayoutRow" ||
    type === "ElCol" ||
    isLayoutContainerType(type)
  );
}

/**
 * 根据 descriptor 判断容器是否允许接受指定类型的子组件
 * @param {string} parentType - 父组件类型
 * @param {string} childType - 子组件类型
 * @param {number} currentChildCount - 当前子项数量（用于 maxChildren 检查）
 * @returns {boolean}
 */
export function canAcceptChildByDescriptor(parentType, childType, currentChildCount = 0) {
  const descriptor = _descriptors.get(parentType);
  if (!descriptor) return true; // 未注册 descriptor 的组件默认允许

  // 检查 maxChildren（单子项容器限制）
  if (typeof descriptor.maxChildren === "number") {
    if (currentChildCount >= descriptor.maxChildren) {
      return false;
    }
  }

  // 检查 acceptChildren
  const acceptChildren = descriptor.acceptChildren;
  if (acceptChildren === false) {
    return false; // 明确不接受子组件
  }
  if (acceptChildren === true) {
    return true; // 接受任意子组件
  }
  if (Array.isArray(acceptChildren)) {
    return acceptChildren.includes(childType); // 限定类型列表
  }

  // 未定义 acceptChildren，默认允许
  return true;
}

/**
 * 判断是否为 El 容器布局区域类型（ElHeader/ElAside/ElMain/ElFooter）
 * @param {string} type - 组件类型
 * @returns {boolean}
 */
export function isRegionType(type) {
  return _descriptors.get(type)?.isRegion === true;
}

/**
 * 获取 Flex 布局方向
 * @param {string} type - 组件类型
 * @returns {'row'|'column'|null}
 */
export function getFlexDirection(type) {
  return _descriptors.get(type)?.flexDirection ?? null;
}

export default {
  registerDescriptor,
  getDescriptor,
  hasDescriptor,
  getRenderTag,
  isContainerType,
  getChildPositioning,
  getChildFlowLayout,
  getChildStyle,
  isChildResizable,
  resolveDescriptorContainerStyle,
  getDisplayContent,
  getRenderKey,
  getPropsFilter,
  getCustomRenderer,
  getDefaultSize,
  getChildLayout,
  isFlexContainer,
  isLayoutContainerType,
  isLayoutType,
  canAcceptChildByDescriptor,
  isRegionType,
  getFlexDirection,
  isTableLikeType,
  isNodeDesignerMovable,
  usesComponentWrapper,
  usesLegacyFlexDirectionProps,
  getRegionDesignerHint,
  getDesignerNodeLayoutClasses,
};
