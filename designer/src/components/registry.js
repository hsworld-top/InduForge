/**
 * 组件描述符注册中心
 *
 * 每个组件通过 registerDescriptor 注册一个描述符（descriptor），
 * 描述符包含渲染标签、容器样式、子项布局策略等元数据，
 * 替代 NodeRenderer / editor-store / factory 中大量硬编码的类型分支。
 *
 * @module components/registry
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
 */

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
 * @returns {string}
 */
export function getRenderTag(type) {
  return _descriptors.get(type)?.renderTag ?? "div";
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
};
