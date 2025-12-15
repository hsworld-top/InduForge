<template>
    <el-select
        class="ui-select-component"
        :model-value="value"
        :placeholder="placeholder"
        :size="size"
        :disabled="disabled"
        :clearable="clearable"
        :multiple="multiple"
        :filterable="filterable"
        :style="mergedStyle">
        <el-option v-for="opt in options" :key="opt.value" :label="opt.label" :value="opt.value" />
    </el-select>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({
    name: 'UISelectComponent',
});

const props = defineProps({
    value: { type: [String, Number, Array], default: '' },
    placeholder: { type: String, default: '请选择' },
    size: { type: String, default: 'default' },
    disabled: { type: Boolean, default: false },
    clearable: { type: Boolean, default: false },
    multiple: { type: Boolean, default: false },
    filterable: { type: Boolean, default: false },
    options: {
        type: Array,
        default: () => [
            { label: '选项1', value: '1' },
            { label: '选项2', value: '2' },
            { label: '选项3', value: '3' },
        ],
    },
});

const attrs = useAttrs();
const mergedStyle = computed(() => extractLayoutFreeStyle(attrs));
</script>

<style scoped>
.ui-select-component {
    width: 100%;
    height: 100%;
}
</style>
