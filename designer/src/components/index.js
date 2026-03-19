/**
 * 组件包入口
 * 注册所有已拆分组件的 descriptor 到组件注册中心
 * manifest 仍由 manifests/index.js 统一注册到 componentRegistry
 *
 * 新增组件时：
 * 1. 创建 components/<ComponentName>/manifest.js + descriptor.js + index.js
 * 2. 在此处 import 并调用 registerDescriptor
 */

import { registerDescriptor } from "./registry.js";
import { registerSimpleDescriptors } from "./simple-descriptors.js";

import { descriptor as ButtonDescriptor } from "./Button/index.js";
import { descriptor as HorizontalLayoutDescriptor } from "./HorizontalLayout/index.js";
import { descriptor as VerticalLayoutDescriptor } from "./VerticalLayout/index.js";

/**
 * 注册所有组件描述符
 * 应在应用启动时（main.js 或 App.vue setup 阶段）调用一次
 * 顺序：先注册标杆组件（Button、HorizontalLayout、VerticalLayout），再批量注册简单组件
 */
export function registerAllDescriptors() {
  registerDescriptor("Button", ButtonDescriptor);
  registerDescriptor("HorizontalLayout", HorizontalLayoutDescriptor);
  registerDescriptor("VerticalLayout", VerticalLayoutDescriptor);
  registerSimpleDescriptors();
}

export {
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
} from "./registry.js";
