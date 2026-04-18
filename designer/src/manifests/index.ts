/**
 * 组件 Manifest 注册总入口
 * 仅保留当前版本支持的布局容器与按钮。
 */

import "@/components/HorizontalLayout/manifest";
import "@/components/VerticalLayout/manifest";
import "@/components/Collapse/manifest";
import "@/components/Tabs/manifest";
import "@/components/FormLayout/manifest";
import "@/components/ElContainer/manifest";
import "@/components/FreeContainer/manifest";
import "@/components/Button/manifest";
import "@/components/Image/manifest";
import "@/components/Video/manifest";
import "@/components/DownloadLink/manifest";

export {
  getAllManifests,
  getManifest,
  getManifestsByCategory,
  registerManifest,
} from "./manifest-registry";

export { default } from "./manifest-registry";
