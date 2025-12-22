<template>
    <ChartPlaceholder
        :title="title || '饼图'"
        type-label="Pie"
        :color="color"
        :data="valueList" />
</template>

<script setup>
import { computed } from 'vue';
import ChartPlaceholder from './ChartPlaceholder.vue';

defineOptions({
    name: 'PieChartComponent',
});

const props = defineProps({
    title: { type: String, default: '饼图' },
    data: {
        type: Array,
        default: () => [
            { name: '分类1', value: 40 },
            { name: '分类2', value: 25 },
            { name: '分类3', value: 35 },
        ],
    },
});

const valueList = computed(() => (props.data?.length ? props.data.map((d) => d.value) : [30, 30, 40]));
const color = '#F56C6C';
</script>
