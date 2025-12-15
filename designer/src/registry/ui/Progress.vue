<template>
    <div class="ui-progress-wrapper" :style="mergedStyle">
        <el-progress
            :percentage="percentage"
            :type="type"
            :stroke-width="strokeWidth"
            :status="status || undefined"
            :color="color"
            :show-text="showText"
            :text-inside="textInside" />
    </div>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({
    name: 'UIProgressComponent',
});

const props = defineProps({
    percentage: { type: Number, default: 0 },
    type: { type: String, default: 'line' },
    strokeWidth: { type: Number, default: 6 },
    status: { type: String, default: '' },
    color: { type: [String, Array, Function], default: '#409EFF' },
    showText: { type: Boolean, default: true },
    textInside: { type: Boolean, default: false },
});

const attrs = useAttrs();
const mergedStyle = computed(() => extractLayoutFreeStyle(attrs));
</script>

<style scoped>
.ui-progress-wrapper {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
}
</style>
