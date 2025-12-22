<template>
  <div class="ui-el-tooltip" :style="mergedStyle">
    <el-tooltip
      :content="content"
      :placement="placement"
      :visible="visible"
      :effect="effect"
      :show-after="showAfter"
      :hide-after="hideAfter"
      v-bind="listeners"
    >
      <slot>
        <el-button size="small">Hover 我</el-button>
      </slot>
    </el-tooltip>
  </div>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElTooltip' });

const props = defineProps({
  content: { type: String, default: '提示内容' },
  placement: { type: String, default: 'top' },
  visible: { type: Boolean, default: true },
  effect: { type: String, default: 'dark' },
  showAfter: { type: Number, default: 0 },
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
.ui-el-tooltip {
  width: 100%;
  height: 100%;
}
</style>
