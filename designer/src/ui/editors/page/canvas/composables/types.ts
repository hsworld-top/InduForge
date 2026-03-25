import type { ComputedRef, Ref, ShallowRef } from "vue";
import type { ComponentNode, SelectableElement } from "@/editor-core/document/types";

export type MaybeRef<T> = Ref<T> | ComputedRef<T>;

export interface CanvasDocLike {
  getNode: (id: string) => ComponentNode | null;
  getParent: (id: string) => ComponentNode | null;
  vars?: {
    pages?: Record<string, Record<string, unknown>>;
  };
}

export interface CanvasMutableDocLike extends CanvasDocLike {
  _moveNode?: (nodeId: string, parentId: string, index: number) => void;
  _updateNode?: (id: string, patch: Record<string, unknown>) => void;
}

export interface CanvasSelectionLike {
  isSelected: (id: string) => boolean;
}

export interface CanvasSelectionController extends CanvasSelectionLike {
  select: (element: SelectableElement) => void;
}

export interface CanvasSelectionManager extends CanvasSelectionController {
  getPrimaryElement?: () => SelectableElement | null;
  getSelectedElements?: () => SelectableElement[];
  selectRange?: (element: SelectableElement) => void;
  toggleSelect?: (element: SelectableElement) => void;
}

export interface HistoryLike {
  beginTransaction?: () => void;
  commitTransaction?: (label?: string) => void;
  execute?: (command: unknown) => void;
  executeInTransaction?: (command: unknown) => void;
  isInTransaction?: () => boolean;
  rollbackTransaction?: () => void;
}

export interface UseNodePropsDeps {
  node: ComputedRef<ComponentNode | null | undefined>;
  doc: ComputedRef<CanvasDocLike | null | undefined>;
  currentPage: ComputedRef<{ id?: string | null } | null | undefined>;
  projectVariables: ComputedRef<Record<string, unknown> | null | undefined>;
  docVersion: ComputedRef<number>;
  readonly: ComputedRef<boolean>;
}

export interface ExpressionContextDeps {
  doc: ComputedRef<CanvasDocLike | null | undefined>;
  currentPage: ComputedRef<{ id?: string | null } | null | undefined>;
  projectVariables: ComputedRef<Record<string, unknown> | null | undefined>;
}

export interface UseNodeRendererDerivationsDeps {
  node: ComputedRef<ComponentNode | null | undefined>;
  detailConfigText: ComputedRef<string>;
  docVersion: Ref<number>;
  tableRenderVersion: Ref<number>;
  resolvedNodeProps: ComputedRef<Record<string, unknown>>;
  selectionVersion: Ref<number>;
  selection: ShallowRef<CanvasSelectionLike | null | undefined>;
  doc: ShallowRef<CanvasDocLike | null | undefined>;
  isContainer: ComputedRef<boolean>;
  isMovable: ComputedRef<boolean>;
  isDropActive: ComputedRef<boolean>;
  activeTabName: Ref<string>;
  tabsList: ComputedRef<Array<Record<string, unknown>>>;
  props: {
    isRoot?: boolean;
    readonly?: boolean;
  };
}

export interface MenuDslApplyContext {
  node: ComputedRef<ComponentNode | null | undefined>;
  readonly: Ref<boolean>;
  applyPreviewPatch: (patch: Record<string, unknown>) => void;
  editorStore: {
    updateNode: (id: string, patch: Record<string, unknown>) => boolean;
  };
  normalizeMenuItems: (items: unknown) => Array<Record<string, unknown>>;
}

export interface ResizeHandle {
  key: string;
  x: -1 | 0 | 1;
  y: -1 | 0 | 1;
  cursor?: string;
}

export interface RegionResizeConfig {
  axis: "x" | "y";
  prop: "width" | "height";
  handles: string[];
  invert?: boolean;
}

export interface UseNodeResizeDeps {
  node: MaybeRef<ComponentNode | null | undefined>;
  doc: MaybeRef<CanvasMutableDocLike | null | undefined>;
  nodeRef: Ref<HTMLElement | null | undefined>;
  readonly: MaybeRef<boolean>;
  isMovable: MaybeRef<boolean>;
  isElColInRow: MaybeRef<boolean>;
  isChildInElCol: MaybeRef<boolean>;
  selection: MaybeRef<CanvasSelectionController | null | undefined>;
  canvasZoom: MaybeRef<number | null | undefined>;
  history: MaybeRef<HistoryLike | null | undefined>;
  editorStore: {
    updateNode: (id: string, patch: Record<string, unknown>) => boolean;
  };
  isChildResizableByDescriptor: (type: string) => boolean;
  getRegionResizeConfig: (type: string) => RegionResizeConfig | null;
}

export interface UseNodeInteractionDeps {
  node: ComputedRef<ComponentNode | null | undefined>;
  doc: ComputedRef<CanvasDocLike | null | undefined>;
  selection: MaybeRef<CanvasSelectionManager | null | undefined>;
  selectionVersion: MaybeRef<number>;
  readonly: ComputedRef<boolean>;
  isRoot: ComputedRef<boolean>;
  isRootCanvasContainer: (node: ComponentNode | null | undefined) => boolean;
  isChildResizableByDescriptor: (type: string) => boolean;
  nodeRef: Ref<HTMLElement | null | undefined>;
  createSelectableElement: (kind: "node" | "graphic", id: string) => SelectableElement;
  runPreviewScript: (eventName: string, event?: Event) => unknown;
  handleSelect: (event: MouseEvent) => void;
  showContextMenu?: ((event: MouseEvent) => void) | null;
}

export interface DragStateLike {
  dragType: string;
  targetContainerId: string;
}

export interface InsertLineStyleLike {
  orientation: "horizontal" | "vertical";
  offset: number;
}

export interface InsertLineBoxLike {
  left: number;
  top: number;
  width: number;
  height: number;
}

export interface RowInsertInfoLike {
  rowId: string;
  index: number;
  lineBox: InsertLineBoxLike;
}

export interface LayoutInsertInfoLike {
  layoutId: string;
  index: number;
  lineBox: InsertLineBoxLike;
}

export interface DragDropManagerLike {
  calculateFlexInsertPosition: (
    containerElement: Element,
    event: MouseEvent | DragEvent,
    direction?: string,
  ) => {
    index: number;
    insertLine: InsertLineStyleLike | null;
  };
}

export interface InsertNodeOptionsLike {
  dropPosition?:
    | {
        x: number;
        y: number;
      }
    | undefined;
}

export interface CanvasEditorStoreLike {
  insertNode: (
    type: string,
    parentId?: string,
    index?: number,
    options?: InsertNodeOptionsLike,
  ) => ComponentNode | null;
  updateNode: (id: string, patch: Record<string, unknown>) => boolean;
}

export interface UseNodeDropDeps {
  node: ComputedRef<ComponentNode | null | undefined>;
  doc: ComputedRef<CanvasDocLike | null | undefined>;
  dragState: DragStateLike;
  dragDropManager: DragDropManagerLike;
  isContainer: ComputedRef<boolean>;
  resolveFlexDirection: (type: string, element: Element | null | undefined) => string;
  isFlexContainer: (type: string) => boolean;
  readonly: ComputedRef<boolean>;
  editorStore: CanvasEditorStoreLike;
  canvasZoom: Ref<number | null | undefined>;
  endDrag: () => void;
  notifyInsertFailure: (message?: string) => void;
  activeTabName: Ref<string>;
  tabsList: ComputedRef<Array<Record<string, unknown>>>;
}

export interface DropTargetPositionLike {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface DropTargetLike {
  containerId: string;
  insertIndex: number;
  position: DropTargetPositionLike;
  layoutType: "flex";
  direction: "row" | "column";
}

export type PointerCaptureTargetLike = Element;

export interface PointerDragHandlers {
  move: (event: MouseEvent | PointerEvent) => void;
  up: (event: MouseEvent | PointerEvent) => void;
  userSelect?: string;
  pointerTarget?: PointerCaptureTargetLike | null | undefined;
  pointerId?: number;
  usePointer: boolean;
  pointerEvents?: string;
  pointerElement?: HTMLElement | null;
}

export interface NodePatchLike extends Record<string, unknown> {
  positioning?: "absolute" | "flow" | undefined;
  absolutePos?: ComponentNode["absolutePos"] | undefined;
  flowLayout?: ComponentNode["flowLayout"] | undefined;
  layoutItem?: ComponentNode["layoutItem"] | undefined;
  props?: ComponentNode["props"] | undefined;
  style?: ComponentNode["style"] | undefined;
}

export interface UseNodePointerDeps {
  node: ComputedRef<ComponentNode | null | undefined>;
  doc: ComputedRef<CanvasMutableDocLike | null | undefined>;
  nodeRef: Ref<HTMLElement | null | undefined>;
  readonly: ComputedRef<boolean>;
  isMovable: ComputedRef<boolean>;
  isContainer: ComputedRef<boolean>;
  selection: MaybeRef<CanvasSelectionManager | null | undefined>;
  canvasZoom: MaybeRef<number | null | undefined>;
  history: MaybeRef<HistoryLike | null | undefined>;
  editorStore: CanvasEditorStoreLike;
  currentPage: ComputedRef<{ rootNodeId?: string | null } | null | undefined>;
  startDrag: (dragType: string) => void;
  endDrag: () => void;
  updateDropTarget: (target: DropTargetLike) => void;
  clearDropTarget: () => void;
  dragDropManager: DragDropManagerLike;
  canAcceptChild: (parentNode: ComponentNode, childType: string) => boolean;
  showInsertLine: Ref<boolean>;
  insertLineStyle: Ref<InsertLineStyleLike | null>;
  rowInsertInfo: Ref<RowInsertInfoLike | null>;
  layoutInsertInfo: Ref<LayoutInsertInfoLike | null>;
  activeTabName: Ref<string>;
  tabsList: ComputedRef<Array<Record<string, unknown>>>;
  resolveFlexDirection: (type: string, element: Element | null | undefined) => string;
}

export interface PreviewPatchLike {
  props?: Record<string, unknown>;
  style?: Record<string, unknown>;
  bindings?: Record<string, unknown>;
  events?: Record<string, unknown>;
  conditions?: Record<string, unknown>;
  permissions?: Record<string, unknown>;
  hidden?: boolean;
  label?: string;
}

export interface RuntimeComponentRefLike {
  $el?: HTMLElement | null | undefined;
  $emit?: (event: string, ...args: unknown[]) => void;
  callECharts?: (method: string, ...args: unknown[]) => unknown;
  setOption?: (
    option: unknown,
    notMergeOrOpts?: unknown,
    lazyUpdate?: boolean,
    silent?: boolean,
    replaceMerge?: unknown,
  ) => void;
  showPreview?: () => void;
  updateKeyChildren?: (key: unknown, data: unknown) => void;
  getCheckedNodes?: (...args: unknown[]) => unknown;
  setCheckedNodes?: (nodes: unknown) => void;
  getCheckedKeys?: (leafOnly?: boolean) => unknown;
  setCheckedKeys?: (keys: unknown, leafOnly?: boolean) => void;
  setChecked?: (keyOrData: unknown, checked?: boolean, deep?: boolean) => void;
  getHalfCheckedNodes?: () => unknown;
  getHalfCheckedKeys?: () => unknown;
  getCurrentKey?: () => unknown;
  getCurrentNode?: () => unknown;
  setCurrentKey?: (key: unknown) => void;
  setCurrentNode?: (nodeData: unknown) => void;
  getNode?: (dataOrKey: unknown) => unknown;
  remove?: (dataOrNode: unknown) => void;
  append?: (data: unknown, parentNode?: unknown) => void;
  insertBefore?: (data: unknown, refNode?: unknown) => void;
  insertAfter?: (data: unknown, refNode?: unknown) => void;
  setExpandedKeys?: (keys: unknown[]) => void;
  getExpandedKeys?: () => unknown;
  filter?: (keyword: string) => void;
  open?: (index?: unknown) => void;
  close?: (index?: unknown) => void;
  handleOpen?: () => void;
  handleClose?: () => void;
  toggleMenu?: (visible?: boolean) => void;
  togglePopperVisible?: (visible?: boolean) => void;
  focus?: () => void;
  blur?: () => void;
  select?: () => void;
  clearQuery?: (area?: unknown) => void;
  clearSelection?: () => void;
  toggleRowSelection?: (row: unknown, selected?: boolean) => void;
  toggleAllSelection?: () => void;
  toggleRowExpansion?: (row: unknown, expanded?: boolean) => void;
  setCurrentRow?: (row: unknown) => void;
  clearSort?: () => void;
  clearFilter?: (columnKeys?: unknown) => void;
  doLayout?: () => void;
  sort?: (prop: unknown, order?: unknown) => void;
  getSelectionRows?: () => unknown[];
  setScrollTop?: (top: number) => void;
  scrollTo?: (keyOrRow: unknown) => void;
  next?: () => void;
  prev?: () => void;
  setActiveItem?: (nameOrIndex: unknown) => void;
}

export interface UseBuildRefInfoDeps {
  node: ComputedRef<ComponentNode | null | undefined>;
  nodeRef: Ref<HTMLElement | null | undefined>;
  contentRef: Ref<RuntimeComponentRefLike | null | undefined>;
  editorStore: {
    updateNode: (id: string, patch: Record<string, unknown>) => boolean;
  };
  readonly: ComputedRef<boolean>;
  tableRenderVersion: Ref<number>;
  docVersion: Ref<number>;
  isRunningDetailConfigFn?: (() => boolean) | null;
}
