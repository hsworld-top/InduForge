<!--
  脚本与变量面板 — 系统脚本折叠块
-->
<script setup lang="ts">
import { useI18n } from "vue-i18n";
import IconEpClose from "~icons/ep/close";
import IconEpPointer from "~icons/ep/pointer";

defineProps<{
  selectedSystemKey: string;
}>();

const emit = defineEmits<{
  selectSystem: [key: string];
  openSystemEditor: [kind: "startup" | "shutdown"];
}>();
const { t } = useI18n();
</script>

<template>
  <el-collapse-item name="system">
    <template #title> {{ t("scriptPanel.sections.system") }} </template>
    <div class="scripts-layout">
      <div class="scripts-list is-full">
        <el-menu
          :default-active="selectedSystemKey"
          class="list-menu"
          @select="(key: string) => emit('selectSystem', key)"
        >
          <el-menu-item index="startup" @dblclick="emit('openSystemEditor', 'startup')">
            <el-icon class="node-icon icon-system"><IconEpPointer /></el-icon>
            {{ t("scriptPanel.sections.startup") }}
          </el-menu-item>
          <el-menu-item index="shutdown" @dblclick="emit('openSystemEditor', 'shutdown')">
            <el-icon class="node-icon icon-system"><IconEpClose /></el-icon>
            {{ t("scriptPanel.sections.shutdown") }}
          </el-menu-item>
        </el-menu>
      </div>
    </div>
  </el-collapse-item>
</template>
