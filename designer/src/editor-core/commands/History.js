/**
 * History - 撤销重做栈
 * 管理命令的撤销和重做
 *
 * 设计原则：
 * - 基于命令模式，而非快照模式
 * - 支持命令合并（连续输入）
 * - 支持最大栈大小限制
 * - 提供事件通知
 */

import { EventEmitter } from "../utils/EventEmitter.js";
import { BatchCommand } from "./Command.js";

/**
 * @typedef {import('./Command.js').Command} Command
 * @typedef {import('../document/DocumentModel.js').DocumentModel} DocumentModel
 */

/**
 * 历史记录条目
 * @typedef {Object} HistoryEntry
 * @property {Command} command - 命令
 * @property {number} timestamp - 执行时间戳
 * @property {string} [userId] - 执行用户 ID
 */

/**
 * History 类
 * 管理命令的撤销和重做
 */
export class History extends EventEmitter {
  /**
   * 创建历史记录管理器
   * @param {DocumentModel} doc - 文档模型
   * @param {Object} [options] - 配置选项
   * @param {number} [options.maxSize=100] - 最大栈大小
   * @param {number} [options.mergeWindow=1000] - 命令合并时间窗口（毫秒）
   */
  constructor(doc, options = {}) {
    super();

    /** @type {DocumentModel} */
    this._doc = doc;

    /** @type {number} */
    this._maxSize = options.maxSize ?? 100;

    /** @type {number} */
    this._mergeWindow = options.mergeWindow ?? 1000;

    /** @type {HistoryEntry[]} */
    this._undoStack = [];

    /** @type {HistoryEntry[]} */
    this._redoStack = [];

    /** @type {number} */
    this._lastExecuteTime = 0;

    /** @type {boolean} */
    this._isExecuting = false;
  }

  // ==================== 状态查询 ====================

  /**
   * 是否可以撤销
   * @returns {boolean}
   */
  canUndo() {
    return this._undoStack.length > 0;
  }

  /**
   * 是否可以重做
   * @returns {boolean}
   */
  canRedo() {
    return this._redoStack.length > 0;
  }

  /**
   * 获取撤销栈长度
   * @returns {number}
   */
  get undoStackSize() {
    return this._undoStack.length;
  }

  /**
   * 获取重做栈长度
   * @returns {number}
   */
  get redoStackSize() {
    return this._redoStack.length;
  }

  /**
   * 获取撤销栈描述列表
   * @returns {string[]}
   */
  getUndoDescriptions() {
    return this._undoStack.map((entry) => entry.command.getDescription());
  }

  /**
   * 获取重做栈描述列表
   * @returns {string[]}
   */
  getRedoDescriptions() {
    return this._redoStack.map((entry) => entry.command.getDescription());
  }

  // ==================== 命令执行 ====================

  /**
   * 执行命令
   * @param {Command} command - 要执行的命令
   * @param {Object} [options] - 执行选项
   * @param {boolean} [options.skipMerge=false] - 是否跳过合并检查
   * @param {string} [options.userId] - 执行用户 ID
   */
  execute(command, options = {}) {
    if (this._isExecuting) {
      console.warn("History: 正在执行命令，忽略重入");
      return;
    }

    this._isExecuting = true;

    try {
      const now = Date.now();

      // 执行命令
      command.execute(this._doc);

      // 尝试合并命令
      const lastEntry = this._undoStack[this._undoStack.length - 1];
      const shouldMerge =
        !options.skipMerge &&
        lastEntry &&
        now - this._lastExecuteTime < this._mergeWindow &&
        lastEntry.command.canMerge?.(command);

      if (shouldMerge) {
        // 合并到上一个命令
        const mergedCommand = lastEntry.command.merge(command);
        this._undoStack[this._undoStack.length - 1] = {
          command: mergedCommand,
          timestamp: now,
          userId: options.userId,
        };
      } else {
        // 添加新命令到撤销栈
        this._undoStack.push({
          command,
          timestamp: now,
          userId: options.userId,
        });
      }

      // 清空重做栈
      this._redoStack = [];

      // 限制栈大小
      while (this._undoStack.length > this._maxSize) {
        this._undoStack.shift();
      }

      this._lastExecuteTime = now;

      // 触发事件
      this.emit("change", {
        type: "execute",
        command,
        canUndo: this.canUndo(),
        canRedo: this.canRedo(),
      });
    } finally {
      this._isExecuting = false;
    }
  }

  /**
   * 撤销
   * @returns {boolean} 是否成功撤销
   */
  undo() {
    if (!this.canUndo()) {
      console.warn("History: 无法撤销，撤销栈为空");
      return false;
    }

    if (this._isExecuting) {
      console.warn("History: 正在执行命令，忽略撤销");
      return false;
    }

    this._isExecuting = true;

    try {
      const entry = this._undoStack.pop();
      entry.command.undo(this._doc);
      this._redoStack.push(entry);

      // 触发事件
      this.emit("change", {
        type: "undo",
        command: entry.command,
        canUndo: this.canUndo(),
        canRedo: this.canRedo(),
      });

      return true;
    } finally {
      this._isExecuting = false;
    }
  }

  /**
   * 重做
   * @returns {boolean} 是否成功重做
   */
  redo() {
    if (!this.canRedo()) {
      console.warn("History: 无法重做，重做栈为空");
      return false;
    }

    if (this._isExecuting) {
      console.warn("History: 正在执行命令，忽略重做");
      return false;
    }

    this._isExecuting = true;

    try {
      const entry = this._redoStack.pop();
      entry.command.execute(this._doc);
      this._undoStack.push(entry);

      // 触发事件
      this.emit("change", {
        type: "redo",
        command: entry.command,
        canUndo: this.canUndo(),
        canRedo: this.canRedo(),
      });

      return true;
    } finally {
      this._isExecuting = false;
    }
  }

  /**
   * 撤销到指定位置
   * @param {number} targetIndex - 目标位置（0 表示撤销全部）
   * @returns {number} 实际撤销的次数
   */
  undoTo(targetIndex) {
    const currentSize = this._undoStack.length;
    const undoCount = currentSize - targetIndex;

    let count = 0;
    for (let i = 0; i < undoCount; i++) {
      if (this.undo()) {
        count++;
      } else {
        break;
      }
    }

    return count;
  }

  /**
   * 重做到指定位置
   * @param {number} count - 重做次数
   * @returns {number} 实际重做的次数
   */
  redoMultiple(count) {
    let actual = 0;
    for (let i = 0; i < count; i++) {
      if (this.redo()) {
        actual++;
      } else {
        break;
      }
    }
    return actual;
  }

  // ==================== 栈管理 ====================

  /**
   * 清空历史记录
   */
  clear() {
    this._undoStack = [];
    this._redoStack = [];
    this._lastExecuteTime = 0;

    this.emit("change", {
      type: "clear",
      canUndo: false,
      canRedo: false,
    });
  }

  /**
   * 保存点（用于判断是否有未保存的更改）
   * @private
   */
  _savePoint = 0;

  /**
   * 标记当前位置为保存点
   */
  markSaved() {
    this._savePoint = this._undoStack.length;
  }

  /**
   * 是否有未保存的更改
   * @returns {boolean}
   */
  isDirty() {
    return this._undoStack.length !== this._savePoint;
  }

  /**
   * 获取自上次保存以来的更改数量
   * @returns {number}
   */
  getChangesSinceSave() {
    return Math.abs(this._undoStack.length - this._savePoint);
  }

  // ==================== 事务支持 ====================

  /** @type {Command[]} */
  _transactionCommands = [];

  /** @type {boolean} */
  _inTransaction = false;

  /**
   * 开始事务
   */
  beginTransaction() {
    if (this._inTransaction) {
      console.warn("History: 已经在事务中");
      return;
    }
    this._inTransaction = true;
    this._transactionCommands = [];
  }

  /**
   * 在事务中执行命令（不立即推入历史）
   * @param {Command} command - 命令
   */
  executeInTransaction(command) {
    if (!this._inTransaction) {
      console.warn("History: 不在事务中，使用普通 execute");
      this.execute(command);
      return;
    }

    // 执行命令但不推入历史
    command.execute(this._doc);
    this._transactionCommands.push(command);
  }

  /**
   * 提交事务
   * @param {string} [description] - 事务描述
   */
  commitTransaction(description = "批量操作") {
    if (!this._inTransaction) {
      console.warn("History: 不在事务中");
      return;
    }

    this._inTransaction = false;

    if (this._transactionCommands.length === 0) {
      return;
    }

    // 创建批量命令并推入历史
    const batchCommand = new BatchCommand(
      this._transactionCommands,
      description,
    );

    // 直接推入历史（命令已执行）
    this._undoStack.push({
      command: batchCommand,
      timestamp: Date.now(),
    });

    // 清空重做栈
    this._redoStack = [];

    // 限制栈大小
    while (this._undoStack.length > this._maxSize) {
      this._undoStack.shift();
    }

    this._transactionCommands = [];

    this.emit("change", {
      type: "execute",
      command: batchCommand,
      canUndo: this.canUndo(),
      canRedo: this.canRedo(),
    });
  }

  /**
   * 回滚事务
   */
  rollbackTransaction() {
    if (!this._inTransaction) {
      console.warn("History: 不在事务中");
      return;
    }

    // 逆序撤销所有事务中的命令
    for (let i = this._transactionCommands.length - 1; i >= 0; i--) {
      this._transactionCommands[i].undo(this._doc);
    }

    this._inTransaction = false;
    this._transactionCommands = [];
  }

  /**
   * 是否在事务中
   * @returns {boolean}
   */
  isInTransaction() {
    return this._inTransaction;
  }
}

export default History;
