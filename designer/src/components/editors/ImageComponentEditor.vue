<template>
  <div class="image-component-editor">
    <div class="editor-section">
      <div class="section-title">图片属性</div>
      
      <!-- 图片URL -->
      <div class="form-item">
        <label>图片地址</label>
        <el-input
          :model-value="componentProps.src || ''"
          @update:model-value="(val) => handleChange('src', val)"
          placeholder="输入图片URL"
        />
      </div>

      <!-- Alt文本 -->
      <div class="form-item">
        <label>Alt文本</label>
        <el-input
          :model-value="componentProps.alt || ''"
          @update:model-value="(val) => handleChange('alt', val)"
          placeholder="图片描述文本"
        />
      </div>

      <!-- Fit模式 -->
      <div class="form-item">
        <label>适应模式</label>
        <el-select
          :model-value="componentProps.fit || 'cover'"
          @update:model-value="(val) => handleChange('fit', val)"
          placeholder="选择适应模式"
        >
          <el-option label="填充 (Fill)" value="fill" />
          <el-option label="包含 (Contain)" value="contain" />
          <el-option label="覆盖 (Cover)" value="cover" />
          <el-option label="无 (None)" value="none" />
          <el-option label="缩小 (Scale-down)" value="scale-down" />
        </el-select>
      </div>

      <!-- 懒加载 -->
      <div class="form-item">
        <el-checkbox
          :model-value="componentProps.lazy || false"
          @update:model-value="(val) => handleChange('lazy', val)"
        >
          懒加载
        </el-checkbox>
      </div>

      <!-- 预览功能 -->
      <div class="form-item">
        <el-checkbox
          :model-value="componentProps.previewSrcList?.length > 0"
          @update:model-value="(val) => handlePreviewChange(val)"
        >
          启用预览
        </el-checkbox>
      </div>
    </div>
  </div>
</template>

<script setup>
/**
 * ImageComponentEditor.vue - 图片组件属性编辑器
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

/**
 * 处理预览功能切换
 */
function handlePreviewChange(enabled) {
  if (enabled) {
    emit('update', {
      props: {
        previewSrcList: [componentProps.value.src || ''],
      },
    });
  } else {
    emit('update', {
      props: {
        previewSrcList: [],
      },
    });
  }
}
</script>

<style scoped>
.image-component-editor {
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

