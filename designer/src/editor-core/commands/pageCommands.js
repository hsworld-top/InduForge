/**
 * 页面操作命令
 * 包含页面更新命令实现
 */

import { Command } from "./Command.js";

/**
 * @typedef {import('../document/DocumentModel.js').DocumentModel} DocumentModel
 * @typedef {import('../document/types.js').PageNode} PageNode
 */

/**
 * 更新页面命令
 */
export class UpdatePageCommand extends Command {
  get type() {
    return "UpdatePage";
  }

  /**
   * 创建更新页面命令
   * @param {string} pageId - 页面 ID
   * @param {Partial<PageNode>} patch - 更新内容
   */
  constructor(pageId, patch) {
    super();
    /** @type {string} */
    this._pageId = pageId;
    /** @type {Partial<PageNode>} */
    this._patch = patch;
    /** @type {Partial<PageNode> | null} */
    this._oldValues = null;
  }

  /**
   * @param {DocumentModel} doc
   */
  execute(doc) {
    const page = doc.getPage(this._pageId);
    if (!page) return;

    this._oldValues = {};
    for (const key of Object.keys(this._patch)) {
      this._oldValues[key] = JSON.parse(
        JSON.stringify(page[key] ?? null)
      );
    }

    doc._updatePage(this._pageId, this._patch);
  }

  /**
   * @param {DocumentModel} doc
   */
  undo(doc) {
    if (this._oldValues) {
      doc._updatePage(this._pageId, this._oldValues);
    }
  }

  /**
   * @param {Command} other
   * @returns {boolean}
   */
  canMerge(other) {
    return other instanceof UpdatePageCommand && other._pageId === this._pageId;
  }

  /**
   * @param {UpdatePageCommand} other
   * @returns {UpdatePageCommand}
   */
  merge(other) {
    const mergedPatch = { ...this._patch, ...other._patch };
    const cmd = new UpdatePageCommand(this._pageId, mergedPatch);
    cmd._oldValues = this._oldValues;
    return cmd;
  }

  getDescription() {
    const keys = Object.keys(this._patch);
    return `更新页面属性: ${keys.join(", ")}`;
  }
}
