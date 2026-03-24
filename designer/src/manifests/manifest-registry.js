/**
 * 组件 Manifest 注册表
 * 定义组件的属性、样式和事件配置
 */

/**
 * 属性类型
 * @typedef {'string' | 'number' | 'boolean' | 'color' | 'enum' | 'object' | 'array'} PropType
 */

/**
 * 属性定义
 * @typedef {{
 *   name: string,
 *   type: PropType,
 *   label: string,
 *   group?: string,
 *   defaultValue?: any,
 *   options?: Array<{ label: string, value: any }>,
 *   min?: number,
 *   max?: number,
 *   step?: number,
 *   placeholder?: string
 * }} PropDefinition
 */

/**
 * 组件 Manifest
 * @typedef {{
 *   type: string,
 *   name: string,
 *   category: string,
 *   props: PropDefinition[],
 *   defaultSize?: { width: number, height: number }
 * }} ComponentManifest
 */

/** @type {Map<string, ComponentManifest>} */
const manifestRegistry = new Map();

/**
 * 注册组件 Manifest
 * @param {ComponentManifest} manifest - 组件 Manifest
 */
export function registerManifest(manifest) {
  if (!manifest?.type) {
    console.warn("Invalid manifest: missing type");
    return;
  }
  manifestRegistry.set(manifest.type, manifest);
}

/**
 * 获取组件 Manifest
 * @param {string} type - 组件类型
 * @returns {ComponentManifest | undefined}
 */
export function getManifest(type) {
  return manifestRegistry.get(type);
}

/**
 * 获取所有已注册的 Manifest
 * @returns {ComponentManifest[]}
 */
export function getAllManifests() {
  return Array.from(manifestRegistry.values());
}

/**
 * 按分类获取 Manifest
 * @param {string} category - 分类名称
 * @returns {ComponentManifest[]}
 */
export function getManifestsByCategory(category) {
  return getAllManifests().filter((m) => m.category === category);
}


export default {
  registerManifest,
  getManifest,
  getAllManifests,
  getManifestsByCategory,
};
