<template>
    <el-date-picker
        class="ui-date-picker-component"
        :model-value="value"
        :type="type"
        :placeholder="placeholder"
        :start-placeholder="startPlaceholder"
        :end-placeholder="endPlaceholder"
        :range-separator="rangeSeparator"
        :size="size"
        :disabled="disabled"
        :clearable="clearable"
        :format="format"
        :value-format="valueFormat"
        :style="mergedStyle" />
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({
    name: 'UIDatePickerComponent',
});

const props = defineProps({
    value: { type: [String, Array, Date], default: '' },
    type: { type: String, default: 'date' },
    placeholder: { type: String, default: '选择日期' },
    startPlaceholder: { type: String, default: '开始日期' },
    endPlaceholder: { type: String, default: '结束日期' },
    rangeSeparator: { type: String, default: '至' },
    size: { type: String, default: 'default' },
    disabled: { type: Boolean, default: false },
    clearable: { type: Boolean, default: true },
    format: { type: String, default: 'YYYY-MM-DD' },
    valueFormat: { type: String, default: 'YYYY-MM-DD' },
});

const attrs = useAttrs();
const mergedStyle = computed(() => extractLayoutFreeStyle(attrs));
</script>

<style scoped>
.ui-date-picker-component {
    width: 100%;
    height: 100%;
}
</style>
