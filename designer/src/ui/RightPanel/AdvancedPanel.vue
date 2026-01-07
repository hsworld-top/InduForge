<template>
  <div class="flex flex-col gap-3">
    <el-tabs v-model="activeTab">
      <el-tab-pane label="事件" name="events">
        <EventPanel />
      </el-tab-pane>
      <el-tab-pane label="绑定" name="bindings">
        <BindingPanel />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup>
import { ref } from "vue";
import EventPanel from "./EventPanel.vue";
import BindingPanel from "./BindingPanel.vue";

const activeTab = ref("events");
</script>
