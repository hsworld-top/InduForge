<template>
    <el-button
        class="ui-button-component"
        :type="type"
        :size="size"
        :disabled="disabled"
        :loading="loading"
        :plain="plain"
        :round="round"
        :circle="circle"
        :icon="resolvedIcon"
        :style="mergedStyle">
        <template v-if="resolvedIcon" #icon>
            <component :is="resolvedIcon" />
        </template>
        <span>{{ text || '按钮' }}</span>
    </el-button>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import * as ElementPlusIconsVue from '@element-plus/icons-vue';

defineOptions({
    name: 'UIButtonComponent',
});

const props = defineProps({
    text: { type: String, default: '按钮' },
    type: { type: String, default: 'primary' },
    size: { type: String, default: 'default' },
    disabled: { type: Boolean, default: false },
    loading: { type: Boolean, default: false },
    plain: { type: Boolean, default: false },
    round: { type: Boolean, default: false },
    circle: { type: Boolean, default: false },
    icon: { type: String, default: '' },
});

const attrs = useAttrs();

const resolvedIcon = computed(() => {
    if (!props.icon) return null;
    const match = Object.keys(ElementPlusIconsVue).find((key) => key.toLowerCase() === props.icon.toLowerCase());
    return match ? ElementPlusIconsVue[match] : null;
});

const mergedStyle = computed(() => {
    const styleAttr = attrs.style || {};
    const { left, top, right, bottom, position, zIndex, ...rest } = styleAttr;
    return {
        width: '100%',
        height: '100%',
        ...rest,
    };
});
</script>

<style scoped>
.ui-button-component {
    width: 100%;
    height: 100%;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 0 12px;
    box-sizing: border-box;
}
</style>
