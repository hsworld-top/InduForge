/**
 * 注册表模块统一导出
 */

export { registerBuiltinComponents } from "./builtin-manifests";
export {
  defaultEventDefinitions,
  getComponentEventDefinitions,
  normalizeEventDefinitions,
} from "./component-events";
export type { EventDefinition } from "./component-events";
export { ComponentCategory, ComponentRegistry, componentRegistry } from "./component-registry";
export type { EditorComponentManifest, RegistryEventDefinition } from "./component-registry";
