<template>
    <div class="layout-flex" :style="flexStyle">
        <slot></slot>
    </div>
</template>

<script setup>
/**
 * Flex - Flexbox 布局组件
 * Task 3.3: 实现 Flex 组件
 */
import { computed } from 'vue';

const props = defineProps({
    flexDirection: {
        type: String,
        default: 'row',
        validator: (value) => ['row', 'row-reverse', 'column', 'column-reverse'].includes(value),
    },
    justifyContent: {
        type: String,
        default: 'flex-start',
    },
    alignItems: {
        type: String,
        default: 'flex-start',
    },
    flexWrap: {
        type: String,
        default: 'nowrap',
        validator: (value) => ['nowrap', 'wrap', 'wrap-reverse'].includes(value),
    },
    gap: {
        type: Number,
        default: 0,
    },
    alignContent: {
        type: String,
        default: 'normal',
    },
});

const flexStyle = computed(() => ({
    display: 'flex',
    flexDirection: props.flexDirection,
    justifyContent: props.justifyContent,
    alignItems: props.alignItems,
    flexWrap: props.flexWrap,
    gap: `${props.gap}px`,
    alignContent: props.alignContent,
}));
</script>

<style scoped>
.layout-flex {
    width: 100%;
    height: 100%;
}
</style>
