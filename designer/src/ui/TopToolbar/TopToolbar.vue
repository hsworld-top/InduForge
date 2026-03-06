<template>
  <header class="designer-toolbar toolbar-v2">
    <div class="toolbar-left">
      <div class="page-chip">
        <span class="page-chip__label">页面：</span>
        <span class="page-chip__name">{{ pageName || "未命名" }}</span>
        <span v-if="isDirty" class="page-chip__dirty">●</span>
      </div>
      <el-button class="icon-btn" @click="handleToggleLock">
        <IconLucideLock v-if="isLocked" />
        <IconLucideLockOpen v-else />
      </el-button>
    </div>

    <div class="toolbar-center">
      <div class="device-group">
        <el-tooltip v-for="item in viewItems" :key="item.key" :content="item.label" placement="bottom">
          <el-button class="icon-btn" :class="{ 'is-active': activeViewKey === item.key }"
            @click="handleViewChange(item.key)">
            <component :is="item.icon" :class="['view-mode-icon', item.iconClass]" />
          </el-button>
        </el-tooltip>
      </div>

      <div class="view-control-group">
        <el-select
          class="preset-select"
          size="small"
          :model-value="activeViewKey"
          @change="handleViewChange"
        >
          <el-option
            v-for="item in viewPresets"
            :key="item.key"
            :label="`${item.label || item.key} (${item.width}x${item.height})`"
            :value="item.key"
          />
        </el-select>
        <span class="metric-text">{{ canvasWidthText }}</span>
        <el-tooltip content="缩小" placement="bottom">
          <el-button class="icon-btn" @click="handleZoomOut">
            <IconLucideZoomOut />
          </el-button>
        </el-tooltip>
        <span class="zoom-pill">{{ Math.round(zoom * 100) }}%</span>
        <el-tooltip content="放大" placement="bottom">
          <el-button class="icon-btn" @click="handleZoomIn">
            <IconLucideZoomIn />
          </el-button>
        </el-tooltip>
        <el-button class="view-btn" @click="handleFitCanvas">适配画布</el-button>
        <el-button class="view-btn" @click="handleFitScreen">适配屏幕</el-button>
        <el-tooltip :content="showRuler ? '隐藏标尺' : '显示标尺'" placement="bottom">
          <el-button class="icon-btn" :class="{ 'is-active': showRuler }" @click="handleToggleRuler">
            <IconLucideRuler />
          </el-button>
        </el-tooltip>
        <el-tooltip :content="showGrid ? '隐藏网格' : '显示网格'" placement="bottom">
          <el-button class="icon-btn" :class="{ 'is-active': showGrid }" @click="handleToggleGrid">
            <IconLucideGrid3x3 />
          </el-button>
        </el-tooltip>
        <el-tooltip :content="enableSnap ? '关闭吸附' : '启用吸附'" placement="bottom">
          <el-button class="icon-btn" :class="{ 'is-active': enableSnap }" @click="handleToggleSnap">
            <IconLucideMagnet />
          </el-button>
        </el-tooltip>
      </div>
    </div>

    <div class="toolbar-right">
      <div class="tool-group">
        <el-button class="icon-btn" disabled>
          <IconLucideBot />
        </el-button>
        <el-button class="icon-btn" disabled>
          <IconLucideSun />
        </el-button>
        <el-tooltip content="清除当前界面" placement="bottom">
          <el-button class="icon-btn" @click="handleClearCanvas">
            <IconLucideTrash2 />
          </el-button>
        </el-tooltip>
        <el-button class="icon-btn" :disabled="!canUndo" @click="handleUndo">
          <IconLucideUndo2 />
        </el-button>
        <el-button class="icon-btn" :disabled="!canRedo" @click="handleRedo">
          <IconLucideRedo2 />
        </el-button>
        <el-dropdown class="preview-action" trigger="click" placement="bottom-end" popper-class="preview-menu-popper"
          split-button @click="handlePreview" @command="handlePreviewCommand">
          <IconLucidePlay />
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="pagePreview">页面预览</el-dropdown-item>
              <el-dropdown-item command="appPreview">应用预览</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>

      <div class="action-group">
        <el-button class="action-btn" @click="handleExport">
          <IconLucideDownload />
          导出
        </el-button>
        <el-dropdown ref="saveDropdownRef" class="save-action" trigger="click" placement="bottom-end"
          popper-class="save-settings-popper" split-button @click="handleSave">
          <span class="save-action__text">
            <IconLucideSave />
            保存
          </span>
          <template #dropdown>
            <div class="save-settings-panel" @click.stop>
              <div class="save-settings-panel__title">保存设置</div>
              <div class="save-settings-panel__row save-settings-panel__row--check">
                <el-checkbox v-model="localSaveSettings.autoSave" />
                <span class="save-settings-panel__label">自动保存</span>
              </div>
              <div class="save-settings-panel__row">
                <span class="save-settings-panel__label">保存间隔</span>
                <el-select v-model="localSaveSettings.intervalMinutes" class="save-settings-panel__select"
                  :disabled="!localSaveSettings.autoSave">
                  <el-option v-for="item in saveIntervalOptions" :key="item.value" :label="item.label"
                    :value="item.value" />
                </el-select>
              </div>
              <el-button class="save-settings-panel__submit" @click="handleSaveSettingsSubmit">
                设置并保存
              </el-button>
            </div>
          </template>
        </el-dropdown>
        <el-dropdown trigger="click" @command="handleMoreCommand">
          <el-button class="icon-btn">
            <IconLucideEllipsis />
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="collaboration">多人协作</el-dropdown-item>
              <el-dropdown-item command="refresh">刷新画布</el-dropdown-item>
              <el-dropdown-item command="locale">中英文切换</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </div>
  </header>
</template>

<script setup>
import { computed, ref, watch } from "vue";
import IconLucideLock from "~icons/lucide/lock";
import IconLucideLockOpen from "~icons/lucide/lock-open";
import IconLucideMonitor from "~icons/lucide/monitor";
import IconLucideLaptop from "~icons/lucide/laptop";
import IconLucideTablet from "~icons/lucide/tablet";
import IconLucideSmartphone from "~icons/lucide/smartphone";
import IconLucideBot from "~icons/lucide/bot";
import IconLucideSun from "~icons/lucide/sun";
import IconLucideZoomIn from "~icons/lucide/zoom-in";
import IconLucideZoomOut from "~icons/lucide/zoom-out";
import IconLucideRuler from "~icons/lucide/ruler";
import IconLucideGrid3x3 from "~icons/lucide/grid-3x3";
import IconLucideMagnet from "~icons/lucide/magnet";
import IconLucideUndo2 from "~icons/lucide/undo-2";
import IconLucideRedo2 from "~icons/lucide/redo-2";
import IconLucidePlay from "~icons/lucide/play";
import IconLucideDownload from "~icons/lucide/download";
import IconLucideSave from "~icons/lucide/save";
import IconLucideEllipsis from "~icons/lucide/ellipsis";
import IconLucideTrash2 from "~icons/lucide/trash-2";

const props = defineProps({
  pageName: { type: String, default: "未命名页面" },
  isLocked: { type: Boolean, default: false },
  isDirty: { type: Boolean, default: false },
  viewPresets: { type: Array, default: () => [] },
  activeViewKey: { type: String, default: "pc" },
  canUndo: { type: Boolean, default: false },
  canRedo: { type: Boolean, default: false },
  canMoveLayer: { type: Boolean, default: false },
  zoom: { type: Number, default: 1 },
  showRuler: { type: Boolean, default: true },
  showGrid: { type: Boolean, default: false },
  enableSnap: { type: Boolean, default: true },
  saving: { type: Boolean, default: false },
  saveSettings: {
    type: Object,
    default: () => ({ autoSave: false, intervalMinutes: 5 }),
  },
});

const emit = defineEmits([
  "update:activeViewKey",
  "toggleLock",
  "export",
  "undo",
  "redo",
  "preview",
  "previewApp",
  "save",
  "moveUp",
  "moveDown",
  "moveToTop",
  "moveToBottom",
  "openCollaboration",
  "refreshCanvas",
  "toggleLocale",
  "clearCanvas",
  "saveSettingsChange",
  "zoomIn",
  "zoomOut",
  "fitCanvas",
  "fitScreen",
  "toggleRuler",
  "toggleGrid",
  "toggleSnap",
]);

const viewItems = [
  { key: "bigscreen", label: "大屏", icon: IconLucideMonitor },
  { key: "pc", label: "PC", icon: IconLucideLaptop },
  { key: "tablet", label: "平板", icon: IconLucideTablet, iconClass: "is-tablet" },
  {
    key: "phoneLandscape",
    label: "手机横向",
    icon: IconLucideSmartphone,
    iconClass: "is-landscape",
  },
  {
    key: "phonePortrait",
    label: "手机竖屏",
    icon: IconLucideSmartphone,
    iconClass: "is-portrait",
  },
];

const canvasWidthText = computed(() => {
  const preset = props.viewPresets.find((item) => item.key === props.activeViewKey);
  return preset ? `${preset.width}px` : "-";
});

const saveIntervalOptions = [
  { value: 1, label: "1分钟" },
  { value: 3, label: "3分钟" },
  { value: 5, label: "5分钟" },
  { value: 10, label: "10分钟" },
  { value: 15, label: "15分钟" },
];

const localSaveSettings = ref({
  autoSave: false,
  intervalMinutes: 5,
});
const saveDropdownRef = ref(null);

watch(
  () => props.saveSettings,
  (value) => {
    localSaveSettings.value = {
      autoSave: Boolean(value?.autoSave),
      intervalMinutes: Number(value?.intervalMinutes) || 5,
    };
  },
  { immediate: true, deep: true }
);

const handleViewChange = (key) => emit("update:activeViewKey", key);
const handleToggleLock = () => emit("toggleLock");
const handleExport = () => emit("export");
const handleUndo = () => emit("undo");
const handleRedo = () => emit("redo");
const handlePreview = () => emit("preview");
const handleSave = () => emit("save");
const handleClearCanvas = () => emit("clearCanvas");
const handleZoomIn = () => emit("zoomIn");
const handleZoomOut = () => emit("zoomOut");
const handleFitCanvas = () => emit("fitCanvas");
const handleFitScreen = () => emit("fitScreen");
const handleToggleRuler = () => emit("toggleRuler");
const handleToggleGrid = () => emit("toggleGrid");
const handleToggleSnap = () => emit("toggleSnap");
const handlePreviewCommand = (command) => {
  if (command === "pagePreview") {
    emit("preview");
    return;
  }
  if (command === "appPreview") {
    emit("previewApp");
  }
};
const handleSaveSettingsSubmit = () => {
  emit("saveSettingsChange", { ...localSaveSettings.value });
  emit("save");
  saveDropdownRef.value?.handleClose?.();
};

const handleMoreCommand = (command) => {
  if (command === "collaboration") {
    emit("openCollaboration");
    return;
  }
  if (command === "refresh") {
    emit("refreshCanvas");
    return;
  }
  if (command === "locale") {
    emit("toggleLocale");
  }
};
</script>

<style scoped>
.toolbar-v2 {
  gap: 12px;
  padding: 0 12px;
}

.toolbar-left,
.toolbar-center,
.toolbar-right {
  display: flex;
  align-items: center;
  min-width: 0;
}

.toolbar-left {
  gap: 8px;
  width: 280px;
  flex-shrink: 0;
}

.toolbar-center {
  flex: 1;
  justify-content: center;
  gap: 12px;
}

.toolbar-right {
  justify-content: flex-end;
  gap: 8px;
  width: 620px;
  flex-shrink: 0;
  flex-wrap: nowrap;
}

.action-group {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-left: 32px;
  white-space: nowrap;
  flex-wrap: nowrap;
}

.page-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 34px;
  padding: 0 12px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #f3f4f6;
}

.page-chip__label {
  font-size: 14px;
  color: #4b5563;
}

.page-chip__name {
  font-size: 20px;
  line-height: 1;
  color: #111827;
  font-weight: 500;
}

.page-chip__dirty {
  color: #ef4444;
  font-size: 12px;
}

.device-group,
.tool-group {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.view-control-group {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 2px 8px;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  background: #f9fafb;
}

.preset-select {
  width: 170px;
}

:deep(.preset-select .el-select__wrapper) {
  min-height: 30px;
  border-radius: 8px;
  box-shadow: inset 0 0 0 1px #e5e7eb;
  background: #ffffff;
}

.metric-text {
  font-size: 13px;
  color: #4b5563;
  white-space: nowrap;
  margin-right: 4px;
}

.zoom-pill {
  min-width: 56px;
  height: 28px;
  padding: 0 8px;
  border-radius: 8px;
  border: 1px solid #e5e7eb;
  background: #ffffff;
  color: #374151;
  font-size: 13px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

:deep(.view-btn) {
  height: 30px;
  padding: 0 10px;
  border-radius: 8px;
  border: 1px solid #e5e7eb;
  background: #ffffff;
  color: #374151;
  font-size: 12px;
}

:deep(.icon-btn) {
  width: 30px;
  height: 30px;
  padding: 0;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  color: #374151;
  background: #ffffff;
}

:deep(.icon-btn.is-active) {
  color: #2563eb;
  border-color: #bfdbfe;
  background: #eff6ff;
}

:deep(.icon-btn .el-icon),
:deep(.icon-btn svg) {
  width: 16px;
  height: 16px;
}

:deep(.view-mode-icon) {
  width: 16px;
  height: 16px;
  transition: transform 0.15s ease;
}

:deep(.view-mode-icon.is-tablet) {
  transform: scale(1.05);
}

:deep(.view-mode-icon.is-landscape) {
  transform: rotate(90deg) scale(0.95);
}

:deep(.view-mode-icon.is-portrait) {
  transform: scale(0.95);
}

:deep(.preview-action .el-button-group) {
  display: inline-flex;
  flex-wrap: nowrap;
  align-items: center;
  vertical-align: middle;
}

:deep(.preview-action .el-button-group > .el-button),
:deep(.preview-action .el-button-group > .el-dropdown__caret-button) {
  width: 30px;
  height: 30px;
  padding: 0;
  border: 1px solid #e5e7eb;
  background: #ffffff;
  color: #374151;
}

:deep(.preview-action .el-button-group > .el-button:first-child) {
  border-top-left-radius: 8px;
  border-bottom-left-radius: 8px;
}

:deep(.preview-action .el-button-group > .el-dropdown__caret-button:last-child) {
  border-top-right-radius: 8px;
  border-bottom-right-radius: 8px;
  border-top-left-radius: 0;
  border-bottom-left-radius: 0;
}

:deep(.preview-action .el-button-group > .el-dropdown__caret-button .el-icon) {
  margin-left: 0;
  width: 14px;
  height: 14px;
}

:global(.preview-menu-popper .el-dropdown-menu) {
  padding: 4px;
}

:deep(.action-btn) {
  height: 34px;
  border-radius: 8px;
  padding: 0 12px;
  border: 1px solid #e5e7eb;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  line-height: 1;
}

:deep(.action-btn .el-icon),
:deep(.action-btn svg) {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
}

:deep(.action-btn--primary) {
  background: #ffffff;
  color: #111827;
  border-color: #d1d5db;
}

.save-action {
  display: inline-flex;
  white-space: nowrap;

  :deep(.el-button-group) {
    display: inline-flex;
    flex-wrap: nowrap;
    align-items: center;
    vertical-align: middle;
    white-space: nowrap;
  }
}

:deep(.save-action .el-button-group > .el-button),
:deep(.save-action .el-button-group > .el-dropdown__caret-button) {
  height: 34px;
  border-color: #d1d5db;
  background: #ffffff;
  color: #111827;
}

:deep(.save-action .el-button-group > .el-button:first-child) {
  padding: 0 12px;
  min-width: 74px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.save-action__text {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

:deep(.save-action .el-button-group > .el-dropdown__caret-button) {
  width: 34px;
  padding: 0;
}

:deep(.save-action .el-button-group > .el-dropdown__caret-button .el-icon) {
  margin-left: 0;
}

:deep(.save-action .el-button-group > .el-button:first-child) {
  border-top-left-radius: 8px;
  border-bottom-left-radius: 8px;
}

:deep(.save-action .el-button-group > .el-dropdown__caret-button:last-child) {
  border-radius: 8px;
  border-top-left-radius: 0;
  border-bottom-left-radius: 0;
}

:global(.save-settings-popper .save-settings-panel) {
  width: 320px;
  padding: 20px;
  border-radius: 12px;
  background: #ffffff;
  box-sizing: border-box;
  font-size: 14px;
}

:global(.save-settings-popper .el-dropdown-menu) {
  padding: 0;
  border: none;
  background: transparent;
}

:global(.save-settings-popper.el-popper) {
  padding: 0;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  box-shadow: 0 12px 32px rgba(15, 23, 42, 0.12);
  overflow: hidden;
}

:global(.save-settings-popper .save-settings-panel__title) {
  font-size: 16px;
  font-weight: 600;
  line-height: 1.4;
  color: #111827;
  margin-bottom: 14px;
}

:global(.save-settings-popper .save-settings-panel__row) {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  margin-bottom: 12px;
}

:global(.save-settings-popper .save-settings-panel__row--check) {
  gap: 10px;
}

:global(.save-settings-popper .save-settings-panel__row--check .el-checkbox) {
  margin-right: 0;
}

:global(.save-settings-popper .save-settings-panel__label) {
  font-size: 14px;
  line-height: 1.4;
  color: #374151;
}

:global(.save-settings-popper .save-settings-panel__select) {
  width: 150px;
  margin-left: auto;
}

:global(.save-settings-popper .save-settings-panel__select .el-input__wrapper) {
  min-height: 36px;
  border-radius: 8px;
  box-shadow: inset 0 0 0 1px #d1d5db;
  background: #f9fafb;
}

:global(.save-settings-popper .save-settings-panel__select .el-input__inner) {
  font-size: 14px;
  color: #374151;
}

:global(.save-settings-popper .save-settings-panel__row--check .el-checkbox__inner) {
  width: 16px;
  height: 16px;
  border-radius: 4px;
}

:global(.save-settings-popper .save-settings-panel__submit) {
  width: 100%;
  height: 36px;
  margin: 8px auto 0;
  display: block;
  border-radius: 8px;
  border: none;
  font-size: 14px;
  font-weight: 500;
  color: #fff;
  background: #18181b;
}
</style>
