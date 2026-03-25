/**
 * 组件方法补全配置
 *
 * 用于 Monaco 编辑器中脚本/变量变更时的补全提示，
 * 根据组件类型提供对应的方法（如 Table.SetData、Button.Click 等）。
 */

export interface CompletionItem {
  label: string;
  insertText: string;
  kind?: string;
  detail?: string;
  prefix?: string | string[];
}

interface ComponentTreeNode {
  type?: string;
  componentName?: string;
  componentType?: string;
  children?: ComponentTreeNode[];
}

export interface FlatComponentRef {
  componentName: string;
  componentType: string;
}

/** 通用属性补全项（Name、Location、Size 等） */
const COMMON_PROPERTIES: Array<Pick<CompletionItem, "label" | "insertText" | "detail">> = [
  { label: "Name", insertText: "Name", detail: "组件名称" },
  { label: "Comment", insertText: "Comment", detail: "组件类型说明" },
  { label: "Location", insertText: "Location", detail: "组件位置对象" },
  { label: "Location.X", insertText: "Location.X", detail: "横坐标" },
  { label: "Location.Y", insertText: "Location.Y", detail: "纵坐标" },
  { label: "Size", insertText: "Size", detail: "组件尺寸对象" },
  { label: "Size.Width", insertText: "Size.Width", detail: "宽度" },
  { label: "Size.Height", insertText: "Size.Height", detail: "高度" },
  { label: "Visible", insertText: "Visible", detail: "显示状态" },
  { label: "Enable", insertText: "Enable", detail: "启用状态" },
  { label: "Caption", insertText: "Caption", detail: "显示文本" },
  { label: "Image", insertText: "Image", detail: "图片地址" },
];

/** 组件类型到方法列表的映射 */
const METHOD_MAP: Record<string, string[]> = {
  Text: ["SetText", "GetText", "SetType", "SetEllipsis", "SetTooltip"],
  Tag: ["SetText", "SetType", "Close"],
  Button: ["SetLoading", "SetDisabled", "Click", "Focus"],
  Image: ["SetSrc", "GetSrc", "Preview", "Reload"],
  Input: [
    "Focus",
    "Blur",
    "Select",
    "SetInputValue",
    "GetInputValue",
    "SetValue",
    "GetValue",
    "Clear",
  ],
  Select: [
    "Focus",
    "Blur",
    "Open",
    "Close",
    "Toggle",
    "SetValue",
    "GetValue",
    "SetData",
    "GetData",
    "Clear",
    "GetCheckedNodes",
    "SetCheckedNodes",
  ],
  Switch: ["SetValue", "GetValue", "Toggle", "Disable"],
  Table: [
    "ClearSelection",
    "AppendRow",
    "ToggleRowSelection",
    "ToggleAllSelection",
    "ToggleRowExpansion",
    "SetCurrentRow",
    "ClearSort",
    "ClearFilter",
    "Dolayout",
    "Sort",
    "SetData",
    "GetData",
    "GetSelection",
    "SetSelectionByKeys",
    "GetSelectionKeys",
    "SetPage",
    "SetPageSize",
    "GetPageData",
    "UpdateRowByKey",
    "RemoveRowByKey",
    "UpsertRowByKey",
    "ScrollToTop",
    "ScrollToRow",
    "DoLayoutSafe",
  ],
  BigDataTable: [
    "ClearSelection",
    "AppendRow",
    "ToggleRowSelection",
    "ToggleAllSelection",
    "ToggleRowExpansion",
    "SetCurrentRow",
    "ClearSort",
    "ClearFilter",
    "Dolayout",
    "Sort",
    "SetData",
    "GetData",
    "GetSelection",
    "SetSelectionByKeys",
    "GetSelectionKeys",
    "SetPage",
    "SetPageSize",
    "GetPageData",
    "UpdateRowByKey",
    "RemoveRowByKey",
    "UpsertRowByKey",
    "ScrollToTop",
    "ScrollToRow",
    "DoLayoutSafe",
  ],
  Tree: [
    "UpdateKeyChildren",
    "GetCheckedNodes",
    "SetCheckedNodes",
    "GetCheckedKeys",
    "SetCheckedKeys",
    "SetChecked",
    "GetHalfCheckedNodes",
    "GetHalfCheckedKeys",
    "GetCurrentKey",
    "GetCurrentNode",
    "SetCurrentKey",
    "SetCurrentNode",
    "GetNode",
    "Remove",
    "Append",
    "InsertBefore",
    "InsertAfter",
    "ExpandAll",
    "CollapseAll",
    "SetExpandedKeys",
    "GetExpandedKeys",
    "Filter",
    "ClearFilter",
  ],
  Dropdown: [
    "InsertItem",
    "GetCommandItem",
    "GetMenuItem",
    "DeleteItem",
    "ClearAll",
    "Open",
    "Close",
    "Toggle",
    "GetVisible",
  ],
  Menu: ["Open", "Close", "SetActive", "GetActive", "Collapse"],
  Radio: [
    "GetRadioChecked",
    "GetRadioValue",
    "GetRadioLabel",
    "SetRadioEnable",
    "GetRadioEnable",
    "SetRadioVisible",
    "GetRadioVisible",
    "SetValue",
    "GetValue",
    "Clear",
  ],
  Checkbox: [
    "GetCheckState",
    "SetCheckState",
    "SetCheckEnable",
    "GetCheckEnable",
    "SetCheckVisible",
    "GetCheckVisible",
    "CheckAll",
    "SetValue",
    "GetValue",
    "Clear",
  ],
  Cascader: [
    "Open",
    "Close",
    "Toggle",
    "GetCheckedNodes",
    "SetData",
    "GetData",
    "SetValue",
    "GetValue",
    "Clear",
  ],
  Tabs: ["SetActive", "GetActive", "Next", "Prev", "AddTab", "RemoveTab"],
  Transfer: [
    "SetData",
    "GetData",
    "ClearQuery",
    "SetValue",
    "GetValue",
    "Clear",
    "MoveToRight",
    "MoveToLeft",
  ],
  InputNumber: ["SetValue", "GetValue", "Increase", "Decrease", "Focus", "Blur", "Select"],
  Timeline: ["SetItems", "AppendItem", "Clear"],
  ImageCarousel: ["SetActiveItem", "Prev", "Next", "Play", "Pause", "SetItems"],
  CarouselComponent: ["SetActiveItem", "Prev", "Next", "Play", "Pause", "SetItems"],
  WebContainer: ["Load", "Reload", "PostMessage", "GetUrl", "Back", "Forward"],
  Steps: ["SetActive", "Next", "Prev", "Reset", "SetItems"],
  Card: ["SetTitle", "SetLoading", "Collapse"],
  Pagination: ["SetPage", "GetPage", "SetPageSize", "GetPageSize", "Reset", "SetTotal", "GetTotal"],
  Collapse: ["Open", "Close", "Toggle", "GetActiveNames", "SetActiveNames"],
  BusinessCard: ["SetData", "GetData", "SetLoading", "Refresh", "OpenDetail", "SetTitle"],
  Barcode: ["SetValue", "GetValue", "Render", "Download"],
  Slider: ["SetValue", "GetValue", "Reset", "Disable"],
  Calendar: ["SetDate", "GetDate", "Today"],
  Signature: ["Clear", "GetImage", "SetImage", "IsEmpty", "SetPen"],
  EChart: ["setOption", "echarts"],
};

/**
 * 扁平化组件树
 */
export function flattenComponentTree(
  tree: ComponentTreeNode[] | undefined | null,
): FlatComponentRef[] {
  const result: FlatComponentRef[] = [];
  const walk = (nodes: ComponentTreeNode[] | undefined) => {
    if (!Array.isArray(nodes)) return;
    nodes.forEach((node) => {
      if (!node) return;
      if (node.type === "component" && node.componentName) {
        result.push({
          componentName: node.componentName,
          componentType: node.componentType || "",
        });
      }
      if (Array.isArray(node.children) && node.children.length > 0) {
        walk(node.children);
      }
    });
  };
  walk(tree ?? undefined);
  return result;
}

/**
 * 构建组件方法补全列表
 */
export function buildComponentMethodCompletions(
  tree: ComponentTreeNode[] | undefined | null,
): CompletionItem[] {
  const components = flattenComponentTree(tree);
  const items: CompletionItem[] = [];
  components.forEach((component) => {
    const prefix = `components.${component.componentName}.`;
    const methods = METHOD_MAP[component.componentType] || [];
    COMMON_PROPERTIES.forEach((prop) => {
      const item: CompletionItem = {
        label: prop.label,
        insertText: prop.insertText,
        kind: "Property",
        prefix,
      };
      if (prop.detail !== undefined) {
        item.detail = prop.detail;
      }
      items.push(item);
    });
    methods.forEach((method) => {
      const text = `${method}()`;
      items.push({
        label: method,
        insertText: text,
        kind: "Function",
        detail: `${component.componentType}方法`,
        prefix,
      });
    });
  });
  return items;
}
