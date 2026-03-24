/**
 * 组件 Manifest 注册表 — 汇总入口
 * 侧向 import 触发各类 registerManifest
 */

export {
  registerManifest,
  getManifest,
  getAllManifests,
  getManifestsByCategory,
} from "./manifest-registry";

import "./layout-manifests";
import "./form-manifests";
import "./display-manifests";
import "./chart-manifests";

export { default } from "./manifest-registry";
