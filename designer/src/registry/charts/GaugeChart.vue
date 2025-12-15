<template>
    <ChartPlaceholder
        :title="title || '仪表盘'"
        type-label="Gauge"
        :color="color"
        :data="[value]" />
</template>

<script setup>
import ChartPlaceholder from './ChartPlaceholder.vue';

defineOptions({
    name: 'GaugeChartComponent',
});

const props = defineProps({
    title: { type: String, default: '仪表盘' },
    value: { type: Number, default: 75 },
});

const color = '#8B5CF6';
</script>
