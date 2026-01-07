/**
 * 节点操作命令
 * 包含插入、删除、更新、移动节点的命令实现
 */

import { Command } from './Command.js';

/**
 * @typedef {import('../document/DocumentModel.js').DocumentModel} DocumentModel
 * @typedef {import('../document/types.js').ComponentNode} ComponentNode
 */

/**
 * 插入节点命令
 */
export class InsertNodeCommand extends Command {
    get type() {
        return 'InsertNode';
    }

    /**
     * 创建插入节点命令
     * @param {string} parentId - 父节点 ID
     * @param {number} index - 插入位置
     * @param {ComponentNode} node - 要插入的节点
     */
    constructor(parentId, index, node) {
        super();
        /** @type {string} */
        this._parentId = parentId;
        /** @type {number} */
        this._index = index;
        /** @type {ComponentNode} */
        this._node = node;
    }

    /**
     * @param {DocumentModel} doc
     */
    execute(doc) {
        doc._insertNode(this._parentId, this._index, this._node);
    }

    /**
     * @param {DocumentModel} doc
     */
    undo(doc) {
        doc._removeNode(this._node.id);
    }

    getDescription() {
        return `插入组件: ${this._node.type}`;
    }
}

/**
 * 删除节点命令
 */
export class RemoveNodeCommand extends Command {
    get type() {
        return 'RemoveNode';
    }

    /**
     * 创建删除节点命令
     * @param {string} nodeId - 要删除的节点 ID
     */
    constructor(nodeId) {
        super();
        /** @type {string} */
        this._nodeId = nodeId;
        /** @type {ComponentNode | null} */
        this._removedNode = null;
        /** @type {string | null} */
        this._parentId = null;
        /** @type {number} */
        this._index = -1;
    }

    /**
     * @param {DocumentModel} doc
     */
    execute(doc) {
        // 保存删除前的状态
        const parent = doc.getParent(this._nodeId);
        this._parentId = parent?.id ?? null;
        if (parent && parent.children) {
            this._index = parent.children.indexOf(this._nodeId);
        }

        // 深拷贝节点及其所有后代（用于撤销）
        const node = doc.getNode(this._nodeId);
        if (node) {
            this._removedNode = this._deepCloneNode(node, doc);
        }

        // 执行删除
        doc._removeNode(this._nodeId);
    }

    /**
     * @param {DocumentModel} doc
     */
    undo(doc) {
        if (this._removedNode && this._parentId !== null) {
            // 恢复节点及其所有后代
            this._restoreNode(doc, this._parentId, this._index, this._removedNode);
        }
    }

    /**
     * 深拷贝节点及其所有后代
     * @param {ComponentNode} node
     * @param {DocumentModel} doc
     * @returns {ComponentNode}
     * @private
     */
    _deepCloneNode(node, doc) {
        const clone = JSON.parse(JSON.stringify(node));

        // 递归获取所有后代节点
        if (clone.children && clone.children.length > 0) {
            clone._childNodes = clone.children.map((childId) => {
                const child = doc.getNode(childId);
                return child ? this._deepCloneNode(child, doc) : null;
            }).filter(Boolean);
        }

        return clone;
    }

    /**
     * 恢复节点及其所有后代
     * @param {DocumentModel} doc
     * @param {string} parentId
     * @param {number} index
     * @param {ComponentNode} node
     * @private
     */
    _restoreNode(doc, parentId, index, node) {
        // 分离子节点数据
        const childNodes = node._childNodes || [];
        delete node._childNodes;

        // 先将节点添加到 nodesById
        doc._schema.nodesById[node.id] = node;

        // 添加到父节点
        const parent = doc.getNode(parentId);
        if (parent) {
            if (!parent.children) {
                parent.children = [];
            }
            const insertIndex = Math.min(Math.max(0, index), parent.children.length);
            parent.children.splice(insertIndex, 0, node.id);
        }

        // 更新索引
        doc._parentIndex.set(node.id, parentId);
        doc._addToTypeIndex(node);
        doc._addToBindingIndex(node);

        // 递归恢复子节点
        for (let i = 0; i < childNodes.length; i++) {
            this._restoreNode(doc, node.id, i, childNodes[i]);
        }

        // 触发变更事件
        doc._emitChange({
            type: 'insert',
            target: 'node',
            id: node.id,
            parentId,
            index,
            newValue: node,
        });
    }

    getDescription() {
        return `删除组件`;
    }
}

/**
 * 更新节点命令
 */
export class UpdateNodeCommand extends Command {
    get type() {
        return 'UpdateNode';
    }

    /**
     * 创建更新节点命令
     * @param {string} nodeId - 节点 ID
     * @param {Partial<ComponentNode>} patch - 更新内容
     */
    constructor(nodeId, patch) {
        super();
        /** @type {string} */
        this._nodeId = nodeId;
        /** @type {Partial<ComponentNode>} */
        this._patch = patch;
        /** @type {Partial<ComponentNode> | null} */
        this._oldValues = null;
    }

    /**
     * @param {DocumentModel} doc
     */
    execute(doc) {
        const node = doc.getNode(this._nodeId);
        if (node) {
            // 保存旧值
            this._oldValues = {};
            for (const key of Object.keys(this._patch)) {
                this._oldValues[key] = JSON.parse(JSON.stringify(node[key] ?? null));
            }

            // 执行更新
            doc._updateNode(this._nodeId, this._patch);
        }
    }

    /**
     * @param {DocumentModel} doc
     */
    undo(doc) {
        if (this._oldValues) {
            doc._updateNode(this._nodeId, this._oldValues);
        }
    }

    /**
     * @param {Command} other
     * @returns {boolean}
     */
    canMerge(other) {
        // 连续更新同一节点可以合并
        return (
            other instanceof UpdateNodeCommand &&
            other._nodeId === this._nodeId
        );
    }

    /**
     * @param {UpdateNodeCommand} other
     * @returns {UpdateNodeCommand}
     */
    merge(other) {
        const mergedPatch = { ...this._patch, ...other._patch };
        const cmd = new UpdateNodeCommand(this._nodeId, mergedPatch);
        cmd._oldValues = this._oldValues;
        return cmd;
    }

    getDescription() {
        const keys = Object.keys(this._patch);
        return `更新组件属性: ${keys.join(', ')}`;
    }
}

/**
 * 移动节点命令
 */
export class MoveNodeCommand extends Command {
    get type() {
        return 'MoveNode';
    }

    /**
     * 创建移动节点命令
     * @param {string} nodeId - 节点 ID
     * @param {string} newParentId - 新父节点 ID
     * @param {number} newIndex - 新位置索引
     */
    constructor(nodeId, newParentId, newIndex) {
        super();
        /** @type {string} */
        this._nodeId = nodeId;
        /** @type {string} */
        this._newParentId = newParentId;
        /** @type {number} */
        this._newIndex = newIndex;
        /** @type {string | null} */
        this._oldParentId = null;
        /** @type {number} */
        this._oldIndex = -1;
    }

    /**
     * @param {DocumentModel} doc
     */
    execute(doc) {
        // 保存旧位置
        const oldParent = doc.getParent(this._nodeId);
        this._oldParentId = oldParent?.id ?? null;
        if (oldParent && oldParent.children) {
            this._oldIndex = oldParent.children.indexOf(this._nodeId);
        }

        // 执行移动
        doc._moveNode(this._nodeId, this._newParentId, this._newIndex);
    }

    /**
     * @param {DocumentModel} doc
     */
    undo(doc) {
        if (this._oldParentId !== null) {
            doc._moveNode(this._nodeId, this._oldParentId, this._oldIndex);
        }
    }

    getDescription() {
        return `移动组件`;
    }
}

/**
 * 复制节点命令
 */
export class DuplicateNodeCommand extends Command {
    get type() {
        return 'DuplicateNode';
    }

    /**
     * 创建复制节点命令
     * @param {string} sourceNodeId - 源节点 ID
     * @param {string} [newId] - 新节点 ID（可选）
     * @param {{x?: number, y?: number}} [offset] - 位置偏移
     */
    constructor(sourceNodeId, newId, offset = { x: 20, y: 20 }) {
        super();
        /** @type {string} */
        this._sourceNodeId = sourceNodeId;
        /** @type {string} */
        this._newId = newId || crypto.randomUUID().replace(/-/g, '').substring(0, 12);
        /** @type {{x?: number, y?: number}} */
        this._offset = offset;
        /** @type {ComponentNode | null} */
        this._createdNode = null;
        /** @type {string | null} */
        this._parentId = null;
        /** @type {number} */
        this._index = -1;
    }

    /**
     * @param {DocumentModel} doc
     */
    execute(doc) {
        const sourceNode = doc.getNode(this._sourceNodeId);
        if (!sourceNode) return;

        // 获取父节点信息
        const parent = doc.getParent(this._sourceNodeId);
        this._parentId = parent?.id ?? null;
        if (parent && parent.children) {
            this._index = parent.children.indexOf(this._sourceNodeId) + 1;
        }

        // 深拷贝节点
        this._createdNode = this._deepCloneWithNewIds(sourceNode, doc);

        // 应用位置偏移
        if (this._createdNode.style) {
            if (typeof this._createdNode.style.left === 'number') {
                this._createdNode.style.left += this._offset.x || 0;
            }
            if (typeof this._createdNode.style.top === 'number') {
                this._createdNode.style.top += this._offset.y || 0;
            }
        }

        // 插入复制的节点
        if (this._parentId) {
            doc._insertNode(this._parentId, this._index, this._createdNode);
        }
    }

    /**
     * @param {DocumentModel} doc
     */
    undo(doc) {
        if (this._createdNode) {
            doc._removeNode(this._createdNode.id);
        }
    }

    /**
     * 深拷贝并生成新 ID
     * @param {ComponentNode} node
     * @param {DocumentModel} doc
     * @returns {ComponentNode}
     * @private
     */
    _deepCloneWithNewIds(node, doc) {
        const clone = JSON.parse(JSON.stringify(node));
        const idMap = new Map();

        // 递归替换 ID
        const replaceIds = (n) => {
            const oldId = n.id;
            const newId = crypto.randomUUID().replace(/-/g, '').substring(0, 12);
            idMap.set(oldId, newId);
            n.id = newId;

            if (n.children && n.children.length > 0) {
                // 获取子节点数据并递归处理
                const newChildren = [];
                for (const childId of n.children) {
                    const child = doc.getNode(childId);
                    if (child) {
                        const childClone = JSON.parse(JSON.stringify(child));
                        replaceIds(childClone);
                        // 将子节点添加到 nodesById
                        doc._schema.nodesById[childClone.id] = childClone;
                        newChildren.push(childClone.id);
                    }
                }
                n.children = newChildren;
            }
        };

        // 主节点使用指定的 ID
        clone.id = this._newId;
        if (clone.children && clone.children.length > 0) {
            const newChildren = [];
            for (const childId of clone.children) {
                const child = doc.getNode(childId);
                if (child) {
                    const childClone = JSON.parse(JSON.stringify(child));
                    replaceIds(childClone);
                    newChildren.push(childClone.id);
                }
            }
            clone.children = newChildren;
        }

        return clone;
    }

    /**
     * 获取创建的节点 ID
     * @returns {string | null}
     */
    getCreatedNodeId() {
        return this._createdNode?.id ?? null;
    }

    getDescription() {
        return `复制组件`;
    }
}

/**
 * 设置节点属性命令
 */
export class SetNodePropsCommand extends Command {
    get type() {
        return 'SetNodeProps';
    }

    /**
     * 创建设置节点属性命令
     * @param {string} nodeId - 节点 ID
     * @param {Record<string, *>} props - 要设置的属性
     */
    constructor(nodeId, props) {
        super();
        /** @type {string} */
        this._nodeId = nodeId;
        /** @type {Record<string, *>} */
        this._props = props;
        /** @type {Record<string, *> | null} */
        this._oldProps = null;
    }

    /**
     * @param {DocumentModel} doc
     */
    execute(doc) {
        const node = doc.getNode(this._nodeId);
        if (node) {
            // 保存旧属性
            this._oldProps = {};
            for (const key of Object.keys(this._props)) {
                this._oldProps[key] = node.props?.[key];
            }

            // 合并新属性
            const newProps = { ...node.props, ...this._props };
            doc._updateNode(this._nodeId, { props: newProps });
        }
    }

    /**
     * @param {DocumentModel} doc
     */
    undo(doc) {
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

    /**
     * @param {Command} other
     * @returns {boolean}
     */
    canMerge(other) {
        return (
            other instanceof SetNodePropsCommand &&
            other._nodeId === this._nodeId
        );
    }

    /**
     * @param {SetNodePropsCommand} other
     * @returns {SetNodePropsCommand}
     */
    merge(other) {
        const mergedProps = { ...this._props, ...other._props };
        const cmd = new SetNodePropsCommand(this._nodeId, mergedProps);
        cmd._oldProps = this._oldProps;
        return cmd;
    }

    getDescription() {
        return `设置组件属性`;
    }
}

/**
 * 设置节点样式命令
 */
export class SetNodeStyleCommand extends Command {
    get type() {
        return 'SetNodeStyle';
    }

    /**
     * 创建设置节点样式命令
     * @param {string} nodeId - 节点 ID
     * @param {Record<string, *>} style - 要设置的样式
     */
    constructor(nodeId, style) {
        super();
        /** @type {string} */
        this._nodeId = nodeId;
        /** @type {Record<string, *>} */
        this._style = style;
        /** @type {Record<string, *> | null} */
        this._oldStyle = null;
    }

    /**
     * @param {DocumentModel} doc
     */
    execute(doc) {
        const node = doc.getNode(this._nodeId);
        if (node) {
            // 保存旧样式
            this._oldStyle = {};
            for (const key of Object.keys(this._style)) {
                this._oldStyle[key] = node.style?.[key];
            }

            // 合并新样式
            const newStyle = { ...node.style, ...this._style };
            doc._updateNode(this._nodeId, { style: newStyle });
        }
    }

    /**
     * @param {DocumentModel} doc
     */
    undo(doc) {
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

    /**
     * @param {Command} other
     * @returns {boolean}
     */
    canMerge(other) {
        return (
            other instanceof SetNodeStyleCommand &&
            other._nodeId === this._nodeId
        );
    }

    /**
     * @param {SetNodeStyleCommand} other
     * @returns {SetNodeStyleCommand}
     */
    merge(other) {
        const mergedStyle = { ...this._style, ...other._style };
        const cmd = new SetNodeStyleCommand(this._nodeId, mergedStyle);
        cmd._oldStyle = this._oldStyle;
        return cmd;
    }

    getDescription() {
        return `设置组件样式`;
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
};

