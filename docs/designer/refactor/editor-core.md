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
  graphicsById: Record<string, GraphicNode>; // Canvas 图形
  symbolsById: Record<string, SymbolDef>; // 符号库定义
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

// Canvas 图形节点
interface GraphicNode {
  id: string;
  type: GraphicType; // 'Canvas.Line' | 'Canvas.Rect' | 'Canvas.Circle' | ...
  props: GraphicProps;
  bindings: Record<string, Binding>;
  events: Record<string, Action[]>;
  animations: Animation[];
  z: number; // 图层顺序
  locked?: boolean; // 锁定状态
  visible?: boolean; // 可见性
}

type GraphicType =
  | "Canvas.Line"
  | "Canvas.Rect"
  | "Canvas.Circle"
  | "Canvas.Ellipse"
  | "Canvas.Polygon"
  | "Canvas.Path"
  | "Canvas.Pipe"
  | "Canvas.Text"
  | "Canvas.Symbol"
  | "Canvas.Group";

// 符号定义
interface SymbolDef {
  id: string;
  name: string;
  category: string;
  graphics: GraphicPrimitive[]; // 组成符号的基础图形
  anchors: Anchor[]; // 连接锚点
  defaultSize: { width: number; height: number };
  isBuiltin?: boolean; // 是否内置符号
}

// 锚点定义（用于管道连接）
interface Anchor {
  name: string;
  x: number; // 相对于符号中心的偏移
  y: number;
  direction?: "up" | "down" | "left" | "right"; // 连接方向
}
```

### 2.3 API 设计

```typescript
class DocumentModel {
  // ===== 组件节点操作 =====
  getNode(id: string): ComponentNode | null;
  getPage(id: string): PageNode | null;
  getParent(nodeId: string): ComponentNode | null;
  getChildren(nodeId: string): ComponentNode[];
  getAncestors(nodeId: string): ComponentNode[];

  // ===== Canvas 图形操作 =====
  getGraphic(id: string): GraphicNode | null;
  getGraphicsByPage(pageId: string): GraphicNode[];
  getGraphicsByType(type: GraphicType): GraphicNode[];
  getSymbol(id: string): SymbolDef | null;
  getAllSymbols(): SymbolDef[];

  // ===== 查询 =====
  findNodesByType(type: string): ComponentNode[];
  findNodesByBinding(datapointPath: string): ComponentNode[];
  findGraphicsByBinding(datapointPath: string): GraphicNode[];

  // ===== 混合查询（节点 + 图形）=====
  getElement(id: string): ComponentNode | GraphicNode | null;
  findElementsByBinding(datapointPath: string): (ComponentNode | GraphicNode)[];

  // ===== 变更（内部使用，外部通过 Command）=====
  _insertNode(parentId: string, index: number, node: ComponentNode): void;
  _removeNode(nodeId: string): ComponentNode;
  _insertGraphic(pageId: string, graphic: GraphicNode): void;
  _removeGraphic(graphicId: string): GraphicNode;
  _updateGraphic(graphicId: string, patch: Partial<GraphicNode>): void;
  _insertSymbol(symbol: SymbolDef): void;
  _removeSymbol(symbolId: string): SymbolDef;
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

### 3.2.5 Canvas 图形命令

```typescript
// 插入图形
class InsertGraphicCommand implements Command {
  readonly type = "InsertGraphic";

  constructor(private pageId: string, private graphic: GraphicNode) {}

  execute(doc: DocumentModel): void {
    doc._insertGraphic(this.pageId, this.graphic);
  }

  undo(doc: DocumentModel): void {
    doc._removeGraphic(this.graphic.id);
  }
}

// 删除图形
class RemoveGraphicCommand implements Command {
  readonly type = "RemoveGraphic";
  private removedGraphic: GraphicNode | null = null;
  private pageId: string | null = null;

  constructor(private graphicId: string) {}

  execute(doc: DocumentModel): void {
    this.pageId = doc.getGraphicPageId(this.graphicId);
    this.removedGraphic = doc._removeGraphic(this.graphicId);
  }

  undo(doc: DocumentModel): void {
    if (this.removedGraphic && this.pageId) {
      doc._insertGraphic(this.pageId, this.removedGraphic);
    }
  }
}

// 更新图形属性
class UpdateGraphicCommand implements Command {
  readonly type = "UpdateGraphic";
  private oldValues: Partial<GraphicNode> | null = null;

  constructor(private graphicId: string, private patch: Partial<GraphicNode>) {}

  execute(doc: DocumentModel): void {
    const graphic = doc.getGraphic(this.graphicId);
    if (graphic) {
      this.oldValues = {};
      for (const key of Object.keys(this.patch)) {
        this.oldValues[key] = graphic[key];
      }
      doc._updateGraphic(this.graphicId, this.patch);
    }
  }

  undo(doc: DocumentModel): void {
    if (this.oldValues) {
      doc._updateGraphic(this.graphicId, this.oldValues);
    }
  }

  // 支持连续修改同一图形时合并命令
  canMerge(other: Command): boolean {
    return (
      other instanceof UpdateGraphicCommand &&
      other.graphicId === this.graphicId
    );
  }

  merge(other: UpdateGraphicCommand): Command {
    return new UpdateGraphicCommand(this.graphicId, {
      ...this.patch,
      ...other.patch,
    });
  }
}

// 移动图形（修改位置）
class MoveGraphicCommand implements Command {
  readonly type = "MoveGraphic";
  private oldProps: Partial<GraphicProps> | null = null;

  constructor(
    private graphicId: string,
    private deltaX: number,
    private deltaY: number
  ) {}

  execute(doc: DocumentModel): void {
    const graphic = doc.getGraphic(this.graphicId);
    if (graphic) {
      this.oldProps = { ...graphic.props };
      // 根据图形类型更新位置
      const newProps = this.applyDelta(graphic.props, this.deltaX, this.deltaY);
      doc._updateGraphic(this.graphicId, { props: newProps });
    }
  }

  undo(doc: DocumentModel): void {
    if (this.oldProps) {
      doc._updateGraphic(this.graphicId, { props: this.oldProps });
    }
  }

  private applyDelta(
    props: GraphicProps,
    dx: number,
    dy: number
  ): GraphicProps {
    const newProps = { ...props };
    // 处理不同图形类型的位置属性
    if ("x" in newProps) newProps.x += dx;
    if ("y" in newProps) newProps.y += dy;
    if ("cx" in newProps) newProps.cx += dx;
    if ("cy" in newProps) newProps.cy += dy;
    if ("points" in newProps) {
      newProps.points = newProps.points.map(([x, y]) => [x + dx, y + dy]);
    }
    return newProps;
  }
}

// 调整图形层级
class ReorderGraphicCommand implements Command {
  readonly type = "ReorderGraphic";
  private oldZ: number = 0;

  constructor(private graphicId: string, private newZ: number) {}

  execute(doc: DocumentModel): void {
    const graphic = doc.getGraphic(this.graphicId);
    if (graphic) {
      this.oldZ = graphic.z;
      doc._updateGraphic(this.graphicId, { z: this.newZ });
    }
  }

  undo(doc: DocumentModel): void {
    doc._updateGraphic(this.graphicId, { z: this.oldZ });
  }
}

// 图形编组
class GroupGraphicsCommand implements Command {
  readonly type = "GroupGraphics";
  private groupId: string;
  private oldGraphicIds: string[] = [];

  constructor(private pageId: string, private graphicIds: string[]) {
    this.groupId = generateId("gfx_group_");
  }

  execute(doc: DocumentModel): void {
    // 创建编组图形
    const group: GraphicNode = {
      id: this.groupId,
      type: "Canvas.Group",
      props: { children: this.graphicIds },
      bindings: {},
      events: {},
      animations: [],
      z: Math.max(...this.graphicIds.map((id) => doc.getGraphic(id)?.z ?? 0)),
    };
    doc._insertGraphic(this.pageId, group);

    // 从页面图形列表中移除被编组的图形（它们现在属于 group）
    this.oldGraphicIds = [...this.graphicIds];
  }

  undo(doc: DocumentModel): void {
    doc._removeGraphic(this.groupId);
  }
}

// 设置图形绑定
class SetGraphicBindingCommand implements Command {
  readonly type = "SetGraphicBinding";
  private oldBinding: Binding | null = null;

  constructor(
    private graphicId: string,
    private propKey: string,
    private binding: Binding | null
  ) {}

  execute(doc: DocumentModel): void {
    const graphic = doc.getGraphic(this.graphicId);
    if (graphic) {
      this.oldBinding = graphic.bindings[this.propKey] ?? null;
      const newBindings = { ...graphic.bindings };
      if (this.binding) {
        newBindings[this.propKey] = this.binding;
      } else {
        delete newBindings[this.propKey];
      }
      doc._updateGraphic(this.graphicId, { bindings: newBindings });
    }
  }

  undo(doc: DocumentModel): void {
    const graphic = doc.getGraphic(this.graphicId);
    if (graphic) {
      const newBindings = { ...graphic.bindings };
      if (this.oldBinding) {
        newBindings[this.propKey] = this.oldBinding;
      } else {
        delete newBindings[this.propKey];
      }
      doc._updateGraphic(this.graphicId, { bindings: newBindings });
    }
  }
}

// 创建自定义符号
class CreateSymbolCommand implements Command {
  readonly type = "CreateSymbol";

  constructor(private symbol: SymbolDef) {}

  execute(doc: DocumentModel): void {
    doc._insertSymbol(this.symbol);
  }

  undo(doc: DocumentModel): void {
    doc._removeSymbol(this.symbol.id);
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

SelectionModel 支持 **混合选择**：同时选中 DOM 组件和 Canvas 图形。

### 4.1 数据结构

```typescript
// 选中元素类型
type SelectableElement =
  | { kind: "node"; id: string } // DOM 组件节点
  | { kind: "graphic"; id: string }; // Canvas 图形

interface SelectionState {
  // 选中的元素列表（支持混合多选）
  selectedElements: SelectableElement[];

  // 当前 hover 的元素
  hoveredElement: SelectableElement | null;

  // 当前拖拽目标（仅限容器节点）
  dropTargetId: string | null;

  // 选中锚点（用于 Shift 连续选择）
  anchorElement: SelectableElement | null;

  // 当前绘图工具
  activeTool: DrawingTool | null;
}

type DrawingTool =
  | "select"
  | "marquee"
  | "pan" // 选择工具
  | "line"
  | "rect"
  | "circle" // 绘图工具
  | "ellipse"
  | "polygon"
  | "pipe"
  | "text";
```

### 4.2 API 设计

```typescript
class SelectionModel {
  private state: SelectionState;

  // ===== 选择操作（支持混合选择）=====
  select(element: SelectableElement): void;
  selectById(id: string): void; // 自动判断是节点还是图形
  toggleSelect(element: SelectableElement): void;
  selectRange(element: SelectableElement): void;
  selectAll(pageId: string): void; // 选中所有节点和图形
  clearSelection(): void;

  // ===== 分类选择 =====
  selectNodes(nodeIds: string[]): void;
  selectGraphics(graphicIds: string[]): void;
  selectOnlyNodes(): void; // 仅保留节点选中
  selectOnlyGraphics(): void; // 仅保留图形选中

  // ===== 查询 =====
  isSelected(id: string): boolean;
  getSelectedElements(): SelectableElement[];
  getSelectedNodes(): ComponentNode[];
  getSelectedGraphics(): GraphicNode[];
  getPrimarySelection(): ComponentNode | GraphicNode | null;
  getSelectionType(): "nodes" | "graphics" | "mixed" | "none";

  // ===== Hover =====
  setHover(element: SelectableElement | null): void;
  setHoverById(id: string | null): void;

  // ===== 拖拽目标 =====
  setDropTarget(nodeId: string | null): void;

  // ===== 绘图工具 =====
  setActiveTool(tool: DrawingTool | null): void;
  getActiveTool(): DrawingTool | null;

  // ===== 事件 =====
  on(event: "change", handler: (state: SelectionState) => void): void;
  on(event: "toolChange", handler: (tool: DrawingTool | null) => void): void;
}
```

### 4.3 多选行为规范

| 操作       | 行为                        |
| ---------- | --------------------------- |
| 单击       | 清除其他选中，选中当前元素  |
| Ctrl+单击  | 切换当前元素选中状态        |
| Shift+单击 | 从锚点到当前元素范围选中    |
| Ctrl+A     | 选中当前页面所有可选元素    |
| Esc        | 清除选中 / 取消当前绘图工具 |
| Delete     | 删除所有选中元素            |

### 4.4 混合选择的特殊处理

当同时选中节点和图形时：

| 场景     | 处理方式                                  |
| -------- | ----------------------------------------- |
| 属性面板 | 显示共同属性（如 bindings），隐藏特有属性 |
| 样式面板 | 显示共同样式（如 opacity），分组显示特有  |
| 批量移动 | 同时移动所有选中元素                      |
| 批量删除 | 同时删除所有选中元素                      |
| 对齐操作 | 基于包围盒对齐                            |
| 复制粘贴 | 保持类型，分别处理                        |

### 4.5 绘图工具状态

```typescript
// 绘图工具激活时的行为
if (
  activeTool !== "select" &&
  activeTool !== "marquee" &&
  activeTool !== "pan"
) {
  // 1. 清除当前选中
  clearSelection();

  // 2. 画布点击进入绘图模式
  // 3. 绘图完成后自动切回 select 工具并选中新图形
}
```

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
- [ ] PageLockManager 锁获取/释放
- [ ] 锁冲突处理与只读模式

## 9. 页面编辑锁（PageLockManager）

### 9.1 设计目标

实现**简化版多人开发**：同一页面同时只能有一人编辑，防止冲突。

| 特性         | 支持 | 说明                               |
| ------------ | ---- | ---------------------------------- |
| 实时协同编辑 | ❌   | 不做多光标同步                     |
| 页面级编辑锁 | ✅   | 同一页面同时只能有一人编辑         |
| 锁冲突提示   | ✅   | 尝试编辑已锁定页面时提示当前编辑者 |
| 只读模式     | ✅   | 无法获取锁时自动进入只读模式       |
| 锁超时释放   | ✅   | 30 分钟无操作或断连自动释放        |

### 9.2 数据模型

```typescript
// 锁状态（本地存储）
interface PageLockState {
  pageId: string;
  locked: boolean;
  lockedBy?: string; // 用户 ID
  lockedByName?: string; // 用户姓名
  lockedAt?: number; // 锁定时间戳
  isOwner: boolean; // 是否是自己持有的锁
}

// 编辑器只读状态
interface EditorReadonlyState {
  readonly: boolean;
  reason?: "no_permission" | "page_locked" | "viewer_role";
  lockedByName?: string;
}
```

### 9.3 PageLockManager 实现

```typescript
class PageLockManager {
  private currentPageId: string | null = null;
  private lockState: PageLockState | null = null;
  private heartbeatTimer: number | null = null;
  private socket: Socket;

  constructor(private api: ApiClient, socket: Socket) {
    this.socket = socket;
    this.setupSocketListeners();
  }

  // ===== 锁操作 =====

  /**
   * 尝试获取页面编辑锁
   * @returns 是否成功获取锁
   */
  async acquireLock(pageId: string): Promise<LockResult> {
    try {
      const response = await this.api.post(`/pages/${pageId}/lock`);

      if (response.success) {
        this.currentPageId = pageId;
        this.lockState = {
          pageId,
          locked: true,
          lockedBy: response.data.lockedBy,
          lockedByName: response.data.lockedByName,
          lockedAt: response.data.lockedAt,
          isOwner: true,
        };
        this.startHeartbeat();
        return { success: true };
      } else {
        // 锁被其他人持有
        this.lockState = {
          pageId,
          locked: true,
          lockedBy: response.data.lockedBy,
          lockedByName: response.data.lockedByName,
          lockedAt: response.data.lockedAt,
          isOwner: false,
        };
        return {
          success: false,
          reason: "locked",
          lockedByName: response.data.lockedByName,
        };
      }
    } catch (error) {
      return { success: false, reason: "error", error };
    }
  }

  /**
   * 释放页面编辑锁
   */
  async releaseLock(): Promise<void> {
    if (!this.currentPageId || !this.lockState?.isOwner) return;

    try {
      await this.api.delete(`/pages/${this.currentPageId}/lock`);
    } catch (error) {
      console.error("Failed to release lock:", error);
    } finally {
      this.stopHeartbeat();
      this.currentPageId = null;
      this.lockState = null;
    }
  }

  /**
   * 查询页面锁状态
   */
  async queryLockStatus(pageId: string): Promise<PageLockState> {
    const response = await this.api.get(`/pages/${pageId}/lock`);
    return {
      pageId,
      locked: response.data.locked,
      lockedBy: response.data.lockedBy,
      lockedByName: response.data.lockedByName,
      lockedAt: response.data.lockedAt,
      isOwner: response.data.lockedBy === this.currentUserId,
    };
  }

  // ===== 心跳续锁 =====

  private startHeartbeat(): void {
    this.stopHeartbeat();
    // 每 5 分钟发送一次心跳续锁
    this.heartbeatTimer = setInterval(async () => {
      if (this.currentPageId && this.lockState?.isOwner) {
        try {
          await this.api.post(`/pages/${this.currentPageId}/lock/heartbeat`);
        } catch (error) {
          console.error("Heartbeat failed:", error);
          // 心跳失败可能意味着锁已被释放
          this.handleLockLost();
        }
      }
    }, 5 * 60 * 1000);
  }

  private stopHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }

  // ===== WebSocket 监听 =====

  private setupSocketListeners(): void {
    // 监听页面锁状态变化
    this.socket.on("page:lock:changed", (data: PageLockChangedEvent) => {
      if (data.pageId === this.currentPageId) {
        if (data.locked && data.lockedBy !== this.currentUserId) {
          // 其他人获取了锁
          this.handleLockLost();
        }
      }
      // 通知 UI 更新页面树中的锁状态显示
      this.emit("lockStatusChanged", data);
    });

    // 监听强制释放通知
    this.socket.on("page:lock:force_release", (data) => {
      if (data.pageId === this.currentPageId && this.lockState?.isOwner) {
        this.handleLockForceReleased(data);
      }
    });
  }

  private handleLockLost(): void {
    this.lockState = { ...this.lockState!, isOwner: false };
    this.stopHeartbeat();
    this.emit("lockLost", { pageId: this.currentPageId });
  }

  private handleLockForceReleased(data: any): void {
    this.lockState = null;
    this.currentPageId = null;
    this.stopHeartbeat();
    this.emit("lockForceReleased", data);
  }

  // ===== 生命周期 =====

  /**
   * 页面离开时调用
   */
  async onPageLeave(): Promise<void> {
    await this.releaseLock();
  }

  /**
   * 应用关闭/刷新时调用
   */
  async onAppUnload(): Promise<void> {
    // 使用 sendBeacon 确保请求发出
    if (this.currentPageId && this.lockState?.isOwner) {
      navigator.sendBeacon(
        `/api/v1/pages/${this.currentPageId}/lock/release`,
        JSON.stringify({ userId: this.currentUserId })
      );
    }
  }
}

interface LockResult {
  success: boolean;
  reason?: "locked" | "error";
  lockedByName?: string;
  error?: any;
}
```

### 9.4 与编辑器集成

```typescript
// Designer 页面切换时
async function onPageChange(newPageId: string) {
  // 1. 释放旧页面的锁
  await lockManager.onPageLeave();

  // 2. 尝试获取新页面的锁
  const result = await lockManager.acquireLock(newPageId);

  if (result.success) {
    // 正常编辑模式
    setEditorReadonly(false);
    loadPage(newPageId);
  } else {
    // 只读模式
    setEditorReadonly(true, {
      reason: "page_locked",
      lockedByName: result.lockedByName,
    });
    loadPage(newPageId, { readonly: true });

    // 显示提示
    showLockConflictDialog(result.lockedByName);
  }
}

// 页面树中显示锁状态
function PageTreeItem({ page, lockStatus }) {
  return (
    <div className="page-tree-item">
      <Icon name="page" />
      <span>{page.name}</span>
      {lockStatus?.locked && (
        <Tooltip content={`正在被 ${lockStatus.lockedByName} 编辑`}>
          <Icon
            name="lock"
            className={lockStatus.isOwner ? "lock-own" : "lock-other"}
          />
        </Tooltip>
      )}
    </div>
  );
}
```

### 9.5 只读模式实现

```typescript
// 编辑器只读状态管理
const editorStore = defineStore("editor", {
  state: () => ({
    readonly: false,
    readonlyReason: null as EditorReadonlyState["reason"] | null,
    lockedByName: null as string | null,
  }),

  actions: {
    setReadonly(readonly: boolean, info?: Partial<EditorReadonlyState>) {
      this.readonly = readonly;
      this.readonlyReason = info?.reason ?? null;
      this.lockedByName = info?.lockedByName ?? null;
    },
  },
});

// 命令执行拦截
class History {
  execute(command: Command): void {
    // 检查只读状态
    if (editorStore.readonly) {
      console.warn("Editor is in readonly mode, command blocked:", command);
      showToast("当前为只读模式，无法编辑");
      return;
    }

    // 正常执行命令
    command.execute(this.doc);
    // ...
  }
}
```

### 9.6 API 定义

| 端点                            | 方法   | 说明       | 权限       |
| ------------------------------- | ------ | ---------- | ---------- |
| `/pages/:pageId/lock`           | GET    | 查询锁状态 | 工程成员   |
| `/pages/:pageId/lock`           | POST   | 获取锁     | DEVELOPER+ |
| `/pages/:pageId/lock`           | DELETE | 释放锁     | 锁持有者   |
| `/pages/:pageId/lock/heartbeat` | POST   | 心跳续锁   | 锁持有者   |
| `/pages/:pageId/lock/force`     | DELETE | 强制释放锁 | ADMIN+     |

### 9.7 锁超时处理（服务端）

```javascript
// 定时任务：每分钟检查过期锁
const LOCK_TIMEOUT = 30 * 60 * 1000; // 30 分钟

async function cleanExpiredLocks() {
  const expiredPages = await DesignPage.findAll({
    where: {
      lockedBy: { [Op.ne]: null },
      lockedAt: { [Op.lt]: new Date(Date.now() - LOCK_TIMEOUT) },
    },
  });

  for (const page of expiredPages) {
    // 释放锁
    await page.update({ lockedBy: null, lockedAt: null });

    // 广播锁释放事件
    socketService.broadcast(`designer:${page.projectId}`, "page:lock:changed", {
      pageId: page.id,
      pageName: page.name,
      locked: false,
    });

    // 通知原锁定者
    socketService.sendToUser(page.lockedBy, "page:lock:force_release", {
      pageId: page.id,
      pageName: page.name,
      reason: "timeout",
    });

    logger.info(`Lock expired and released for page ${page.id}`);
  }
}

// 每分钟执行一次
setInterval(cleanExpiredLocks, 60 * 1000);
```

---

**相关文档**：

- [Schema 设计](./schema-design.md)
- [数据绑定 v2](./data-binding-v2.md)
- [WebSocket 协议](../../backend/websocket.md)
