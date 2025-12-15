<template>
    <ChartPlaceholder
        :title="title || '折线图'"
        type-label="Line"
        :color="primaryColor"
        :data="seriesData" />
</template>

<script setup>
import { computed } from 'vue';
import ChartPlaceholder from './ChartPlaceholder.vue';

defineOptions({
    name: 'LineChartComponent',
});

const props = defineProps({
    title: { type: String, default: '折线图' },
    series: {
        type: Array,
        default: () => [
            { name: '系列1', data: [120, 200, 150, 80, 70, 110, 130], color: '#409EFF' },
        ],
    },
});

const seriesData = computed(() => (props.series?.[0]?.data ? props.series[0].data : [20, 50, 80]));
const primaryColor = computed(() => props.series?.[0]?.color || '#409EFF');
</script>
