/**
 * 页面操作命令
 * 包含页面更新命令实现
 */

import type { PageNode } from "../document/types.ts";
import type { CommandDocument } from "./Command";
import { Command } from "./Command";

/**
 * 更新页面命令
 */
export class UpdatePageCommand extends Command {
  _pageId!: string;
  _patch!: Partial<PageNode>;
  _oldValues!: Partial<PageNode> | null;

  get type() {
    return "UpdatePage";
  }

  constructor(pageId: string, patch: Partial<PageNode>) {
    super();
    this._pageId = pageId;
    this._patch = patch;
    this._oldValues = null;
  }

  execute(doc: CommandDocument) {
    const page = doc.getPage(this._pageId);
    if (!page) return;

    const oldValues: Partial<PageNode> = {};
    const pageRec = page as unknown as Record<string, unknown>;
    for (const key of Object.keys(this._patch)) {
      oldValues[key as keyof PageNode] = JSON.parse(JSON.stringify(pageRec[key] ?? null)) as never;
    }
    this._oldValues = oldValues;

    doc._updatePage(this._pageId, this._patch);
  }

  undo(doc: CommandDocument) {
    if (this._oldValues) {
      doc._updatePage(this._pageId, this._oldValues);
    }
  }

  canMerge(other: Command) {
    return other instanceof UpdatePageCommand && other._pageId === this._pageId;
  }

  merge(other: UpdatePageCommand) {
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
