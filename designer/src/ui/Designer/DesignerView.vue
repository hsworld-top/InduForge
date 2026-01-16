<template>
  <div class="designer-layout">
    <!-- 顶部工具栏 -->
    <TopToolbar
      :page-name="pageName"
      :is-locked="isLocked"
      :is-dirty="isDirty"
      :view-presets="viewPresets"
      :active-view-key="activeViewKey"
      :can-undo="canUndoEnabled"
      :can-redo="canRedoEnabled"
      :can-move-layer="canMoveLayer"
      :zoom="zoom"
      :saving="saving"
      @update:activeViewKey="handleViewChange"
      @undo="handleUndo"
      @redo="handleRedo"
      @preview="handlePreview"
      @save="handleSave"
      @export="handleExport"
      @toggleLock="handleToggleLock"
      @moveUp="handleLayerMoveUp"
      @moveDown="handleLayerMoveDown"
      @moveToTop="handleLayerMoveToTop"
      @moveToBottom="handleLayerMoveToBottom"
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
          <template #actions>
            <el-tooltip v-if="leftActiveKey === 'pages'" content="新建页面">
              <el-button size="small" text @click="handlePageCreate">
                <IconEpPlus />
              </el-button>
            </el-tooltip>
          </template>
          <component
            :is="leftPanelComponent"
            v-bind="leftPanelProps"
            ref="leftPanelRef"
            @update:drawingTool="(value) => (drawingTool.value = value)"
          />
        </DockPanel>

        <div class="designer-canvas">
          <!-- 页面标签栏 -->
          <div v-if="pageTabs.length > 0" class="page-tabs-bar">
            <el-tabs
              v-model="activePageTabId"
              type="card"
              closable
              @tab-remove="handleClosePageTab"
            >
              <el-tab-pane
                v-for="tab in pageTabs"
                :key="tab.id"
                :name="tab.id"
                closable
              >
                <template #label>
                  <span class="page-tab-label">
                    <IconEpDocument class="tab-icon" />
                    <span class="tab-name">{{ tab.name }}</span>
                    <IconEpWarning
                      v-if="tab.isDirty"
                      class="tab-dirty-icon"
                      title="未保存"
                    />
                  </span>
                </template>
              </el-tab-pane>
            </el-tabs>
          </div>

          <!-- 画布容器 -->
          <template v-if="hasPages">
            <CanvasContainer
              :width="canvasWidth"
              :height="canvasHeight"
              :zoom="zoom"
              @zoomChange="handleZoomChange"
            />
          </template>

          <!-- 空页面提示 -->
          <div v-else class="empty-canvas-placeholder">
            <div class="empty-content">
              <IconEpDocument class="empty-icon" />
              <h3 class="empty-title">暂无页面</h3>
              <p class="empty-desc">创建一个新页面开始设计</p>
              <el-button type="primary" @click="handlePageCreate">
                <IconEpPlus class="mr-1" />
                新建页面
              </el-button>
            </div>
          </div>

          <DockPanel
            v-if="leftActiveKey && leftFloating"
            side="left"
            :title="leftPanelTitle"
            :floating="leftFloating"
            @close="handleLeftClose"
            @toggleFloating="toggleLeftFloating"
          >
            <template #actions>
              <el-tooltip v-if="leftActiveKey === 'pages'" content="新建页面">
                <el-button size="small" text @click="handlePageCreate">
                  <IconEpPlus />
                </el-button>
              </el-tooltip>
            </template>
            <component
              :is="leftPanelComponent"
              v-bind="leftPanelProps"
              ref="leftPanelRef"
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
import { computed, onBeforeUnmount, onMounted, ref, watch, provide } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
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
import {
  PropertyPanel,
  AdvancedPanel,
  VariablesPanel,
} from "@/ui/RightPanel";
import { VIEW_PRESETS } from "@/constants";
import IconEpDocument from "~icons/ep/document";
import IconEpMenu from "~icons/ep/menu";
import IconEpBox from "~icons/ep/box";
import IconEpDataAnalysis from "~icons/ep/data-analysis";
import IconEpEdit from "~icons/ep/edit";
import IconEpChatDotRound from "~icons/ep/chat-dot-round";
import IconEpUser from "~icons/ep/user";
import IconEpTools from "~icons/ep/tools";
import IconEpSetting from "~icons/ep/setting";
import IconEpList from "~icons/ep/list";
import IconEpPlus from "~icons/ep/plus";
import IconEpWarning from "~icons/ep/warning";

const route = useRoute();
const router = useRouter();

const editorStore = useEditorStore();
const {
  canUndo,
  canRedo,
  saving,
  selection,
  currentPageId,
  currentPage,
  isLocked,
  readonlyState,
} = storeToRefs(editorStore);

const zoom = ref(1);
const activeViewKey = ref("pc");
const viewPresets = VIEW_PRESETS;

const drawingTool = ref("");

// ==================== 页面标签页系统 ====================
/**
 * @typedef {Object} PageTab
 * @property {string} id - 页面ID
 * @property {string} name - 页面名称
 * @property {boolean} isDirty - 是否有未保存的修改
 */

/** @type {import('vue').Ref<PageTab[]>} */
const pageTabs = ref([]);
const activePageTabId = ref("");

/**
 * 打开页面标签页
 * @param {string} pageId - 页面ID
 */
const openPageTab = (pageId) => {
  const page = editorStore.pages.find((p) => p.id === pageId);
  if (!page) return;

  // 检查是否已打开
  const existingTab = pageTabs.value.find((t) => t.id === pageId);
  if (existingTab) {
    // 只更新 activePageTabId，watch 会自动调用 setCurrentPage
    activePageTabId.value = pageId;
    return;
  }

  // 添加新标签页
  pageTabs.value.push({
    id: pageId,
    name: page.name || "未命名页面",
    isDirty: false,
  });

  // 只更新 activePageTabId，watch 会自动调用 setCurrentPage
  activePageTabId.value = pageId;
};

/**
 * 关闭页面标签页
 * @param {string} tabId - 标签页ID
 */
const handleClosePageTab = async (tabId) => {
  const index = pageTabs.value.findIndex((t) => t.id === tabId);
  if (index === -1) return;

  const tab = pageTabs.value[index];

  // 如果有未保存的修改，弹窗确认
  if (tab.isDirty) {
    try {
      const action = await ElMessageBox.confirm(
        `页面 "${tab.name}" 有未保存的修改，是否保存后关闭？`,
        "关闭确认",
        {
          distinguishCancelAndClose: true,
          confirmButtonText: "保存并关闭",
          cancelButtonText: "不保存",
          type: "warning",
        }
      );

      if (action === "confirm") {
        // 先切换到该页面再保存
        if (currentPageId.value !== tabId) {
          editorStore.setCurrentPage(tabId);
          await new Promise((resolve) => setTimeout(resolve, 100));
        }
        await editorStore.saveCurrentPage();
        ElMessage.success("页面已保存");
      }
    } catch (action) {
      if (action === "close") {
        // 用户点击关闭按钮，取消操作
        return;
      }
      // action === 'cancel'，不保存直接关闭
    }
  }

  // 执行关闭
  pageTabs.value.splice(index, 1);

  // 如果关闭的是当前标签页
  if (activePageTabId.value === tabId) {
    if (pageTabs.value.length > 0) {
      // 切换到其他标签页
      const newActiveTab =
        pageTabs.value[Math.min(index, pageTabs.value.length - 1)];
      activePageTabId.value = newActiveTab.id;
      editorStore.setCurrentPage(newActiveTab.id);
    } else {
      // 没有标签页了，清空状态
      activePageTabId.value = "";
    }
  }
};

/**
 * 更新标签页脏状态
 */
const updateTabDirtyState = () => {
  const tab = pageTabs.value.find((t) => t.id === currentPageId.value);
  if (tab) {
    tab.isDirty = canUndo.value;
  }
};

/**
 * 更新标签页名称
 */
const updateTabName = () => {
  const tab = pageTabs.value.find((t) => t.id === currentPageId.value);
  if (tab) {
    // 从 pages 列表获取最新的页面名称
    const page = editorStore.pages.find((p) => p.id === currentPageId.value);
    tab.name = page?.name || "未命名页面";
  }
};

// 监听 canUndo 变化，更新标签页脏状态
watch(canUndo, updateTabDirtyState);

// 监听页面列表变化，更新标签页名称
watch(
  [() => editorStore.pages, currentPageId],
  () => {
    updateTabName();
  },
  { deep: true }
);

// 监听标签页切换，同步到 currentPageId
watch(
  activePageTabId,
  async (newTabId, oldTabId) => {
    // 标签页切换时，直接同步到 store
    if (newTabId && newTabId !== oldTabId) {
      await editorStore.setCurrentPage(newTabId);
    }
  },
  { flush: "sync" }
);

// 初始化时打开当前页面
watch(
  currentPageId,
  (newPageId) => {
    if (newPageId && !pageTabs.value.find((t) => t.id === newPageId)) {
      // ✅ 验证页面是否真的存在
      const pageExists = editorStore.pages.some((p) => p.id === newPageId);
      if (pageExists) {
        openPageTab(newPageId);
      }
    }
  },
  { immediate: true }
);

// 提供 openPageTab 方法给子组件
provide("openPageTab", openPageTab);

/**
 * 是否有页面
 */
const hasPages = computed(() => {
  // ✅ 确保页面列表不为空且当前页面确实存在
  if (editorStore.pages.length === 0) return false;
  if (!currentPageId.value) return false;
  return editorStore.pages.some((p) => p.id === currentPageId.value);
});
// ==================== 页面标签页系统结束 ====================
const canUndoEnabled = computed(
  () => canUndo.value && !readonlyState.value?.readonly
);
const canRedoEnabled = computed(
  () => canRedo.value && !readonlyState.value?.readonly
);

/**
 * 是否可以移动图层
 */
const canMoveLayer = computed(() => {
  if (readonlyState.value?.readonly) return false;
  const primary = selection.value?.getPrimaryElement();
  if (!primary || primary.kind !== "node") return false;
  // 不能移动根节点
  const rootNodeId = currentPage.value?.rootNodeId;
  return primary.id !== rootNodeId;
});
const leftActiveKey = ref("pages");
const rightActiveKey = ref("props");
const leftFloating = ref(false);
const rightFloating = ref(false);
const leftPanelRef = ref(null);

const pageName = computed(() => {
  // ✅ 如果没有页面，返回空
  if (!hasPages.value) return "";
  // 优先从 pages 列表获取名称（更可靠）
  const pageFromList = editorStore.pages.find(
    (p) => p.id === currentPageId.value
  );
  if (pageFromList?.name) return pageFromList.name;
  // 其次从 doc 中获取
  const page = currentPage.value;
  return page?.name || "";
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
  { key: "advanced", label: "高级", icon: IconEpSetting },
  { key: "variables", label: "变量", icon: IconEpList },
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
    case "advanced":
      return AdvancedPanel;
    case "variables":
      return VariablesPanel;
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

    // 保存成功后清除当前标签页的脏状态
    const tab = pageTabs.value.find((t) => t.id === currentPageId.value);
    if (tab) {
      tab.isDirty = false;
    }

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
const handleToggleLock = async () => {
  const result = await editorStore.togglePageLock();
  if (!result) return;

  if (result.success) {
    const message =
      result.action === "release" ? "已释放页面锁" : "已获取页面锁";
    ElMessage.success(message);
    return;
  }

  if (result.reason === "locked") {
    ElMessage.warning(`页面已被${result.lockedByName || "其他用户"}锁定`);
    return;
  }

  ElMessage.error(result.error?.message || "页面锁操作失败");
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
 * 更新缩放比例
 * @param {number} value - 缩放比例
 */
const handleZoomChange = (value) => {
  zoom.value = value;
};

/**
 * 打开页面新建弹窗
 * @returns {void}
 */
const handlePageCreate = () => {
  if (leftActiveKey.value !== "pages") return;
  leftPanelRef.value?.openCreateDialog?.();
};

/**
 * 工具栏:上移图层
 */
const handleLayerMoveUp = () => {
  if (editorStore.moveNodeUp()) {
    ElMessage.success("已上移");
  }
};

/**
 * 工具栏:下移图层
 */
const handleLayerMoveDown = () => {
  if (editorStore.moveNodeDown()) {
    ElMessage.success("已下移");
  }
};

/**
 * 工具栏:置顶
 */
const handleLayerMoveToTop = () => {
  if (editorStore.moveNodeToTop()) {
    ElMessage.success("已置顶");
  }
};

/**
 * 工具栏:置底
 */
const handleLayerMoveToBottom = () => {
  if (editorStore.moveNodeToBottom()) {
    ElMessage.success("已置底");
  }
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

onBeforeUnmount(() => {
  void editorStore.releasePageLock();
});
</script>

<style scoped>
/* 页面标签栏样式 */
.page-tabs-bar {
  flex-shrink: 0;
  background: #f5f7fa;
  border-bottom: 1px solid #e4e7ed;
}

.dark .page-tabs-bar {
  background: #1a1a1a;
  border-bottom-color: #3a3a3a;
}

.page-tabs-bar :deep(.el-tabs__header) {
  margin: 0;
  border-bottom: none;
}

.page-tabs-bar :deep(.el-tabs__nav-wrap::after) {
  display: none;
}

.page-tabs-bar :deep(.el-tabs__item) {
  height: 36px;
  line-height: 36px;
  border: none !important;
  background: transparent;
  color: #606266;
  padding: 0 16px;
}

.page-tabs-bar :deep(.el-tabs__item.is-active) {
  background: white;
  color: #409eff;
  border-bottom: 2px solid #409eff !important;
}

.dark .page-tabs-bar :deep(.el-tabs__item.is-active) {
  background: #2a2a2a;
}

.page-tab-label {
  display: flex;
  align-items: center;
  gap: 6px;
}

.page-tab-label .tab-icon {
  width: 14px;
  height: 14px;
  color: #909399;
}

.page-tab-label .tab-name {
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.page-tab-label .tab-dirty-icon {
  width: 12px;
  height: 12px;
  color: #e6a23c;
}

/* 空页面提示 */
.empty-canvas-placeholder {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #f5f7fa 0%, #e4e7ed 100%);
}

.dark .empty-canvas-placeholder {
  background: linear-gradient(135deg, #1a1a1a 0%, #2a2a2a 100%);
}

.empty-content {
  text-align: center;
  padding: 40px;
}

.empty-icon {
  width: 80px;
  height: 80px;
  color: #c0c4cc;
  margin-bottom: 20px;
}

.dark .empty-icon {
  color: #4a4a4a;
}

.empty-title {
  font-size: 20px;
  font-weight: 500;
  color: #606266;
  margin: 0 0 8px 0;
}

.dark .empty-title {
  color: #d1d5db;
}

.empty-desc {
  font-size: 14px;
  color: #909399;
  margin: 0 0 24px 0;
}
</style>
