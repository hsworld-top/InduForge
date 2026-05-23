/**
 * Command - 命令接口
 * 所有编辑操作必须通过命令执行，支持撤销重做
 */

import type { DocumentModel } from '../document/DocumentModel'

/** 文档模型：由 DocumentModel 实现 */
export type CommandDocument = DocumentModel

/**
 * 命令基类
 */
export class Command {
  get type(): string {
    throw new Error('Command.type must be implemented')
  }

  execute(_doc: CommandDocument): void {
    throw new Error('Command.execute must be implemented')
  }

  undo(_doc: CommandDocument): void {
    throw new Error('Command.undo must be implemented')
  }

  canMerge(_other: Command): boolean {
    return false
  }

  merge(_other: Command): Command {
    throw new Error('Command.merge not supported')
  }

  getDescription(): string {
    return this.type
  }
}

/**
 * 批量命令（事务）
 */
export class BatchCommand extends Command {
  private readonly _commands: Command[]
  private readonly _description: string

  get type(): string {
    return 'Batch'
  }

  constructor(commands: Command[], description = '批量操作') {
    super()
    this._commands = commands
    this._description = description
  }

  execute(doc: DocumentModel): void {
    for (const cmd of this._commands) {
      cmd.execute(doc)
    }
  }

  undo(doc: DocumentModel): void {
    for (let i = this._commands.length - 1; i >= 0; i--) {
      const cmd = this._commands[i]
      if (cmd) cmd.undo(doc)
    }
  }

  override getDescription(): string {
    return this._description
  }

  getCommands(): Command[] {
    return [...this._commands]
  }
}

export default { Command, BatchCommand }
