<template>
    <div class="layout-grid" :style="gridStyle">
        <slot></slot>
    </div>
</template>

<script setup>
/**
 * Grid - CSS Grid 布局组件
 * Task 3.4: 实现 Grid 组件
 */
import { computed } from 'vue';

const props = defineProps({
    gridTemplateColumns: {
        type: String,
        default: 'repeat(3, 1fr)',
    },
    gridTemplateRows: {
        type: String,
        default: 'auto',
    },
    gap: {
        type: Number,
        default: 10,
    },
    gridAutoFlow: {
        type: String,
        default: 'row',
        validator: (value) => ['row', 'column', 'dense', 'row dense', 'column dense'].includes(value),
    },
    justifyItems: {
        type: String,
        default: 'stretch',
    },
    alignItems: {
        type: String,
        default: 'stretch',
    },
});

const gridStyle = computed(() => ({
    display: 'grid',
    gridTemplateColumns: props.gridTemplateColumns,
    gridTemplateRows: props.gridTemplateRows,
    gap: `${props.gap}px`,
    gridAutoFlow: props.gridAutoFlow,
    justifyItems: props.justifyItems,
    alignItems: props.alignItems,
}));
</script>

<style scoped>
.layout-grid {
    width: 100%;
    height: 100%;
}
</style>
