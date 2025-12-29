<template>
  <div class="input-component-editor">
    <div class="editor-section">
      <div class="section-title">输入框属性</div>
      
      <!-- 占位符 -->
      <div class="form-item">
        <label>占位符</label>
        <el-input
          :model-value="componentProps.placeholder || ''"
          @update:model-value="(val) => handleChange('placeholder', val)"
          placeholder="输入占位符文本"
        />
      </div>

      <!-- 输入框类型 -->
      <div class="form-item">
        <label>类型</label>
        <el-select
          :model-value="componentProps.type || 'text'"
          @update:model-value="(val) => handleChange('type', val)"
          placeholder="选择类型"
        >
          <el-option label="文本" value="text" />
          <el-option label="文本域" value="textarea" />
          <el-option label="密码" value="password" />
          <el-option label="数字" value="number" />
          <el-option label="邮箱" value="email" />
          <el-option label="URL" value="url" />
          <el-option label="日期" value="date" />
        </el-select>
      </div>

      <!-- 尺寸 -->
      <div class="form-item">
        <label>尺寸</label>
        <el-select
          :model-value="componentProps.size || 'default'"
          @update:model-value="(val) => handleChange('size', val)"
          placeholder="选择尺寸"
        >
          <el-option label="大" value="large" />
          <el-option label="默认" value="default" />
          <el-option label="小" value="small" />
        </el-select>
      </div>

      <!-- 最大长度 -->
      <div class="form-item">
        <label>最大长度</label>
        <el-input-number
          :model-value="componentProps.maxlength || 100"
          @update:model-value="(val) => handleChange('maxlength', val)"
          :min="1"
          :max="10000"
        />
      </div>

      <!-- 文本域行数 -->
      <div v-if="componentProps.type === 'textarea'" class="form-item">
        <label>行数</label>
        <el-input-number
          :model-value="componentProps.rows || 3"
          @update:model-value="(val) => handleChange('rows', val)"
          :min="1"
          :max="20"
        />
      </div>

      <!-- 是否显示字数统计 -->
      <div class="form-item">
        <el-checkbox
          :model-value="componentProps.showWordLimit || false"
          @update:model-value="(val) => handleChange('showWordLimit', val)"
        >
          显示字数统计
        </el-checkbox>
      </div>

      <!-- 是否可清空 -->
      <div class="form-item">
        <el-checkbox
          :model-value="componentProps.clearable !== false"
          @update:model-value="(val) => handleChange('clearable', val)"
        >
          可清空
        </el-checkbox>
      </div>

      <!-- 是否禁用 -->
      <div class="form-item">
        <el-checkbox
          :model-value="componentProps.disabled || false"
          @update:model-value="(val) => handleChange('disabled', val)"
        >
          禁用
        </el-checkbox>
      </div>

      <!-- 是否只读 -->
      <div class="form-item">
        <el-checkbox
          :model-value="componentProps.readonly || false"
          @update:model-value="(val) => handleChange('readonly', val)"
        >
          只读
        </el-checkbox>
      </div>
    </div>
  </div>
</template>

<script setup>
/**
 * InputComponentEditor.vue - 输入框组件属性编辑器
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
.input-component-editor {
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

