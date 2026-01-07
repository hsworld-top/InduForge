<template>
  <div class="flex flex-col h-screen bg-gray-100 dark:bg-gray-900">
    <!-- 工具栏 -->
    <header
      class="flex items-center justify-between h-12 px-3 bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700"
    >
      <div class="flex items-center gap-2">
        <el-button size="small" @click="handleBack">
          <IconEpArrowLeft />
          返回编辑
        </el-button>
        <el-divider direction="vertical" />
        <span class="text-sm font-medium">预览模式</span>
      </div>

      <div class="flex items-center gap-2">
        <el-radio-group v-model="viewKey" size="small">
          <el-radio-button
            v-for="preset in viewPresets"
            :key="preset.key"
            :value="preset.key"
          >
            {{ preset.label }}
          </el-radio-button>
        </el-radio-group>
      </div>

      <div class="flex items-center gap-2">
        <el-button size="small" type="primary" @click="handleRefresh">
          <IconEpRefresh />
          刷新
        </el-button>
      </div>
    </header>

    <!-- 预览内容 -->
    <main class="flex-1 flex items-center justify-center p-6 overflow-auto">
      <div
        class="bg-white dark:bg-gray-800 shadow-lg rounded-lg overflow-hidden transition-all duration-300"
        :style="frameStyle"
      >
        <div
          class="flex flex-col items-center justify-center h-full min-h-[400px] text-gray-400"
        >
          <IconEpView class="text-6xl text-blue-500 mb-4" />
          <h3 class="text-lg font-medium mb-2">预览模式</h3>
          <p class="text-sm">这里将渲染设计器中的页面内容</p>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup>
import { ref, computed } from "vue";
import { useRouter, useRoute } from "vue-router";

// 图标导入
import IconEpArrowLeft from "~icons/ep/arrow-left";
import IconEpRefresh from "~icons/ep/refresh";
import IconEpView from "~icons/ep/view";
import { VIEW_PRESETS } from "@/constants";

const router = useRouter();
const route = useRoute();

const viewKey = ref("pc");
const viewPresets = VIEW_PRESETS;

// 预览框样式
const frameStyle = computed(() => {
  const preset = viewPresets.find((item) => item.key === viewKey.value);
  const size = preset
    ? { width: `${preset.width}px`, height: `${preset.height}px` }
    : { width: "100%", height: "100%" };
  return {
    width: size.width,
    height: size.height,
    maxWidth: "100%",
    maxHeight: "100%",
  };
});

/**
 * 返回编辑
 */
const handleBack = () => {
  const projectId = route.query.pid;
  router.push({ path: "/", query: { pid: projectId } });
};

/**
 * 刷新预览
 */
const handleRefresh = () => {
  // TODO: 重新加载数据
  console.log("刷新预览");
};
</script>

