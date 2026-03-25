/**
 * 组件 Manifest 注册表 — 汇总入口
 * 侧向 import 触发各类 registerManifest
 */

import "./layout-manifests-flexbox";
import "@/components/HorizontalLayout/manifest";
import "@/components/VerticalLayout/manifest";
import "./layout-manifests-page-layouts";
import "./form-manifests";
import "./display-manifests-drawing-text";
import "@/components/Button/manifest";
import "./display-manifests-pc-elements";
import "./chart-manifests";

export {
  getAllManifests,
  getManifest,
  getManifestsByCategory,
  registerManifest,
} from "./manifest-registry";

export { default } from "./manifest-registry";
