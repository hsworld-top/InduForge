<template>
    <div class="layout-row" :style="rowStyle">
        <slot></slot>
    </div>
</template>

<script setup>
/**
 * Row - 行组件（24栅格系统）
 * Task 3.2: 实现 Row/Col 组件
 */
import { computed } from 'vue';

const props = defineProps({
    gutter: {
        type: Number,
        default: 0,
    },
    justify: {
        type: String,
        default: 'start',
    },
    align: {
        type: String,
        default: 'top',
    },
});

const rowStyle = computed(() => ({
    display: 'flex',
    flexWrap: 'wrap',
    marginLeft: `-${props.gutter / 2}px`,
    marginRight: `-${props.gutter / 2}px`,
    justifyContent:
        props.justify === 'start'
            ? 'flex-start'
            : props.justify === 'end'
              ? 'flex-end'
              : props.justify === 'center'
                ? 'center'
                : props.justify === 'space-around'
                  ? 'space-around'
                  : props.justify === 'space-between'
                    ? 'space-between'
                    : 'flex-start',
    alignItems: props.align === 'top' ? 'flex-start' : props.align === 'middle' ? 'center' : props.align === 'bottom' ? 'flex-end' : 'flex-start',
}));
</script>

<style scoped>
.layout-row {
    width: 100%;
}
</style>
