<template>
  <div class="data-panel">
    <DatapointPanel class="data-panel__content" />
  </div>
</template>

<script setup>
import DatapointPanel from "./DatapointPanel.vue";
</script>

<style scoped>
.data-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.data-panel__content {
  flex: 1;
  min-height: 0;
}
</style>
