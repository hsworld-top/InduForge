<template>
    <ChartPlaceholder
        :title="title || '散点图'"
        type-label="Scatter"
        :color="primaryColor"
        :data="flattenData" />
</template>

<script setup>
import { computed } from 'vue';
import ChartPlaceholder from './ChartPlaceholder.vue';

defineOptions({
    name: 'ScatterChartComponent',
});

const props = defineProps({
    title: { type: String, default: '散点图' },
    series: {
        type: Array,
        default: () => [
            {
                name: '系列1',
                data: [
                    [10, 20],
                    [20, 40],
                    [30, 10],
                    [40, 60],
                ],
                color: '#E6A23C',
            },
        ],
    },
});

const flattenData = computed(() => {
    const first = props.series?.[0]?.data || [];
    return first.map((pair) => (Array.isArray(pair) ? pair[1] ?? pair[0] : pair));
});

const primaryColor = computed(() => props.series?.[0]?.color || '#E6A23C');
</script>
