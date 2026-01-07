<template>
  <div class="designer-layout">
    <!-- 顶部工具栏 -->
    <TopToolbar
      :page-name="pageName"
      :is-locked="isLocked"
      :is-dirty="isDirty"
      :view-presets="viewPresets"
      :active-view-key="activeViewKey"
      :can-undo="canUndo"
      :can-redo="canRedo"
      :zoom="zoom"
      :saving="saving"
      @update:activeViewKey="handleViewChange"
      @undo="handleUndo"
      @redo="handleRedo"
      @preview="handlePreview"
      @save="handleSave"
      @export="handleExport"
      @toggleLock="handleToggleLock"
    />

    <!-- 主体区域 -->
    <div class="designer-main">
      <ToolRail
        side="left"
        :items="leftRailItems"
        :active-key="leftActiveKey"
        @select="handleLeftSelect"
      />

      <div class="designer-workspace">
        <DockPanel
          v-if="leftActiveKey && !leftFloating"
          side="left"
          :title="leftPanelTitle"
          :floating="leftFloating"
          @close="handleLeftClose"
          @toggleFloating="toggleLeftFloating"
        >
          <component
            :is="leftPanelComponent"
            v-bind="leftPanelProps"
            @update:drawingTool="(value) => (drawingTool.value = value)"
          />
        </DockPanel>

        <div class="designer-canvas">
          <CanvasContainer
            :width="canvasWidth"
            :height="canvasHeight"
            :zoom="zoom"
          />

          <DockPanel
            v-if="leftActiveKey && leftFloating"
            side="left"
            :title="leftPanelTitle"
            :floating="leftFloating"
            @close="handleLeftClose"
            @toggleFloating="toggleLeftFloating"
          >
            <component
              :is="leftPanelComponent"
              v-bind="leftPanelProps"
              @update:drawingTool="(value) => (drawingTool.value = value)"
            />
          </DockPanel>

          <DockPanel
            v-if="rightActiveKey && rightFloating"
            side="right"
            :title="rightPanelTitle"
            :floating="rightFloating"
            @close="handleRightClose"
            @toggleFloating="toggleRightFloating"
          >
            <component :is="rightPanelComponent" />
          </DockPanel>
        </div>

        <DockPanel
          v-if="rightActiveKey && !rightFloating"
          side="right"
          :title="rightPanelTitle"
          :floating="rightFloating"
          @close="handleRightClose"
          @toggleFloating="toggleRightFloating"
        >
          <component :is="rightPanelComponent" />
        </DockPanel>
      </div>

      <ToolRail
        side="right"
        :items="rightRailItems"
        :active-key="rightActiveKey"
        @select="handleRightSelect"
      />
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ElMessage } from "element-plus";
import { storeToRefs } from "pinia";
import { useEditorStore } from "@/stores/editor-store";
import { CanvasContainer } from "@/ui/Canvas";
import { TopToolbar } from "@/ui/TopToolbar";
import { ToolRail } from "@/ui/ToolRail";
import { DockPanel } from "@/ui/DockPanel";
import {
  PageTree,
  OutlineTree,
  MaterialPanel,
  DataPanel,
  I18nPanel,
  ScriptVarsPanel,
  AiPanel,
  RolePanel,
} from "@/ui/LeftPanel";
import { PropertyPanel, StylePanel, AdvancedPanel } from "@/ui/RightPanel";
import { VIEW_PRESETS } from "@/constants";
import IconEpDocument from "~icons/ep/document";
import IconEpMenu from "~icons/ep/menu";
import IconEpBox from "~icons/ep/box";
import IconEpDataAnalysis from "~icons/ep/data-analysis";
import IconEpEdit from "~icons/ep/edit";
import IconEpChatDotRound from "~icons/ep/chat-dot-round";
import IconEpUser from "~icons/ep/user";
import IconEpTools from "~icons/ep/tools";
import IconEpBrush from "~icons/ep/brush";
import IconEpSetting from "~icons/ep/setting";

const route = useRoute();
const router = useRouter();

const editorStore = useEditorStore();
const { canUndo, canRedo, saving, currentPageId, currentPage } =
  storeToRefs(editorStore);

const zoom = ref(1);
const activeViewKey = ref("pc");
const viewPresets = VIEW_PRESETS;

const drawingTool = ref("");
const isLocked = ref(false);
const leftActiveKey = ref("pages");
const rightActiveKey = ref("props");
const leftFloating = ref(false);
const rightFloating = ref(false);

const pageName = computed(() => {
  const page = currentPage.value;
  return page?.name || page?.id || "未命名页面";
});

const isDirty = computed(() => canUndo.value);

const activeView = computed(() =>
  viewPresets.find((preset) => preset.key === activeViewKey.value)
);

const canvasWidth = computed(() => activeView.value?.width || 1920);
const canvasHeight = computed(() => activeView.value?.height || 1080);

const leftRailItems = [
  { key: "pages", label: "页面", icon: IconEpDocument },
  { key: "outline", label: "大纲", icon: IconEpMenu },
  { key: "material", label: "物料", icon: IconEpBox },
  { key: "data", label: "数据", icon: IconEpDataAnalysis },
  { key: "i18n", label: "国际", icon: IconEpDocument },
  { key: "script", label: "脚本", icon: IconEpEdit },
  { key: "ai", label: "AI", icon: IconEpChatDotRound },
  { key: "role", label: "角色", icon: IconEpUser, placement: "bottom" },
];

const rightRailItems = [
  { key: "props", label: "属性", icon: IconEpTools },
  { key: "style", label: "样式", icon: IconEpBrush },
  { key: "advanced", label: "高级", icon: IconEpSetting },
];

const leftPanelComponent = computed(() => {
  switch (leftActiveKey.value) {
    case "pages":
      return PageTree;
    case "outline":
      return OutlineTree;
    case "material":
      return MaterialPanel;
    case "data":
      return DataPanel;
    case "i18n":
      return I18nPanel;
    case "script":
      return ScriptVarsPanel;
    case "ai":
      return AiPanel;
    case "role":
      return RolePanel;
    default:
      return PageTree;
  }
});

const leftPanelTitle = computed(() => {
  const item = leftRailItems.find((entry) => entry.key === leftActiveKey.value);
  return item?.label || "面板";
});

const leftPanelProps = computed(() => {
  if (leftActiveKey.value === "material") {
    return { drawingTool: drawingTool.value };
  }
  return {};
});

const rightPanelComponent = computed(() => {
  switch (rightActiveKey.value) {
    case "props":
      return PropertyPanel;
    case "style":
      return StylePanel;
    case "advanced":
      return AdvancedPanel;
    default:
      return PropertyPanel;
  }
});

const rightPanelTitle = computed(() => {
  const item = rightRailItems.find(
    (entry) => entry.key === rightActiveKey.value
  );
  return item?.label || "面板";
});


/**
 * 撤销操作
 */
const handleUndo = () => {
  if (!editorStore.undo()) {
    ElMessage.info("没有可撤销的操作");
  }
};

/**
 * 重做操作
 */
const handleRedo = () => {
  if (!editorStore.redo()) {
    ElMessage.info("没有可重做的操作");
  }
};

/**
 * 预览
 */
const handlePreview = () => {
  const projectId = route.meta.project?.id;
  router.push({ path: "/preview", query: { pid: projectId } });
};

/**
 * 保存
 */
const handleSave = async () => {
  try {
    await editorStore.saveCurrentPage();
    ElMessage.success("保存成功");
  } catch (error) {
    const message = error instanceof Error ? error.message : "未知错误";
    ElMessage.error("保存失败: " + message);
  }
};

/**
 * 切换视图
 * @param {string} key - 视图键值
 */
const handleViewChange = (key) => {
  activeViewKey.value = key;
  const targetView = viewPresets.find((preset) => preset.key === key);
  if (!targetView || !currentPage.value) return;
  const nextConfig = {
    ...currentPage.value.config,
    width: targetView.width,
    height: targetView.height,
  };
  editorStore.updateCurrentPage({ config: nextConfig });
};

/**
 * 切换锁定状态
 */
const handleToggleLock = () => {
  isLocked.value = !isLocked.value;
};

/**
 * 导出页面 Schema
 */
const handleExport = () => {
  if (!editorStore.doc || !currentPageId.value) {
    ElMessage.warning("暂无可导出的页面");
    return;
  }
  const payload = editorStore.serializer.exportPage(
    editorStore.doc,
    currentPageId.value
  );
  const json = JSON.stringify(payload, null, 2);
  const blob = new Blob([json], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const name = `${pageName.value || "page"}.json`;
  const link = document.createElement("a");
  link.href = url;
  link.download = name;
  link.click();
  URL.revokeObjectURL(url);
  ElMessage.success("已导出页面");
};

const handleLeftSelect = (key) => {
  leftActiveKey.value = leftActiveKey.value === key ? "" : key;
};

const handleRightSelect = (key) => {
  rightActiveKey.value = rightActiveKey.value === key ? "" : key;
};

const handleLeftClose = () => {
  leftActiveKey.value = "";
};

const handleRightClose = () => {
  rightActiveKey.value = "";
};

const toggleLeftFloating = () => {
  leftFloating.value = !leftFloating.value;
};

const toggleRightFloating = () => {
  rightFloating.value = !rightFloating.value;
};

/**
 * 加载工程数据
 */
const loadProject = async () => {
  const project = route.meta.project;
  if (!project?.id) return;

  const result = await editorStore.loadProject(project.id);
  if (!result.ok) {
    ElMessage.error(result.error?.message || "加载工程失败");
  }
};

onMounted(() => {
  void loadProject();
});
</script>

