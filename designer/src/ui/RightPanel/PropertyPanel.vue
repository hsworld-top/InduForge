<template>
  <div class="flex flex-col gap-3">
    <!-- 页面设置面板：未选中任何元素 -->
    <PageInspectorPanel v-if="panelState === 'page'" />

    <!-- 多选面板：选中多个元素 -->
    <MultiInspectorPanel
      v-else-if="panelState === 'multi'"
      :elements="selectedElements"
    />

    <!-- 单选面板：选中单个元素 -->
    <div v-else class="element-inspector">
      <!-- 基础信息 -->
      <el-descriptions :column="1" size="small" border>
        <el-descriptions-item label="ID">
          {{ elementId }}
        </el-descriptions-item>
        <el-descriptions-item label="类型">
          {{ elementType }}
        </el-descriptions-item>
        <el-descriptions-item label="名称">
          <el-input
            v-model="elementLabel"
            size="small"
            placeholder="未命名"
            @change="handleLabelChange"
          />
        </el-descriptions-item>
      </el-descriptions>

      <el-divider />

      <!-- 属性表单：根据 Manifest 生成 -->
      <template v-if="manifest && manifest.props.length > 0">
        <el-collapse v-model="activeGroupNames">
          <el-collapse-item
            v-for="group in groupedProps"
            :key="group.name"
            :title="group.name"
            :name="group.name"
          >
            <div class="prop-list">
              <div
                v-for="propDef in group.props"
                :key="propDef.name"
                class="prop-item"
              >
                <div class="prop-label">
                  <span>{{ propDef.label }}</span>
                  <el-tooltip content="绑定数据" placement="top">
                    <el-button
                      size="small"
                      text
                      class="bind-btn"
                      @click="handleBindClick(propDef.name)"
                    >
                      <IconEpLink />
                    </el-button>
                  </el-tooltip>
                </div>
                <PropEditor
                  :prop="propDef"
                  :model-value="getPropValue(propDef.name)"
                  @update:model-value="
                    (val) => handlePropChange(propDef.name, val)
                  "
                />
              </div>
            </div>
          </el-collapse-item>
        </el-collapse>
      </template>

      <!-- 无 Manifest 时显示原始 Props -->
      <template v-else>
        <div class="text-xs text-gray-500 mb-2">Props</div>
        <pre
          class="text-xs bg-gray-50 dark:bg-gray-900 p-2 rounded overflow-auto max-h-60"
          >{{ formattedProps }}</pre
        >
      </template>
    </div>
  </div>
</template>

<script setup>
/**
 * 属性面板
 * 根据选中状态显示不同的面板：
 * - 未选中：Page Inspector（页面设置）
 * - 单选：Element Inspector（元素属性）
 * - 多选：Multi Inspector（批量编辑）
 */

import { computed, ref, watch } from "vue";
import PageInspectorPanel from "./PageInspectorPanel.vue";
import MultiInspectorPanel from "./MultiInspectorPanel.vue";
import PropEditor from "./PropEditor.vue";
import { usePanelState } from "./use-panel-state";
import { getManifest } from "@/manifests";
import { useEditorStore } from "@/stores/editor-store";
import { ElMessage } from "element-plus";
import IconEpLink from "~icons/ep/link";

const editorStore = useEditorStore();

/** 当前展开的分组 */
const activeGroupNames = ref([]);

const { panelState, selectedElements, selectedNode, selectedGraphic } =
  usePanelState();

/** 当前选中的元素 */
const currentElement = computed(
  () => selectedNode.value || selectedGraphic.value
);

/** 元素 ID */
const elementId = computed(() => currentElement.value?.id || "-");

/** 元素类型 */
const elementType = computed(() => currentElement.value?.type || "-");

/** 元素名称（可编辑） */
const elementLabel = ref("");

// 监听选中元素变化，更新名称
watch(
  currentElement,
  (el) => {
    elementLabel.value = el?.label || "";
  },
  { immediate: true }
);

/**
 * 获取组件 Manifest
 */
const manifest = computed(() => {
  const type = currentElement.value?.type;
  if (!type) return null;
  return getManifest(type);
});

/**
 * 按分组整理属性
 */
const groupedProps = computed(() => {
  if (!manifest.value) return [];

  const groups = new Map();
  for (const prop of manifest.value.props) {
    const groupName = prop.group || "基础";
    if (!groups.has(groupName)) {
      groups.set(groupName, { name: groupName, props: [] });
    }
    groups.get(groupName).props.push(prop);
  }
  const result = Array.from(groups.values());

  // 初始化展开所有分组
  if (result.length > 0 && activeGroupNames.value.length === 0) {
    activeGroupNames.value = result.map((g) => g.name);
  }

  return result;
});

/**
 * 获取属性值
 * @param {string} propName - 属性名
 */
const getPropValue = (propName) => {
  const el = currentElement.value;
  if (!el?.props) return undefined;
  return el.props[propName];
};

/**
 * Props 格式化展示（无 Manifest 时使用）
 */
const formattedProps = computed(() => {
  const target = currentElement.value;
  if (!target) return "{}";
  return JSON.stringify(target.props || {}, null, 2);
});

/**
 * 处理名称变更
 */
const handleLabelChange = () => {
  const el = currentElement.value;
  if (!el) return;

  const nextLabel = (elementLabel.value || "").trim();
  if (!nextLabel) {
    elementLabel.value = el.label || "";
    return;
  }
  if (!editorStore.isLabelUnique?.(nextLabel, el.id)) {
    ElMessage.warning("组件名称已存在，请更换");
    elementLabel.value = el.label || "";
    return;
  }

  if (selectedNode.value) {
    editorStore.updateNode(el.id, { label: nextLabel });
  } else if (selectedGraphic.value) {
    editorStore.updateGraphic(el.id, { label: nextLabel });
  }
};

/**
 * 处理属性变更
 * @param {string} propName - 属性名
 * @param {any} value - 新值
 */
const handlePropChange = (propName, value) => {
  const el = currentElement.value;
  if (!el) return;

  const newProps = { ...el.props, [propName]: value };

  if (selectedNode.value) {
    editorStore.updateNode(el.id, { props: newProps });
  } else if (selectedGraphic.value) {
    editorStore.updateGraphic(el.id, { props: newProps });
  }
};

/**
 * 处理绑定按钮点击
 * @param {string} propName - 属性名
 */
const handleBindClick = (propName) => {
  // 切换到绑定面板，传递属性路径
  // TODO: 通过事件或 store 通知父组件切换面板
  console.log("绑定属性:", propName);
};
</script>

<style scoped>
.element-inspector {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.element-inspector :deep(.el-collapse) {
  border: none;
}

.element-inspector :deep(.el-collapse-item__header) {
  font-size: 13px;
  font-weight: 500;
  background: var(--el-fill-color-lighter);
  padding: 0 12px;
  border-radius: 4px;
}

.element-inspector :deep(.el-collapse-item__wrap) {
  border: none;
}

.element-inspector :deep(.el-collapse-item__content) {
  padding: 12px 8px;
}

.prop-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.prop-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.prop-label {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 12px;
  color: var(--el-text-color-regular);
}

.bind-btn {
  padding: 2px;
  height: auto;
  opacity: 0.5;
  transition: opacity 0.2s;
}

.bind-btn:hover {
  opacity: 1;
}
</style>
