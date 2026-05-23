/**
 * 组件注册表：可用组件类型的注册与查询
 */

import type { PropDefinition } from '@/materials/manifests/manifest-registry'

/** 事件项：字符串名或完整定义 */
export interface RegistryEventDefinition {
  name: string
  label: string
  description?: string
}

/**
 * 编辑器内使用的组件清单（由 manifest + builtin 注册逻辑组装）
 */
export interface EditorComponentManifest {
  type: string
  name: string
  category: string
  icon?: string
  description?: string
  defaultProps: Record<string, unknown>
  defaultStyle: Record<string, string | number>
  defaultSize?: { width: number; height: number }
  propsSchema: PropDefinition[]
  styleSchema?: Record<string, unknown>
  events: Array<string | RegistryEventDefinition>
  isContainer: boolean
  allowedChildren?: string[]
  slots?: Record<string, unknown>
}

export const ComponentCategory = {
  BASIC: 'basic',
  CONTAINER: 'container',
  FORM: 'form',
  DATA: 'data',
  CHART: 'chart',
  NAVIGATION: 'navigation',
  LAYOUT: 'layout',
  MEDIA: 'media',
  CUSTOM: 'custom',
} as const

export class ComponentRegistry {
  private _manifests = new Map<string, EditorComponentManifest>()
  private _byCategory = new Map<string, EditorComponentManifest[]>()

  register(manifest: EditorComponentManifest): void {
    if (!manifest.type) {
      throw new Error('组件清单缺少 type 字段')
    }

    if (this._manifests.has(manifest.type)) {
      console.warn(`组件 ${manifest.type} 已存在，将被覆盖`)
    }

    this._manifests.set(manifest.type, manifest)

    const category = manifest.category || ComponentCategory.CUSTOM
    if (!this._byCategory.has(category)) {
      this._byCategory.set(category, [])
    }
    const list = this._byCategory.get(category)
    if (list) {
      list.push(manifest)
    }
  }

  registerAll(manifests: EditorComponentManifest[]): void {
    for (const manifest of manifests) {
      this.register(manifest)
    }
  }

  get(type: string): EditorComponentManifest | undefined {
    return this._manifests.get(type)
  }

  has(type: string): boolean {
    return this._manifests.has(type)
  }

  getAll(): EditorComponentManifest[] {
    return Array.from(this._manifests.values())
  }

  getByCategory(category: string): EditorComponentManifest[] {
    return this._byCategory.get(category) ?? []
  }

  getCategories(): string[] {
    return Array.from(this._byCategory.keys())
  }

  search(keyword: string): EditorComponentManifest[] {
    const lowerKeyword = keyword.toLowerCase()
    return this.getAll().filter(
      (m) =>
        m.type.toLowerCase().includes(lowerKeyword) ||
        m.name.toLowerCase().includes(lowerKeyword) ||
        (m.description?.toLowerCase().includes(lowerKeyword) ?? false),
    )
  }

  getDefaultNode(type: string): {
    type: string
    label: string
    props: Record<string, unknown>
    style: Record<string, string | number>
    children: unknown[]
  } | null {
    const manifest = this._manifests.get(type)
    if (!manifest) return null

    return {
      type,
      label: manifest.name,
      props: { ...manifest.defaultProps },
      style: { ...manifest.defaultStyle },
      children: [],
    }
  }

  unregister(type: string): void {
    const manifest = this._manifests.get(type)
    if (manifest) {
      this._manifests.delete(type)

      const category = manifest.category || ComponentCategory.CUSTOM
      const categoryList = this._byCategory.get(category)
      if (categoryList) {
        const index = categoryList.indexOf(manifest)
        if (index > -1) {
          categoryList.splice(index, 1)
        }
      }
    }
  }

  clear(): void {
    this._manifests.clear()
    this._byCategory.clear()
  }
}

export const componentRegistry = new ComponentRegistry()

export default ComponentRegistry
