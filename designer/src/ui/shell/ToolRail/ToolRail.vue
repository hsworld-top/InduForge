<!--
  ToolRail - 工具轨
  左右侧垂直工具按钮栏，用于切换页面树、组件、数据等面板
-->
<script setup lang="ts">
import type { ToolRailItem } from "../tool-rail-types";
import { computed } from "vue";

const props = withDefaults(
  defineProps<{
    side?: "left" | "right";
    items?: ToolRailItem[];
    activeKey?: string;
  }>(),
  {
    side: "left",
    items: () => [],
    activeKey: "",
  },
);

const emit = defineEmits<{
  select: [key: string];
}>();

const topItems = computed(() => props.items.filter((item) => item.placement !== "bottom"));

const bottomItems = computed(() => props.items.filter((item) => item.placement === "bottom"));

function handleSelect(key: string) {
  emit("select", key);
}
</script>

<template>
  <nav class="tool-rail" :class="`tool-rail--${side}`">
    <div class="tool-rail__section">
      <el-tooltip
        v-for="item in topItems"
        :key="item.key"
        :content="item.label"
        :placement="side === 'left' ? 'right-end' : 'left-end'"
        :show-after="300"
      >
        <button
          type="button"
          class="tool-rail__button"
          :class="{ 'is-active': item.key === activeKey }"
          @click="handleSelect(item.key)"
        >
          <component :is="item.icon" v-if="item.icon" class="tool-rail__icon" />
        </button>
      </el-tooltip>
    </div>
    <div class="tool-rail__section tool-rail__section--bottom">
      <el-tooltip
        v-for="item in bottomItems"
        :key="item.key"
        :content="item.label"
        :placement="side === 'left' ? 'right-end' : 'left-end'"
        :show-after="300"
      >
        <button
          type="button"
          class="tool-rail__button"
          :class="{ 'is-active': item.key === activeKey }"
          @click="handleSelect(item.key)"
        >
          <component :is="item.icon" v-if="item.icon" class="tool-rail__icon" />
        </button>
      </el-tooltip>
    </div>
  </nav>
</template>
