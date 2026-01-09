<template>
  <div class="prop-editor">
    <!-- 字符串类型 -->
    <el-input
      v-if="prop.type === 'string'"
      :model-value="modelValue"
      :placeholder="prop.placeholder || '请输入'"
      size="small"
      @update:model-value="handleChange"
    />

    <!-- 数字类型 -->
    <el-input-number
      v-else-if="prop.type === 'number'"
      :model-value="modelValue"
      :min="prop.min"
      :max="prop.max"
      :step="prop.step || 1"
      size="small"
      controls-position="right"
      @update:model-value="handleChange"
    />

    <!-- 布尔类型 -->
    <el-switch
      v-else-if="prop.type === 'boolean'"
      :model-value="modelValue"
      size="small"
      @update:model-value="handleChange"
    />

    <!-- 颜色类型 -->
    <el-color-picker
      v-else-if="prop.type === 'color'"
      :model-value="modelValue"
      size="small"
      show-alpha
      @update:model-value="handleChange"
    />

    <!-- 枚举类型 -->
    <el-select
      v-else-if="prop.type === 'enum'"
      :model-value="modelValue"
      size="small"
      @update:model-value="handleChange"
    >
      <el-option
        v-for="opt in prop.options"
        :key="opt.value"
        :label="opt.label"
        :value="opt.value"
      />
    </el-select>

    <!-- 默认：文本输入 -->
    <el-input
      v-else
      :model-value="String(modelValue ?? '')"
      size="small"
      @update:model-value="handleChange"
    />
  </div>
</template>

<script setup>
/**
 * 通用属性编辑器
 * 根据属性类型渲染对应的 Element Plus 组件
 */

defineOptions({ name: 'PropEditor' });

const props = defineProps({
  /** 属性定义 */
  prop: {
    type: Object,
    required: true,
  },
  /** 当前值 */
  modelValue: {
    type: [String, Number, Boolean, Object, Array],
    default: undefined,
  },
});

const emit = defineEmits(['update:modelValue']);

/**
 * 处理值变更
 * @param {any} value - 新值
 */
const handleChange = (value) => {
  emit('update:modelValue', value);
};
</script>

<style scoped>
.prop-editor {
  width: 100%;
}

.prop-editor :deep(.el-input-number) {
  width: 100%;
}

.prop-editor :deep(.el-select) {
  width: 100%;
}
</style>
