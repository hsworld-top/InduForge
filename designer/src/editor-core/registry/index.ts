/**
 * 注册表模块统一导出
 */

export {
  ComponentRegistry,
  ComponentCategory,
  componentRegistry,
} from "./component-registry";
export type {
  EditorComponentManifest,
  RegistryEventDefinition,
} from "./component-registry";
export {
  defaultEventDefinitions,
  getComponentEventDefinitions,
  normalizeEventDefinitions,
} from "./component-events";
export type { EventDefinition } from "./component-events";
export { registerBuiltinComponents } from "./builtin-manifests";
