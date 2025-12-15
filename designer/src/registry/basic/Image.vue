<template>
    <el-image
        class="basic-image-component"
        :src="src"
        :alt="alt"
        :fit="fit"
        :style="imageStyle"
        :draggable="false">
        <template #error>
            <div class="basic-image-placeholder">加载图片失败</div>
        </template>
    </el-image>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({
    name: 'BasicImageComponent',
});

const props = defineProps({
    src: { type: String, default: 'https://via.placeholder.com/150' },
    alt: { type: String, default: '图片' },
    fit: { type: String, default: 'contain' },
    opacity: { type: Number, default: 1 },
});

const attrs = useAttrs();
const mergedStyle = computed(() => extractLayoutFreeStyle(attrs));

const imageStyle = computed(() => ({
    ...mergedStyle.value,
    opacity: props.opacity,
    objectFit: props.fit,
}));
</script>

<style scoped>
.basic-image-component {
    width: 100%;
    height: 100%;
    background: #f5f5f5;
    border: 1px dashed #e5e7eb;
}

.basic-image-placeholder {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #909399;
    font-size: 12px;
    background: #fafafa;
}
</style>
