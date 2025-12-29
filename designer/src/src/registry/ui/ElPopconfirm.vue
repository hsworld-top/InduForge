<template>
  <div class="ui-el-popconfirm" :style="mergedStyle">
    <el-popconfirm
      :title="title"
      :width="width"
      :confirm-button-text="confirmButtonText"
      :cancel-button-text="cancelButtonText"
      :icon="icon"
      :icon-color="iconColor"
      :hide-after="hideAfter"
      v-bind="listeners"
      @confirm="() => emit('confirm')"
      @cancel="() => emit('cancel')"
    >
      <template #reference>
        <slot name="reference">
          <el-button size="small" type="danger">删除</el-button>
        </slot>
      </template>
      <slot>{{ title }}</slot>
    </el-popconfirm>
  </div>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElPopconfirm' });

const emit = defineEmits(['confirm', 'cancel']);

const props = defineProps({
  title: { type: String, default: '确认删除？' },
  width: { type: [String, Number], default: 200 },
  confirmButtonText: { type: String, default: '确认' },
  cancelButtonText: { type: String, default: '取消' },
  icon: { type: String, default: 'QuestionFilled' },
  iconColor: { type: String, default: '#f56c6c' },
  hideAfter: { type: Number, default: 0 },
});

const attrs = useAttrs();

const listeners = computed(() => {
  const res = {};
  Object.entries(attrs).forEach(([k, v]) => {
    if (k.startsWith('on')) res[k] = v;
  });
  return res;
});

const mergedStyle = computed(() => extractLayoutFreeStyle(attrs));
</script>

<style scoped>
.ui-el-popconfirm {
  width: 100%;
  height: 100%;
}
</style>
