<template>
  <div class="ui-el-popover" :style="mergedStyle">
    <el-popover
      :trigger="trigger"
      :width="width"
      :placement="placement"
      :visible="visible"
      :title="title"
      v-bind="listeners"
    >
      <template #reference>
        <slot name="reference">
          <el-button size="small">参考元素</el-button>
        </slot>
      </template>
      <slot>{{ content }}</slot>
    </el-popover>
  </div>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElPopover' });

const props = defineProps({
  trigger: { type: String, default: 'click' },
  width: { type: [String, Number], default: 200 },
  placement: { type: String, default: 'top' },
  visible: { type: Boolean, default: true },
  title: { type: String, default: '标题' },
  content: { type: String, default: '气泡卡片内容' },
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
.ui-el-popover {
  width: 100%;
  height: 100%;
}
</style>
