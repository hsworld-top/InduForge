<!--
  页面树 — 基础页面固定槽位列表
-->
<script setup>
import IconEpDocument from "~icons/ep/document";
import IconEpMoreFilled from "~icons/ep/more-filled";

defineProps({
  /** @type {unknown[]} */
  basicSlots: { type: Array, default: () => [] },
  isPageActive: { type: Function, required: true },
});

const emit = defineEmits(["slotClick", "slotDblClick", "rowAction"]);
</script>

<template>
  <section class="page-section">
    <div class="page-section__header">
      <span class="page-section__title">基础页面</span>
    </div>
    <div v-if="basicSlots.length" class="page-section__body">
      <div
        v-for="basicSlot in basicSlots"
        :key="basicSlot.type"
        class="tree-node page-node page-node--basic"
        :class="{
          'is-active': basicSlot.page ? isPageActive(basicSlot.page.id) : false,
          'is-empty': !basicSlot.page,
        }"
        @click="emit('slotClick', basicSlot)"
        @dblclick="emit('slotDblClick', basicSlot)"
      >
        <div class="node-indent node-indent--basic" />
        <IconEpDocument class="node-icon page" />
        <span class="node-label">{{ basicSlot.label }}</span>
        <el-dropdown
          trigger="click"
          placement="bottom-end"
          @command="(command) => emit('rowAction', command, basicSlot)"
        >
          <el-button class="node-action-btn" text @click.stop>
            <IconEpMoreFilled />
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item v-if="basicSlot.page" command="open">打开</el-dropdown-item>
              <el-dropdown-item v-else command="create">创建</el-dropdown-item>
              <el-dropdown-item v-if="basicSlot.page" command="export">导出页面</el-dropdown-item>
              <el-dropdown-item
                v-if="basicSlot.page && basicSlot.type !== 'home'"
                command="delete"
                divided
              >
                删除
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </div>
    <div v-else class="page-section__empty">暂无基础页面</div>
  </section>
</template>
