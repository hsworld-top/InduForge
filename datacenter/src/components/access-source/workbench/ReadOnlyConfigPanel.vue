<template>
  <div class="aws-readonly-config">
    <EmptyState
      title="工业协议仅配置"
      description="该协议在开发态仅持有配置契约，真实采集由节点侧执行。"
      icon-name="access-source"
    >
      <el-button size="small" @click="$emit('back')">返回接入源列表</el-button>
    </EmptyState>
  </div>
</template>

<script setup lang="ts">
import EmptyState from "@/components/shared/EmptyState.vue";

defineProps<{
  connection: Record<string, any>;
  projectId: string;
}>();

defineEmits<{
  (event: "back"): void;
}>();
</script>

<style scoped>
.aws-readonly-config {
  height: 100%;
  min-height: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>
