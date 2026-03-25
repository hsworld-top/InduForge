/**
 * Editor Core - 编辑器内核
 * 统一导出所有模块
 */

import type { ProjectSchema } from "./document/types.ts";
import type { PageLockApi, PageLockSocket } from "./lock/PageLockManager.ts";
import { History as HistoryClass } from "./commands/History.ts";
import { DocumentModel as DocumentModelClass } from "./document/DocumentModel.ts";
import { Serializer as SerializerClass } from "./document/Serializer.ts";
import {
  createMockApiClient,
  PageLockManager as PageLockManagerClass,
} from "./lock/PageLockManager.ts";
import { SelectionModel as SelectionModelClass } from "./selection/SelectionModel.ts";

export {
  AlignElementsCommand,
  DistributeElementsCommand,
  MatchSizeCommand,
} from "./commands/alignCommands.ts";
export {
  ClearAllBindingsCommand,
  CopyBindingsCommand,
  SetBindingCommand,
  SetBindingFallbackCommand,
  SetMultipleBindingsCommand,
  UpdateBindingTransformCommand,
} from "./commands/bindingCommands.ts";

// 命令系统
export { BatchCommand, Command } from "./commands/Command.ts";

export { UpdateEntryCommand } from "./commands/entryCommands.ts";
export {
  GroupGraphicsCommand,
  InsertGraphicCommand,
  MoveGraphicCommand,
  RemoveGraphicCommand,
  ReorderGraphicCommand,
  SetGraphicBindingCommand,
  UngroupGraphicsCommand,
  UpdateGraphicCommand,
} from "./commands/graphicCommands.ts";

export { History } from "./commands/History.ts";

export {
  DuplicateNodeCommand,
  InsertNodeCommand,
  MoveNodeCommand,
  RemoveNodeCommand,
  ReorderNodeCommand,
  SetNodePropsCommand,
  SetNodeStyleCommand,
  ToggleNodeLockCommand,
  ToggleNodeVisibilityCommand,
  UpdateNodeCommand,
} from "./commands/nodeCommands.ts";
export { UpdatePageCommand } from "./commands/pageCommands.ts";

// 文档模型
export { DocumentModel } from "./document/DocumentModel.ts";
// 工厂函数
export {
  cloneComponentNode,
  createComponentNode,
  createDiagramData,
  createDiagramNode,
  createShape,
  inferPositioning,
} from "./document/factory.ts";
export * from "./document/indexes.ts";
export { getSchemaVersion, migrate, needsMigration } from "./document/migrations.ts";
// 序列化（从 document/ 目录导出）
export { Serializer } from "./document/Serializer.ts";
// 类型定义（从 document/ 目录导出）
export * from "./document/types.ts";
export { default as types } from "./document/types.ts";
// 页面锁
export { createMockApiClient, PageLockManager } from "./lock/PageLockManager.ts";

// 组件注册表
export {
  ComponentCategory,
  ComponentRegistry,
  componentRegistry,
} from "./registry/component-registry.ts";

// 选中管理
export { SelectionModel } from "./selection/SelectionModel.ts";
// 工具类
export { EventEmitter } from "./utils/EventEmitter.ts";

export { BindingValidator, Validator } from "./validate/index.ts";

export interface CreateEditorOptions {
  schema?: ProjectSchema;
  api?: PageLockApi;
  socket?: PageLockSocket | null;
  userId?: string;
}

/**
 * 创建编辑器实例
 */
export function createEditor(options: CreateEditorOptions = {}) {
  const doc = new DocumentModelClass(options.schema);

  const history = new HistoryClass(doc);

  const selection = new SelectionModelClass(doc);

  const serializer = new SerializerClass();

  const lockManager = new PageLockManagerClass({
    api: options.api ?? createMockApiClient(),
    socket: options.socket ?? null,
    currentUserId: options.userId ?? "",
  });

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
