<template>
  <div class="datapoint-workspace">
    <DataPointList :project-id="projectId" mode="management" />
  </div>
</template>

<script setup lang="ts">
import DataPointList from "@/components/datapoint/DataPointList.vue";

defineProps<{
  projectId: string;
}>();
</script>

<style scoped>
.datapoint-workspace {
  height: calc(100vh - 32px);
  min-height: 0;
  overflow: hidden;
}
</style>
