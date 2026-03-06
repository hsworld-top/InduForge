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
          <component v-if="item.icon" :is="item.icon" class="tool-rail__icon" />
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
          <component v-if="item.icon" :is="item.icon" class="tool-rail__icon" />
        </button>
      </el-tooltip>
    </div>
  </nav>
</template>

<script setup>
import { computed } from "vue";

const props = defineProps({
  side: {
    type: String,
    default: "left",
  },
  items: {
    type: Array,
    default: () => [],
  },
  activeKey: {
    type: String,
    default: "",
  },
});

const emit = defineEmits(["select"]);

const topItems = computed(() =>
  props.items.filter((item) => item.placement !== "bottom")
);

const bottomItems = computed(() =>
  props.items.filter((item) => item.placement === "bottom")
);

/**
 * 选择工具栏入口
 * @param {string} key - 工具键值
 */
const handleSelect = (key) => {
  emit("select", key);
};
</script>
