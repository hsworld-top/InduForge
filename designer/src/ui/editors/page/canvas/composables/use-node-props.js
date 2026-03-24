/**
 * 节点属性解析 Composable
 *
 * 从 NodeRenderer 抽取的 resolvedProps、resolvedNodeProps、表达式绑定、变量上下文等逻辑。
 * 供递归渲染器复用。
 *
 * @module ui/Canvas/composables/use-node-props
 */

import { computed } from "vue";
import { getPropsFilter } from "@/components/descriptors/registry";
import { evaluate, evaluateTemplate } from "@/data";
import { getPreviewRuntime } from "@/ui/editors/page/preview/previewRuntime";
import { buildVarValuesFromDefinitions } from "@/editor-core/utils/variable-utils";

/**
 * 构建表达式上下文
 * @param {Object} propsValue - 当前组件属性
 * @param {Object} deps - 依赖（doc, currentPage, projectVariables）
 * @returns {import("@/data").ExpressionContext}
 */
export function buildExpressionContext(propsValue = {}, deps) {
  const { doc, currentPage, projectVariables } = deps;
  const pageId = currentPage.value?.id;
  const pageDefs = doc.value?.vars?.pages?.[pageId] || {};
  const globalDefs = projectVariables.value || {};
  const runtimeGlobals = getPreviewRuntime()?.globals;
  const globalValues =
    runtimeGlobals && typeof runtimeGlobals === "object"
      ? runtimeGlobals
      : buildVarValuesFromDefinitions(globalDefs);
  return {
    $dp: {},
    $vars: buildVarValuesFromDefinitions(pageDefs),
    $global: globalValues,
    $props: propsValue,
    state: globalValues,
  };
}

/**
 * 解析表达式值
 * @param {string} expr - 表达式
 * @param {import("@/data").ExpressionContext} context - 上下文
 * @param {*} fallback - 兜底值
 * @returns {*}
 */
export function resolveExpressionValue(expr, context, fallback) {
  if (typeof expr !== "string" || !expr.trim()) return fallback;
  const text = expr.trim();
  const value = text.includes("{{")
    ? evaluateTemplate(text, context)
    : evaluate(text, context);
  return value === undefined ? fallback : value;
}

/**
 * 解析表达式绑定
 * @param {Record<string, any>} bindings - 绑定配置
 * @param {Record<string, any>} propsValue - 当前组件属性
 * @param {Object} deps - 依赖（doc, currentPage, projectVariables）
 * @returns {Record<string, any>}
 */
function resolveExprBindings(bindings, propsValue, deps) {
  if (!bindings || typeof bindings !== "object") return {};
  const context = buildExpressionContext(propsValue, deps);
  const resolved = {};
  Object.entries(bindings).forEach(([key, binding]) => {
    if (!binding || typeof binding !== "object") return;
    if (binding.kind !== "expr" || typeof binding.expr !== "string") return;
    const value = resolveExpressionValue(
      binding.expr,
      context,
      binding.fallback,
    );
    if (value !== undefined) {
      resolved[key] = value;
    }
  });
  return resolved;
}

/**
 * 创建节点属性解析器
 * @param {Object} deps - 依赖
 * @param {import('vue').ComputedRef<Object>} deps.node - 节点 computed
 * @param {import('vue').ComputedRef<Object>} deps.doc - 文档 computed
 * @param {import('vue').ComputedRef<Object>} deps.currentPage - 当前页面 computed
 * @param {import('vue').ComputedRef<Object>} deps.projectVariables - 项目变量 computed
 * @param {import('vue').ComputedRef<number>} deps.docVersion - 文档版本 computed
 * @param {import('vue').ComputedRef<boolean>} deps.readonly - 只读模式 computed
 * @returns {Object} 返回 resolvedNodeProps、resolvedProps、filteredProps
 */
export function useNodeProps(deps) {
  const {
    node,
    doc,
    currentPage,
    projectVariables,
    docVersion,
    readonly,
  } = deps;

  // 表达式绑定依赖
  const exprDeps = { doc, currentPage, projectVariables };

  /**
   * 解析后的节点属性（应用表达式绑定）
   */
  const resolvedNodeProps = computed(() => {
    docVersion.value;
    if (!node.value) return {};
    const baseProps = node.value.props || {};
    const bindingValues = resolveExprBindings(
      node.value.bindings,
      baseProps,
      exprDeps,
    );
    if (Object.keys(bindingValues).length === 0) return baseProps;
    return { ...baseProps, ...bindingValues };
  });

  /**
   * 解析后的 props（包含 ElCol/ElLayoutRow 特化逻辑）
   * 注意：Tabs modelValue 处理已移到 useNodeContent 中，避免循环依赖
   */
  const resolvedProps = computed(() => {
    docVersion.value;
    if (!node.value) return {};
    // 直接操作 resolvedNodeProps，不再调用 filterRenderProps（已清空为浅拷贝）
    const nextProps = { ...(resolvedNodeProps.value || {}) };
    if (node.value.type === "ElCol") {
      const parentNode = doc.value?.getParent?.(node.value.id);
      const rawColumns = Number(parentNode?.props?.columns);
      if (
        parentNode?.type === "ElLayoutRow" &&
        Number.isFinite(rawColumns) &&
        !Object.prototype.hasOwnProperty.call(nextProps, "span")
      ) {
        const columns = Math.max(1, Math.min(24, rawColumns));
        const colIds = (parentNode.children || []).filter((childId) => {
          const childNode = doc.value?.getNode?.(childId);
          return childNode?.type === "ElCol";
        });
        const colIndex = Math.max(0, colIds.indexOf(node.value.id));
        const offsets = colIds.map((colId) => {
          const colNode = doc.value?.getNode?.(colId);
          const offset = Number(colNode?.props?.offset) || 0;
          return Math.max(0, Math.min(24, offset));
        });
        // 按剩余格数等分列宽，避免只压缩右侧区域
        const totalOffset = offsets.reduce((sum, value) => sum + value, 0);
        const remainingUnits = Math.max(columns, 24 - totalOffset);
        const base = Math.floor(remainingUnits / columns);
        const rem = remainingUnits - base * columns;
        const span = base + (colIndex < rem ? 1 : 0);
        nextProps.span = Math.max(1, span);
      }
    }
    if (readonly.value && node.value.type === "ElLayoutRow") {
      nextProps.gutter = 0;
    }
    return nextProps;
  });

  /**
   * 过滤后的 props（应用 descriptor.propsFilter）
   * 注意：必须基于 resolvedProps 计算，以包含布局派生值（ElCol span、ElLayoutRow gutter 等）
   */
  const filteredProps = computed(() => {
    if (!node.value) return {};
    // 优先从 descriptor 读取 propsFilter（新架构组件）
    // 基于 resolvedProps 而非 resolvedNodeProps，确保布局派生值被包含
    const descriptorFiltered = getPropsFilter(
      node.value.type,
      resolvedProps.value || {},
    );
    // 如果 descriptor 返回了过滤后的对象，使用它
    if (descriptorFiltered) {
      return descriptorFiltered;
    }
    // fallback: 未注册 propsFilter 的组件直接返回 resolvedProps（包含布局派生值）
    return resolvedProps.value || {};
  });

  return {
    resolvedNodeProps,
    resolvedProps,
    filteredProps,
  };
}

export default { useNodeProps };
