/**
 * 绑定操作命令
 * 包含数据绑定的设置、更新、删除命令
 */

import type { DocumentModel } from '../document/DocumentModel'
import type { Binding, TransformOp } from '../document/types'
import { Command } from './Command'

/** 设置节点绑定命令 */
export class SetBindingCommand extends Command {
  private _nodeId!: string
  private _propKey!: string
  private _binding: Binding | null
  private _oldBinding: Binding | null = null

  get type(): string {
    return 'SetBinding'
  }

  constructor(nodeId: string, propKey: string, binding: Binding | null) {
    super()
    this._nodeId = nodeId
    this._propKey = propKey
    this._binding = binding
  }

  execute(doc: DocumentModel): void {
    const node = doc.getNode(this._nodeId)
    if (node) {
      this._oldBinding = node.bindings?.[this._propKey] ?? null
      const newBindings = { ...node.bindings }
      if (this._binding) {
        newBindings[this._propKey] = this._binding
      } else {
        delete newBindings[this._propKey]
      }
      doc._updateNode(this._nodeId, { bindings: newBindings })
    }
  }

  undo(doc: DocumentModel): void {
    const node = doc.getNode(this._nodeId)
    if (node) {
      const newBindings = { ...node.bindings }
      if (this._oldBinding) {
        newBindings[this._propKey] = this._oldBinding
      } else {
        delete newBindings[this._propKey]
      }
      doc._updateNode(this._nodeId, { bindings: newBindings })
    }
  }

  getDescription(): string {
    return this._binding ? `设置绑定: ${this._propKey}` : `移除绑定: ${this._propKey}`
  }
}

/** 批量设置绑定命令 */
export class SetMultipleBindingsCommand extends Command {
  private _nodeId!: string
  private _bindings!: Record<string, Binding | null>
  private _oldBindings: Record<string, Binding | null> = {}

  get type(): string {
    return 'SetMultipleBindings'
  }

  constructor(nodeId: string, bindings: Record<string, Binding | null>) {
    super()
    this._nodeId = nodeId
    this._bindings = bindings
  }

  execute(doc: DocumentModel): void {
    const node = doc.getNode(this._nodeId)
    if (node) {
      for (const propKey of Object.keys(this._bindings)) {
        this._oldBindings[propKey] = node.bindings?.[propKey] ?? null
      }
      const newBindings = { ...node.bindings }
      for (const [propKey, binding] of Object.entries(this._bindings)) {
        if (binding) {
          newBindings[propKey] = binding
        } else {
          delete newBindings[propKey]
        }
      }
      doc._updateNode(this._nodeId, { bindings: newBindings })
    }
  }

  undo(doc: DocumentModel): void {
    const node = doc.getNode(this._nodeId)
    if (node) {
      const newBindings = { ...node.bindings }
      for (const [propKey, oldBinding] of Object.entries(this._oldBindings)) {
        if (oldBinding) {
          newBindings[propKey] = oldBinding
        } else {
          delete newBindings[propKey]
        }
      }
      doc._updateNode(this._nodeId, { bindings: newBindings })
    }
  }

  getDescription(): string {
    const count = Object.keys(this._bindings).length
    return `批量设置绑定 (${count} 个)`
  }
}

/** 清除所有绑定命令 */
export class ClearAllBindingsCommand extends Command {
  private _nodeId!: string
  private _oldBindings: Record<string, Binding> = {}

  get type(): string {
    return 'ClearAllBindings'
  }

  constructor(nodeId: string) {
    super()
    this._nodeId = nodeId
  }

  execute(doc: DocumentModel): void {
    const node = doc.getNode(this._nodeId)
    if (node && node.bindings) {
      this._oldBindings = JSON.parse(JSON.stringify(node.bindings)) as Record<string, Binding>
      doc._updateNode(this._nodeId, { bindings: {} })
    }
  }

  undo(doc: DocumentModel): void {
    if (Object.keys(this._oldBindings).length > 0) {
      doc._updateNode(this._nodeId, { bindings: this._oldBindings })
    }
  }

  getDescription(): string {
    return `清除所有绑定`
  }
}

/** 更新绑定转换命令 */
export class UpdateBindingTransformCommand extends Command {
  private _nodeId!: string
  private _propKey!: string
  private _transform!: TransformOp[]
  private _oldTransform: TransformOp[] | undefined = undefined

  get type(): string {
    return 'UpdateBindingTransform'
  }

  constructor(nodeId: string, propKey: string, transform: TransformOp[]) {
    super()
    this._nodeId = nodeId
    this._propKey = propKey
    this._transform = transform
  }

  execute(doc: DocumentModel): void {
    const node = doc.getNode(this._nodeId)
    const binding = node?.bindings?.[this._propKey]
    if (!node || !binding) return
    if (binding.kind !== 'datapoint' && binding.kind !== 'var') return
    this._oldTransform = binding.transform ? [...binding.transform] : undefined
    const newBindings = {
      ...node.bindings,
      [this._propKey]: { ...binding, transform: this._transform },
    }
    doc._updateNode(this._nodeId, {
      bindings: newBindings as Record<string, Binding>,
    })
  }

  undo(doc: DocumentModel): void {
    const node = doc.getNode(this._nodeId)
    const binding = node?.bindings?.[this._propKey]
    if (!node || !binding) return
    if (binding.kind !== 'datapoint' && binding.kind !== 'var') return
    const newBindings = {
      ...node.bindings,
      [this._propKey]: { ...binding, transform: this._oldTransform },
    }
    doc._updateNode(this._nodeId, {
      bindings: newBindings as Record<string, Binding>,
    })
  }

  getDescription(): string {
    return `更新绑定转换: ${this._propKey}`
  }
}

/** 设置绑定降级值命令 */
export class SetBindingFallbackCommand extends Command {
  private _nodeId!: string
  private _propKey!: string
  private _fallback!: unknown
  private _oldFallback: unknown = undefined

  get type(): string {
    return 'SetBindingFallback'
  }

  constructor(nodeId: string, propKey: string, fallback: unknown) {
    super()
    this._nodeId = nodeId
    this._propKey = propKey
    this._fallback = fallback
  }

  execute(doc: DocumentModel): void {
    const node = doc.getNode(this._nodeId)
    const binding = node?.bindings?.[this._propKey]
    if (!node || !binding) return
    this._oldFallback = binding.fallback
    const newBindings = {
      ...node.bindings,
      [this._propKey]: { ...binding, fallback: this._fallback },
    }
    doc._updateNode(this._nodeId, {
      bindings: newBindings as Record<string, Binding>,
    })
  }

  undo(doc: DocumentModel): void {
    const node = doc.getNode(this._nodeId)
    const binding = node?.bindings?.[this._propKey]
    if (!node || !binding) return
    const newBindings = {
      ...node.bindings,
      [this._propKey]: { ...binding, fallback: this._oldFallback },
    }
    doc._updateNode(this._nodeId, {
      bindings: newBindings as Record<string, Binding>,
    })
  }

  getDescription(): string {
    return `设置绑定降级值: ${this._propKey}`
  }
}

/** 复制绑定命令 */
export class CopyBindingsCommand extends Command {
  private _sourceNodeId!: string
  private _targetNodeId!: string
  private _propKeys: string[] | undefined
  private _oldTargetBindings: Record<string, Binding | null> = {}
  private _copiedBindings: Record<string, Binding> = {}

  get type(): string {
    return 'CopyBindings'
  }

  constructor(sourceNodeId: string, targetNodeId: string, propKeys?: string[]) {
    super()
    this._sourceNodeId = sourceNodeId
    this._targetNodeId = targetNodeId
    this._propKeys = propKeys
  }

  execute(doc: DocumentModel): void {
    const sourceNode = doc.getNode(this._sourceNodeId)
    const targetNode = doc.getNode(this._targetNodeId)
    if (!sourceNode || !targetNode || !sourceNode.bindings) return

    const keys = this._propKeys ?? Object.keys(sourceNode.bindings)
    for (const key of keys) {
      const src = sourceNode.bindings[key]
      if (src) {
        this._copiedBindings[key] = JSON.parse(JSON.stringify(src)) as Binding
        this._oldTargetBindings[key] = targetNode.bindings?.[key] ?? null
      }
    }
    const newBindings = { ...targetNode.bindings, ...this._copiedBindings }
    doc._updateNode(this._targetNodeId, { bindings: newBindings })
  }

  undo(doc: DocumentModel): void {
    const targetNode = doc.getNode(this._targetNodeId)
    if (!targetNode) return
    const newBindings = { ...targetNode.bindings }
    for (const [key, oldBinding] of Object.entries(this._oldTargetBindings)) {
      if (oldBinding) {
        newBindings[key] = oldBinding
      } else {
        delete newBindings[key]
      }
    }
    doc._updateNode(this._targetNodeId, { bindings: newBindings })
  }

  getDescription(): string {
    return `复制绑定`
  }
}

export default {
  SetBindingCommand,
  SetMultipleBindingsCommand,
  ClearAllBindingsCommand,
  UpdateBindingTransformCommand,
  SetBindingFallbackCommand,
  CopyBindingsCommand,
}
