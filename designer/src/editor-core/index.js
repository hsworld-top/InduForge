/**
 * Editor Core - 编辑器内核
 * 统一导出所有模块
 */

import { DocumentModel as DocumentModelClass } from "./document/DocumentModel.ts";
import { History as HistoryClass } from "./commands/History.ts";
import { SelectionModel as SelectionModelClass } from "./selection/SelectionModel.ts";
import { Serializer as SerializerClass } from "./document/Serializer.ts";
import { PageLockManager as PageLockManagerClass } from "./lock/PageLockManager.ts";

// 类型定义（从 document/ 目录导出）
export * from "./document/types.js";
export { default as types } from "./document/types.js";

// 工具类
export { EventEmitter } from "./utils/EventEmitter.ts";

// 文档模型
export { DocumentModel } from "./document/DocumentModel.ts";
export * from "./document/indexes.js";

// 工厂函数
export {
  createComponentNode,
  cloneComponentNode,
  inferPositioning,
  createDiagramNode,
  createDiagramData,
  createShape,
} from "./document/factory.ts";

// 序列化（从 document/ 目录导出）
export { Serializer } from "./document/Serializer.ts";
export {
  migrate,
  needsMigration,
  getSchemaVersion,
} from "./document/migrations.js";

// 命令系统
export { Command, BatchCommand } from "./commands/Command.ts";
export {
  InsertNodeCommand,
  RemoveNodeCommand,
  UpdateNodeCommand,
  MoveNodeCommand,
  DuplicateNodeCommand,
  SetNodePropsCommand,
  SetNodeStyleCommand,
  ReorderNodeCommand,
  ToggleNodeVisibilityCommand,
  ToggleNodeLockCommand,
} from "./commands/nodeCommands.ts";
export {
  InsertGraphicCommand,
  RemoveGraphicCommand,
  UpdateGraphicCommand,
  MoveGraphicCommand,
  ReorderGraphicCommand,
  GroupGraphicsCommand,
  UngroupGraphicsCommand,
  SetGraphicBindingCommand,
} from "./commands/graphicCommands.ts";
export {
  AlignElementsCommand,
  DistributeElementsCommand,
  MatchSizeCommand,
} from "./commands/alignCommands.ts";
export {
  SetBindingCommand,
  SetMultipleBindingsCommand,
  ClearAllBindingsCommand,
  UpdateBindingTransformCommand,
  SetBindingFallbackCommand,
  CopyBindingsCommand,
} from "./commands/bindingCommands.ts";
export { UpdatePageCommand } from "./commands/pageCommands.ts";
export { UpdateEntryCommand } from "./commands/entryCommands.ts";
export { History } from "./commands/History.ts";

// 选中管理
export { SelectionModel } from "./selection/SelectionModel.ts";

// 校验器
export { Validator } from "./validate/validator.js";
export { BindingValidator } from "./validate/binding-validator";

// 组件注册表
export {
  ComponentRegistry,
  ComponentCategory,
  componentRegistry,
} from "./registry/component-registry";

// 页面锁
export {
  PageLockManager,
  createMockApiClient,
} from "./lock/PageLockManager.ts";

/**
 * 创建编辑器实例
 * @param {Object} [options] - 配置选项
 * @param {import('./document/types.js').ProjectSchema} [options.schema] - 初始 Schema
 * @param {Object} [options.api] - API 客户端
 * @param {Object} [options.socket] - Socket.IO 实例
 * @param {string} [options.userId] - 当前用户 ID
 * @returns {{doc: DocumentModel, history: History, selection: SelectionModel, serializer: Serializer, lockManager: PageLockManager}}
 */
export function createEditor(options = {}) {
  // 创建文档模型
  const doc = new DocumentModelClass(options.schema);

  // 创建历史记录管理器
  const history = new HistoryClass(doc);

  // 创建选中管理器
  const selection = new SelectionModelClass(doc);

  // 创建序列化器
  const serializer = new SerializerClass();

  // 创建页面锁管理器
  const lockManager = new PageLockManagerClass({
    api: options.api,
    socket: options.socket,
    currentUserId: options.userId,
  });

  // 初始化页面锁
  lockManager.init();

  return {
    doc,
    history,
    selection,
    serializer,
    lockManager,
  };
}

export default {
  createEditor,
};
