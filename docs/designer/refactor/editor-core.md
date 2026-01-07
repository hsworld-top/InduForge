# 编辑器内核（Editor Core）

编辑器内核是 Designer 的核心基础设施，提供文档模型、命令系统、历史记录、选中管理等能力。

**核心原则**：UI 层永远不要直接修改 JSON，所有修改必须走命令系统。

## 1. 架构概览

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            Editor Core                                       │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐             │
│  │  DocumentModel  │  │ Command/History │  │ SelectionModel  │             │
│  │  (文档模型)      │  │ (命令/历史)      │  │ (选中管理)       │             │
│  └────────┬────────┘  └────────┬────────┘  └────────┬────────┘             │
│           │                    │                    │                       │
│           └────────────────────┼────────────────────┘                       │
│                                │                                            │
│  ┌─────────────────┐  ┌───────┴───────┐  ┌─────────────────┐               │
│  │   Validator     │  │  Serializer   │  │    Registry     │               │
│  │   (校验器)       │  │  (序列化)      │  │  (组件注册)      │               │
│  └─────────────────┘  └───────────────┘  └─────────────────┘               │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## 2. DocumentModel（文档模型）

### 2.1 核心职责

- 存储规范化的工程 Schema
- 提供节点 CRUD 操作
- 维护索引便于快速查询
- 触发变更事件

### 2.2 数据结构

```typescript
interface DocumentModel {
  // 工程元信息
  project: ProjectMeta;

  // 规范化存储
  pagesById: Record<string, PageNode>;
  nodesById: Record<string, ComponentNode>;
  assetsById: Record<string, AssetRef>;

  // 配置
  securityDecl: SecurityDecl;
  dataProviders: Record<string, DataProvider>;
  entry: EntryConfig;
  vars: VarsConfig;
}

interface ComponentNode {
  id: string;
  type: string;
  props: Record<string, any>;
  style: Record<string, any>;
  layoutItem: LayoutItem | null;
  bindings: Record<string, Binding>;
  permissions: PermissionConfig;
  events: Record<string, Action[]>;
  children: string[]; // 子节点 ID 数组
}
```

### 2.3 API 设计

```typescript
class DocumentModel {
  // 节点操作
  getNode(id: string): ComponentNode | null;
  getPage(id: string): PageNode | null;
  getParent(nodeId: string): ComponentNode | null;
  getChildren(nodeId: string): ComponentNode[];
  getAncestors(nodeId: string): ComponentNode[];

  // 查询
  findNodesByType(type: string): ComponentNode[];
  findNodesByBinding(datapointPath: string): ComponentNode[];

  // 变更（内部使用，外部通过 Command）
  _insertNode(parentId: string, index: number, node: ComponentNode): void;
  _removeNode(nodeId: string): ComponentNode;
  _updateNode(nodeId: string, patch: Partial<ComponentNode>): void;
  _moveNode(nodeId: string, newParentId: string, newIndex: number): void;

  // 事件
  on(event: "change", handler: (changes: Change[]) => void): void;
}
```

### 2.4 索引维护

```typescript
class DocumentModel {
  // 内部索引
  private parentIndex: Map<string, string>; // nodeId → parentId
  private typeIndex: Map<string, Set<string>>; // type → nodeIds
  private bindingIndex: Map<string, Set<string>>; // datapointPath → nodeIds

  // 索引更新
  private updateIndexes(changes: Change[]): void {
    for (const change of changes) {
      if (change.type === "insert") {
        this.parentIndex.set(change.nodeId, change.parentId);
        this.addToTypeIndex(change.node);
        this.addToBindingIndex(change.node);
      }
      // ...
    }
  }
}
```

## 3. Command/History（命令系统）

### 3.1 命令接口

```typescript
interface Command {
  // 命令标识
  readonly type: string;

  // 执行与撤销
  execute(doc: DocumentModel): void;
  undo(doc: DocumentModel): void;

  // 合并（用于连续输入等场景）
  canMerge?(other: Command): boolean;
  merge?(other: Command): Command;
}
```

### 3.2 基础命令实现

```typescript
// 插入节点
class InsertNodeCommand implements Command {
  readonly type = "InsertNode";

  constructor(
    private parentId: string,
    private index: number,
    private node: ComponentNode
  ) {}

  execute(doc: DocumentModel): void {
    doc._insertNode(this.parentId, this.index, this.node);
  }

  undo(doc: DocumentModel): void {
    doc._removeNode(this.node.id);
  }
}

// 删除节点
class RemoveNodeCommand implements Command {
  readonly type = "RemoveNode";
  private removedNode: ComponentNode | null = null;
  private parentId: string | null = null;
  private index: number = -1;

  constructor(private nodeId: string) {}

  execute(doc: DocumentModel): void {
    const parent = doc.getParent(this.nodeId);
    this.parentId = parent?.id ?? null;
    this.index = parent?.children.indexOf(this.nodeId) ?? -1;
    this.removedNode = doc._removeNode(this.nodeId);
  }

  undo(doc: DocumentModel): void {
    if (this.removedNode && this.parentId !== null) {
      doc._insertNode(this.parentId, this.index, this.removedNode);
    }
  }
}

// 更新节点属性
class UpdateNodeCommand implements Command {
  readonly type = "UpdateNode";
  private oldValues: Partial<ComponentNode> | null = null;

  constructor(private nodeId: string, private patch: Partial<ComponentNode>) {}

  execute(doc: DocumentModel): void {
    const node = doc.getNode(this.nodeId);
    if (node) {
      this.oldValues = {};
      for (const key of Object.keys(this.patch)) {
        this.oldValues[key] = node[key];
      }
      doc._updateNode(this.nodeId, this.patch);
    }
  }

  undo(doc: DocumentModel): void {
    if (this.oldValues) {
      doc._updateNode(this.nodeId, this.oldValues);
    }
  }
}

// 移动节点
class MoveNodeCommand implements Command {
  readonly type = "MoveNode";
  private oldParentId: string | null = null;
  private oldIndex: number = -1;

  constructor(
    private nodeId: string,
    private newParentId: string,
    private newIndex: number
  ) {}

  execute(doc: DocumentModel): void {
    const oldParent = doc.getParent(this.nodeId);
    this.oldParentId = oldParent?.id ?? null;
    this.oldIndex = oldParent?.children.indexOf(this.nodeId) ?? -1;
    doc._moveNode(this.nodeId, this.newParentId, this.newIndex);
  }

  undo(doc: DocumentModel): void {
    if (this.oldParentId !== null) {
      doc._moveNode(this.nodeId, this.oldParentId, this.oldIndex);
    }
  }
}

// 设置绑定
class SetBindingCommand implements Command {
  readonly type = "SetBinding";
  private oldBinding: Binding | null = null;

  constructor(
    private nodeId: string,
    private propKey: string,
    private binding: Binding | null
  ) {}

  execute(doc: DocumentModel): void {
    const node = doc.getNode(this.nodeId);
    if (node) {
      this.oldBinding = node.bindings[this.propKey] ?? null;
      if (this.binding) {
        node.bindings[this.propKey] = this.binding;
      } else {
        delete node.bindings[this.propKey];
      }
      doc._updateNode(this.nodeId, { bindings: { ...node.bindings } });
    }
  }

  undo(doc: DocumentModel): void {
    const node = doc.getNode(this.nodeId);
    if (node) {
      if (this.oldBinding) {
        node.bindings[this.propKey] = this.oldBinding;
      } else {
        delete node.bindings[this.propKey];
      }
      doc._updateNode(this.nodeId, { bindings: { ...node.bindings } });
    }
  }
}
```

### 3.3 History（历史管理）

```typescript
class History {
  private undoStack: Command[] = [];
  private redoStack: Command[] = [];
  private maxSize: number = 100;

  constructor(private doc: DocumentModel) {}

  // 执行命令
  execute(command: Command): void {
    command.execute(this.doc);

    // 尝试合并
    const last = this.undoStack[this.undoStack.length - 1];
    if (last?.canMerge?.(command)) {
      this.undoStack[this.undoStack.length - 1] = last.merge!(command);
    } else {
      this.undoStack.push(command);
    }

    // 清空重做栈
    this.redoStack = [];

    // 限制栈大小
    if (this.undoStack.length > this.maxSize) {
      this.undoStack.shift();
    }
  }

  // 撤销
  undo(): boolean {
    const command = this.undoStack.pop();
    if (command) {
      command.undo(this.doc);
      this.redoStack.push(command);
      return true;
    }
    return false;
  }

  // 重做
  redo(): boolean {
    const command = this.redoStack.pop();
    if (command) {
      command.execute(this.doc);
      this.undoStack.push(command);
      return true;
    }
    return false;
  }

  // 状态
  canUndo(): boolean {
    return this.undoStack.length > 0;
  }
  canRedo(): boolean {
    return this.redoStack.length > 0;
  }
}
```

### 3.4 批量命令（事务）

```typescript
class BatchCommand implements Command {
  readonly type = "Batch";

  constructor(private commands: Command[], private description?: string) {}

  execute(doc: DocumentModel): void {
    for (const cmd of this.commands) {
      cmd.execute(doc);
    }
  }

  undo(doc: DocumentModel): void {
    // 逆序撤销
    for (let i = this.commands.length - 1; i >= 0; i--) {
      this.commands[i].undo(doc);
    }
  }
}

// 使用示例
function deleteMultipleNodes(history: History, nodeIds: string[]): void {
  const commands = nodeIds.map((id) => new RemoveNodeCommand(id));
  history.execute(new BatchCommand(commands, "批量删除"));
}
```

## 4. SelectionModel（选中管理）

### 4.1 数据结构

```typescript
interface SelectionState {
  // 选中的节点 ID 列表（支持多选）
  selectedIds: string[];

  // 当前 hover 的节点
  hoveredId: string | null;

  // 当前拖拽目标
  dropTargetId: string | null;

  // 选中锚点（用于 Shift 连续选择）
  anchorId: string | null;
}
```

### 4.2 API 设计

```typescript
class SelectionModel {
  private state: SelectionState;

  // 选择操作
  select(nodeId: string): void;
  toggleSelect(nodeId: string): void; // Ctrl+Click
  selectRange(nodeId: string): void; // Shift+Click
  selectAll(pageId: string): void; // Ctrl+A
  clearSelection(): void;

  // 查询
  isSelected(nodeId: string): boolean;
  getSelectedNodes(): ComponentNode[];
  getPrimarySelection(): ComponentNode | null;

  // Hover
  setHover(nodeId: string | null): void;

  // 拖拽目标
  setDropTarget(nodeId: string | null): void;

  // 事件
  on(event: "change", handler: (state: SelectionState) => void): void;
}
```

### 4.3 多选行为规范

| 操作       | 行为                     |
| ---------- | ------------------------ |
| 单击       | 清除其他选中，选中当前   |
| Ctrl+单击  | 切换当前节点选中状态     |
| Shift+单击 | 从锚点到当前节点范围选中 |
| Ctrl+A     | 选中当前页面所有可选节点 |
| Esc        | 清除选中                 |
| Delete     | 删除所有选中节点         |

## 5. Validator（校验器）

### 5.1 校验规则

```typescript
interface ValidationResult {
  valid: boolean;
  errors: ValidationError[];
  warnings: ValidationWarning[];
}

interface ValidationError {
  code: string;
  message: string;
  path: string; // 错误位置，如 'nodesById.node_001.bindings.text'
  nodeId?: string;
}

class Validator {
  validate(doc: DocumentModel): ValidationResult {
    const errors: ValidationError[] = [];
    const warnings: ValidationWarning[] = [];

    // 1. Schema 版本检查
    this.validateSchemaVersion(doc, errors);

    // 2. 入口页面检查
    this.validateEntry(doc, errors);

    // 3. 节点完整性检查
    this.validateNodes(doc, errors);

    // 4. 绑定检查
    this.validateBindings(doc, errors, warnings);

    // 5. 资源引用检查
    this.validateAssets(doc, errors);

    // 6. 权限配置检查
    this.validatePermissions(doc, errors);

    return {
      valid: errors.length === 0,
      errors,
      warnings,
    };
  }
}
```

### 5.2 绑定校验（含数据点状态）

```typescript
class BindingValidator {
  constructor(private diagnosticsStore: DiagnosticsStore) {}

  async validateBindings(doc: DocumentModel): Promise<BindingValidationResult> {
    const issues: BindingIssue[] = [];

    // 收集所有绑定
    const bindings = this.collectAllBindings(doc);

    // 批量查询数据点状态
    const paths = bindings
      .filter((b) => b.binding.kind === "datapoint")
      .map((b) => b.binding.path);

    const statuses = await this.diagnosticsStore.fetchStatuses(paths);

    // 检查每个绑定
    for (const { nodeId, propKey, binding } of bindings) {
      if (binding.kind === "datapoint") {
        const status = statuses.get(binding.path);
        if (status === "invalid") {
          issues.push({
            type: "error",
            nodeId,
            propKey,
            message: `数据点 ${binding.path} 已失效`,
            datapointPath: binding.path,
          });
        } else if (status === "unknown") {
          issues.push({
            type: "warning",
            nodeId,
            propKey,
            message: `数据点 ${binding.path} 不存在`,
            datapointPath: binding.path,
          });
        }
      }
    }

    return { issues };
  }
}
```

## 6. Serializer（序列化）

### 6.1 导入导出

```typescript
class Serializer {
  // 导出为 JSON
  export(doc: DocumentModel): string {
    return JSON.stringify(
      {
        schemaVersion: 2,
        project: doc.project,
        securityDecl: doc.securityDecl,
        entry: doc.entry,
        dataProviders: doc.dataProviders,
        vars: doc.vars,
        assetsById: doc.assetsById,
        pagesById: doc.pagesById,
        nodesById: doc.nodesById,
      },
      null,
      2
    );
  }

  // 从 JSON 导入
  import(json: string): DocumentModel {
    const data = JSON.parse(json);

    // 版本迁移
    const migrated = this.migrate(data);

    // 构建文档模型
    return new DocumentModel(migrated);
  }

  // 版本迁移
  private migrate(data: any): any {
    let version = data.schemaVersion ?? 1;

    while (version < CURRENT_SCHEMA_VERSION) {
      data = migrations[version](data);
      version++;
    }

    return data;
  }
}
```

### 6.2 Patch（差量保存）

```typescript
interface Patch {
  ops: PatchOp[];
  timestamp: number;
  userId?: string;
}

type PatchOp =
  | { op: "add"; path: string; value: any }
  | { op: "remove"; path: string }
  | { op: "replace"; path: string; value: any };

class PatchGenerator {
  // 生成差量
  generate(oldDoc: any, newDoc: any): Patch {
    const ops = this.diff(oldDoc, newDoc, "");
    return {
      ops,
      timestamp: Date.now(),
    };
  }

  // 应用差量
  apply(doc: any, patch: Patch): any {
    let result = JSON.parse(JSON.stringify(doc));
    for (const op of patch.ops) {
      result = this.applyOp(result, op);
    }
    return result;
  }
}
```

## 7. 实现步骤

### Step 1：定义类型（types.ts）

```typescript
// 1. 定义 Schema v2 所有类型
// 2. 定义 Command 接口
// 3. 定义 ValidationResult 等
```

### Step 2：实现 DocumentModel

```typescript
// 1. 实现基本 CRUD
// 2. 实现索引维护
// 3. 实现事件触发
// 4. 单元测试
```

### Step 3：实现 Command/History

```typescript
// 1. 实现基础命令（Insert/Remove/Update/Move）
// 2. 实现 History 栈管理
// 3. 实现 BatchCommand
// 4. 单元测试
```

### Step 4：实现 SelectionModel

```typescript
// 1. 实现选中状态管理
// 2. 实现多选行为
// 3. 集成到 UI
```

### Step 5：实现 Validator

```typescript
// 1. 实现基础校验规则
// 2. 集成 BindingValidator
// 3. 集成到保存/发布流程
```

### Step 6：实现 Serializer

```typescript
// 1. 实现导入导出
// 2. 实现版本迁移
// 3. 实现 Patch 生成
```

## 8. 测试要点

- [ ] DocumentModel CRUD 操作正确性
- [ ] Command execute/undo 对称性
- [ ] History 撤销/重做栈管理
- [ ] BatchCommand 批量操作与撤销
- [ ] SelectionModel 多选行为
- [ ] Validator 各类错误检测
- [ ] Serializer 导入导出一致性
- [ ] 版本迁移正确性

---

**相关文档**：

- [Schema 设计](./schema-design.md)
- [数据绑定 v2](./data-binding-v2.md)
