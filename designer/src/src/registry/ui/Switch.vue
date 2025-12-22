<template>
    <div class="ui-switch-wrapper" :style="mergedStyle">
        <el-switch
            class="ui-switch-component"
            :model-value="value"
            :disabled="disabled"
            :active-text="activeText"
            :inactive-text="inactiveText"
            :active-color="activeColor"
            :inactive-color="inactiveColor"
            :active-value="activeValue"
            :inactive-value="inactiveValue" />
    </div>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({
    name: 'UISwitchComponent',
});

const props = defineProps({
    value: { type: [Boolean, String, Number], default: false },
    disabled: { type: Boolean, default: false },
    activeText: { type: String, default: '' },
    inactiveText: { type: String, default: '' },
    activeColor: { type: String, default: '#409EFF' },
    inactiveColor: { type: String, default: '#C0CCDA' },
    activeValue: { type: [Boolean, String, Number], default: true },
    inactiveValue: { type: [Boolean, String, Number], default: false },
});

const attrs = useAttrs();
const mergedStyle = computed(() => extractLayoutFreeStyle(attrs));
</script>

<style scoped>
.ui-switch-wrapper {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
}
</style>
