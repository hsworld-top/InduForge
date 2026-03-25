/**
 * 组件包入口
 * 注册所有已拆分组件的 descriptor 到组件注册中心
 * 各组件 manifest 在对应 manifest.ts 内自注册；manifests/index.ts 通过分片 import 保证加载顺序
 *
 * 新增组件时：
 * 1. 创建 components/<ComponentName>/manifest.ts + descriptors/<name>.ts + index.ts
 * 2. 在此处 import 并调用 registerDescriptor
 */

import { descriptor as ButtonDescriptor } from "./Button";
import { registerDescriptor } from "./descriptors/registry";

import { registerSimpleDescriptors } from "./descriptors/simple-descriptors";
import { descriptor as HorizontalLayoutDescriptor } from "./HorizontalLayout";
import { descriptor as VerticalLayoutDescriptor } from "./VerticalLayout";

/**
 * 注册所有组件描述符
 * 应在应用启动时（main.ts 或 App.vue setup 阶段）调用一次
 * 顺序：先注册标杆组件（Button、HorizontalLayout、VerticalLayout），再批量注册简单组件
 */
export function registerAllDescriptors() {
  registerDescriptor("Button", ButtonDescriptor);
  registerDescriptor("HorizontalLayout", HorizontalLayoutDescriptor);
  registerDescriptor("VerticalLayout", VerticalLayoutDescriptor);
  registerSimpleDescriptors();
}

export {
  canAcceptChildByDescriptor,
  getChildFlowLayout,
  getChildLayout,
  getChildPositioning,
  getChildStyle,
  getCustomRenderer,
  getDefaultSize,
  getDescriptor,
  getDisplayContent,
  getFlexDirection,
  getPropsFilter,
  getRenderKey,
  getRenderTag,
  hasDescriptor,
  isChildResizable,
  isContainerType,
  isFlexContainer,
  isLayoutContainerType,
  isLayoutType,
  isRegionType,
  registerDescriptor,
  resolveDescriptorContainerStyle,
} from "./descriptors/registry";
