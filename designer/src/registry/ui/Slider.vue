<template>
    <div class="ui-slider-wrapper" :style="mergedStyle">
        <el-slider
            class="ui-slider-component"
            :model-value="value"
            :min="min"
            :max="max"
            :step="step"
            :disabled="disabled"
            :show-stops="showStops"
            :show-tooltip="showTooltip"
            :range="range" />
    </div>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({
    name: 'UISliderComponent',
});

const props = defineProps({
    value: { type: [Number, Array], default: 0 },
    min: { type: Number, default: 0 },
    max: { type: Number, default: 100 },
    step: { type: Number, default: 1 },
    disabled: { type: Boolean, default: false },
    showStops: { type: Boolean, default: false },
    showTooltip: { type: Boolean, default: true },
    range: { type: Boolean, default: false },
});

const attrs = useAttrs();
const mergedStyle = computed(() => extractLayoutFreeStyle(attrs));
</script>

<style scoped>
.ui-slider-wrapper {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
}

.ui-slider-component {
    width: 100%;
}
</style>
