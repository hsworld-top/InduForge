<script setup>
/**
 * 多选检查器面板
 * 显示多选元素的共同属性，支持批量编辑
 */

import { reactive, watch } from "vue";
import FriendlyColorPicker from "@/components/common/FriendlyColorPicker.vue";
import { useMultiSelect } from "../composables/use-multi-select";

const props = defineProps({
  /** 选中的元素列表 */
  elements: {
    type: Array,
    default: () => [],
  },
});

const { selectedCount, getMultiSelectValue, setUnifiedValue } = useMultiSelect(props);

// 表单数据
const form = reactive({
  opacity: 100,
  visible: true,
  locked: false,
  fill: "",
  stroke: "",
  strokeWidth: 1,
});

// 多选值状态
const fillValue = reactive({ type: "same", value: "" });
const strokeValue = reactive({ type: "same", value: "" });
const strokeWidthValue = reactive({ type: "same", value: 1 });

/**
 * 同步表单数据
 */
function syncForm() {
  if (!props.elements?.length) return;

  // 透明度
  const opacityResult = getMultiSelectValue("style.opacity");
  if (opacityResult.type === "same") {
    form.opacity = (opacityResult.value ?? 1) * 100;
  }

  // 可见性
  const visibleResult = getMultiSelectValue("visible");
  if (visibleResult.type === "same") {
    form.visible = visibleResult.value !== false;
  }

  // 锁定状态
  const lockedResult = getMultiSelectValue("locked");
  if (lockedResult.type === "same") {
    form.locked = lockedResult.value === true;
  }

  // 填充颜色
  const fillResult = getMultiSelectValue("style.fill");
  fillValue.type = fillResult.type;
  fillValue.value = fillResult.value;
  if (fillResult.type === "same") {
    form.fill = fillResult.value || "";
  }

  // 描边颜色
  const strokeResult = getMultiSelectValue("style.stroke");
  strokeValue.type = strokeResult.type;
  strokeValue.value = strokeResult.value;
  if (strokeResult.type === "same") {
    form.stroke = strokeResult.value || "";
  }

  // 描边宽度
  const strokeWidthResult = getMultiSelectValue("style.strokeWidth");
  strokeWidthValue.type = strokeWidthResult.type;
  strokeWidthValue.value = strokeWidthResult.value;
  if (strokeWidthResult.type === "same") {
    form.strokeWidth = strokeWidthResult.value ?? 1;
  }
}

// 监听元素变化
watch(
  () => props.elements,
  () => syncForm(),
  { immediate: true, deep: true },
);

/**
 * 处理透明度变化
 * @param {number} value - 透明度百分比
 */
function handleOpacityChange(value) {
  setUnifiedValue("style.opacity", value / 100);
}

/**
 * 处理可见性变化
 * @param {boolean} value - 是否可见
 */
function handleVisibleChange(value) {
  setUnifiedValue("visible", value);
}

/**
 * 处理锁定状态变化
 * @param {boolean} value - 是否锁定
 */
function handleLockedChange(value) {
  setUnifiedValue("locked", value);
}

/**
 * 处理填充颜色变化
 * @param {string} value - 颜色值
 */
function handleFillChange(value) {
  setUnifiedValue("style.fill", value);
  fillValue.type = "same";
  fillValue.value = value;
}

/**
 * 处理描边颜色变化
 * @param {string} value - 颜色值
 */
function handleStrokeChange(value) {
  setUnifiedValue("style.stroke", value);
  strokeValue.type = "same";
  strokeValue.value = value;
}

/**
 * 处理描边宽度变化
 * @param {number} value - 宽度值
 */
function handleStrokeWidthChange(value) {
  setUnifiedValue("style.strokeWidth", value);
  strokeWidthValue.type = "same";
  strokeWidthValue.value = value;
}
</script>

<template>
  <div class="multi-inspector-panel">
    <div class="multi-inspector-header">
      <span class="text-sm text-gray-500">已选中</span>
      <span class="text-lg font-medium ml-1">{{ selectedCount }}</span>
      <span class="text-sm text-gray-500 ml-1">个元素</span>
    </div>

    <el-divider />

    <!-- 通用属性 -->
    <div class="multi-inspector-section">
      <div class="section-title">通用属性</div>
      <el-form label-width="72px" size="small">
        <el-form-item label="透明度">
          <el-slider
            v-model="form.opacity"
            :min="0"
            :max="100"
            :format-tooltip="(val) => `${val}%`"
            @change="handleOpacityChange"
          />
        </el-form-item>
        <el-form-item label="可见性">
          <el-switch
            v-model="form.visible"
            :active-value="true"
            :inactive-value="false"
            @change="handleVisibleChange"
          />
          <span class="ml-2 text-xs text-gray-400">
            {{ form.visible ? "显示" : "隐藏" }}
          </span>
        </el-form-item>
        <el-form-item label="锁定">
          <el-switch
            v-model="form.locked"
            :active-value="true"
            :inactive-value="false"
            @change="handleLockedChange"
          />
          <span class="ml-2 text-xs text-gray-400">
            {{ form.locked ? "已锁定" : "未锁定" }}
          </span>
        </el-form-item>
      </el-form>
    </div>

    <el-divider />

    <!-- 样式属性 -->
    <div class="multi-inspector-section">
      <div class="section-title">样式属性</div>
      <el-form label-width="72px" size="small">
        <el-form-item label="填充颜色">
          <div class="flex items-center gap-2">
            <FriendlyColorPicker
              v-model="form.fill"
              :disabled="fillValue.type === 'mixed'"
              @change="handleFillChange"
            />
            <span v-if="fillValue.type === 'mixed'" class="text-xs text-gray-400"> 混合值 </span>
          </div>
        </el-form-item>
        <el-form-item label="描边颜色">
          <div class="flex items-center gap-2">
            <FriendlyColorPicker
              v-model="form.stroke"
              :disabled="strokeValue.type === 'mixed'"
              @change="handleStrokeChange"
            />
            <span v-if="strokeValue.type === 'mixed'" class="text-xs text-gray-400"> 混合值 </span>
          </div>
        </el-form-item>
        <el-form-item label="描边宽度">
          <el-input-number
            v-model="form.strokeWidth"
            :min="0"
            :max="100"
            :placeholder="strokeWidthValue.type === 'mixed' ? '--' : ''"
            controls-position="right"
            @change="handleStrokeWidthChange"
          />
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<style scoped>
.multi-inspector-panel {
  padding: 12px;
}

.multi-inspector-header {
  display: flex;
  align-items: baseline;
  margin-bottom: 8px;
}

.multi-inspector-section {
  margin-bottom: 16px;
}

.section-title {
  font-size: 12px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  margin-bottom: 12px;
}
</style>
