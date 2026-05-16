<template>
  <el-empty class="dc-empty-state" :description="description">
    <template #image>
      <component :is="iconComponent" class="dc-empty-state__icon" />
    </template>

    <div class="dc-empty-state__content">
      <h3 class="dc-empty-state__title">{{ title }}</h3>
      <p v-if="description" class="dc-empty-state__description">
        {{ description }}
      </p>
      <div v-if="$slots.default" class="dc-empty-state__actions">
        <slot />
      </div>
    </div>
  </el-empty>
</template>

<script setup lang="ts">
import { computed } from "vue";
import {
  Bell,
  Box,
  Collection,
  Connection,
  Cpu,
  DataLine,
  Files,
  FolderOpened,
  Search,
  Warning,
} from "@element-plus/icons-vue";

type EmptyIconName =
  | "box"
  | "search"
  | "datapoint"
  | "access-source"
  | "compute"
  | "alarm"
  | "folder"
  | "files"
  | "warning"
  | "collection";

const props = withDefaults(
  defineProps<{
    title: string;
    description?: string;
    iconName?: EmptyIconName;
  }>(),
  {
    description: "",
    iconName: "box",
  },
);

const iconMap = {
  box: Box,
  search: Search,
  datapoint: DataLine,
  "access-source": Connection,
  compute: Cpu,
  alarm: Bell,
  folder: FolderOpened,
  files: Files,
  warning: Warning,
  collection: Collection,
};

const iconComponent = computed(() => iconMap[props.iconName] || Box);
</script>

<style scoped>
.dc-empty-state {
  --el-empty-padding: 24px 0;
}

.dc-empty-state :deep(.el-empty__image) {
  width: 72px;
  height: 72px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 10px;
  border: 1px solid var(--dc-border);
  border-radius: 18px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
}

.dc-empty-state :deep(.el-empty__description) {
  display: none;
}

.dc-empty-state__icon {
  width: 32px;
  height: 32px;
}

.dc-empty-state__content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  text-align: center;
}

.dc-empty-state__title {
  margin: 0;
  color: var(--dc-text);
  font-size: 15px;
  font-weight: 700;
  letter-spacing: 0;
  line-height: 1.45;
}

.dc-empty-state__description {
  max-width: 360px;
  margin: 0;
  color: var(--dc-text-muted);
  font-size: 13px;
  line-height: 1.6;
}

.dc-empty-state__actions {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin-top: 4px;
}
</style>
