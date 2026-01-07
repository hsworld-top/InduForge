/**
 * Command - 命令接口
 * 所有编辑操作必须通过命令执行，支持撤销重做
 *
 * 设计原则：
 * - UI 层永远不要直接修改 JSON
 * - 所有修改必须走命令系统
 * - 命令必须支持 execute 和 undo
 */

/**
 * @typedef {import('../document/DocumentModel.js').DocumentModel} DocumentModel
 */

/**
 * 命令接口
 * @interface
 */
export class Command {
    /**
     * 命令类型
     * @type {string}
     * @readonly
     */
    get type() {
        throw new Error('Command.type must be implemented');
    }

    /**
     * 执行命令
     * @param {DocumentModel} doc - 文档模型
     */
    execute(doc) {
        throw new Error('Command.execute must be implemented');
    }

    /**
     * 撤销命令
     * @param {DocumentModel} doc - 文档模型
     */
    undo(doc) {
        throw new Error('Command.undo must be implemented');
    }

    /**
     * 是否可以与另一个命令合并
     * @param {Command} other - 另一个命令
     * @returns {boolean}
     */
    canMerge(other) {
        return false;
    }

    /**
     * 与另一个命令合并
     * @param {Command} other - 另一个命令
     * @returns {Command}
     */
    merge(other) {
        throw new Error('Command.merge not supported');
    }

    /**
     * 获取命令描述（用于历史记录显示）
     * @returns {string}
     */
    getDescription() {
        return this.type;
    }
}

/**
 * 批量命令（事务）
 * 将多个命令合并为一个，支持原子性撤销重做
 */
export class BatchCommand extends Command {
    /**
     * @type {string}
     * @readonly
     */
    get type() {
        return 'Batch';
    }

    /**
     * 创建批量命令
     * @param {Command[]} commands - 命令列表
     * @param {string} [description] - 批量操作描述
     */
    constructor(commands, description = '批量操作') {
        super();
        /** @type {Command[]} */
        this._commands = commands;
        /** @type {string} */
        this._description = description;
    }

    /**
     * 执行所有命令
     * @param {DocumentModel} doc
     */
    execute(doc) {
        for (const cmd of this._commands) {
            cmd.execute(doc);
        }
    }

    /**
     * 逆序撤销所有命令
     * @param {DocumentModel} doc
     */
    undo(doc) {
        for (let i = this._commands.length - 1; i >= 0; i--) {
            this._commands[i].undo(doc);
        }
    }

    /**
     * @returns {string}
     */
    getDescription() {
        return this._description;
    }

    /**
     * 获取子命令列表
     * @returns {Command[]}
     */
    getCommands() {
        return [...this._commands];
    }
}

export default {
    Command,
    BatchCommand,
};

