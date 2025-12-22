<template>
  <div class="ui-el-result" :style="mergedStyle">
    <el-result :icon="icon" :title="title" :sub-title="subTitle" v-bind="listeners">
      <template #extra>
        <slot name="extra"></slot>
      </template>
    </el-result>
  </div>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElResult' });

const props = defineProps({
  icon: { type: String, default: 'success' },
  title: { type: String, default: '提交成功' },
  subTitle: { type: String, default: '您的请求已完成' },
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
.ui-el-result {
  width: 100%;
  height: 100%;
}
</style>
