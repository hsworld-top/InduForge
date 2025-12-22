<template>
    <div class="ui-table-wrapper" :style="mergedStyle">
        <el-table
            class="ui-table-component"
            :data="data"
            :stripe="stripe"
            :border="border"
            :size="size"
            :show-header="showHeader"
            :highlight-current-row="highlightCurrentRow"
            height="100%"
            style="width: 100%">
            <el-table-column v-for="col in columns" :key="col.prop || col.label" v-bind="col" />
        </el-table>
    </div>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({
    name: 'UITableComponent',
});

const props = defineProps({
    data: { type: Array, default: () => [] },
    columns: {
        type: Array,
        default: () => [],
    },
    stripe: { type: Boolean, default: false },
    border: { type: Boolean, default: false },
    size: { type: String, default: 'default' },
    showHeader: { type: Boolean, default: true },
    highlightCurrentRow: { type: Boolean, default: false },
});

const attrs = useAttrs();
const mergedStyle = computed(() => extractLayoutFreeStyle(attrs));
</script>

<style scoped>
.ui-table-wrapper {
    width: 100%;
    height: 100%;
}

.ui-table-component {
    width: 100%;
    height: 100%;
}
</style>
