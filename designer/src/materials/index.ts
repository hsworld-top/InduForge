/**
 * 组件入口：注册当前版本保留组件的 descriptor 到注册中心
 */

import { descriptor as ButtonDescriptor } from "./Button";
import { descriptor as CollapseDescriptor } from "./Collapse";
import { descriptor as DownloadLinkDescriptor } from "./DownloadLink";
import { descriptor as ElContainerDescriptor } from "./ElContainer";
import { descriptor as FormLayoutDescriptor } from "./FormLayout";
import { descriptor as FreeContainerDescriptor } from "./FreeContainer";
import { descriptor as HorizontalLayoutDescriptor } from "./HorizontalLayout";
import { descriptor as ImageDescriptor } from "./Image";
import { descriptor as LanguageSwitcherDescriptor } from "./LanguageSwitcher";
import { descriptor as TabsDescriptor } from "./Tabs";
import { descriptor as VerticalLayoutDescriptor } from "./VerticalLayout";
import { descriptor as VideoDescriptor } from "./Video";
import {
  asideDescriptor,
  footerDescriptor,
  headerDescriptor,
  mainDescriptor,
} from "@/editor-core/descriptors/el-container";
import { registerDescriptor } from "@/editor-core/descriptors/registry";

/**
 * 注册所有组件描述符
 */
export function registerAllDescriptors() {
  registerDescriptor("Button", ButtonDescriptor);
  registerDescriptor("HorizontalLayout", HorizontalLayoutDescriptor);
  registerDescriptor("VerticalLayout", VerticalLayoutDescriptor);
  registerDescriptor("Tabs", TabsDescriptor);
  registerDescriptor("Collapse", CollapseDescriptor);
  registerDescriptor("FormLayout", FormLayoutDescriptor);
  registerDescriptor("ElContainer", ElContainerDescriptor);
  registerDescriptor("ElHeader", headerDescriptor);
  registerDescriptor("ElAside", asideDescriptor);
  registerDescriptor("ElMain", mainDescriptor);
  registerDescriptor("ElFooter", footerDescriptor);
  registerDescriptor("FreeContainer", FreeContainerDescriptor);
  registerDescriptor("Image", ImageDescriptor);
  registerDescriptor("Video", VideoDescriptor);
  registerDescriptor("DownloadLink", DownloadLinkDescriptor);
  registerDescriptor("LanguageSwitcher", LanguageSwitcherDescriptor);
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
} from "@/editor-core/descriptors/registry";
