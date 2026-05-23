/**
 * History - 撤销重做栈
 */

import type { Command, CommandDocument } from './Command'
import { EventEmitter } from '../utils/EventEmitter'
import { BatchCommand } from './Command'

export interface HistoryEntry {
  command: Command
  timestamp: number
  userId?: string
}

export interface HistoryOptions {
  maxSize?: number
  mergeWindow?: number
}

export class History extends EventEmitter {
  private readonly _doc: CommandDocument
  private readonly _maxSize: number
  private readonly _mergeWindow: number
  private _undoStack: HistoryEntry[] = []
  private _redoStack: HistoryEntry[] = []
  private _lastExecuteTime = 0
  private _isExecuting = false
  private _savePoint = 0
  private _transactionCommands: Command[] = []
  private _inTransaction = false

  constructor(doc: CommandDocument, options: HistoryOptions = {}) {
    super()
    this._doc = doc
    this._maxSize = options.maxSize ?? 100
    this._mergeWindow = options.mergeWindow ?? 1000
  }

  canUndo(): boolean {
    return this._undoStack.length > 0
  }

  canRedo(): boolean {
    return this._redoStack.length > 0
  }

  get undoStackSize(): number {
    return this._undoStack.length
  }

  get redoStackSize(): number {
    return this._redoStack.length
  }

  getUndoDescriptions(): string[] {
    return this._undoStack.map((entry) => entry.command.getDescription())
  }

  getRedoDescriptions(): string[] {
    return this._redoStack.map((entry) => entry.command.getDescription())
  }

  execute(command: Command, options: { skipMerge?: boolean; userId?: string } = {}): void {
    if (this._isExecuting) {
      console.warn('History: 正在执行命令，忽略重入')
      return
    }

    this._isExecuting = true

    try {
      const now = Date.now()

      command.execute(this._doc)

      const lastEntry = this._undoStack.at(-1)
      const shouldMerge =
        !options.skipMerge &&
        lastEntry !== undefined &&
        now - this._lastExecuteTime < this._mergeWindow &&
        lastEntry.command.canMerge?.(command)

      if (shouldMerge && lastEntry) {
        const mergedCommand = lastEntry.command.merge(command)
        const mergedEntry: HistoryEntry = {
          command: mergedCommand,
          timestamp: now,
        }
        if (options.userId !== undefined) mergedEntry.userId = options.userId
        this._undoStack[this._undoStack.length - 1] = mergedEntry
      } else {
        const newEntry: HistoryEntry = {
          command,
          timestamp: now,
        }
        if (options.userId !== undefined) newEntry.userId = options.userId
        this._undoStack.push(newEntry)
      }

      this._redoStack = []

      while (this._undoStack.length > this._maxSize) {
        this._undoStack.shift()
      }

      this._lastExecuteTime = now

      this.emit('change', {
        type: 'execute',
        command,
        canUndo: this.canUndo(),
        canRedo: this.canRedo(),
      })
    } finally {
      this._isExecuting = false
    }
  }

  undo(): boolean {
    if (!this.canUndo()) {
      console.warn('History: 无法撤销，撤销栈为空')
      return false
    }

    if (this._isExecuting) {
      console.warn('History: 正在执行命令，忽略撤销')
      return false
    }

    this._isExecuting = true

    try {
      const entry = this._undoStack.pop()
      if (!entry) return false
      entry.command.undo(this._doc)
      this._redoStack.push(entry)

      this.emit('change', {
        type: 'undo',
        command: entry.command,
        canUndo: this.canUndo(),
        canRedo: this.canRedo(),
      })

      return true
    } finally {
      this._isExecuting = false
    }
  }

  redo(): boolean {
    if (!this.canRedo()) {
      console.warn('History: 无法重做，重做栈为空')
      return false
    }

    if (this._isExecuting) {
      console.warn('History: 正在执行命令，忽略重做')
      return false
    }

    this._isExecuting = true

    try {
      const entry = this._redoStack.pop()
      if (!entry) return false
      entry.command.execute(this._doc)
      this._undoStack.push(entry)

      this.emit('change', {
        type: 'redo',
        command: entry.command,
        canUndo: this.canUndo(),
        canRedo: this.canRedo(),
      })

      return true
    } finally {
      this._isExecuting = false
    }
  }

  undoTo(targetIndex: number): number {
    const currentSize = this._undoStack.length
    const undoCount = currentSize - targetIndex

    let count = 0
    for (let i = 0; i < undoCount; i++) {
      if (this.undo()) {
        count++
      } else {
        break
      }
    }

    return count
  }

  redoMultiple(count: number): number {
    let actual = 0
    for (let i = 0; i < count; i++) {
      if (this.redo()) {
        actual++
      } else {
        break
      }
    }
    return actual
  }

  clear(): void {
    this._undoStack = []
    this._redoStack = []
    this._lastExecuteTime = 0

    this.emit('change', {
      type: 'clear',
      canUndo: false,
      canRedo: false,
    })
  }

  markSaved(): void {
    this._savePoint = this._undoStack.length
  }

  isDirty(): boolean {
    return this._undoStack.length !== this._savePoint
  }

  getChangesSinceSave(): number {
    return Math.abs(this._undoStack.length - this._savePoint)
  }

  beginTransaction(): void {
    if (this._inTransaction) {
      console.warn('History: 已经在事务中')
      return
    }
    this._inTransaction = true
    this._transactionCommands = []
  }

  executeInTransaction(command: Command): void {
    if (!this._inTransaction) {
      console.warn('History: 不在事务中，使用普通 execute')
      this.execute(command)
      return
    }

    command.execute(this._doc)
    this._transactionCommands.push(command)
  }

  commitTransaction(description = '批量操作'): void {
    if (!this._inTransaction) {
      console.warn('History: 不在事务中')
      return
    }

    this._inTransaction = false

    if (this._transactionCommands.length === 0) {
      return
    }

    const batchCommand = new BatchCommand(this._transactionCommands, description)

    this._undoStack.push({
      command: batchCommand,
      timestamp: Date.now(),
    })

    this._redoStack = []

    while (this._undoStack.length > this._maxSize) {
      this._undoStack.shift()
    }

    this._transactionCommands = []

    this.emit('change', {
      type: 'execute',
      command: batchCommand,
      canUndo: this.canUndo(),
      canRedo: this.canRedo(),
    })
  }

  rollbackTransaction(): void {
    if (!this._inTransaction) {
      console.warn('History: 不在事务中')
      return
    }

    for (let i = this._transactionCommands.length - 1; i >= 0; i--) {
      const cmd = this._transactionCommands[i]
      if (cmd) cmd.undo(this._doc)
    }

    this._inTransaction = false
    this._transactionCommands = []
  }

  isInTransaction(): boolean {
    return this._inTransaction
  }
}

export default History
