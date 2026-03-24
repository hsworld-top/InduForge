/**
 * 组件 Manifest 注册表 — 汇总入口
 * 侧向 import 触发各类 registerManifest
 */

export {
  registerManifest,
  getManifest,
  getAllManifests,
  getManifestsByCategory,
} from "./manifest-registry.js";

import "./layout-manifests.js";
import "./form-manifests.js";
import "./display-manifests.js";
import "./chart-manifests.js";

export { default } from "./manifest-registry.js";
