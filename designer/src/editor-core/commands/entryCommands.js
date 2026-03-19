/**
 * 入口配置命令
 * 用于更新工程入口信息（首页/登录页）
 */

import { Command } from "./Command.js";

/**
 * @typedef {import('../document/DocumentModel.js').DocumentModel} DocumentModel
 * @typedef {import('../document/types.js').EntryConfig} EntryConfig
 */

/**
 * 更新入口配置命令
 */
export class UpdateEntryCommand extends Command {
  get type() {
    return "UpdateEntry";
  }

  /**
   * @param {Partial<EntryConfig>} patch - 更新内容
   */
  constructor(patch) {
    super();
    /** @type {Partial<EntryConfig>} */
    this._patch = patch;
    /** @type {Partial<EntryConfig> | null} */
    this._oldValues = null;
  }

  /**
   * @param {DocumentModel} doc
   */
  execute(doc) {
    const entry = doc.entry;
    if (!entry) return;
    this._oldValues = {};
    for (const key of Object.keys(this._patch)) {
      this._oldValues[key] = JSON.parse(JSON.stringify(entry[key] ?? null));
    }
    doc._updateEntry(this._patch);
  }

  /**
   * @param {DocumentModel} doc
   */
  undo(doc) {
    if (this._oldValues) {
      doc._updateEntry(this._oldValues);
    }
  }

  /**
   * @param {Command} other
   * @returns {boolean}
   */
  canMerge(other) {
    return other instanceof UpdateEntryCommand;
  }

  /**
   * @param {UpdateEntryCommand} other
   * @returns {UpdateEntryCommand}
   */
  merge(other) {
    const mergedPatch = { ...this._patch, ...other._patch };
    const cmd = new UpdateEntryCommand(mergedPatch);
    cmd._oldValues = this._oldValues;
    return cmd;
  }

  getDescription() {
    const keys = Object.keys(this._patch);
    return `更新入口配置: ${keys.join(", ")}`;
  }
}
