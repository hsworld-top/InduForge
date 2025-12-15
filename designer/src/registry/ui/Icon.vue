<template>
    <div class="ui-icon-wrapper" :style="mergedStyle">
        <component v-if="resolvedIcon" :is="resolvedIcon" :size="size" :color="color" />
        <span v-else class="ui-icon-fallback">Icon</span>
    </div>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';
import { resolveEpIcon } from '../utils/iconHelpers.js';

defineOptions({
    name: 'UIIconComponent',
});

const props = defineProps({
    name: { type: String, default: 'star' },
    size: { type: [Number, String], default: 16 },
    color: { type: String, default: '#606266' },
});

const attrs = useAttrs();
const resolvedIcon = computed(() => resolveEpIcon(props.name));
const mergedStyle = computed(() => extractLayoutFreeStyle(attrs));
</script>

<style scoped>
.ui-icon-wrapper {
    width: 100%;
    height: 100%;
    display: inline-flex;
    align-items: center;
    justify-content: center;
}

.ui-icon-fallback {
    font-size: 12px;
    color: #909399;
}
</style>
