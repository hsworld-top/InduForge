<template>
  <div class="button-component-editor">
    <div class="editor-section">
      <div class="section-title">按钮属性</div>
      
      <!-- 按钮文本 -->
      <div class="form-item">
        <label>文本</label>
        <el-input
          :model-value="props.text || '按钮'"
          @update:model-value="(val) => handleChange('text', val)"
          placeholder="按钮文本"
        />
      </div>

      <!-- 按钮类型 -->
      <div class="form-item">
        <label>类型</label>
        <el-select
          :model-value="props.type || 'primary'"
          @update:model-value="(val) => handleChange('type', val)"
          placeholder="选择类型"
        >
          <el-option label="主要按钮" value="primary" />
          <el-option label="成功按钮" value="success" />
          <el-option label="警告按钮" value="warning" />
          <el-option label="危险按钮" value="danger" />
          <el-option label="信息按钮" value="info" />
          <el-option label="文本按钮" value="text" />
        </el-select>
      </div>

      <!-- 按钮大小 -->
      <div class="form-item">
        <label>尺寸</label>
        <el-select
          :model-value="props.size || 'default'"
          @update:model-value="(val) => handleChange('size', val)"
          placeholder="选择尺寸"
        >
          <el-option label="大" value="large" />
          <el-option label="默认" value="default" />
          <el-option label="小" value="small" />
        </el-select>
      </div>

      <!-- 是否圆角 -->
      <div class="form-item">
        <el-checkbox
          :model-value="props.round || false"
          @update:model-value="(val) => handleChange('round', val)"
        >
          圆角按钮
        </el-checkbox>
      </div>

      <!-- 是否圆形 -->
      <div class="form-item">
        <el-checkbox
          :model-value="props.circle || false"
          @update:model-value="(val) => handleChange('circle', val)"
        >
          圆形按钮
        </el-checkbox>
      </div>

      <!-- 是否禁用 -->
      <div class="form-item">
        <el-checkbox
          :model-value="props.disabled || false"
          @update:model-value="(val) => handleChange('disabled', val)"
        >
          禁用
        </el-checkbox>
      </div>

      <!-- 图标 -->
      <div class="form-item">
        <label>图标</label>
        <el-input
          :model-value="props.icon || ''"
          @update:model-value="(val) => handleChange('icon', val)"
          placeholder="图标名称 (Element Plus Icons)"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
/**
 * ButtonComponentEditor.vue - 按钮组件属性编辑器
 * Task 6.3: 实现组件专有属性编辑器
 */
import { computed } from 'vue';

const props = defineProps({
  component: {
    type: Object,
    required: true,
  },
});

const emit = defineEmits(['update']);

/**
 * 组件 props
 */
const componentProps = computed(() => props.component?.props || {});

/**
 * 处理属性变化
 */
function handleChange(key, value) {
  emit('update', {
    props: {
      [key]: value,
    },
  });
}
</script>

<style scoped>
.button-component-editor {
  padding: 12px 0;
}

.editor-section {
  margin-bottom: 16px;
}

.section-title {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid #e4e7ed;
}

.form-item {
  margin-bottom: 16px;
}

.form-item label {
  display: block;
  font-size: 13px;
  color: #606266;
  margin-bottom: 8px;
}

.el-input,
.el-select {
  width: 100%;
}
</style>

