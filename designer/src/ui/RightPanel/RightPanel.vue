<!--
  @deprecated 此组件已废弃
  请使用 ToolRail + DockPanel 架构替代
  新架构参见 DesignerView.vue 中的 rightRailItems 配置
  各功能面板已拆分为独立组件：PropertyPanel、StylePanel、AdvancedPanel
-->
<template>
  <aside class="panel panel-right">
    <div class="panel-header">
      <el-tabs v-model="activeTab" class="flex-1">
        <el-tab-pane label="属性" name="props" />
        <el-tab-pane label="样式" name="style" />
        <el-tab-pane label="事件" name="events" />
        <el-tab-pane label="绑定" name="bindings" />
      </el-tabs>
    </div>
    <div class="panel-body">
      <component :is="currentPanel" />
    </div>
  </aside>
</template>

<script setup>
import { computed, ref } from "vue";
import PropertyPanel from "./PropertyPanel.vue";
import StylePanel from "./StylePanel.vue";
import EventPanel from "./EventPanel.vue";
import BindingPanel from "./BindingPanel.vue";

/**
 * 当前激活的面板标签
 */
const activeTab = ref("props");

/**
 * 根据标签切换面板组件
 */
const currentPanel = computed(() => {
  const mapping = {
    props: PropertyPanel,
    style: StylePanel,
    events: EventPanel,
    bindings: BindingPanel,
  };
  return mapping[activeTab.value] || PropertyPanel;
});
</script>
