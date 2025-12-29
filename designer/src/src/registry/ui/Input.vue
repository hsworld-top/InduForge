<template>
    <el-input
        class="ui-input-component"
        :model-value="value"
        :type="type"
        :placeholder="placeholder"
        :size="size"
        :disabled="disabled"
        :clearable="clearable"
        :show-password="showPassword"
        :maxlength="maxlength"
        :show-word-limit="showWordLimit"
        :prefix-icon="resolvedPrefixIcon"
        :suffix-icon="resolvedSuffixIcon"
        :style="mergedStyle" />
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';
import { resolveEpIcon } from '../utils/iconHelpers.js';

defineOptions({
    name: 'UIInputComponent',
});

const props = defineProps({
    value: { type: [String, Number], default: '' },
    placeholder: { type: String, default: '请输入内容' },
    type: { type: String, default: 'text' },
    size: { type: String, default: 'default' },
    disabled: { type: Boolean, default: false },
    clearable: { type: Boolean, default: false },
    showPassword: { type: Boolean, default: false },
    maxlength: { type: [Number, null], default: null },
    showWordLimit: { type: Boolean, default: false },
    prefixIcon: { type: String, default: '' },
    suffixIcon: { type: String, default: '' },
});

const attrs = useAttrs();

const resolvedPrefixIcon = computed(() => resolveEpIcon(props.prefixIcon));
const resolvedSuffixIcon = computed(() => resolveEpIcon(props.suffixIcon));

const mergedStyle = computed(() => extractLayoutFreeStyle(attrs));
</script>

<style scoped>
.ui-input-component {
    width: 100%;
    height: 100%;
}
</style>
