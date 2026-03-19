/**
 * 构建预览运行时组件 ref 接口（DSL 方法映射）
 *
 * 从 NodeRenderer 抽取的 buildRefInfo、applyPreviewPatch 及相关辅助逻辑。
 *
 * @module ui/Canvas/composables/use-build-ref-info
 */

import { nextTick } from "vue";
import { componentRegistry } from "@/editor-core";

/**
 * 创建 buildRefInfo / applyPreviewPatch
 * @param {Object} deps - 依赖
 * @param {import('vue').ComputedRef} deps.node - 当前节点
 * @param {import('vue').Ref} deps.nodeRef - 节点根 DOM/组件 ref
 * @param {import('vue').Ref} deps.contentRef - 内容区组件 ref
 * @param {import('vue').Store} deps.editorStore - 编辑器 store（含 updateNode）
 * @param {import('vue').ComputedRef<boolean>} deps.readonly - 是否只读/预览
 * @param {import('vue').Ref<number>} deps.tableRenderVersion - 表格强制刷新版本
 * @param {import('vue').Ref<number>} deps.docVersion - 文档版本（applyPreviewPatch 递增）
 * @param {Function} deps.isRunningDetailConfigFn - 是否正在执行 detailConfig（防循环）
 * @returns {{ buildRefInfo: Function, applyPreviewPatch: Function }}
 */
export function useBuildRefInfo({
  node,
  nodeRef,
  contentRef,
  editorStore,
  readonly,
  tableRenderVersion,
  docVersion,
  isRunningDetailConfigFn,
}) {
  const applyPreviewPatch = (patch) => {
    if (!node.value || !patch || typeof patch !== "object") return;
    if (patch.props) {
      node.value.props = { ...(node.value.props || {}), ...patch.props };
    }
    if (patch.style) {
      node.value.style = { ...(node.value.style || {}), ...patch.style };
    }
    if (patch.bindings) {
      node.value.bindings = { ...(node.value.bindings || {}), ...patch.bindings };
    }
    if (patch.events) {
      node.value.events = { ...(node.value.events || {}), ...patch.events };
    }
    if (patch.conditions) {
      node.value.conditions = {
        ...(node.value.conditions || {}),
        ...patch.conditions,
      };
    }
    if (patch.permissions) {
      node.value.permissions = {
        ...(node.value.permissions || {}),
        ...patch.permissions,
      };
    }
    if (typeof patch.hidden === "boolean") {
      node.value.hidden = patch.hidden;
    }
    if (patch.label !== undefined) {
      node.value.label = patch.label;
    }
    docVersion.value += 1;
  };
  /**
   * @typedef {Object} ButtonDslOnClick
   * @property {string} action - 点击动作脚本
   * @property {string} [confirm] - 二次确认提示
   */
  
  /**
   * @typedef {Object} ButtonDslConfig
   * @property {string} id - DOM 唯一 ID
   * @property {string} [text] - 静态文字
   * @property {string} [textExpr] - 初始化表达式
   * @property {boolean | string} [visible] - 显隐（支持表达式）
   * @property {string} [permission] - 权限码
   * @property {'primary' | 'success' | 'warning' | 'danger' | 'info' | 'text'} [type] - 类型
   * @property {'large' | 'default' | 'small'} [size] - 尺寸
   * @property {boolean} [plain] - 朴素按钮
   * @property {boolean} [round] - 圆角按钮
   * @property {boolean} [circle] - 圆形按钮
   * @property {boolean} [disabled] - 禁用
   * @property {boolean} [loading] - 加载中
   * @property {string} [icon] - 图标
   * @property {Record<string, string>} [style] - 样式
   * @property {string} [className] - 自定义类名
   * @property {string | ButtonDslOnClick} [onClick] - 点击事件
   * @property {*} [plugin] - 插件扩展
   */
  
  /**
   * 生成按钮 DSL 的点击脚本
   * @param {string | ButtonDslOnClick} onClick - 点击配置
   * @returns {string}
   */
  const buildButtonClickScript = (onClick) => {
    if (typeof onClick === "string") return onClick;
    if (!onClick || typeof onClick !== "object") return "";
    const action =
      typeof onClick.action === "string" ? onClick.action.trim() : "";
    if (!action) return "";
    if (typeof onClick.confirm === "string" && onClick.confirm.trim()) {
      const confirmText = JSON.stringify(onClick.confirm.trim());
      return `if (confirm(${confirmText})) {\n  ${action}\n}`;
    }
    return action;
  };
  
  /**
   * 生成按钮 DSL 更新补丁
   * @param {ButtonDslConfig} config - 按钮 DSL 配置
   * @returns {{
   *   propsPatch: Record<string, *>,
   *   stylePatch: Record<string, *>,
   *   bindingsPatch: Record<string, *>,
   *   eventsPatch: Record<string, *>,
   *   conditionsPatch: Record<string, *>,
   *   hidden: boolean | undefined,
   * }}
   */
  const buildButtonDslPatch = (config) => {
    const propsPatch = {};
    const stylePatch = {};
    const bindingsPatch = {};
    const eventsPatch = {};
    const conditionsPatch = {};
    let hidden;
  
    if (!config || typeof config !== "object") {
      return {
        propsPatch,
        stylePatch,
        bindingsPatch,
        eventsPatch,
        conditionsPatch,
        hidden,
      };
    }
  
    if (Object.prototype.hasOwnProperty.call(config, "id")) {
      const value = String(config.id || "");
      if (value) propsPatch.id = value;
    }
    if (Object.prototype.hasOwnProperty.call(config, "text")) {
      propsPatch.text = String(config.text ?? "");
    }
    if (typeof config.textExpr === "string" && config.textExpr.trim()) {
      bindingsPatch.text = {
        kind: "expr",
        expr: config.textExpr.trim(),
        fallback:
          Object.prototype.hasOwnProperty.call(config, "text") &&
            config.text !== undefined
            ? String(config.text ?? "")
            : undefined,
      };
    }
    if (typeof config.visible === "boolean") {
      hidden = !config.visible;
    } else if (typeof config.visible === "string" && config.visible.trim()) {
      conditionsPatch.visible = config.visible.trim();
    }
    if (typeof config.permission === "string" && config.permission.trim()) {
      propsPatch.permission = config.permission.trim();
    }
    if (typeof config.type === "string" && config.type.trim()) {
      propsPatch.type = config.type.trim();
    }
    if (typeof config.size === "string" && config.size.trim()) {
      propsPatch.size = config.size.trim();
    }
    if (typeof config.plain === "boolean") propsPatch.plain = config.plain;
    if (typeof config.round === "boolean") propsPatch.round = config.round;
    if (typeof config.circle === "boolean") propsPatch.circle = config.circle;
    if (typeof config.disabled === "boolean")
      propsPatch.disabled = config.disabled;
    if (typeof config.loading === "boolean") propsPatch.loading = config.loading;
    if (typeof config.icon === "string" && config.icon.trim()) {
      propsPatch.icon = config.icon.trim();
    }
    if (config.style && typeof config.style === "object") {
      Object.entries(config.style).forEach(([key, value]) => {
        if (typeof value === "string") {
          stylePatch[key] = value;
        }
      });
    }
    if (typeof config.className === "string" && config.className.trim()) {
      propsPatch.class = config.className.trim();
    }
    if (config.onClick) {
      const code = buildButtonClickScript(config.onClick);
      if (code) {
        eventsPatch.click = [{ type: "script", code, enabled: true }];
      }
    }
    if (Object.prototype.hasOwnProperty.call(config, "plugin")) {
      propsPatch.plugin = config.plugin;
    }
  
    return {
      propsPatch,
      stylePatch,
      bindingsPatch,
      eventsPatch,
      conditionsPatch,
      hidden,
    };
  };
  
  function buildRefInfo() {
    if (!node.value) return null;
    const dslMethodMap = {
      Input: "input",
      InputNumber: "inputNumber",
      Select: "select",
      Cascader: "cascader",
      Radio: "radio",
      Checkbox: "checkbox",
      Switch: "switch",
      Table: "table",
      BigDataTable: "bigDataTable",
      Tree: "tree",
      Dropdown: "dropdown",
      Menu: "menu",
      Tabs: "tabs",
      Transfer: "transfer",
      Tag: "tag",
      Timeline: "timeline",
      Steps: "steps",
      ImageCarousel: "imageCarousel",
      CarouselComponent: "carouselComponent",
      Card: "card",
      Pagination: "pagination",
      Collapse: "collapse",
      Slider: "slider",
      Calendar: "calendar",
      WebContainer: "webContainer",
      Text: "text",
      ElContainer: "elContainer",
      ElMain: "elMain",
      ElLayout: "elLayout",
      ElLayoutRow: "elLayoutRow",
      ElCol: "elCol",
    };
    const applyCommonDslConfig = (config) => {
      if (!config || typeof config !== "object") return;
      const reservedKeys = new Set([
        "id",
        "label",
        "type",
        "props",
        "state",
        "style",
        "className",
        "events",
        "visible",
        "disabled",
        "loading",
        "permission",
        "onClick",
      ]);
      const rawProps =
        config.props && typeof config.props === "object"
          ? { ...config.props }
          : {};
      Object.entries(config).forEach(([key, value]) => {
        if (reservedKeys.has(key)) return;
        if (rawProps[key] === undefined) {
          rawProps[key] = value;
        }
      });
      if (node.value?.type === "Menu" && Array.isArray(rawProps.items)) {
        // 兼容 command/key 写法，确保 Menu 有可用的 index
        rawProps.items = rawProps.items
          .map((item) => {
            if (!item) return null;
            if (typeof item === "string") {
              return { label: item, index: item };
            }
            if (typeof item !== "object") return null;
            const label = item.label ?? item.title ?? item.name ?? "";
            const index =
              item.index ??
              item.command ??
              item.key ??
              (label ? String(label) : undefined);
            return { ...item, label, index };
          })
          .filter(Boolean);
      }
      if (
        Object.prototype.hasOwnProperty.call(rawProps, "value") &&
        !Object.prototype.hasOwnProperty.call(rawProps, "modelValue")
      ) {
        rawProps.modelValue = rawProps.value;
        delete rawProps.value;
      }
      const nextPatch = {};
      if (Object.keys(rawProps).length > 0) {
        nextPatch.props = { ...(node.value.props || {}), ...rawProps };
      }
      if (config.style && typeof config.style === "object") {
        nextPatch.style = { ...(node.value.style || {}), ...config.style };
      }
      if (typeof config.className === "string" && config.className.trim()) {
        const className = config.className.trim();
        nextPatch.props = {
          ...(nextPatch.props || node.value.props || {}),
          class: className,
        };
      }
      if (
        typeof config.label === "string" &&
        config.label.trim() &&
        !isRunningDetailConfigFn?.()
      ) {
        nextPatch.label = config.label.trim();
      }
      if (typeof config.visible === "boolean") {
        nextPatch.hidden = !config.visible;
      }
      if (typeof config.disabled === "boolean") {
        nextPatch.props = {
          ...(nextPatch.props || node.value.props || {}),
          disabled: config.disabled,
        };
      }
      if (typeof config.loading === "boolean") {
        nextPatch.props = {
          ...(nextPatch.props || node.value.props || {}),
          loading: config.loading,
        };
      }
      if (Object.keys(nextPatch).length === 0) return;
      if (readonly.value) {
        applyPreviewPatch(nextPatch);
        return;
      }
      const ok = editorStore.updateNode(node.value.id, nextPatch);
      if (!ok) {
        applyPreviewPatch(nextPatch);
      }
    };
    /**
     * 规范化下拉菜单项数据
     * @param {Record<string, any>} input - 菜单项数据
     * @returns {Record<string, any> | null} 规范化后的菜单项
     */
    const normalizeDropdownItem = (input) => {
      if (!input || typeof input !== "object") return null;
      const text = input.text ?? input.label ?? "";
      const command = input.command ?? input.value ?? input.key ?? "";
      const disabled = Boolean(input.disabled);
      const divided = Boolean(
        Object.prototype.hasOwnProperty.call(input, "divided")
          ? input.divided
          : input.diveded,
      );
      const icon = typeof input.icon === "string" ? input.icon : "";
      return {
        label: text || String(command ?? ""),
        value: command || String(text ?? ""),
        disabled,
        divided,
        icon,
      };
    };
    /**
     * 获取当前下拉菜单项列表
     * @returns {Array} 菜单项列表
     */
    const getDropdownItems = () => {
      const items = node.value?.props?.items;
      return Array.isArray(items) ? [...items] : [];
    };
    /**
     * 更新下拉菜单项列表
     * @param {Array} nextItems - 新的菜单项列表
     * @returns {void}
     */
    const updateDropdownItems = (nextItems) => {
      if (readonly.value) {
        applyPreviewPatch({ props: { items: nextItems } });
        return;
      }
      editorStore.updateNode(node.value.id, {
        props: { ...(node.value.props || {}), items: nextItems },
      });
    };
    /**
     * 更新组件属性
     * @param {Record<string, any>} patch - 属性补丁
     * @returns {void}
     */
    const updateNodeProps = (patch) => {
      if (!patch || typeof patch !== "object") return;
      if (readonly.value) {
        applyPreviewPatch({ props: patch });
        return;
      }
      editorStore.updateNode(node.value.id, {
        props: { ...(node.value.props || {}), ...patch },
      });
    };
    /**
     * 更新组件样式
     * @param {Record<string, any>} patch - 样式补丁
     * @returns {void}
     */
    const updateNodeStyle = (patch) => {
      if (!patch || typeof patch !== "object") return;
      if (readonly.value) {
        applyPreviewPatch({ style: patch });
        return;
      }
      editorStore.updateNode(node.value.id, {
        style: { ...(node.value.style || {}), ...patch },
      });
    };
    /**
     * 更新组件显隐
     * @param {boolean} visible - 是否显示
     * @returns {void}
     */
    const updateNodeVisibility = (visible) => {
      const hidden = !visible;
      if (readonly.value) {
        applyPreviewPatch({ hidden });
        return;
      }
      editorStore.updateNode(node.value.id, { hidden });
    };
    /**
     * 获取样式数值
     * @param {string} key - 样式字段
     * @returns {number}
     */
    const getStyleNumber = (key) => {
      const raw = node.value?.style?.[key];
      if (typeof raw === "number") return raw;
      if (typeof raw === "string") {
        const parsed = Number.parseFloat(raw);
        return Number.isFinite(parsed) ? parsed : 0;
      }
      return 0;
    };
    /**
     * 应用布局 DSL 配置，或返回当前节点 DOM 引用
     * @param {string} expectedType - 期望组件类型
     * @param {Record<string, any>} [config] - 布局 DSL 配置
     * @returns {HTMLElement | null}
     */
    const applyLayoutDslOrGetElement = (expectedType, config) => {
      if (node.value?.type !== expectedType) return nodeRef.value || null;
      if (config && typeof config === "object") {
        applyCommonDslConfig(config);
      }
      return nodeRef.value || null;
    };
    /**
     * 规范化单选/多选项数据
     * @param {any} input - 选项数据
     * @returns {Record<string, any> | null} 规范化后的选项
     */
    const normalizeChoiceOption = (input) => {
      if (input == null) return null;
      if (typeof input === "object") {
        const label = input.label ?? input.text ?? "";
        const value = input.value ?? input.label ?? input.text ?? "";
        return {
          ...input,
          label,
          value,
        };
      }
      const text = String(input);
      return { label: text, value: text };
    };
    /**
     * 获取单选/多选项列表
     * @returns {Array} 选项列表
     */
    const getChoiceOptions = () => {
      const options = node.value?.props?.options;
      if (!Array.isArray(options)) return [];
      return options.map((item) => normalizeChoiceOption(item)).filter(Boolean);
    };
    /**
     * 更新单选/多选项列表
     * @param {Array} nextOptions - 新的选项列表
     * @returns {void}
     */
    const updateChoiceOptions = (nextOptions) => {
      if (readonly.value) {
        applyPreviewPatch({ props: { options: nextOptions } });
        return;
      }
      editorStore.updateNode(node.value.id, {
        props: { ...(node.value.props || {}), options: nextOptions },
      });
    };
    /**
     * 获取表格数据
     * @returns {Array} 表格数据
     */
    const getTableData = () => {
      const data = node.value?.props?.data;
      return Array.isArray(data) ? [...data] : [];
    };
    /**
     * 更新表格数据
     * @param {Array} nextData - 表格数据
     * @returns {void}
     */
    const updateTableData = (nextData) => {
      if (readonly.value) {
        applyPreviewPatch({ props: { data: nextData } });
        tableRenderVersion.value += 1;
        return;
      }
      editorStore.updateNode(node.value.id, {
        props: { ...(node.value.props || {}), data: nextData },
      });
      tableRenderVersion.value += 1;
    };
    /**
     * 更新树组件数据
     * @param {Array} nextData - 树数据
     * @returns {void}
     */
    const updateTreeData = (nextData) => {
      updateNodeProps({ data: Array.isArray(nextData) ? nextData : [] });
    };
    /**
     * 更新级联选择器数据
     * @param {Array} nextOptions - 级联数据
     * @returns {void}
     */
    const updateCascaderOptions = (nextOptions) => {
      updateNodeProps({ options: Array.isArray(nextOptions) ? nextOptions : [] });
    };
    /**
     * 更新选择器数据
     * @param {Array} nextOptions - 选择器选项
     * @returns {void}
     */
    const updateSelectOptions = (nextOptions) => {
      updateNodeProps({ options: Array.isArray(nextOptions) ? nextOptions : [] });
    };
    /**
     * 更新输入值
     * @param {any} value - 输入值
     * @returns {void}
     */
    const updateInputValue = (value) => {
      updateNodeProps({ modelValue: value });
    };
    /**
     * 更新开关值
     * @param {boolean} value - 开关状态
     * @returns {void}
     */
    const updateSwitchValue = (value) => {
      updateNodeProps({ modelValue: Boolean(value) });
    };
    /**
     * 更新穿梭框数据
     * @param {Array} leftData - 左侧数据
     * @param {Array} rightData - 右侧数据
     * @returns {void}
     */
    const updateTransferData = (leftData, rightData) => {
      updateNodeProps({
        data: Array.isArray(leftData) ? leftData : [],
        modelValue: Array.isArray(rightData) ? rightData : [],
      });
    };
    /**
     * 获取表格行键字段
     * @returns {string}
     */
    const getTableRowKeyProp = () => {
      const propsValue = node.value?.props || {};
      return (
        propsValue.rowKey || propsValue["row-key"] || propsValue.keyField || "id"
      );
    };
    /**
     * 获取表格行键值
     * @param {Record<string, any>} row - 行数据
     * @returns {string|number|undefined}
     */
    const getTableRowKeyValue = (row) => {
      if (!row || typeof row !== "object") return undefined;
      const keyProp = getTableRowKeyProp();
      return row[keyProp];
    };
    /**
     * 获取树节点键字段
     * @returns {string}
     */
    const getTreeNodeKeyProp = () => {
      const propsValue = node.value?.props || {};
      return propsValue.nodeKey || propsValue["node-key"] || "id";
    };
    /**
     * 收集树数据中的节点 key
     * @param {Array} list - 树节点列表
     * @returns {Array<string|number>}
     */
    const collectTreeKeys = (list) => {
      const keys = [];
      const keyProp = getTreeNodeKeyProp();
      const walk = (items) => {
        if (!Array.isArray(items)) return;
        items.forEach((item) => {
          if (!item || typeof item !== "object") return;
          if (Object.prototype.hasOwnProperty.call(item, keyProp)) {
            keys.push(item[keyProp]);
          }
          walk(item.children);
        });
      };
      walk(list);
      return keys;
    };
    const refInfo = {
      get Name() {
        return node.value?.label || "";
      },
      get Comment() {
        const type = node.value?.type || "";
        const manifest = type ? componentRegistry.get(type) : null;
        return manifest?.name || type || "";
      },
      get Location() {
        return {
          get X() {
            return getStyleNumber("left");
          },
          set X(value) {
            const next = Number(value);
            if (!Number.isFinite(next)) return;
            updateNodeStyle({ left: `${next}px` });
          },
          get Y() {
            return getStyleNumber("top");
          },
          set Y(value) {
            const next = Number(value);
            if (!Number.isFinite(next)) return;
            updateNodeStyle({ top: `${next}px` });
          },
        };
      },
      get Size() {
        return {
          get Width() {
            return getStyleNumber("width");
          },
          set Width(value) {
            const next = Number(value);
            if (!Number.isFinite(next)) return;
            updateNodeStyle({ width: `${next}px` });
          },
          get Height() {
            return getStyleNumber("height");
          },
          set Height(value) {
            const next = Number(value);
            if (!Number.isFinite(next)) return;
            updateNodeStyle({ height: `${next}px` });
          },
        };
      },
      get Visible() {
        return !node.value?.hidden;
      },
      set Visible(value) {
        updateNodeVisibility(Boolean(value));
      },
      get Enable() {
        return !node.value?.props?.disabled;
      },
      set Enable(value) {
        updateNodeProps({ disabled: !value });
      },
      get Caption() {
        if (node.value?.type === "Button" || node.value?.type === "Tag") {
          return node.value?.props?.text ?? "";
        }
        return undefined;
      },
      set Caption(value) {
        if (node.value?.type === "Button" || node.value?.type === "Tag") {
          updateNodeProps({ text: String(value ?? "") });
        }
      },
      get Image() {
        if (node.value?.type !== "Image") return undefined;
        return node.value?.props?.src ?? "";
      },
      set Image(value) {
        if (node.value?.type !== "Image") return;
        updateNodeProps({ src: String(value ?? "") });
      },
      name: node.value.label,
      id: node.value.id,
      el: nodeRef.value || null,
      component: contentRef.value || null,
      elContainer: (config) => applyLayoutDslOrGetElement("ElContainer", config),
      elMain: (config) => applyLayoutDslOrGetElement("ElMain", config),
      elLayout: (config) => applyLayoutDslOrGetElement("ElLayout", config),
      elLayoutRow: (config) => applyLayoutDslOrGetElement("ElLayoutRow", config),
      elCol: (config) => applyLayoutDslOrGetElement("ElCol", config),
      node: node.value,
      setProps: (patch) => {
        if (!patch || typeof patch !== "object") return;
        if (readonly.value) {
          applyPreviewPatch({ props: patch });
          return;
        }
        editorStore.updateNode(node.value.id, {
          props: { ...(node.value.props || {}), ...patch },
        });
      },
      echarts: (method, ...args) => {
        if (node.value?.type !== "EChart") return;
        const chartApi = contentRef.value;
        if (chartApi?.callECharts) {
          return chartApi.callECharts(method, ...args);
        }
      },
      setStyle: (patch) => {
        if (!patch || typeof patch !== "object") return;
        if (readonly.value) {
          applyPreviewPatch({ style: patch });
          return;
        }
        editorStore.updateNode(node.value.id, {
          style: { ...(node.value.style || {}), ...patch },
        });
      },
      setOption: (
        option,
        notMergeOrOpts,
        lazyUpdate = false,
        silent = false,
        replaceMerge,
      ) => {
        if (node.value?.type !== "EChart") return;
        if (option && typeof option.then === "function") {
          option.then((resolved) => {
            refInfo.setOption?.(
              resolved,
              notMergeOrOpts,
              lazyUpdate,
              silent,
              replaceMerge,
            );
          });
          return;
        }
        const chartApi = contentRef.value;
        if (chartApi?.setOption) {
          chartApi.setOption(
            option,
            notMergeOrOpts,
            lazyUpdate,
            silent,
            replaceMerge,
          );
        }
        if (readonly.value) {
          applyPreviewPatch({ props: { option } });
          return;
        }
        editorStore.updateNode(node.value.id, {
          props: { ...(node.value.props || {}), option },
        });
      },
      setText: (text) => {
        const value = String(text ?? "");
        if (readonly.value) {
          const propsPatch = { text: value };
          if (
            node.value?.type === "Card" ||
            node.value?.type === "BusinessCard"
          ) {
            propsPatch.content = value;
          }
          applyPreviewPatch({ props: propsPatch });
          return;
        }
        editorStore.updateNode(node.value.id, {
          props: { ...(node.value.props || {}), text: value },
        });
      },
      SetText: (text) => {
        if (
          node.value?.type !== "Text" &&
          node.value?.type !== "Tag" &&
          node.value?.type !== "Button"
        ) {
          return;
        }
        const value = String(text ?? "");
        updateNodeProps({ text: value });
      },
      GetText: () => {
        if (
          node.value?.type !== "Text" &&
          node.value?.type !== "Tag" &&
          node.value?.type !== "Button"
        ) {
          return undefined;
        }
        return node.value?.props?.text ?? "";
      },
      SetType: (type) => {
        if (
          node.value?.type !== "Button" &&
          node.value?.type !== "Tag" &&
          node.value?.type !== "Text"
        ) {
          return;
        }
        updateNodeProps({ type: String(type ?? "") });
      },
      SetEllipsis: (value) => {
        if (node.value?.type !== "Text") return;
        updateNodeProps({ truncate: Boolean(value) });
      },
      SetTooltip: (value) => {
        if (node.value?.type !== "Text") return;
        updateNodeProps({ showTooltip: Boolean(value) });
      },
      SetLoading: (value) => {
        if (
          node.value?.type !== "Button" &&
          node.value?.type !== "Card" &&
          node.value?.type !== "BusinessCard"
        ) {
          return;
        }
        updateNodeProps({ loading: Boolean(value) });
      },
      SetDisabled: (value) => {
        if (node.value?.type !== "Button") return;
        updateNodeProps({ disabled: Boolean(value) });
      },
      Click: () => {
        if (node.value?.type !== "Button") return;
        const el = contentRef.value?.$el || contentRef.value || nodeRef.value;
        if (el?.click) {
          el.click();
          return;
        }
        contentRef.value?.$emit?.("click");
      },
      SetSrc: (src) => {
        if (node.value?.type !== "Image") return;
        updateNodeProps({ src: String(src ?? "") });
      },
      GetSrc: () => {
        if (node.value?.type !== "Image") return undefined;
        return node.value?.props?.src ?? "";
      },
      Preview: (urls, startIndex = 0) => {
        if (node.value?.type !== "Image") return;
        if (Array.isArray(urls) && urls.length > 0) {
          updateNodeProps({
            previewSrcList: urls,
            initialIndex: Number(startIndex) || 0,
          });
        }
        contentRef.value?.showPreview?.();
      },
      Reload: () => {
        if (node.value?.type === "Image") {
          const src = String(node.value?.props?.src || "");
          if (!src) return;
          const url = new URL(src, window.location.href);
          url.searchParams.set("_t", String(Date.now()));
          updateNodeProps({ src: url.toString() });
          return;
        }
        if (node.value?.type === "WebContainer") {
          const src = String(node.value?.props?.url || "");
          if (!src) return;
          const url = new URL(src, window.location.href);
          url.searchParams.set("_t", String(Date.now()));
          updateNodeProps({ url: url.toString() });
        }
      },
      setTableHeader: (columns) => {
        if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
          return;
        }
        if (columns && typeof columns.then === "function") {
          columns.then((resolved) => {
            refInfo.setTableHeader(resolved);
          });
          return;
        }
        let input = columns;
        if (input && typeof input === "object" && !Array.isArray(input)) {
          if (Array.isArray(input.columns)) {
            input = input.columns;
          } else if (Array.isArray(input.data)) {
            input = input.data;
          }
        }
        if (!Array.isArray(input)) return;
        const normalized = input
          .map((item) => {
            if (item && typeof item === "object") {
              const label = item.label ?? item.title ?? item.name ?? item.prop;
              const prop =
                item.prop ?? item.field ?? item.key ?? item.name ?? item.label;
              return { ...item, label, prop };
            }
            if (typeof item === "string") {
              return { label: item, prop: item };
            }
            return null;
          })
          .filter(Boolean);
        if (readonly.value) {
          applyPreviewPatch({ props: { columns: normalized } });
          tableRenderVersion.value += 1;
          return;
        }
        editorStore.updateNode(node.value.id, {
          props: { ...(node.value.props || {}), columns: normalized },
        });
        tableRenderVersion.value += 1;
      },
      setTableData: (data, header) => {
        if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
          return;
        }
        if (!data) return;
        if (data && typeof data.then === "function") {
          data.then((resolved) => {
            refInfo.setTableData(resolved, header);
          });
          return;
        }
        let rows = data;
        let columns = header || null;
        if (rows && typeof rows === "object" && !Array.isArray(rows)) {
          if (Array.isArray(rows.rows)) {
            rows = rows.rows;
          } else if (Array.isArray(rows.data)) {
            rows = rows.data;
          }
          if (Array.isArray(rows.columns)) {
            columns = rows.columns;
          } else if (Array.isArray(data.columns)) {
            columns = data.columns;
          }
        }
        if (!Array.isArray(rows)) return;
        const normalizeColumns = (input) => {
          if (!Array.isArray(input)) return [];
          return input
            .map((item) => {
              if (item && typeof item === "object") {
                const label = item.label ?? item.title ?? item.name ?? item.prop;
                const prop =
                  item.prop ?? item.field ?? item.key ?? item.name ?? item.label;
                return { ...item, label, prop };
              }
              if (typeof item === "string") {
                return { label: item, prop: item };
              }
              return null;
            })
            .filter(Boolean);
        };
        const normalizedColumns = normalizeColumns(
          columns || node.value?.props?.columns || [],
        );
        const normalizedData =
          Array.isArray(rows) && Array.isArray(rows[0])
            ? rows.map((row) => {
              if (!Array.isArray(row)) return row;
              if (normalizedColumns.length === 0) return row;
              const next = {};
              normalizedColumns.forEach((col, index) => {
                const key = col.prop ?? col.label ?? `col${index}`;
                next[key] = row[index];
              });
              return next;
            })
            : rows;
        if (readonly.value) {
          applyPreviewPatch({ props: { data: normalizedData } });
          tableRenderVersion.value += 1;
          return;
        }
        editorStore.updateNode(node.value.id, {
          props: { ...(node.value.props || {}), data: normalizedData },
        });
        tableRenderVersion.value += 1;
      },
      /**
       * 应用按钮 DSL 配置
       * @param {ButtonDslConfig} config - 按钮 DSL 配置
       * @returns {void}
       */
      button: (config) => {
        if (node.value?.type !== "Button") return;
        const patch = buildButtonDslPatch(config);
        const nextPatch = {};
        if (Object.keys(patch.propsPatch).length > 0) {
          nextPatch.props = {
            ...(node.value.props || {}),
            ...patch.propsPatch,
          };
        }
        if (Object.keys(patch.stylePatch).length > 0) {
          nextPatch.style = {
            ...(node.value.style || {}),
            ...patch.stylePatch,
          };
        }
        if (Object.keys(patch.bindingsPatch).length > 0) {
          nextPatch.bindings = {
            ...(node.value.bindings || {}),
            ...patch.bindingsPatch,
          };
        }
        if (Object.keys(patch.eventsPatch).length > 0) {
          nextPatch.events = {
            ...(node.value.events || {}),
            ...patch.eventsPatch,
          };
        }
        if (Object.keys(patch.conditionsPatch).length > 0) {
          nextPatch.conditions = {
            ...(node.value.conditions || {}),
            ...patch.conditionsPatch,
          };
        }
        if (typeof patch.hidden === "boolean") {
          nextPatch.hidden = patch.hidden;
        }
        if (Object.keys(nextPatch).length === 0) return;
        if (readonly.value) {
          applyPreviewPatch(nextPatch);
          return;
        }
        editorStore.updateNode(node.value.id, nextPatch);
      },
      /**
       * 应用 Tabs DSL 配置
       * @param {Record<string, any>} config - Tabs DSL 配置
       * @returns {void}
       */
      tabs: (config) => {
        if (
          node.value?.type !== "Tabs" ||
          !config ||
          typeof config !== "object"
        ) {
          return;
        }
        if (config.type && String(config.type) !== "Tabs") return;
        const nextPatch = {};
        if (
          Object.prototype.hasOwnProperty.call(config, "label") &&
          !isRunningDetailConfigFn?.()
        ) {
          nextPatch.label = String(config.label ?? "");
        }
        const propsPatch = { ...(node.value.props || {}) };
        if (Object.prototype.hasOwnProperty.call(config, "id")) {
          const value = String(config.id ?? "");
          if (value) propsPatch.id = value;
        }
        if (config.className && String(config.className).trim()) {
          propsPatch.class = String(config.className).trim();
        }
        if (config.props && typeof config.props === "object") {
          const rawProps = { ...config.props };
          if (Array.isArray(rawProps.items) && !Array.isArray(rawProps.tabs)) {
            rawProps.tabs = rawProps.items;
            delete rawProps.items;
          }
          Object.assign(propsPatch, rawProps);
        }
        nextPatch.props = propsPatch;
        if (config.style && typeof config.style === "object") {
          nextPatch.style = {
            ...(node.value.style || {}),
            ...config.style,
          };
        }
        if (Object.keys(nextPatch).length === 0) return;
        if (readonly.value) {
          applyPreviewPatch(nextPatch);
          return;
        }
        editorStore.updateNode(node.value.id, nextPatch);
      },
      InsertItem: (item) => {
        if (node.value?.type !== "Dropdown") return;
        const normalized = normalizeDropdownItem(item);
        if (!normalized) return;
        const items = getDropdownItems();
        items.push(normalized);
        updateDropdownItems(items);
      },
      GetCommandItem: (menuItem) => {
        if (node.value?.type !== "Dropdown") return undefined;
        const items = getDropdownItems();
        if (menuItem && typeof menuItem === "object") {
          return menuItem.value ?? menuItem.command ?? menuItem.key;
        }
        const found = items.find(
          (item) =>
            item?.label === menuItem ||
            item?.text === menuItem ||
            item?.value === menuItem,
        );
        return found?.value;
      },
      GetMenuItem: (command) => {
        if (node.value?.type !== "Dropdown") return undefined;
        const items = getDropdownItems();
        return items.find((item) => item?.value === command);
      },
      DeleteItem: (menuItem) => {
        if (node.value?.type !== "Dropdown") return;
        const items = getDropdownItems();
        const index = items.findIndex(
          (item) =>
            item?.label === menuItem ||
            item?.text === menuItem ||
            item?.value === menuItem,
        );
        if (index === -1) return;
        items.splice(index, 1);
        updateDropdownItems(items);
      },
      ClearAll: () => {
        if (node.value?.type === "Dropdown") {
          updateDropdownItems([]);
          return;
        }
        if (node.value?.type === "Cascader") {
          updateCascaderOptions([]);
          updateInputValue([]);
          return;
        }
        if (node.value?.type === "Select") {
          const multiple = Boolean(node.value?.props?.multiple);
          updateInputValue(multiple ? [] : "");
          return;
        }
      },
      UpdateKeyChildren: (key, data) => {
        if (node.value?.type !== "Tree") return;
        contentRef.value?.updateKeyChildren?.(key, data);
      },
      GetCheckedNodes: (leafOnly, includeHalfChecked) => {
        if (node.value?.type === "Tree") {
          return contentRef.value?.getCheckedNodes?.(
            leafOnly,
            includeHalfChecked,
          );
        }
        if (node.value?.type === "Cascader") {
          return contentRef.value?.getCheckedNodes?.(leafOnly);
        }
        if (node.value?.type === "Select") {
          const options = getChoiceOptions();
          const value = node.value?.props?.modelValue;
          if (Array.isArray(value)) {
            return options.filter((item) => value.includes(item.value));
          }
          const hit = options.find((item) => item.value === value);
          return hit ? [hit] : [];
        }
        return undefined;
      },
      SetCheckedNodes: (nodes) => {
        if (node.value?.type === "Tree") {
          contentRef.value?.setCheckedNodes?.(nodes);
          return;
        }
        if (node.value?.type === "Select") {
          const options = getChoiceOptions();
          const target = options.find((item) => item.label === nodes);
          updateNodeProps({ modelValue: target?.value ?? "" });
        }
      },
      GetCheckedKeys: (leafOnly) => {
        if (node.value?.type !== "Tree") return undefined;
        return contentRef.value?.getCheckedKeys?.(leafOnly);
      },
      SetCheckedKeys: (keys, leafOnly) => {
        if (node.value?.type !== "Tree") return;
        contentRef.value?.setCheckedKeys?.(keys, leafOnly);
      },
      SetChecked: (keyOrData, checked, deep) => {
        if (node.value?.type !== "Tree") return;
        contentRef.value?.setChecked?.(keyOrData, checked, deep);
      },
      GetHalfCheckedNodes: () => {
        if (node.value?.type !== "Tree") return undefined;
        return contentRef.value?.getHalfCheckedNodes?.();
      },
      GetHalfCheckedKeys: () => {
        if (node.value?.type !== "Tree") return undefined;
        return contentRef.value?.getHalfCheckedKeys?.();
      },
      GetCurrentKey: () => {
        if (node.value?.type !== "Tree") return undefined;
        return contentRef.value?.getCurrentKey?.();
      },
      GetCurrentNode: () => {
        if (node.value?.type !== "Tree") return undefined;
        return contentRef.value?.getCurrentNode?.();
      },
      SetCurrentKey: (key) => {
        if (node.value?.type !== "Tree") return;
        contentRef.value?.setCurrentKey?.(key);
      },
      SetCurrentNode: (nodeData) => {
        if (node.value?.type !== "Tree") return;
        contentRef.value?.setCurrentNode?.(nodeData);
      },
      GetNode: (dataOrKey) => {
        if (node.value?.type !== "Tree") return undefined;
        return contentRef.value?.getNode?.(dataOrKey);
      },
      Remove: (dataOrNode) => {
        if (node.value?.type !== "Tree") return;
        contentRef.value?.remove?.(dataOrNode);
      },
      Append: (data, parentNode) => {
        if (node.value?.type !== "Tree") return;
        contentRef.value?.append?.(data, parentNode);
      },
      InsertBefore: (data, refNode) => {
        if (node.value?.type !== "Tree") return;
        contentRef.value?.insertBefore?.(data, refNode);
      },
      InsertAfter: (data, refNode) => {
        if (node.value?.type !== "Tree") return;
        contentRef.value?.insertAfter?.(data, refNode);
      },
      ExpandAll: () => {
        if (node.value?.type !== "Tree") return;
        const data = node.value?.props?.data || [];
        const keys = collectTreeKeys(data);
        contentRef.value?.setExpandedKeys?.(keys);
      },
      CollapseAll: () => {
        if (node.value?.type !== "Tree") return;
        contentRef.value?.setExpandedKeys?.([]);
      },
      SetExpandedKeys: (keys) => {
        if (node.value?.type !== "Tree") return;
        contentRef.value?.setExpandedKeys?.(Array.isArray(keys) ? keys : []);
      },
      GetExpandedKeys: () => {
        if (node.value?.type !== "Tree") return [];
        return contentRef.value?.getExpandedKeys?.() || [];
      },
      Filter: (keyword) => {
        if (node.value?.type !== "Tree") return;
        contentRef.value?.filter?.(keyword ?? "");
      },
      Open: (index) => {
        if (node.value?.type === "Menu") {
          contentRef.value?.open?.(index);
          return;
        }
        if (node.value?.type === "Dropdown") {
          contentRef.value?.handleOpen?.();
          updateNodeProps({ visible: true });
          return;
        }
        if (node.value?.type === "Select" || node.value?.type === "Cascader") {
          contentRef.value?.toggleMenu?.(true);
          contentRef.value?.togglePopperVisible?.(true);
          contentRef.value?.focus?.();
          return;
        }
        if (node.value?.type === "Collapse") {
          const names = Array.isArray(index) ? index : [index];
          updateNodeProps({ modelValue: names });
        }
      },
      Close: (index) => {
        if (node.value?.type === "Menu") {
          contentRef.value?.close?.(index);
          return;
        }
        if (node.value?.type === "Dropdown") {
          contentRef.value?.handleClose?.();
          updateNodeProps({ visible: false });
          return;
        }
        if (node.value?.type === "Select" || node.value?.type === "Cascader") {
          contentRef.value?.toggleMenu?.(false);
          contentRef.value?.togglePopperVisible?.(false);
          contentRef.value?.blur?.();
          return;
        }
        if (node.value?.type === "Tag") {
          updateNodeVisibility(false);
          return;
        }
        if (node.value?.type === "Collapse") {
          const current = Array.isArray(node.value?.props?.modelValue)
            ? node.value.props.modelValue
            : [];
          const remove = new Set(Array.isArray(index) ? index : [index]);
          updateNodeProps({
            modelValue: current.filter((name) => !remove.has(name)),
          });
        }
      },
      Toggle: () => {
        if (node.value?.type === "Dropdown") {
          const visible = Boolean(node.value?.props?.visible);
          if (visible) {
            refInfo.Close();
            return;
          }
          refInfo.Open();
          return;
        }
        if (node.value?.type === "Cascader" || node.value?.type === "Select") {
          contentRef.value?.toggleMenu?.();
          contentRef.value?.togglePopperVisible?.();
        }
        if (node.value?.type === "Switch") {
          const current = Boolean(node.value?.props?.modelValue);
          updateSwitchValue(!current);
        }
        if (node.value?.type === "Collapse") {
          const name = node.value?.props?.modelValue?.[0];
          if (name !== undefined) {
            refInfo.Close([name]);
          }
        }
      },
      GetVisible: () => {
        if (node.value?.type !== "Dropdown") return undefined;
        return Boolean(node.value?.props?.visible);
      },
      Focus: () => {
        if (
          node.value?.type !== "Input" &&
          node.value?.type !== "Select" &&
          node.value?.type !== "Cascader" &&
          node.value?.type !== "InputNumber" &&
          node.value?.type !== "Button"
        ) {
          return;
        }
        const el = contentRef.value?.$el || contentRef.value || nodeRef.value;
        el?.focus?.();
        contentRef.value?.focus?.();
      },
      Blur: () => {
        if (
          node.value?.type !== "Input" &&
          node.value?.type !== "Select" &&
          node.value?.type !== "Cascader" &&
          node.value?.type !== "InputNumber" &&
          node.value?.type !== "Button"
        ) {
          return;
        }
        const el = contentRef.value?.$el || contentRef.value || nodeRef.value;
        el?.blur?.();
        contentRef.value?.blur?.();
      },
      Select: () => {
        if (node.value?.type !== "Input" && node.value?.type !== "InputNumber") {
          return;
        }
        contentRef.value?.select?.();
      },
      GetInputValue: () => {
        if (node.value?.type !== "Input" && node.value?.type !== "InputNumber") {
          return undefined;
        }
        return node.value?.props?.modelValue ?? "";
      },
      SetInputValue: (value) => {
        if (node.value?.type === "Input") {
          updateInputValue(String(value ?? ""));
          return;
        }
        if (node.value?.type === "InputNumber") {
          const next = Number(value);
          if (!Number.isFinite(next)) return;
          updateInputValue(next);
        }
      },
      ClearQuery: (area) => {
        if (node.value?.type !== "Transfer") return;
        contentRef.value?.clearQuery?.(area);
      },
      SetValue: (value) => {
        if (node.value?.type === "Switch") {
          updateSwitchValue(value);
          return;
        }
        if (node.value?.type === "Input") {
          updateInputValue(String(value ?? ""));
          return;
        }
        if (node.value?.type === "InputNumber") {
          const next = Number(value);
          if (!Number.isFinite(next)) return;
          updateInputValue(next);
          return;
        }
        if (node.value?.type === "Select" || node.value?.type === "Cascader") {
          updateInputValue(value);
          return;
        }
        if (node.value?.type === "Radio") {
          updateNodeProps({ modelValue: value });
          return;
        }
        if (node.value?.type === "Checkbox") {
          updateNodeProps({ modelValue: Array.isArray(value) ? value : [] });
          return;
        }
        if (node.value?.type === "Slider") {
          const next = Number(value);
          if (!Number.isFinite(next)) return;
          updateNodeProps({ modelValue: next });
          return;
        }
        if (node.value?.type === "Transfer") {
          updateNodeProps({ modelValue: Array.isArray(value) ? value : [] });
          return;
        }
        if (node.value?.type === "Barcode") {
          updateNodeProps({ value: String(value ?? "") });
        }
      },
      GetValue: () => {
        if (node.value?.type === "Switch") {
          return Boolean(node.value?.props?.modelValue);
        }
        if (
          node.value?.type === "Input" ||
          node.value?.type === "InputNumber" ||
          node.value?.type === "Select" ||
          node.value?.type === "Cascader" ||
          node.value?.type === "Radio" ||
          node.value?.type === "Slider"
        ) {
          return node.value?.props?.modelValue ?? "";
        }
        if (node.value?.type === "Checkbox" || node.value?.type === "Transfer") {
          return Array.isArray(node.value?.props?.modelValue)
            ? node.value.props.modelValue
            : [];
        }
        if (node.value?.type === "Barcode") {
          return node.value?.props?.value ?? "";
        }
        return undefined;
      },
      Clear: () => {
        if (
          node.value?.type === "Input" ||
          node.value?.type === "InputNumber" ||
          node.value?.type === "Select" ||
          node.value?.type === "Cascader"
        ) {
          updateInputValue(
            node.value?.type === "Select" && node.value?.props?.multiple
              ? []
              : "",
          );
          return;
        }
        if (node.value?.type === "Radio") {
          updateNodeProps({ modelValue: "" });
          return;
        }
        if (node.value?.type === "Checkbox" || node.value?.type === "Transfer") {
          updateNodeProps({ modelValue: [] });
          return;
        }
        if (node.value?.type === "Timeline") {
          updateNodeProps({ items: [] });
          return;
        }
        if (node.value?.type === "Signature") {
          updateInputValue("");
        }
      },
      ClearSelection: () => {
        if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
          return;
        }
        contentRef.value?.clearSelection?.();
      },
      AppendRow: (row) => {
        if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
          return;
        }
        const data = getTableData();
        data.push(row);
        updateTableData(data);
      },
      ToggleRowSelection: (row, selected) => {
        if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
          return;
        }
        if (contentRef.value?.toggleRowSelection) {
          contentRef.value.toggleRowSelection(row, selected);
        }
      },
      ToggleAllSelection: () => {
        if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
          return;
        }
        contentRef.value?.toggleAllSelection?.();
      },
      ToggleRowExpansion: (row, expanded) => {
        if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
          return;
        }
        if (contentRef.value?.toggleRowExpansion) {
          contentRef.value.toggleRowExpansion(row, expanded);
        }
      },
      SetCurrentRow: (row) => {
        if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
          return;
        }
        contentRef.value?.setCurrentRow?.(row);
      },
      ClearSort: () => {
        if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
          return;
        }
        contentRef.value?.clearSort?.();
      },
      ClearFilter: (columnKeys) => {
        if (node.value?.type === "Table" || node.value?.type === "BigDataTable") {
          if (typeof columnKeys === "undefined") {
            contentRef.value?.clearFilter?.();
            return;
          }
          contentRef.value?.clearFilter?.(columnKeys);
          return;
        }
        if (node.value?.type === "Tree") {
          contentRef.value?.filter?.("");
        }
      },
      Dolayout: () => {
        if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
          return;
        }
        contentRef.value?.doLayout?.();
      },
      Sort: (prop, order) => {
        if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
          return;
        }
        contentRef.value?.sort?.(prop, order);
      },
      GetSelection: () => {
        if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
          return [];
        }
        return contentRef.value?.getSelectionRows?.() || [];
      },
      GetSelectionKeys: () => {
        if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
          return [];
        }
        const selected = contentRef.value?.getSelectionRows?.() || [];
        return selected
          .map((row) => getTableRowKeyValue(row))
          .filter((value) => value !== undefined);
      },
      SetPage: (page) => {
        const next = Number(page);
        if (!Number.isFinite(next)) return;
        if (node.value?.type === "Pagination") {
          updateNodeProps({ currentPage: next });
          return;
        }
        if (node.value?.type === "Table" || node.value?.type === "BigDataTable") {
          updateNodeProps({ currentPage: next, page: next });
        }
      },
      SetPageSize: (size) => {
        const next = Number(size);
        if (!Number.isFinite(next)) return;
        if (node.value?.type === "Pagination") {
          updateNodeProps({ pageSize: next });
          return;
        }
        if (node.value?.type === "Table" || node.value?.type === "BigDataTable") {
          updateNodeProps({ pageSize: next });
        }
      },
      GetPageData: () => {
        if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
          return [];
        }
        const data = getTableData();
        const page =
          Number(node.value?.props?.currentPage ?? node.value?.props?.page) || 1;
        const size = Number(node.value?.props?.pageSize) || data.length || 1;
        const start = Math.max(0, (page - 1) * size);
        return data.slice(start, start + size);
      },
      UpdateRowByKey: (key, patch) => {
        if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
          return;
        }
        const data = getTableData();
        const index = data.findIndex((row) => getTableRowKeyValue(row) === key);
        if (index === -1) return;
        data[index] = { ...data[index], ...(patch || {}) };
        updateTableData(data);
      },
      RemoveRowByKey: (key) => {
        if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
          return;
        }
        const data = getTableData().filter(
          (row) => getTableRowKeyValue(row) !== key,
        );
        updateTableData(data);
      },
      UpsertRowByKey: (key, row) => {
        if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
          return;
        }
        const data = getTableData();
        const index = data.findIndex((item) => getTableRowKeyValue(item) === key);
        if (index === -1) {
          data.push(row);
        } else {
          data[index] = { ...data[index], ...(row || {}) };
        }
        updateTableData(data);
      },
      ScrollToTop: () => {
        if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
          return;
        }
        contentRef.value?.setScrollTop?.(0);
        const wrapper = contentRef.value?.$el?.querySelector?.(
          ".el-scrollbar__wrap",
        );
        if (wrapper) wrapper.scrollTop = 0;
      },
      ScrollToRow: (keyOrRow) => {
        if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
          return;
        }
        if (contentRef.value?.scrollTo) {
          contentRef.value.scrollTo(keyOrRow);
          return;
        }
        const data = getTableData();
        const key =
          typeof keyOrRow === "object" ? getTableRowKeyValue(keyOrRow) : keyOrRow;
        const index = data.findIndex((row) => getTableRowKeyValue(row) === key);
        if (index < 0) return;
        const wrapper = contentRef.value?.$el?.querySelector?.(
          ".el-scrollbar__wrap",
        );
        if (wrapper) {
          wrapper.scrollTop = index * 32;
        }
      },
      DoLayoutSafe: () => {
        if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
          return;
        }
        nextTick(() => {
          contentRef.value?.doLayout?.();
        });
      },
      SetData: (data, rightData) => {
        if (node.value?.type === "Table" || node.value?.type === "BigDataTable") {
          updateTableData(Array.isArray(data) ? data : []);
          return;
        }
        if (node.value?.type === "Cascader") {
          updateCascaderOptions(data);
          return;
        }
        if (node.value?.type === "Select") {
          updateSelectOptions(data);
          return;
        }
        if (node.value?.type === "Transfer") {
          updateTransferData(data, rightData);
          return;
        }
        if (node.value?.type === "Tree") {
          updateTreeData(data);
          return;
        }
        if (node.value?.type === "BusinessCard") {
          updateNodeProps({ data });
        }
      },
      GetData: () => {
        if (node.value?.type === "Table" || node.value?.type === "BigDataTable") {
          return getTableData();
        }
        if (node.value?.type === "Cascader") {
          return node.value?.props?.options ?? [];
        }
        if (node.value?.type === "Select") {
          return node.value?.props?.options ?? [];
        }
        if (node.value?.type === "Transfer") {
          return {
            leftData: node.value?.props?.data ?? [],
            rightData: node.value?.props?.modelValue ?? [],
          };
        }
        if (node.value?.type === "Tree") {
          return node.value?.props?.data ?? [];
        }
        if (node.value?.type === "BusinessCard") {
          return node.value?.props?.data;
        }
        return undefined;
      },
      GetRadioChecked: () => {
        if (node.value?.type !== "Radio") return undefined;
        const options = getChoiceOptions();
        const value = node.value?.props?.modelValue;
        return options.findIndex((item) => item?.value === value);
      },
      GetRadioValue: (labelIndex) => {
        if (node.value?.type !== "Radio") return undefined;
        const options = getChoiceOptions();
        return options?.[Number(labelIndex)]?.value;
      },
      GetRadioLabel: (radioValue) => {
        if (node.value?.type !== "Radio") return undefined;
        const options = getChoiceOptions();
        return options.find((item) => item?.value === radioValue)?.label;
      },
      SetRadioEnable: (labelIndex, enable) => {
        if (node.value?.type !== "Radio") return;
        const options = getChoiceOptions();
        const index = Number(labelIndex);
        if (!options[index]) return;
        options[index] = { ...options[index], disabled: !enable };
        updateChoiceOptions(options);
      },
      GetRadioEnable: (labelIndex) => {
        if (node.value?.type !== "Radio") return undefined;
        const options = getChoiceOptions();
        const index = Number(labelIndex);
        if (!options[index]) return undefined;
        return !options[index].disabled;
      },
      SetRadioVisible: (labelIndex, visible) => {
        if (node.value?.type !== "Radio") return;
        const options = getChoiceOptions();
        const index = Number(labelIndex);
        if (!options[index]) return;
        options[index] = { ...options[index], visible: Boolean(visible) };
        updateChoiceOptions(options);
      },
      GetRadioVisible: (labelIndex) => {
        if (node.value?.type !== "Radio") return undefined;
        const options = getChoiceOptions();
        const index = Number(labelIndex);
        if (!options[index]) return undefined;
        return options[index].visible !== false;
      },
      GetCheckState: (labelIndex) => {
        if (node.value?.type !== "Checkbox") return undefined;
        const options = getChoiceOptions();
        const index = Number(labelIndex);
        const option = options[index];
        const values = Array.isArray(node.value?.props?.modelValue)
          ? node.value.props.modelValue
          : [];
        if (!option) return undefined;
        return values.includes(option.value);
      },
      SetCheckState: (labelIndex, state) => {
        if (node.value?.type !== "Checkbox") return;
        const options = getChoiceOptions();
        const index = Number(labelIndex);
        const option = options[index];
        if (!option) return;
        const values = Array.isArray(node.value?.props?.modelValue)
          ? [...node.value.props.modelValue]
          : [];
        const exists = values.includes(option.value);
        if (Boolean(state) && !exists) values.push(option.value);
        if (!state && exists) {
          const nextValues = values.filter((value) => value !== option.value);
          if (readonly.value) {
            applyPreviewPatch({ props: { modelValue: nextValues } });
            return;
          }
          editorStore.updateNode(node.value.id, {
            props: { ...(node.value.props || {}), modelValue: nextValues },
          });
          return;
        }
        if (readonly.value) {
          applyPreviewPatch({ props: { modelValue: values } });
          return;
        }
        editorStore.updateNode(node.value.id, {
          props: { ...(node.value.props || {}), modelValue: values },
        });
      },
      SetCheckEnable: (labelIndex, enable) => {
        if (node.value?.type !== "Checkbox") return;
        const options = getChoiceOptions();
        const index = Number(labelIndex);
        if (!options[index]) return;
        options[index] = { ...options[index], disabled: !enable };
        updateChoiceOptions(options);
      },
      GetCheckEnable: (labelIndex) => {
        if (node.value?.type !== "Checkbox") return undefined;
        const options = getChoiceOptions();
        const index = Number(labelIndex);
        if (!options[index]) return undefined;
        return !options[index].disabled;
      },
      SetCheckVisible: (labelIndex, visible) => {
        if (node.value?.type !== "Checkbox") return;
        const options = getChoiceOptions();
        const index = Number(labelIndex);
        if (!options[index]) return;
        options[index] = { ...options[index], visible: Boolean(visible) };
        updateChoiceOptions(options);
      },
      GetCheckVisible: (labelIndex) => {
        if (node.value?.type !== "Checkbox") return undefined;
        const options = getChoiceOptions();
        const index = Number(labelIndex);
        if (!options[index]) return undefined;
        return options[index].visible !== false;
      },
      CheckAll: (value) => {
        if (node.value?.type !== "Checkbox") return;
        const options = getChoiceOptions();
        const next = value ? options.map((item) => item.value) : [];
        updateNodeProps({ modelValue: next });
      },
      SetActive: (name) => {
        if (node.value?.type === "Menu") {
          updateNodeProps({ defaultActive: String(name ?? "") });
          return;
        }
        if (node.value?.type === "Tabs") {
          updateNodeProps({ activeName: String(name ?? "") });
          return;
        }
        if (node.value?.type === "Steps") {
          const next = Number(name);
          if (!Number.isFinite(next)) return;
          updateNodeProps({ active: next });
        }
      },
      GetActive: () => {
        if (node.value?.type === "Menu") {
          return node.value?.props?.defaultActive ?? "";
        }
        if (node.value?.type === "Tabs") {
          return node.value?.props?.activeName ?? "";
        }
        return undefined;
      },
      Collapse: (value) => {
        if (node.value?.type === "Menu") {
          updateNodeProps({ collapse: Boolean(value) });
          return;
        }
        if (node.value?.type === "Card") {
          updateNodeProps({ collapsed: Boolean(value) });
        }
      },
      Next: () => {
        if (node.value?.type === "Tabs") {
          const tabs = Array.isArray(node.value?.props?.tabs)
            ? node.value.props.tabs
            : [];
          const current = node.value?.props?.activeName;
          const index = tabs.findIndex((item) => item.name === current);
          const next = tabs[index + 1] || tabs[0];
          if (next?.name) updateNodeProps({ activeName: next.name });
          return;
        }
        if (node.value?.type === "Steps") {
          const active = Number(node.value?.props?.active) || 0;
          updateNodeProps({ active: active + 1 });
          return;
        }
        if (
          node.value?.type === "ImageCarousel" ||
          node.value?.type === "CarouselComponent"
        ) {
          contentRef.value?.next?.();
        }
      },
      Prev: () => {
        if (node.value?.type === "Tabs") {
          const tabs = Array.isArray(node.value?.props?.tabs)
            ? node.value.props.tabs
            : [];
          const current = node.value?.props?.activeName;
          const index = tabs.findIndex((item) => item.name === current);
          const prev = tabs[index - 1] || tabs[tabs.length - 1];
          if (prev?.name) updateNodeProps({ activeName: prev.name });
          return;
        }
        if (node.value?.type === "Steps") {
          const active = Number(node.value?.props?.active) || 0;
          updateNodeProps({ active: Math.max(0, active - 1) });
          return;
        }
        if (
          node.value?.type === "ImageCarousel" ||
          node.value?.type === "CarouselComponent"
        ) {
          contentRef.value?.prev?.();
        }
      },
      AddTab: (tab) => {
        if (node.value?.type !== "Tabs") return;
        const tabs = Array.isArray(node.value?.props?.tabs)
          ? [...node.value.props.tabs]
          : [];
        tabs.push(tab);
        updateNodeProps({ tabs });
      },
      RemoveTab: (name) => {
        if (node.value?.type !== "Tabs") return;
        const tabs = Array.isArray(node.value?.props?.tabs)
          ? node.value.props.tabs.filter((item) => item.name !== name)
          : [];
        updateNodeProps({ tabs });
      },
      SetSelectionByKeys: (keys) => {
        if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
          return;
        }
        const data = getTableData();
        const keySet = new Set(Array.isArray(keys) ? keys : []);
        contentRef.value?.clearSelection?.();
        data.forEach((row) => {
          const rowKey = getTableRowKeyValue(row);
          if (keySet.has(rowKey)) {
            contentRef.value?.toggleRowSelection?.(row, true);
          }
        });
      },
      MoveToRight: (keys) => {
        if (node.value?.type !== "Transfer") return;
        const current = Array.isArray(node.value?.props?.modelValue)
          ? [...node.value.props.modelValue]
          : [];
        const nextKeys = Array.isArray(keys) ? keys : [];
        nextKeys.forEach((key) => {
          if (!current.includes(key)) current.push(key);
        });
        updateNodeProps({ modelValue: current });
      },
      MoveToLeft: (keys) => {
        if (node.value?.type !== "Transfer") return;
        const current = Array.isArray(node.value?.props?.modelValue)
          ? [...node.value.props.modelValue]
          : [];
        const remove = new Set(Array.isArray(keys) ? keys : []);
        const next = current.filter((key) => !remove.has(key));
        updateNodeProps({ modelValue: next });
      },
      Increase: (step) => {
        if (node.value?.type !== "InputNumber") return;
        const current = Number(node.value?.props?.modelValue) || 0;
        const delta = Number(step ?? node.value?.props?.step ?? 1);
        updateInputValue(current + (Number.isFinite(delta) ? delta : 1));
      },
      Decrease: (step) => {
        if (node.value?.type !== "InputNumber") return;
        const current = Number(node.value?.props?.modelValue) || 0;
        const delta = Number(step ?? node.value?.props?.step ?? 1);
        updateInputValue(current - (Number.isFinite(delta) ? delta : 1));
      },
      SetItems: (items) => {
        if (
          node.value?.type !== "Timeline" &&
          node.value?.type !== "ImageCarousel" &&
          node.value?.type !== "CarouselComponent" &&
          node.value?.type !== "Steps"
        ) {
          return;
        }
        updateNodeProps({ items: Array.isArray(items) ? items : [] });
      },
      AppendItem: (item) => {
        if (node.value?.type !== "Timeline") return;
        const items = Array.isArray(node.value?.props?.items)
          ? [...node.value.props.items]
          : [];
        items.push(item);
        updateNodeProps({ items });
      },
      Play: () => {
        if (
          node.value?.type !== "ImageCarousel" &&
          node.value?.type !== "CarouselComponent"
        ) {
          return;
        }
        updateNodeProps({ autoplay: true });
      },
      Pause: () => {
        if (
          node.value?.type !== "ImageCarousel" &&
          node.value?.type !== "CarouselComponent"
        ) {
          return;
        }
        updateNodeProps({ autoplay: false });
      },
      SetActiveItem: (nameOrIndex) => {
        if (
          node.value?.type !== "ImageCarousel" &&
          node.value?.type !== "CarouselComponent"
        ) {
          return;
        }
        contentRef.value?.setActiveItem?.(nameOrIndex);
      },
      Load: (url) => {
        if (node.value?.type !== "WebContainer") return;
        updateNodeProps({ url: String(url ?? "") });
      },
      PostMessage: (_data) => {
        if (node.value?.type !== "WebContainer") return;
      },
      GetUrl: () => {
        if (node.value?.type !== "WebContainer") return undefined;
        return node.value?.props?.url ?? "";
      },
      Back: () => {
        if (node.value?.type !== "WebContainer") return;
      },
      Forward: () => {
        if (node.value?.type !== "WebContainer") return;
      },
      Reset: () => {
        if (node.value?.type === "Steps") {
          updateNodeProps({ active: 1 });
          return;
        }
        if (node.value?.type === "Pagination") {
          updateNodeProps({ currentPage: 1 });
          return;
        }
        if (node.value?.type === "Slider") {
          const min = Number(node.value?.props?.min);
          updateNodeProps({ modelValue: Number.isFinite(min) ? min : 0 });
        }
      },
      SetTitle: (title) => {
        if (node.value?.type !== "Card" && node.value?.type !== "BusinessCard") {
          return;
        }
        updateNodeProps({ title: String(title ?? "") });
      },
      GetPage: () => {
        if (node.value?.type !== "Pagination") return undefined;
        return Number(node.value?.props?.currentPage) || 1;
      },
      GetPageSize: () => {
        if (node.value?.type !== "Pagination") return undefined;
        return Number(node.value?.props?.pageSize) || 0;
      },
      SetTotal: (total) => {
        if (node.value?.type !== "Pagination") return;
        const next = Number(total);
        if (!Number.isFinite(next)) return;
        updateNodeProps({ total: next });
      },
      GetTotal: () => {
        if (node.value?.type !== "Pagination") return undefined;
        return Number(node.value?.props?.total) || 0;
      },
      GetActiveNames: () => {
        if (node.value?.type !== "Collapse") return [];
        return Array.isArray(node.value?.props?.modelValue)
          ? node.value.props.modelValue
          : [];
      },
      SetActiveNames: (names) => {
        if (node.value?.type !== "Collapse") return;
        updateNodeProps({ modelValue: Array.isArray(names) ? names : [] });
      },
      Refresh: () => {
        if (node.value?.type !== "BusinessCard") return;
        updateNodeProps({ refreshAt: Date.now() });
      },
      OpenDetail: (_id) => {
        if (node.value?.type !== "BusinessCard") return;
      },
      Render: () => {
        if (node.value?.type !== "Barcode") return;
      },
      Download: (_format) => {
        if (node.value?.type !== "Barcode") return undefined;
        return undefined;
      },
      Disable: (value) => {
        if (node.value?.type !== "Switch" && node.value?.type !== "Slider") {
          return;
        }
        updateNodeProps({ disabled: Boolean(value) });
      },
      SetDate: (date) => {
        if (node.value?.type !== "Calendar") return;
        updateNodeProps({ date });
      },
      GetDate: () => {
        if (node.value?.type !== "Calendar") return undefined;
        return node.value?.props?.date ?? null;
      },
      Today: () => {
        if (node.value?.type !== "Calendar") return;
        updateNodeProps({ date: new Date() });
      },
      ClearSignature: () => {
        if (node.value?.type !== "Signature") return;
        updateInputValue("");
      },
      GetImage: () => {
        if (node.value?.type !== "Signature") return undefined;
        return node.value?.props?.modelValue ?? "";
      },
      SetImage: (value) => {
        if (node.value?.type !== "Signature") return;
        updateInputValue(String(value ?? ""));
      },
      IsEmpty: () => {
        if (node.value?.type !== "Signature") return true;
        const value = node.value?.props?.modelValue;
        return !value;
      },
      SetPen: (_color, _width) => {
        if (node.value?.type !== "Signature") return;
      },
    };
    const currentType = node.value.type;
    const dslMethodName = dslMethodMap[currentType];
    if (
      dslMethodName &&
      !Object.prototype.hasOwnProperty.call(refInfo, dslMethodName)
    ) {
      refInfo[dslMethodName] = (config) => {
        if (node.value?.type !== currentType) return;
        applyCommonDslConfig(config);
      };
    }
    return refInfo;
  }

  return { buildRefInfo, applyPreviewPatch };
}
