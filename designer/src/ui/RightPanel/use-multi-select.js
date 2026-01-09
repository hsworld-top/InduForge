/**
 * 多选属性计算 Composable
 * 计算多选元素的共同属性值，支持批量修改
 */

import { computed, toRef } from "vue";
import { storeToRefs } from "pinia";
import { useEditorStore } from "@/stores/editor-store";

/**
 * 多选属性值类型
 * @typedef {{ type: 'same', value: any } | { type: 'mixed', values: any[] }} MultiSelectValue
 */

/**
 * 获取对象的嵌套属性值
 * @param {Object} obj - 对象
 * @param {string} path - 属性路径，如 'style.opacity'
 * @returns {any} 属性值
 */
function getNestedValue(obj, path) {
  if (!obj || !path) return undefined;
  const keys = path.split(".");
  let value = obj;
  for (const key of keys) {
    if (value === null || value === undefined) return undefined;
    value = value[key];
  }
  return value;
}

/**
 * 设置对象的嵌套属性值
 * @param {Object} obj - 对象
 * @param {string} path - 属性路径
 * @param {any} value - 属性值
 * @returns {Object} 新对象
 */
function setNestedValue(obj, path, value) {
  if (!path) return obj;
  const keys = path.split(".");
  const result = { ...obj };
  let current = result;

  for (let i = 0; i < keys.length - 1; i++) {
    const key = keys[i];
    current[key] = current[key] ? { ...current[key] } : {};
    current = current[key];
  }

  current[keys[keys.length - 1]] = value;
  return result;
}

/**
 * 计算多选元素的共同属性值
 * @param {Array<Object>} elements - 元素列表
 * @param {string} path - 属性路径
 * @returns {MultiSelectValue} 多选属性值
 */
export function getMultiSelectValueFromElements(elements, path) {
  if (!elements || elements.length === 0) {
    return { type: "same", value: undefined };
  }

  const values = elements.map((el) => getNestedValue(el, path));
  const uniqueValues = [];
  const seen = new Set();

  for (const val of values) {
    const key = JSON.stringify(val);
    if (!seen.has(key)) {
      seen.add(key);
      uniqueValues.push(val);
    }
  }

  if (uniqueValues.length === 1) {
    return { type: "same", value: uniqueValues[0] };
  }

  return { type: "mixed", values };
}

/**
 * 多选 Composable
 * @param {{ elements: Array<Object> }} props - 组件 props
 * @returns {{
 *   selectedCount: import('vue').ComputedRef<number>,
 *   getMultiSelectValue: (path: string) => MultiSelectValue,
 *   setUnifiedValue: (path: string, value: any) => void
 * }}
 */
export function useMultiSelect(props) {
  const editorStore = useEditorStore();
  const { doc, history } = storeToRefs(editorStore);

  const elementsRef = toRef(props, "elements");

  /**
   * 选中元素数量
   */
  const selectedCount = computed(() => elementsRef.value?.length || 0);

  /**
   * 获取多选属性值
   * @param {string} path - 属性路径
   * @returns {MultiSelectValue}
   */
  const getMultiSelectValue = (path) => {
    if (!elementsRef.value?.length || !doc.value) {
      return { type: "same", value: undefined };
    }

    // 从 doc 中获取完整的元素数据
    const fullElements = elementsRef.value.map((el) => {
      if (el.kind === "node") {
        return doc.value.getNode?.(el.id);
      } else if (el.kind === "graphic") {
        return doc.value.getGraphic?.(el.id);
      }
      return el;
    }).filter(Boolean);

    return getMultiSelectValueFromElements(fullElements, path);
  };

  /**
   * 批量设置属性值
   * @param {string} path - 属性路径
   * @param {any} value - 新值
   */
  const setUnifiedValue = (path, value) => {
    if (!elementsRef.value?.length || !doc.value || !history.value) return;

    // 使用批量操作
    history.value.startBatch?.();

    try {
      for (const el of elementsRef.value) {
        if (el.kind === "node") {
          const node = doc.value.getNode?.(el.id);
          if (node) {
            const patch = buildPatch(path, value);
            // 使用 UpdateNodeCommand 更新节点
            const UpdateNodeCommand = editorStore.doc?.constructor?.UpdateNodeCommand;
            if (UpdateNodeCommand) {
              history.value.execute(new UpdateNodeCommand(el.id, patch));
            }
          }
        } else if (el.kind === "graphic") {
          const graphic = doc.value.getGraphic?.(el.id);
          if (graphic) {
            const patch = buildPatch(path, value);
            // 使用 UpdateGraphicCommand 更新图形
            const UpdateGraphicCommand = editorStore.doc?.constructor?.UpdateGraphicCommand;
            if (UpdateGraphicCommand) {
              history.value.execute(new UpdateGraphicCommand(el.id, patch));
            }
          }
        }
      }
    } finally {
      history.value.endBatch?.();
    }
  };

  /**
   * 构建更新补丁
   * @param {string} path - 属性路径
   * @param {any} value - 属性值
   * @returns {Object} 补丁对象
   */
  const buildPatch = (path, value) => {
    return setNestedValue({}, path, value);
  };

  return {
    selectedCount,
    getMultiSelectValue,
    setUnifiedValue,
  };
}

export default useMultiSelect;
