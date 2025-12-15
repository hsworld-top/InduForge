<template>
    <div class="chart-placeholder" :style="mergedStyle">
        <div class="chart-header">
            <span class="chart-title">{{ title }}</span>
            <span class="chart-type">{{ typeLabel }}</span>
        </div>
        <div class="chart-body">
            <div v-for="(val, idx) in displayData" :key="idx" class="chart-bar" :style="barStyle(val)">
                <span class="chart-value">{{ val }}</span>
            </div>
        </div>
    </div>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({
    name: 'ChartPlaceholder',
});

const props = defineProps({
    title: { type: String, default: '图表' },
    typeLabel: { type: String, default: 'Chart' },
    color: { type: String, default: '#409EFF' },
    data: { type: Array, default: () => [30, 60, 45, 80, 55] },
});

const attrs = useAttrs();
const mergedStyle = computed(() => extractLayoutFreeStyle(attrs));

const displayData = computed(() => (props.data.length ? props.data : [20, 40, 60, 80]));

const barStyle = (val) => ({
    height: `${Math.max(10, Math.min(100, val))}%`,
    background: props.color,
});
</script>

<style scoped>
.chart-placeholder {
    width: 100%;
    height: 100%;
    border: 1px dashed #dcdfe6;
    border-radius: 4px;
    background: linear-gradient(180deg, #f9fafb 0%, #fff 100%);
    padding: 12px;
    box-sizing: border-box;
    display: flex;
    flex-direction: column;
    gap: 8px;
}

.chart-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-size: 12px;
    color: #606266;
}

.chart-title {
    font-weight: 600;
    color: #303133;
}

.chart-body {
    flex: 1;
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(20px, 1fr));
    align-items: end;
    gap: 6px;
}

.chart-bar {
    width: 100%;
    min-height: 8px;
    border-radius: 2px 2px 0 0;
    position: relative;
    display: flex;
    align-items: flex-end;
    justify-content: center;
    color: #fff;
    font-size: 10px;
    padding-bottom: 2px;
    box-sizing: border-box;
}

.chart-value {
    background: rgba(0, 0, 0, 0.35);
    border-radius: 8px;
    padding: 0 4px;
}

.chart-type {
    color: #909399;
}
</style>
