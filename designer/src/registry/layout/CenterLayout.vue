<template>
    <div class="layout-center" :style="centerStyle">
        <slot></slot>
    </div>
</template>

<script setup>
/**
 * CenterLayout - 居中布局组件
 * Task 3.5: 实现 CenterLayout 组件
 */
import { computed } from 'vue';

const props = defineProps({
    horizontal: {
        type: Boolean,
        default: true,
    },
    vertical: {
        type: Boolean,
        default: true,
    },
});

const centerStyle = computed(() => {
    const style = {
        display: 'flex',
        width: '100%',
        height: '100%',
    };

    if (props.horizontal) {
        style.justifyContent = 'center';
    }

    if (props.vertical) {
        style.alignItems = 'center';
    }

    return style;
});
</script>

<style scoped>
.layout-center {
    /* Styles are computed dynamically */
}
</style>
