<template>
  <div class="flex flex-col gap-3">
    <el-tabs v-model="activeTab">
      <el-tab-pane label="组件" name="components">
        <ComponentPanel />
      </el-tab-pane>
      <el-tab-pane label="绘图" name="drawing">
        <DrawingPanel v-model="drawingTool" />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup>
import { computed, ref } from "vue";
import ComponentPanel from "./ComponentPanel.vue";
import DrawingPanel from "./DrawingPanel.vue";

const props = defineProps({
  drawingTool: {
    type: String,
    default: "",
  },
});

const emit = defineEmits(["update:drawingTool"]);

const activeTab = ref("components");

const drawingTool = computed({
  get: () => props.drawingTool,
  set: (value) => emit("update:drawingTool", value),
});
</script>
