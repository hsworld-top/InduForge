/**
 * 节点操作命令
 * 包含插入、删除、更新、移动节点的命令实现
 */

import type { DocumentModel } from "../document/DocumentModel";
import type { Change, ComponentNode } from "../document/types";
import { Command } from "./Command";

const UUID_DASH_REGEX = /-/g;

/** 深拷贝删除命令在内存中挂的子树 */
type NodeCloneWithChildren = ComponentNode & {
  _childNodes?: ComponentNode[];
};

export class InsertNodeCommand extends Command {
  private _parentId!: string;
  private _index!: number;
  private _node!: ComponentNode;

  get type(): string {
    return "InsertNode";
  }

  constructor(parentId: string, index: number, node: ComponentNode) {
    super();
    this._parentId = parentId;
    this._index = index;
    this._node = node;
  }

  execute(doc: DocumentModel): void {
    doc._insertNode(this._parentId, this._index, this._node);
  }

  undo(doc: DocumentModel): void {
    doc._removeNode(this._node.id);
  }

  getDescription(): string {
    return `插入组件: ${this._node.type}`;
  }
}

export class RemoveNodeCommand extends Command {
  private _nodeId!: string;
  private _removedNode: NodeCloneWithChildren | null = null;
  private _parentId: string | null = null;
  private _index = -1;

  get type(): string {
    return "RemoveNode";
  }

  constructor(nodeId: string) {
    super();
    this._nodeId = nodeId;
  }

  execute(doc: DocumentModel): void {
    const parent = doc.getParent(this._nodeId);
    this._parentId = parent?.id ?? null;
    if (parent?.children) {
      this._index = parent.children.indexOf(this._nodeId);
    }
    const node = doc.getNode(this._nodeId);
    if (node) {
      this._removedNode = this._deepCloneNode(node, doc);
    }
    doc._removeNode(this._nodeId);
  }

  undo(doc: DocumentModel): void {
    if (this._removedNode && this._parentId !== null) {
      this._restoreNode(doc, this._parentId, this._index, this._removedNode);
    }
  }

  private _deepCloneNode(node: ComponentNode, doc: DocumentModel): NodeCloneWithChildren {
    const clone = JSON.parse(JSON.stringify(node)) as NodeCloneWithChildren;
    if (clone.children?.length) {
      clone._childNodes = clone.children
        .map((childId) => {
          const child = doc.getNode(childId);
          return child ? this._deepCloneNode(child, doc) : null;
        })
        .filter((c): c is NodeCloneWithChildren => c != null);
    }
    return clone;
  }

  private _restoreNode(
    doc: DocumentModel,
    parentId: string,
    index: number,
    node: NodeCloneWithChildren,
  ): void {
    const childNodes = node._childNodes ?? [];
    delete node._childNodes;

    doc._schema.nodesById[node.id] = node;

    const parent = doc.getNode(parentId);
    if (parent) {
      if (!parent.children) {
        parent.children = [];
      }
      const insertIndex = Math.min(Math.max(0, index), parent.children.length);
      parent.children.splice(insertIndex, 0, node.id);
    }

    doc._parentIndex.set(node.id, parentId);
    doc._addToTypeIndex(node);
    doc._addToBindingIndex(node);

    for (let i = 0; i < childNodes.length; i++) {
      const child = childNodes[i];
      if (child) {
        this._restoreNode(doc, node.id, i, child);
      }
    }

    doc._emitChange({
      type: "insert",
      target: "node",
      id: node.id,
      parentId,
      index,
      newValue: node,
    });
  }

  getDescription(): string {
    return `删除组件`;
  }
}

export class UpdateNodeCommand extends Command {
  private _nodeId!: string;
  private _patch!: Partial<ComponentNode>;
  private _oldValues: Partial<ComponentNode> | null = null;

  get type(): string {
    return "UpdateNode";
  }

  constructor(nodeId: string, patch: Partial<ComponentNode>) {
    super();
    this._nodeId = nodeId;
    this._patch = patch;
  }

  execute(doc: DocumentModel): void {
    const node = doc.getNode(this._nodeId);
    if (node) {
      this._oldValues = {};
      const nRec = node as unknown as Record<string, unknown>;
      for (const key of Object.keys(this._patch)) {
        (this._oldValues as Record<string, unknown>)[key] = JSON.parse(
          JSON.stringify(nRec[key] ?? null),
        );
      }
      doc._updateNode(this._nodeId, this._patch);
    }
  }

  undo(doc: DocumentModel): void {
    if (this._oldValues) {
      doc._updateNode(this._nodeId, this._oldValues);
    }
  }

  canMerge(other: Command): boolean {
    return other instanceof UpdateNodeCommand && other._nodeId === this._nodeId;
  }

  merge(other: UpdateNodeCommand): UpdateNodeCommand {
    const mergedPatch = { ...this._patch, ...other._patch };
    const cmd = new UpdateNodeCommand(this._nodeId, mergedPatch);
    cmd._oldValues = this._oldValues;
    return cmd;
  }

  getDescription(): string {
    const keys = Object.keys(this._patch);
    return `更新组件属性: ${keys.join(", ")}`;
  }
}

export class MoveNodeCommand extends Command {
  private _nodeId!: string;
  private _newParentId!: string;
  private _newIndex!: number;
  private _oldParentId: string | null = null;
  private _oldIndex = -1;

  get type(): string {
    return "MoveNode";
  }

  constructor(nodeId: string, newParentId: string, newIndex: number) {
    super();
    this._nodeId = nodeId;
    this._newParentId = newParentId;
    this._newIndex = newIndex;
  }

  execute(doc: DocumentModel): void {
    const oldParent = doc.getParent(this._nodeId);
    this._oldParentId = oldParent?.id ?? null;
    if (oldParent?.children) {
      this._oldIndex = oldParent.children.indexOf(this._nodeId);
    }
    doc._moveNode(this._nodeId, this._newParentId, this._newIndex);
  }

  undo(doc: DocumentModel): void {
    if (this._oldParentId !== null) {
      doc._moveNode(this._nodeId, this._oldParentId, this._oldIndex);
    }
  }

  getDescription(): string {
    return `移动组件`;
  }
}

export class DuplicateNodeCommand extends Command {
  private _sourceNodeId!: string;
  private _newId: string;
  private _offset: { x?: number; y?: number };
  private _createdNode: ComponentNode | null = null;
  private _parentId: string | null = null;
  private _index = -1;

  get type(): string {
    return "DuplicateNode";
  }

  constructor(
    sourceNodeId: string,
    newId?: string,
    offset: { x?: number; y?: number } = { x: 20, y: 20 },
  ) {
    super();
    this._sourceNodeId = sourceNodeId;
    this._newId = newId || crypto.randomUUID().replace(UUID_DASH_REGEX, "").substring(0, 12);
    this._offset = offset;
  }

  execute(doc: DocumentModel): void {
    const sourceNode = doc.getNode(this._sourceNodeId);
    if (!sourceNode) return;

    const parent = doc.getParent(this._sourceNodeId);
    this._parentId = parent?.id ?? null;
    if (parent?.children) {
      this._index = parent.children.indexOf(this._sourceNodeId) + 1;
    }

    this._createdNode = this._deepCloneWithNewIds(sourceNode, doc);

    const dx = this._offset.x || 0;
    const dy = this._offset.y || 0;
    if (this._createdNode.absolutePos) {
      this._createdNode.absolutePos.x = (this._createdNode.absolutePos.x ?? 0) + dx;
      this._createdNode.absolutePos.y = (this._createdNode.absolutePos.y ?? 0) + dy;
    } else if (this._createdNode.layoutItem?.free?.abs) {
      const abs = this._createdNode.layoutItem.free.abs;
      abs.x = (abs.x ?? 0) + dx;
      abs.y = (abs.y ?? 0) + dy;
    }
    if (
      this._createdNode.style &&
      !this._createdNode.absolutePos &&
      !this._createdNode.layoutItem?.free?.abs
    ) {
      if (typeof this._createdNode.style.left === "number") {
        this._createdNode.style.left += dx;
      }
      if (typeof this._createdNode.style.top === "number") {
        this._createdNode.style.top += dy;
      }
    }

    if (this._parentId) {
      doc._insertNode(this._parentId, this._index, this._createdNode);
    }
  }

  undo(doc: DocumentModel): void {
    if (this._createdNode) {
      doc._removeNode(this._createdNode.id);
    }
  }

  private _deepCloneWithNewIds(node: ComponentNode, doc: DocumentModel): ComponentNode {
    const clone = JSON.parse(JSON.stringify(node)) as ComponentNode;

    const replaceIds = (n: ComponentNode): void => {
      const newId = crypto.randomUUID().replace(UUID_DASH_REGEX, "").substring(0, 12);
      n.id = newId;
      if (n.children?.length) {
        const newChildren: string[] = [];
        for (const childId of n.children) {
          const child = doc.getNode(childId);
          if (child) {
            const childClone = JSON.parse(JSON.stringify(child)) as ComponentNode;
            replaceIds(childClone);
            doc._schema.nodesById[childClone.id] = childClone;
            newChildren.push(childClone.id);
          }
        }
        n.children = newChildren;
      }
    };

    clone.id = this._newId;
    if (clone.children?.length) {
      const newChildren: string[] = [];
      for (const childId of clone.children) {
        const child = doc.getNode(childId);
        if (child) {
          const childClone = JSON.parse(JSON.stringify(child)) as ComponentNode;
          replaceIds(childClone);
          newChildren.push(childClone.id);
        }
      }
      clone.children = newChildren;
    }

    return clone;
  }

  getCreatedNodeId(): string | null {
    return this._createdNode?.id ?? null;
  }

  getDescription(): string {
    return `复制组件`;
  }
}

export class SetNodePropsCommand extends Command {
  private _nodeId!: string;
  private _props!: Record<string, unknown>;
  private _oldProps: Record<string, unknown> | null = null;

  get type(): string {
    return "SetNodeProps";
  }

  constructor(nodeId: string, props: Record<string, unknown>) {
    super();
    this._nodeId = nodeId;
    this._props = props;
  }

  execute(doc: DocumentModel): void {
    const node = doc.getNode(this._nodeId);
    if (node) {
      this._oldProps = {};
      for (const key of Object.keys(this._props)) {
        this._oldProps[key] = node.props?.[key];
      }
      const newProps = { ...node.props, ...this._props };
      doc._updateNode(this._nodeId, { props: newProps });
    }
  }

  undo(doc: DocumentModel): void {
    const node = doc.getNode(this._nodeId);
    if (node && this._oldProps) {
      const restoredProps = { ...node.props };
      for (const [key, value] of Object.entries(this._oldProps)) {
        if (value === undefined) {
          delete restoredProps[key];
        } else {
          restoredProps[key] = value;
        }
      }
      doc._updateNode(this._nodeId, { props: restoredProps });
    }
  }

  canMerge(other: Command): boolean {
    return other instanceof SetNodePropsCommand && other._nodeId === this._nodeId;
  }

  merge(other: SetNodePropsCommand): SetNodePropsCommand {
    const mergedProps = { ...this._props, ...other._props };
    const cmd = new SetNodePropsCommand(this._nodeId, mergedProps);
    cmd._oldProps = this._oldProps;
    return cmd;
  }

  getDescription(): string {
    return `设置组件属性`;
  }
}

export class SetNodeStyleCommand extends Command {
  private _nodeId!: string;
  private _style!: Record<string, unknown>;
  private _oldStyle: Record<string, unknown> | null = null;

  get type(): string {
    return "SetNodeStyle";
  }

  constructor(nodeId: string, style: Record<string, unknown>) {
    super();
    this._nodeId = nodeId;
    this._style = style;
  }

  execute(doc: DocumentModel): void {
    const node = doc.getNode(this._nodeId);
    if (node) {
      this._oldStyle = {};
      for (const key of Object.keys(this._style)) {
        this._oldStyle[key] = node.style?.[key];
      }
      const newStyle = { ...node.style, ...this._style };
      doc._updateNode(this._nodeId, { style: newStyle });
    }
  }

  undo(doc: DocumentModel): void {
    const node = doc.getNode(this._nodeId);
    if (node && this._oldStyle) {
      const restoredStyle = { ...node.style };
      for (const [key, value] of Object.entries(this._oldStyle)) {
        if (value === undefined) {
          delete restoredStyle[key];
        } else {
          restoredStyle[key] = value;
        }
      }
      doc._updateNode(this._nodeId, { style: restoredStyle });
    }
  }

  canMerge(other: Command): boolean {
    return other instanceof SetNodeStyleCommand && other._nodeId === this._nodeId;
  }

  merge(other: SetNodeStyleCommand): SetNodeStyleCommand {
    const mergedStyle = { ...this._style, ...other._style };
    const cmd = new SetNodeStyleCommand(this._nodeId, mergedStyle);
    cmd._oldStyle = this._oldStyle;
    return cmd;
  }

  getDescription(): string {
    return `设置组件样式`;
  }
}

export class ReorderNodeCommand extends Command {
  private _nodeId!: string;
  private _direction!: "up" | "down" | "top" | "bottom";
  private _parentId: string | null = null;
  private _oldIndex = -1;
  private _newIndex = -1;

  get type(): string {
    return "ReorderNode";
  }

  constructor(nodeId: string, direction: "up" | "down" | "top" | "bottom") {
    super();
    this._nodeId = nodeId;
    this._direction = direction;
  }

  execute(doc: DocumentModel): void {
    const parent = doc.getParent(this._nodeId);
    if (!parent?.children) return;

    this._parentId = parent.id;
    this._oldIndex = parent.children.indexOf(this._nodeId);
    if (this._oldIndex === -1) return;

    const childrenCount = parent.children.length;

    switch (this._direction) {
      case "up":
        this._newIndex = Math.min(childrenCount - 1, this._oldIndex + 1);
        break;
      case "down":
        this._newIndex = Math.max(0, this._oldIndex - 1);
        break;
      case "top":
        this._newIndex = childrenCount - 1;
        break;
      case "bottom":
        this._newIndex = 0;
        break;
    }

    if (this._newIndex === this._oldIndex) return;

    const children = [...parent.children];
    const [moved] = children.splice(this._oldIndex, 1);
    if (moved === undefined) return;
    children.splice(this._newIndex, 0, moved);
    parent.children = children;

    doc._emitChange({
      type: "reorder",
      target: "node",
      id: this._nodeId,
      parentId: this._parentId ?? undefined,
      oldIndex: this._oldIndex,
      newIndex: this._newIndex,
    } as unknown as Change);
  }

  undo(doc: DocumentModel): void {
    if (this._parentId === null) return;

    const parent = doc.getNode(this._parentId);
    if (!parent?.children) return;

    const children = [...parent.children];
    const [moved] = children.splice(this._newIndex, 1);
    if (moved === undefined) return;
    children.splice(this._oldIndex, 0, moved);
    parent.children = children;

    doc._emitChange({
      type: "reorder",
      target: "node",
      id: this._nodeId,
      parentId: this._parentId,
      oldIndex: this._newIndex,
      newIndex: this._oldIndex,
    } as unknown as Change);
  }

  getDescription(): string {
    const directionText: Record<string, string> = {
      up: "上移图层",
      down: "下移图层",
      top: "置顶",
      bottom: "置底",
    };
    return directionText[this._direction] || "调整图层顺序";
  }
}

export class ToggleNodeVisibilityCommand extends Command {
  private _nodeId!: string;
  private _targetHidden: boolean | undefined;
  private _oldHidden = false;

  get type(): string {
    return "ToggleNodeVisibility";
  }

  constructor(nodeId: string, hidden?: boolean) {
    super();
    this._nodeId = nodeId;
    this._targetHidden = hidden;
  }

  execute(doc: DocumentModel): void {
    const node = doc.getNode(this._nodeId);
    if (!node) return;

    this._oldHidden = node.hidden ?? false;
    const newHidden = this._targetHidden !== undefined ? this._targetHidden : !this._oldHidden;

    doc._updateNode(this._nodeId, { hidden: newHidden });
  }

  undo(doc: DocumentModel): void {
    doc._updateNode(this._nodeId, { hidden: this._oldHidden });
  }

  getDescription(): string {
    return this._oldHidden ? "显示组件" : "隐藏组件";
  }
}

export class ToggleNodeLockCommand extends Command {
  private _nodeId!: string;
  private _targetLocked: boolean | undefined;
  private _oldLocked = false;

  get type(): string {
    return "ToggleNodeLock";
  }

  constructor(nodeId: string, locked?: boolean) {
    super();
    this._nodeId = nodeId;
    this._targetLocked = locked;
  }

  execute(doc: DocumentModel): void {
    const node = doc.getNode(this._nodeId);
    if (!node) return;

    this._oldLocked = node.locked ?? false;
    const newLocked = this._targetLocked !== undefined ? this._targetLocked : !this._oldLocked;

    doc._updateNode(this._nodeId, { locked: newLocked });
  }

  undo(doc: DocumentModel): void {
    doc._updateNode(this._nodeId, { locked: this._oldLocked });
  }

  getDescription(): string {
    return this._oldLocked ? "解锁组件" : "锁定组件";
  }
}

export default {
  InsertNodeCommand,
  RemoveNodeCommand,
  UpdateNodeCommand,
  MoveNodeCommand,
  DuplicateNodeCommand,
  SetNodePropsCommand,
  SetNodeStyleCommand,
  ReorderNodeCommand,
  ToggleNodeVisibilityCommand,
  ToggleNodeLockCommand,
};
