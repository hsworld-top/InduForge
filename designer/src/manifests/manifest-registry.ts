/**
 * 组件 Manifest 注册表：属性、样式、事件等元数据
 */

export type PropType = "string" | "number" | "boolean" | "color" | "enum" | "object" | "array";

export interface PropOption {
  label: string;
  value: unknown;
}

export interface PropDefinition {
  name: string;
  type: PropType;
  label: string;
  group?: string;
  defaultValue?: unknown;
  options?: PropOption[];
  min?: number;
  max?: number;
  step?: number;
  placeholder?: string;
  /** 是否允许数据绑定 */
  bindable?: boolean;
  /** 属性面板编辑器类型，如 code */
  editor?: string;
  language?: string;
  height?: string;
}

export interface EventDefinition {
  name: string;
  label: string;
  description?: string;
}

/** 画布/属性面板消费的组件清单（与 editor-core registry 对齐的公共子集） */
export interface ComponentManifest {
  type: string;
  name: string;
  category: string;
  props: PropDefinition[];
  defaultSize?: { width: number; height: number };
  isContainer?: boolean;
  events?: EventDefinition[];
  defaultStyle?: Record<string, string | number>;
  icon?: string;
  description?: string;
  allowedChildren?: string[];
  slots?: Record<string, unknown>;
}

const manifestRegistry = new Map<string, ComponentManifest>();

export function registerManifest(manifest: ComponentManifest): void {
  if (!manifest?.type) {
    throw new Error("Invalid manifest: missing type");
  }
  manifestRegistry.set(manifest.type, manifest);
}

export function getManifest(type: string): ComponentManifest | undefined {
  return manifestRegistry.get(type);
}

export function getAllManifests(): ComponentManifest[] {
  return Array.from(manifestRegistry.values());
}

export function getManifestsByCategory(category: string): ComponentManifest[] {
  return getAllManifests().filter((m) => m.category === category);
}

export default {
  registerManifest,
  getManifest,
  getAllManifests,
  getManifestsByCategory,
};
