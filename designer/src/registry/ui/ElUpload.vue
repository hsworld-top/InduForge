<template>
  <el-upload
    class="ui-el-upload"
    :action="action"
    :multiple="multiple"
    :limit="limit"
    :drag="drag"
    :list-type="listType"
    :headers="headers"
    :data="data"
    :style="mergedStyle"
    v-bind="listeners"
    @change="(file, fileList) => emit('change', file, fileList)"
    @success="(res, file, fileList) => emit('success', res, file, fileList)"
    @error="(err, file, fileList) => emit('error', err, file, fileList)"
    @remove="(file, fileList) => emit('remove', file, fileList)"
    @preview="(file) => emit('preview', file)"
  >
    <div class="ui-el-upload__slot">
      <slot>点击上传</slot>
    </div>
  </el-upload>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElUpload' });

const emit = defineEmits(['change', 'success', 'error', 'remove', 'preview']);

const props = defineProps({
  action: { type: String, default: '' },
  multiple: { type: Boolean, default: true },
  limit: { type: Number, default: 3 },
  drag: { type: Boolean, default: true },
  listType: { type: String, default: 'text' },
  headers: { type: Object, default: () => ({}) },
  data: { type: Object, default: () => ({}) },
});

const attrs = useAttrs();

const listeners = computed(() => {
  const res = {};
  Object.entries(attrs).forEach(([key, val]) => {
    if (key.startsWith('on')) res[key] = val;
  });
  return res;
});

const mergedStyle = computed(() => extractLayoutFreeStyle(attrs));
</script>

<style scoped>
.ui-el-upload {
  width: 100%;
  height: 100%;
}

.ui-el-upload__slot {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px dashed #dcdfe6;
  color: #606266;
  box-sizing: border-box;
  padding: 12px;
}
</style>
