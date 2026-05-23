/**
 * 图形操作命令
 * 包含 Canvas 图形的插入、删除、更新、移动命令
 */

import type { DocumentModel } from '../document/DocumentModel'
import type { Binding, GraphicNode, GraphicProps } from '../document/types'
import { Command } from './Command'

const UUID_DASH_REGEX = /-/g

/** 插入图形命令 */
export class InsertGraphicCommand extends Command {
  private _pageId!: string
  private _graphic!: GraphicNode

  get type(): string {
    return 'InsertGraphic'
  }

  constructor(pageId: string, graphic: GraphicNode) {
    super()
    this._pageId = pageId
    this._graphic = graphic
  }

  execute(doc: DocumentModel): void {
    doc._insertGraphic(this._pageId, this._graphic)
  }

  undo(doc: DocumentModel): void {
    doc._removeGraphic(this._graphic.id)
  }

  getDescription(): string {
    return `插入图形: ${this._graphic.type}`
  }
}

/** 删除图形命令 */
export class RemoveGraphicCommand extends Command {
  private _graphicId!: string
  private _removedGraphic: GraphicNode | null = null
  private _pageId: string | null = null

  get type(): string {
    return 'RemoveGraphic'
  }

  constructor(graphicId: string) {
    super()
    this._graphicId = graphicId
  }

  execute(doc: DocumentModel): void {
    this._pageId = doc.getGraphicPageId(this._graphicId)
    const graphic = doc.getGraphic(this._graphicId)
    if (graphic) {
      this._removedGraphic = JSON.parse(JSON.stringify(graphic)) as GraphicNode
    }
    doc._removeGraphic(this._graphicId)
  }

  undo(doc: DocumentModel): void {
    if (this._removedGraphic && this._pageId) {
      doc._insertGraphic(this._pageId, this._removedGraphic)
    }
  }

  getDescription(): string {
    return `删除图形`
  }
}

/** 更新图形命令 */
export class UpdateGraphicCommand extends Command {
  private _graphicId!: string
  private _patch!: Partial<GraphicNode>
  private _oldValues: Partial<GraphicNode> | null = null

  get type(): string {
    return 'UpdateGraphic'
  }

  constructor(graphicId: string, patch: Partial<GraphicNode>) {
    super()
    this._graphicId = graphicId
    this._patch = patch
  }

  execute(doc: DocumentModel): void {
    const graphic = doc.getGraphic(this._graphicId)
    if (graphic) {
      this._oldValues = {}
      const gRec = graphic as unknown as Record<string, unknown>
      for (const key of Object.keys(this._patch)) {
        ;(this._oldValues as Record<string, unknown>)[key] = JSON.parse(
          JSON.stringify(gRec[key] ?? null),
        )
      }
      doc._updateGraphic(this._graphicId, this._patch)
    }
  }

  undo(doc: DocumentModel): void {
    if (this._oldValues) {
      doc._updateGraphic(this._graphicId, this._oldValues)
    }
  }

  canMerge(other: Command): boolean {
    return other instanceof UpdateGraphicCommand && other._graphicId === this._graphicId
  }

  merge(other: UpdateGraphicCommand): UpdateGraphicCommand {
    const mergedPatch = { ...this._patch, ...other._patch }
    const cmd = new UpdateGraphicCommand(this._graphicId, mergedPatch)
    cmd._oldValues = this._oldValues
    return cmd
  }

  getDescription(): string {
    return `更新图形属性`
  }
}

/** 移动图形命令（位置偏移） */
export class MoveGraphicCommand extends Command {
  private _graphicId!: string
  private _deltaX!: number
  private _deltaY!: number
  private _oldProps: GraphicProps | null = null

  get type(): string {
    return 'MoveGraphic'
  }

  constructor(graphicId: string, deltaX: number, deltaY: number) {
    super()
    this._graphicId = graphicId
    this._deltaX = deltaX
    this._deltaY = deltaY
  }

  execute(doc: DocumentModel): void {
    const graphic = doc.getGraphic(this._graphicId)
    if (graphic) {
      this._oldProps = JSON.parse(JSON.stringify(graphic.props)) as GraphicProps
      const newProps = this._applyDelta(graphic.props as GraphicProps, this._deltaX, this._deltaY)
      doc._updateGraphic(this._graphicId, { props: newProps })
    }
  }

  undo(doc: DocumentModel): void {
    if (this._oldProps) {
      doc._updateGraphic(this._graphicId, { props: this._oldProps })
    }
  }

  canMerge(other: Command): boolean {
    return other instanceof MoveGraphicCommand && other._graphicId === this._graphicId
  }

  merge(other: MoveGraphicCommand): MoveGraphicCommand {
    const cmd = new MoveGraphicCommand(
      this._graphicId,
      this._deltaX + other._deltaX,
      this._deltaY + other._deltaY,
    )
    cmd._oldProps = this._oldProps
    return cmd
  }

  private _applyDelta(props: GraphicProps, dx: number, dy: number): GraphicProps {
    const newProps = JSON.parse(JSON.stringify(props)) as GraphicProps
    if (typeof newProps.x === 'number') newProps.x += dx
    if (typeof newProps.y === 'number') newProps.y += dy
    if (typeof newProps.cx === 'number') newProps.cx += dx
    if (typeof newProps.cy === 'number') newProps.cy += dy
    if (Array.isArray(newProps.points)) {
      newProps.points = newProps.points.map(([x, y]) => [x + dx, y + dy])
    }
    return newProps
  }

  getDescription(): string {
    return `移动图形`
  }
}

/** 调整图形层级命令 */
export class ReorderGraphicCommand extends Command {
  private _graphicId!: string
  private _newZ!: number
  private _oldZ = 0

  get type(): string {
    return 'ReorderGraphic'
  }

  constructor(graphicId: string, newZ: number) {
    super()
    this._graphicId = graphicId
    this._newZ = newZ
  }

  execute(doc: DocumentModel): void {
    const graphic = doc.getGraphic(this._graphicId)
    if (graphic) {
      this._oldZ = graphic.z
      doc._updateGraphic(this._graphicId, { z: this._newZ })
    }
  }

  undo(doc: DocumentModel): void {
    doc._updateGraphic(this._graphicId, { z: this._oldZ })
  }

  getDescription(): string {
    return `调整图形层级`
  }
}

/** 图形编组命令 */
export class GroupGraphicsCommand extends Command {
  private _pageId!: string
  private _graphicIds!: string[]
  private _groupId: string

  get type(): string {
    return 'GroupGraphics'
  }

  constructor(pageId: string, graphicIds: string[]) {
    super()
    this._pageId = pageId
    this._graphicIds = graphicIds
    this._groupId = `gfx_group_${crypto.randomUUID().replace(UUID_DASH_REGEX, '').substring(0, 8)}`
  }

  execute(doc: DocumentModel): void {
    const maxZ = Math.max(...this._graphicIds.map((id) => doc.getGraphic(id)?.z ?? 0))
    const group: GraphicNode = {
      id: this._groupId,
      type: 'Canvas.Group',
      props: { children: [...this._graphicIds] },
      bindings: {},
      events: {},
      animations: [],
      z: maxZ,
    }
    doc._insertGraphic(this._pageId, group)
  }

  undo(doc: DocumentModel): void {
    doc._removeGraphic(this._groupId)
  }

  getGroupId(): string {
    return this._groupId
  }

  getDescription(): string {
    return `编组图形 (${this._graphicIds.length} 个)`
  }
}

/** 取消图形编组命令 */
export class UngroupGraphicsCommand extends Command {
  private _groupId!: string
  private _removedGroup: GraphicNode | null = null
  private _pageId: string | null = null

  get type(): string {
    return 'UngroupGraphics'
  }

  constructor(groupId: string) {
    super()
    this._groupId = groupId
  }

  execute(doc: DocumentModel): void {
    const group = doc.getGraphic(this._groupId)
    if (!group || group.type !== 'Canvas.Group') return
    this._removedGroup = JSON.parse(JSON.stringify(group)) as GraphicNode
    this._pageId = doc.getGraphicPageId(this._groupId)
    doc._removeGraphic(this._groupId)
  }

  undo(doc: DocumentModel): void {
    if (this._removedGroup && this._pageId) {
      doc._insertGraphic(this._pageId, this._removedGroup)
    }
  }

  getDescription(): string {
    return `取消编组`
  }
}

/** 设置图形绑定命令 */
export class SetGraphicBindingCommand extends Command {
  private _graphicId!: string
  private _propKey!: string
  private _binding: Binding | null
  private _oldBinding: Binding | null = null

  get type(): string {
    return 'SetGraphicBinding'
  }

  constructor(graphicId: string, propKey: string, binding: Binding | null) {
    super()
    this._graphicId = graphicId
    this._propKey = propKey
    this._binding = binding
  }

  execute(doc: DocumentModel): void {
    const graphic = doc.getGraphic(this._graphicId)
    if (graphic) {
      this._oldBinding = graphic.bindings?.[this._propKey] ?? null
      const newBindings = { ...graphic.bindings }
      if (this._binding) {
        newBindings[this._propKey] = this._binding
      } else {
        delete newBindings[this._propKey]
      }
      doc._updateGraphic(this._graphicId, { bindings: newBindings })
    }
  }

  undo(doc: DocumentModel): void {
    const graphic = doc.getGraphic(this._graphicId)
    if (graphic) {
      const newBindings = { ...graphic.bindings }
      if (this._oldBinding) {
        newBindings[this._propKey] = this._oldBinding
      } else {
        delete newBindings[this._propKey]
      }
      doc._updateGraphic(this._graphicId, { bindings: newBindings })
    }
  }

  getDescription(): string {
    return this._binding ? `设置图形绑定: ${this._propKey}` : `移除图形绑定: ${this._propKey}`
  }
}

export default {
  InsertGraphicCommand,
  RemoveGraphicCommand,
  UpdateGraphicCommand,
  MoveGraphicCommand,
  ReorderGraphicCommand,
  GroupGraphicsCommand,
  UngroupGraphicsCommand,
  SetGraphicBindingCommand,
}
