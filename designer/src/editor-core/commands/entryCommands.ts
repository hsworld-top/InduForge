/**
 * 入口配置命令
 * 用于更新工程入口信息（首页/登录页）
 */

import type { EntryConfig } from "../document/types.ts";
import type { CommandDocument } from "./Command";
import { Command } from "./Command";

/**
 * 更新入口配置命令
 */
export class UpdateEntryCommand extends Command {
  _patch!: Partial<EntryConfig>;
  _oldValues!: Partial<EntryConfig> | null;

  get type() {
    return "UpdateEntry";
  }

  constructor(patch: Partial<EntryConfig>) {
    super();
    this._patch = patch;
    this._oldValues = null;
  }

  execute(doc: CommandDocument) {
    const entry = doc.entry;
    if (!entry) return;
    const oldValues: Partial<EntryConfig> = {};
    const entryRec = entry as unknown as Record<string, unknown>;
    for (const key of Object.keys(this._patch)) {
      oldValues[key as keyof EntryConfig] = JSON.parse(
        JSON.stringify(entryRec[key] ?? null),
      ) as never;
    }
    this._oldValues = oldValues;
    doc._updateEntry(this._patch);
  }

  undo(doc: CommandDocument) {
    if (this._oldValues) {
      doc._updateEntry(this._oldValues);
    }
  }

  canMerge(other: Command) {
    return other instanceof UpdateEntryCommand;
  }

  merge(other: UpdateEntryCommand) {
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
