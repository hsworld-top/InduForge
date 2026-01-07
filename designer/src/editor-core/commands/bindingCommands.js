/**
 * 绑定操作命令
 * 包含数据绑定的设置、更新、删除命令
 */

import { Command } from './Command.js';

/**
 * @typedef {import('../document/DocumentModel.js').DocumentModel} DocumentModel
 * @typedef {import('../document/types.js').Binding} Binding
 * @typedef {import('../document/types.js').ComponentNode} ComponentNode
 */

/**
 * 设置节点绑定命令
 */
export class SetBindingCommand extends Command {
    get type() {
        return 'SetBinding';
    }

    /**
     * 创建设置绑定命令
     * @param {string} nodeId - 节点 ID
     * @param {string} propKey - 属性键
     * @param {Binding | null} binding - 绑定配置，null 表示删除绑定
     */
    constructor(nodeId, propKey, binding) {
        super();
        /** @type {string} */
        this._nodeId = nodeId;
        /** @type {string} */
        this._propKey = propKey;
        /** @type {Binding | null} */
        this._binding = binding;
        /** @type {Binding | null} */
        this._oldBinding = null;
    }

    /**
     * @param {DocumentModel} doc
     */
    execute(doc) {
        const node = doc.getNode(this._nodeId);
        if (node) {
            // 保存旧绑定
            this._oldBinding = node.bindings?.[this._propKey] ?? null;

            // 构建新绑定对象
            const newBindings = { ...node.bindings };
            if (this._binding) {
                newBindings[this._propKey] = this._binding;
            } else {
                delete newBindings[this._propKey];
            }

            // 更新节点
            doc._updateNode(this._nodeId, { bindings: newBindings });
        }
    }

    /**
     * @param {DocumentModel} doc
     */
    undo(doc) {
        const node = doc.getNode(this._nodeId);
        if (node) {
            const newBindings = { ...node.bindings };
            if (this._oldBinding) {
                newBindings[this._propKey] = this._oldBinding;
            } else {
                delete newBindings[this._propKey];
            }

            doc._updateNode(this._nodeId, { bindings: newBindings });
        }
    }

    getDescription() {
        return this._binding ? `设置绑定: ${this._propKey}` : `移除绑定: ${this._propKey}`;
    }
}

/**
 * 批量设置绑定命令
 */
export class SetMultipleBindingsCommand extends Command {
    get type() {
        return 'SetMultipleBindings';
    }

    /**
     * 创建批量设置绑定命令
     * @param {string} nodeId - 节点 ID
     * @param {Record<string, Binding | null>} bindings - 绑定配置映射
     */
    constructor(nodeId, bindings) {
        super();
        /** @type {string} */
        this._nodeId = nodeId;
        /** @type {Record<string, Binding | null>} */
        this._bindings = bindings;
        /** @type {Record<string, Binding | null>} */
        this._oldBindings = {};
    }

    /**
     * @param {DocumentModel} doc
     */
    execute(doc) {
        const node = doc.getNode(this._nodeId);
        if (node) {
            // 保存旧绑定
            for (const propKey of Object.keys(this._bindings)) {
                this._oldBindings[propKey] = node.bindings?.[propKey] ?? null;
            }

            // 构建新绑定对象
            const newBindings = { ...node.bindings };
            for (const [propKey, binding] of Object.entries(this._bindings)) {
                if (binding) {
                    newBindings[propKey] = binding;
                } else {
                    delete newBindings[propKey];
                }
            }

            doc._updateNode(this._nodeId, { bindings: newBindings });
        }
    }

    /**
     * @param {DocumentModel} doc
     */
    undo(doc) {
        const node = doc.getNode(this._nodeId);
        if (node) {
            const newBindings = { ...node.bindings };
            for (const [propKey, oldBinding] of Object.entries(this._oldBindings)) {
                if (oldBinding) {
                    newBindings[propKey] = oldBinding;
                } else {
                    delete newBindings[propKey];
                }
            }

            doc._updateNode(this._nodeId, { bindings: newBindings });
        }
    }

    getDescription() {
        const count = Object.keys(this._bindings).length;
        return `批量设置绑定 (${count} 个)`;
    }
}

/**
 * 清除所有绑定命令
 */
export class ClearAllBindingsCommand extends Command {
    get type() {
        return 'ClearAllBindings';
    }

    /**
     * 创建清除所有绑定命令
     * @param {string} nodeId - 节点 ID
     */
    constructor(nodeId) {
        super();
        /** @type {string} */
        this._nodeId = nodeId;
        /** @type {Record<string, Binding>} */
        this._oldBindings = {};
    }

    /**
     * @param {DocumentModel} doc
     */
    execute(doc) {
        const node = doc.getNode(this._nodeId);
        if (node && node.bindings) {
            // 保存所有旧绑定
            this._oldBindings = JSON.parse(JSON.stringify(node.bindings));

            // 清空绑定
            doc._updateNode(this._nodeId, { bindings: {} });
        }
    }

    /**
     * @param {DocumentModel} doc
     */
    undo(doc) {
        if (Object.keys(this._oldBindings).length > 0) {
            doc._updateNode(this._nodeId, { bindings: this._oldBindings });
        }
    }

    getDescription() {
        return `清除所有绑定`;
    }
}

/**
 * 更新绑定转换命令
 */
export class UpdateBindingTransformCommand extends Command {
    get type() {
        return 'UpdateBindingTransform';
    }

    /**
     * 创建更新绑定转换命令
     * @param {string} nodeId - 节点 ID
     * @param {string} propKey - 属性键
     * @param {import('../document/types.js').TransformOp[]} transform - 转换操作列表
     */
    constructor(nodeId, propKey, transform) {
        super();
        /** @type {string} */
        this._nodeId = nodeId;
        /** @type {string} */
        this._propKey = propKey;
        /** @type {import('../document/types.js').TransformOp[]} */
        this._transform = transform;
        /** @type {import('../document/types.js').TransformOp[] | undefined} */
        this._oldTransform = undefined;
    }

    /**
     * @param {DocumentModel} doc
     */
    execute(doc) {
        const node = doc.getNode(this._nodeId);
        if (node && node.bindings?.[this._propKey]) {
            const binding = node.bindings[this._propKey];
            this._oldTransform = binding.transform ? [...binding.transform] : undefined;

            // 更新 transform
            const newBindings = {
                ...node.bindings,
                [this._propKey]: {
                    ...binding,
                    transform: this._transform,
                },
            };

            doc._updateNode(this._nodeId, { bindings: newBindings });
        }
    }

    /**
     * @param {DocumentModel} doc
     */
    undo(doc) {
        const node = doc.getNode(this._nodeId);
        if (node && node.bindings?.[this._propKey]) {
            const binding = node.bindings[this._propKey];
            const newBindings = {
                ...node.bindings,
                [this._propKey]: {
                    ...binding,
                    transform: this._oldTransform,
                },
            };

            doc._updateNode(this._nodeId, { bindings: newBindings });
        }
    }

    getDescription() {
        return `更新绑定转换: ${this._propKey}`;
    }
}

/**
 * 设置绑定降级值命令
 */
export class SetBindingFallbackCommand extends Command {
    get type() {
        return 'SetBindingFallback';
    }

    /**
     * 创建设置绑定降级值命令
     * @param {string} nodeId - 节点 ID
     * @param {string} propKey - 属性键
     * @param {*} fallback - 降级值
     */
    constructor(nodeId, propKey, fallback) {
        super();
        /** @type {string} */
        this._nodeId = nodeId;
        /** @type {string} */
        this._propKey = propKey;
        /** @type {*} */
        this._fallback = fallback;
        /** @type {*} */
        this._oldFallback = undefined;
    }

    /**
     * @param {DocumentModel} doc
     */
    execute(doc) {
        const node = doc.getNode(this._nodeId);
        if (node && node.bindings?.[this._propKey]) {
            const binding = node.bindings[this._propKey];
            this._oldFallback = binding.fallback;

            const newBindings = {
                ...node.bindings,
                [this._propKey]: {
                    ...binding,
                    fallback: this._fallback,
                },
            };

            doc._updateNode(this._nodeId, { bindings: newBindings });
        }
    }

    /**
     * @param {DocumentModel} doc
     */
    undo(doc) {
        const node = doc.getNode(this._nodeId);
        if (node && node.bindings?.[this._propKey]) {
            const binding = node.bindings[this._propKey];
            const newBindings = {
                ...node.bindings,
                [this._propKey]: {
                    ...binding,
                    fallback: this._oldFallback,
                },
            };

            doc._updateNode(this._nodeId, { bindings: newBindings });
        }
    }

    getDescription() {
        return `设置绑定降级值: ${this._propKey}`;
    }
}

/**
 * 复制绑定命令
 */
export class CopyBindingsCommand extends Command {
    get type() {
        return 'CopyBindings';
    }

    /**
     * 创建复制绑定命令
     * @param {string} sourceNodeId - 源节点 ID
     * @param {string} targetNodeId - 目标节点 ID
     * @param {string[]} [propKeys] - 要复制的属性键列表，不传则复制全部
     */
    constructor(sourceNodeId, targetNodeId, propKeys) {
        super();
        /** @type {string} */
        this._sourceNodeId = sourceNodeId;
        /** @type {string} */
        this._targetNodeId = targetNodeId;
        /** @type {string[] | undefined} */
        this._propKeys = propKeys;
        /** @type {Record<string, Binding>} */
        this._oldTargetBindings = {};
        /** @type {Record<string, Binding>} */
        this._copiedBindings = {};
    }

    /**
     * @param {DocumentModel} doc
     */
    execute(doc) {
        const sourceNode = doc.getNode(this._sourceNodeId);
        const targetNode = doc.getNode(this._targetNodeId);

        if (!sourceNode || !targetNode || !sourceNode.bindings) return;

        // 确定要复制的绑定
        const keys = this._propKeys || Object.keys(sourceNode.bindings);
        for (const key of keys) {
            if (sourceNode.bindings[key]) {
                this._copiedBindings[key] = JSON.parse(JSON.stringify(sourceNode.bindings[key]));
                this._oldTargetBindings[key] = targetNode.bindings?.[key] ?? null;
            }
        }

        // 合并到目标节点
        const newBindings = { ...targetNode.bindings, ...this._copiedBindings };
        doc._updateNode(this._targetNodeId, { bindings: newBindings });
    }

    /**
     * @param {DocumentModel} doc
     */
    undo(doc) {
        const targetNode = doc.getNode(this._targetNodeId);
        if (!targetNode) return;

        // 恢复旧绑定
        const newBindings = { ...targetNode.bindings };
        for (const [key, oldBinding] of Object.entries(this._oldTargetBindings)) {
            if (oldBinding) {
                newBindings[key] = oldBinding;
            } else {
                delete newBindings[key];
            }
        }

        doc._updateNode(this._targetNodeId, { bindings: newBindings });
    }

    getDescription() {
        return `复制绑定`;
    }
}

export default {
    SetBindingCommand,
    SetMultipleBindingsCommand,
    ClearAllBindingsCommand,
    UpdateBindingTransformCommand,
    SetBindingFallbackCommand,
    CopyBindingsCommand,
};

