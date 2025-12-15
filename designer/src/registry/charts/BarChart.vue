<template>
    <ChartPlaceholder
        :title="title || '柱状图'"
        type-label="Bar"
        :color="primaryColor"
        :data="seriesData" />
</template>

<script setup>
import { computed } from 'vue';
import ChartPlaceholder from './ChartPlaceholder.vue';

defineOptions({
    name: 'BarChartComponent',
});

const props = defineProps({
    title: { type: String, default: '柱状图' },
    series: {
        type: Array,
        default: () => [
            { name: '系列1', data: [40, 90, 60, 120], color: '#67C23A' },
        ],
    },
});

const seriesData = computed(() => (props.series?.[0]?.data ? props.series[0].data : [40, 70, 110]));
const primaryColor = computed(() => props.series?.[0]?.color || '#67C23A');
</script>
