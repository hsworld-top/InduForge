<!--
  脚本与变量面板 — 系统脚本折叠块
-->
<script setup lang="ts">
import IconEpClose from "~icons/ep/close";
import IconEpPointer from "~icons/ep/pointer";

defineProps<{
  selectedSystemKey: string;
}>();

const emit = defineEmits<{
  selectSystem: [key: string];
  openSystemEditor: [kind: "startup" | "shutdown"];
}>();
</script>

<template>
  <el-collapse-item name="system">
    <template #title> 系统脚本 </template>
    <div class="scripts-layout">
      <div class="scripts-list is-full">
        <el-menu
          :default-active="selectedSystemKey"
          class="list-menu"
          @select="(key: string) => emit('selectSystem', key)"
        >
          <el-menu-item index="startup" @dblclick="emit('openSystemEditor', 'startup')">
            <el-icon class="node-icon icon-system"><IconEpPointer /></el-icon>
            系统启动
          </el-menu-item>
          <el-menu-item index="shutdown" @dblclick="emit('openSystemEditor', 'shutdown')">
            <el-icon class="node-icon icon-system"><IconEpClose /></el-icon>
            系统关闭
          </el-menu-item>
        </el-menu>
      </div>
    </div>
  </el-collapse-item>
</template>
