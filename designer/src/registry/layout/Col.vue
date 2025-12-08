<template>
    <div class="layout-col" :style="colStyle">
        <slot></slot>
    </div>
</template>

<script setup>
/**
 * Col - 列组件（24栅格系统）
 * Task 3.2: 实现 Row/Col 组件
 */
import { computed, inject } from 'vue';

const props = defineProps({
    span: {
        type: Number,
        default: 24,
        validator: (value) => value >= 0 && value <= 24,
    },
    offset: {
        type: Number,
        default: 0,
    },
    push: {
        type: Number,
        default: 0,
    },
    pull: {
        type: Number,
        default: 0,
    },
});

const gutter = inject('gutter', 0);

const colStyle = computed(() => {
    const width = `${(props.span / 24) * 100}%`;
    const marginLeft = props.offset > 0 ? `${(props.offset / 24) * 100}%` : undefined;
    const left = props.push > 0 ? `${(props.push / 24) * 100}%` : undefined;
    const right = props.pull > 0 ? `${(props.pull / 24) * 100}%` : undefined;

    return {
        width,
        marginLeft,
        left,
        right,
        paddingLeft: `${gutter / 2}px`,
        paddingRight: `${gutter / 2}px`,
    };
});
</script>

<style scoped>
.layout-col {
    position: relative;
    box-sizing: border-box;
}
</style>
