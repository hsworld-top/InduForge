<!--
  页面树 — 基础页面固定槽位列表
-->
<script setup lang="ts">
import { computed } from "vue";
import IconEpDocument from "~icons/ep/document";
import IconEpMoreFilled from "~icons/ep/more-filled";

interface BasicSlotLike {
  type: string;
  label: string;
  page?: { id: string; name?: string | null } | null;
}

const props = defineProps<{
  basicSlots: BasicSlotLike[];
  isPageActive: (pageId: string) => boolean;
  creatingBasicType?: string | null;
}>();

const emit = defineEmits(["slotClick", "slotDblClick", "rowAction"]);

const boundBasicPageCount = computed<number>(
  () => props.basicSlots.filter((basicSlot) => basicSlot.page).length,
);

/**
 * 处理基础页面行尾菜单命令并向父组件透传。
 * @param {string} command - 下拉菜单命令
 * @param {BasicSlotLike} basicSlot - 当前基础页面槽位
 * @returns {void}
 */
function handleRowActionCommand(command: string, basicSlot: BasicSlotLike): void {
  emit("rowAction", command, basicSlot);
}

/**
 * 获取基础页面副标题。
 * @param {BasicSlotLike} basicSlot - 基础页面槽位
 * @returns {string}
 */
function getBasicSlotMeta(basicSlot: BasicSlotLike): string {
  if (basicSlot.page) {
    return `已绑定：${basicSlot.page.name || basicSlot.page.id}`;
  }
  return "通过右侧菜单创建固定入口页";
}

/**
 * 获取空槽位创建文案。
 * @param {BasicSlotLike} basicSlot - 基础页面槽位
 * @returns {string}
 */
function getCreateActionLabel(basicSlot: BasicSlotLike): string {
  return `创建${basicSlot.label}`;
}
</script>

<template>
  <section class="page-section">
    <div class="page-section__header">
      <span class="page-section__title">基础页面</span>
      <div class="page-section__meta">
        <span class="page-section__badge">{{ boundBasicPageCount }}</span>
        <span class="page-section__hint">固定入口</span>
      </div>
    </div>
    <div v-if="basicSlots.length" class="page-section__body">
      <div
        v-for="basicSlot in basicSlots"
        :key="basicSlot.type"
        class="tree-node page-node page-node--basic"
        :class="{
          'is-active': basicSlot.page ? isPageActive(basicSlot.page.id) : false,
          'is-empty': !basicSlot.page,
          'is-creating': creatingBasicType === basicSlot.type,
        }"
        @click="emit('slotClick', basicSlot)"
        @dblclick="emit('slotDblClick', basicSlot)"
      >
        <div class="node-indent node-indent--basic" />
        <IconEpDocument class="node-icon page" />
        <div class="node-main">
          <div class="node-heading">
            <span class="node-label">{{ basicSlot.label }}</span>
            <span class="node-inline-tag" :class="{ 'is-empty': !basicSlot.page }">
              {{ basicSlot.page ? "固定入口" : "待创建" }}
            </span>
          </div>
          <span class="node-meta">
            {{ getBasicSlotMeta(basicSlot) }}
          </span>
        </div>
        <div class="node-side">
          <span class="node-chip" :class="{ 'is-empty': !basicSlot.page }">
            {{ basicSlot.page ? "已绑定" : "未创建" }}
          </span>
          <el-dropdown
            trigger="click"
            placement="bottom-end"
            @command="(command: string) => handleRowActionCommand(command, basicSlot)"
          >
            <el-button
              class="node-action-btn"
              :class="{ 'is-always-visible': !basicSlot.page }"
              text
              @click.stop
            >
              <IconEpMoreFilled />
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item v-if="basicSlot.page" command="open">打开</el-dropdown-item>
                <el-dropdown-item v-else command="create">
                  {{ getCreateActionLabel(basicSlot) }}
                </el-dropdown-item>
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
    </div>
    <div v-else class="page-section__empty">暂无基础页面</div>
  </section>
</template>
