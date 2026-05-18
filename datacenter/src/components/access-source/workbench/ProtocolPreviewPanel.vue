<template>
  <div class="aws-protocol-preview">
    <EmptyState
      title="短时预览能力未启用"
      description="该协议的开发态预览将在 data_service preview-session 接通后启用。当前仅支持配置查看与编辑。"
      icon-name="access-source"
    >
      <!-- 查看配置按钮占位，A3·2 接入后启用 -->
      <el-tooltip content="即将由 A3·2 接入" placement="top">
        <el-button size="small" disabled>查看配置</el-button>
      </el-tooltip>
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
.aws-protocol-preview {
  height: 100%;
  min-height: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>
