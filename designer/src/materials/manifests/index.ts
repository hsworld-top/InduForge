/**
 * 组件 Manifest 注册总入口
 * 仅保留当前版本支持的布局容器与按钮。
 */

import "@/materials/HorizontalLayout/manifest";
import "@/materials/VerticalLayout/manifest";
import "@/materials/Collapse/manifest";
import "@/materials/Tabs/manifest";
import "@/materials/FormLayout/manifest";
import "@/materials/ElContainer/manifest";
import "@/materials/FreeContainer/manifest";
import "@/materials/Button/manifest";
import "@/materials/ElementPlusCore/manifest";
import "@/materials/Image/manifest";
import "@/materials/Video/manifest";
import "@/materials/DownloadLink/manifest";
import "@/materials/LanguageSwitcher/manifest";
import "@/materials/UserAvatarMenu/manifest";

export {
  getAllManifests,
  getManifest,
  getManifestsByCategory,
  registerManifest,
} from "./manifest-registry";

export { default } from "./manifest-registry";
