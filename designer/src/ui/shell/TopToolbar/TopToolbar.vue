<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import IconEpArrowDownBold from "~icons/ep/arrow-down-bold";
import IconLucideBot from "~icons/lucide/bot";
import IconLucideClipboardPaste from "~icons/lucide/clipboard-paste";
import IconLucideCopy from "~icons/lucide/copy";
import IconLucideDownload from "~icons/lucide/download";
import IconLucideEllipsis from "~icons/lucide/ellipsis";
import IconLucideLock from "~icons/lucide/lock";
import IconLucideLockOpen from "~icons/lucide/lock-open";
import IconLucidePlay from "~icons/lucide/play";
import IconLucideRedo2 from "~icons/lucide/redo-2";
import IconLucideSave from "~icons/lucide/save";
import IconLucideSettings2 from "~icons/lucide/settings-2";
import IconLucideTrash from "~icons/lucide/trash";
import IconLucideTrash2 from "~icons/lucide/trash-2";
import IconLucideUndo2 from "~icons/lucide/undo-2";
import IconLucideZoomIn from "~icons/lucide/zoom-in";
import IconLucideZoomOut from "~icons/lucide/zoom-out";
import { ElMessage } from "../el-message-compat";

interface ToolbarSaveSettings {
  autoSave: boolean;
  intervalMinutes: number;
}

/** 与 constants 中 ViewPreset 结构一致；单独声明避免 props 默认 [] 与 readonly 字面量联合退化为 never */
interface ViewPresetProp {
  key: string;
  label?: string;
  width: number;
  height: number;
}

interface RuntimeUserProp {
  id: string;
  username: string;
  displayName?: string;
  status?: string;
}

const props = withDefaults(
  defineProps<{
    pageName?: string;
    isLocked?: boolean;
    isDirty?: boolean;
    viewPresets?: readonly ViewPresetProp[];
    activeViewKey?: string;
    canvasWidth?: number;
    canvasHeight?: number;
    isCustomView?: boolean;
    canUndo?: boolean;
    canRedo?: boolean;
    canMoveLayer?: boolean;
    zoom?: number;
    showRuler?: boolean;
    showGrid?: boolean;
    enableSnap?: boolean;
    isSaving?: boolean;
    saveSettings?: ToolbarSaveSettings;
    hasSelection?: boolean;
    hasClipboard?: boolean;
    runtimeUsers?: readonly RuntimeUserProp[];
    selectedPreviewRuntimeUserId?: string;
  }>(),
  {
    pageName: "",
    isLocked: false,
    isDirty: false,
    viewPresets: () => [],
    activeViewKey: "pc",
    canvasWidth: 1366,
    canvasHeight: 768,
    isCustomView: false,
    canUndo: false,
    canRedo: false,
    canMoveLayer: false,
    zoom: 1,
    showRuler: true,
    showGrid: false,
    enableSnap: true,
    isSaving: false,
    saveSettings: () => ({ autoSave: false, intervalMinutes: 5 }),
    hasSelection: false,
    hasClipboard: false,
    runtimeUsers: () => [],
    selectedPreviewRuntimeUserId: "",
  },
);

const { t } = useI18n();

const emit = defineEmits<{
  "update:activeViewKey": [key: string];
  applyCustomSize: [size: { width: number; height: number }];
  toggleLock: [];
  export: [];
  undo: [];
  redo: [];
  preview: [];
  previewApp: [];
  save: [];
  openAi: [];
  moveUp: [];
  moveDown: [];
  moveToTop: [];
  moveToBottom: [];
  openCollaboration: [];
  refreshCanvas: [];
  clearCanvas: [];
  saveSettingsChange: [settings: ToolbarSaveSettings];
  zoomIn: [];
  zoomOut: [];
  fitCanvas: [];
  fitScreen: [];
  toggleRuler: [];
  toggleGrid: [];
  toggleSnap: [];
  copy: [];
  paste: [];
  deleteSelected: [];
  previewUserChange: [runtimeUserId: string];
}>();

const MIN_CANVAS_WIDTH = 120;
const MAX_CANVAS_WIDTH = 7680;
const MIN_CANVAS_HEIGHT = 120;
const MAX_CANVAS_HEIGHT = 4320;

const viewItems = computed(() =>
  props.viewPresets.map((item) => ({
    key: item.key,
    label: item.label || item.key,
    width: item.width,
    height: item.height,
  })),
);

const currentCanvasSizeText = computed(() => `${Math.round(props.canvasWidth)}px`);

const saveStatusText = computed(() => {
  if (props.isSaving) return t("toolbar.saveStatus.saving");
  if (props.isDirty) return t("toolbar.saveStatus.dirty");
  return t("toolbar.saveStatus.saved");
});

const saveStatusClass = computed(() => {
  if (props.isSaving) return "is-saving";
  if (props.isDirty) return "is-dirty";
  return "is-saved";
});

const saveIntervalOptions = computed(() => [
  { value: 1, label: t("toolbar.saveIntervalOption", { value: 1 }) as string },
  { value: 3, label: t("toolbar.saveIntervalOption", { value: 3 }) as string },
  { value: 5, label: t("toolbar.saveIntervalOption", { value: 5 }) as string },
  { value: 10, label: t("toolbar.saveIntervalOption", { value: 10 }) as string },
  { value: 15, label: t("toolbar.saveIntervalOption", { value: 15 }) as string },
]);

const previewUserOptions = computed(() =>
  props.runtimeUsers
    .filter((user) => user.status !== "disabled")
    .map((user) => ({
      value: user.id,
      label: user.username,
    })),
);

const selectedPreviewUser = computed<string>({
  get: () => props.selectedPreviewRuntimeUserId || "",
  set: (value) => emit("previewUserChange", value),
});

const localSaveSettings = ref({
  autoSave: false,
  intervalMinutes: 5,
});
const localCustomSize = ref({
  width: props.canvasWidth,
  height: props.canvasHeight,
});
const saveDropdownRef = ref<{ handleClose?: () => void } | null>(null);
const previewDropdownRef = ref<{ handleClose?: () => void } | null>(null);

watch(
  () => props.saveSettings,
  (value: ToolbarSaveSettings | undefined) => {
    localSaveSettings.value = {
      autoSave: Boolean(value?.autoSave),
      intervalMinutes: Number(value?.intervalMinutes) || 5,
    };
  },
  { immediate: true, deep: true },
);

watch(
  () => [props.canvasWidth, props.canvasHeight] as const,
  ([width, height]) => {
    localCustomSize.value = {
      width: Math.round(width),
      height: Math.round(height),
    };
  },
  { immediate: true },
);

const handleViewChange = (key: string) => emit("update:activeViewKey", key);
const handleToggleLock = () => emit("toggleLock");
const handleUndo = () => emit("undo");
const handleRedo = () => emit("redo");
const handlePreview = () => emit("preview");
const handleSave = () => emit("save");
const handleZoomIn = () => emit("zoomIn");
const handleZoomOut = () => emit("zoomOut");
const handleCopy = () => emit("copy");
const handlePaste = () => emit("paste");
const handleDeleteSelected = () => emit("deleteSelected");

function handlePreviewCommand(command: string) {
  if (command === "pagePreview") {
    emit("preview");
    previewDropdownRef.value?.handleClose?.();
    return;
  }
  if (command === "appPreview") {
    emit("previewApp");
    previewDropdownRef.value?.handleClose?.();
  }
}

function handleViewMenuCommand(command: string) {
  if (command === "resetZoom") {
    emit("fitScreen");
    return;
  }
  if (command === "toggleRuler") {
    emit("toggleRuler");
    return;
  }
  if (command === "toggleGrid") {
    emit("toggleGrid");
    return;
  }
  if (command === "toggleSnap") {
    emit("toggleSnap");
  }
}

function handleApplyCustomSize() {
  const width = Math.round(Number(localCustomSize.value.width));
  const height = Math.round(Number(localCustomSize.value.height));
  if (!Number.isFinite(width) || width < MIN_CANVAS_WIDTH || width > MAX_CANVAS_WIDTH) {
    ElMessage.warning(
      t("toolbar.widthRange", { min: MIN_CANVAS_WIDTH, max: MAX_CANVAS_WIDTH }) as string,
    );
    return;
  }
  if (!Number.isFinite(height) || height < MIN_CANVAS_HEIGHT || height > MAX_CANVAS_HEIGHT) {
    ElMessage.warning(
      t("toolbar.heightRange", { min: MIN_CANVAS_HEIGHT, max: MAX_CANVAS_HEIGHT }) as string,
    );
    return;
  }
  emit("applyCustomSize", { width, height });
}

function handleSaveSettingsSubmit() {
  emit("saveSettingsChange", { ...localSaveSettings.value });
  emit("save");
  saveDropdownRef.value?.handleClose?.();
}

function handleMoreCommand(command: string) {
  if (command === "openAi") {
    emit("openAi");
    return;
  }
  if (command === "export") {
    emit("export");
    return;
  }
  if (command === "clearCanvas") {
    emit("clearCanvas");
    return;
  }
  if (command === "collaboration") {
    emit("openCollaboration");
    return;
  }
  if (command === "refresh") {
    emit("refreshCanvas");
  }
}
</script>

<template>
  <header class="designer-toolbar toolbar-v2">
    <div class="toolbar-section toolbar-left">
      <div class="page-chip">
        <span class="page-chip__label">{{ t("toolbar.pageChipLabel") }}</span>
        <span class="page-chip__name">{{ pageName || t("toolbar.untitledPage") }}</span>
        <span v-if="isDirty" class="page-chip__dirty">●</span>
      </div>
      <el-tooltip
        :content="isLocked ? t('toolbar.unlockPage') : t('toolbar.lockPage')"
        placement="bottom"
      >
        <el-button class="icon-btn" @click="handleToggleLock">
          <IconLucideLock v-if="isLocked" />
          <IconLucideLockOpen v-else />
        </el-button>
      </el-tooltip>
    </div>

    <div class="toolbar-section toolbar-center">
      <div class="toolbar-center-shell">
        <div class="toolbar-group toolbar-group--clipboard">
          <el-tooltip :content="`${t('toolbar.copy')} (Ctrl+C)`" placement="bottom">
            <el-button class="icon-btn" :disabled="!hasSelection" @click="handleCopy">
              <IconLucideCopy />
            </el-button>
          </el-tooltip>
          <el-tooltip :content="`${t('toolbar.paste')} (Ctrl+V)`" placement="bottom">
            <el-button class="icon-btn" :disabled="!hasClipboard" @click="handlePaste">
              <IconLucideClipboardPaste />
            </el-button>
          </el-tooltip>
          <el-tooltip :content="`${t('toolbar.delete')} (Del)`" placement="bottom">
            <el-button class="icon-btn" :disabled="!hasSelection" @click="handleDeleteSelected">
              <IconLucideTrash />
            </el-button>
          </el-tooltip>
        </div>
        <div class="toolbar-group toolbar-group--canvas toolbar-center-left">
          <el-popover
            trigger="click"
            placement="bottom"
            popper-class="designer-size-popper"
            :width="272"
          >
            <template #reference>
              <el-button class="view-btn view-btn--selector">
                <span class="view-selector__size">{{ currentCanvasSizeText }}</span>
              </el-button>
            </template>
            <div class="size-panel">
              <div class="size-panel__section">
                <div class="size-panel__title">{{ t("toolbar.presetSize") }}</div>
                <div class="size-preset-list">
                  <button
                    v-for="item in viewItems"
                    :key="item.key"
                    type="button"
                    class="size-preset-item"
                    :class="{
                      'is-active': item.key === activeViewKey && !isCustomView,
                    }"
                    @click="handleViewChange(item.key)"
                  >
                    <span class="size-preset-item__label">{{ item.label }}</span>
                    <span class="size-preset-item__meta">{{ item.width }} x {{ item.height }}</span>
                  </button>
                </div>
              </div>
              <div class="size-panel__section size-panel__section--custom">
                <div class="size-panel__title">{{ t("toolbar.customSize") }}</div>
                <div class="custom-size-grid">
                  <label class="custom-size-field">
                    <span class="custom-size-field__label">{{ t("toolbar.width") }}</span>
                    <el-input-number
                      v-model="localCustomSize.width"
                      :min="120"
                      :max="7680"
                      :step="10"
                      controls-position="right"
                    />
                  </label>
                  <label class="custom-size-field">
                    <span class="custom-size-field__label">{{ t("toolbar.height") }}</span>
                    <el-input-number
                      v-model="localCustomSize.height"
                      :min="120"
                      :max="4320"
                      :step="10"
                      controls-position="right"
                    />
                  </label>
                </div>
                <el-button class="size-panel__submit" type="primary" @click="handleApplyCustomSize">
                  {{ t("toolbar.applyCustomSize") }}
                </el-button>
              </div>
            </div>
          </el-popover>
          <div class="zoom-group">
            <el-tooltip :content="t('toolbar.zoomOut')" placement="bottom">
              <el-button class="icon-btn icon-btn--subtle" @click="handleZoomOut">
                <IconLucideZoomOut />
              </el-button>
            </el-tooltip>
            <span class="zoom-pill">{{ Math.round(zoom * 100) }}%</span>
            <el-tooltip :content="t('toolbar.zoomIn')" placement="bottom">
              <el-button class="icon-btn icon-btn--subtle" @click="handleZoomIn">
                <IconLucideZoomIn />
              </el-button>
            </el-tooltip>
          </div>
          <el-dropdown trigger="click" placement="bottom" @command="handleViewMenuCommand">
            <el-button class="view-btn">
              <IconLucideSettings2 />
              <span class="view-btn__text">{{ t("toolbar.view") }}</span>
              <IconEpArrowDownBold class="caret-icon" />
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="resetZoom">{{ t("toolbar.resetZoom") }}</el-dropdown-item>
                <el-dropdown-item command="toggleRuler">
                  {{ showRuler ? t("toolbar.hideRuler") : t("toolbar.showRuler") }}
                </el-dropdown-item>
                <el-dropdown-item command="toggleGrid">
                  {{ showGrid ? t("toolbar.hideGrid") : t("toolbar.showGrid") }}
                </el-dropdown-item>
                <el-dropdown-item command="toggleSnap">
                  {{ enableSnap ? t("toolbar.disableSnap") : t("toolbar.enableSnap") }}
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>

        <div class="toolbar-group toolbar-group--edit toolbar-center-right">
          <el-tooltip :content="t('toolbar.undo')" placement="bottom">
            <el-button class="icon-btn" :disabled="!canUndo" @click="handleUndo">
              <IconLucideUndo2 />
            </el-button>
          </el-tooltip>
          <el-tooltip :content="t('toolbar.redo')" placement="bottom">
            <el-button class="icon-btn" :disabled="!canRedo" @click="handleRedo">
              <IconLucideRedo2 />
            </el-button>
          </el-tooltip>
        </div>
      </div>
    </div>

    <div class="toolbar-section toolbar-right">
      <div class="toolbar-group toolbar-group--primary">
        <span class="save-status" :class="saveStatusClass">{{ saveStatusText }}</span>
        <el-dropdown
          ref="previewDropdownRef"
          class="preview-action"
          trigger="click"
          placement="bottom-end"
          popper-class="preview-menu-popper"
          split-button
          :hide-on-click="false"
          @click="handlePreview"
          @command="handlePreviewCommand"
        >
          <span class="split-action__text">
            <IconLucidePlay />
            <span>{{ t("toolbar.preview") }}</span>
          </span>
          <template #dropdown>
            <el-dropdown-menu>
              <div class="preview-identity-panel" @click.stop>
                <div class="preview-identity-panel__label">预览身份</div>
                <el-select
                  v-model="selectedPreviewUser"
                  size="small"
                  clearable
                  filterable
                  :teleported="false"
                  placeholder="未选择用户"
                >
                  <el-option
                    v-for="item in previewUserOptions"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value"
                  />
                </el-select>
              </div>
              <el-dropdown-item command="pagePreview">{{ t("toolbar.pagePreview") }}</el-dropdown-item>
              <el-dropdown-item command="appPreview">{{ t("toolbar.appPreview") }}</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <el-dropdown
          ref="saveDropdownRef"
          class="save-action"
          trigger="click"
          placement="bottom-end"
          popper-class="save-settings-popper"
          split-button
          @click="handleSave"
        >
          <span class="split-action__text">
            <IconLucideSave />
            <span>{{ t("toolbar.save") }}</span>
          </span>
          <template #dropdown>
            <div class="save-settings-panel" @click.stop>
              <div class="save-settings-panel__title">{{ t("toolbar.saveSettings") }}</div>
              <div class="save-settings-panel__row save-settings-panel__row--check">
                <el-checkbox v-model="localSaveSettings.autoSave" />
                <span class="save-settings-panel__label">{{ t("toolbar.autoSave") }}</span>
              </div>
              <div class="save-settings-panel__row">
                <span class="save-settings-panel__label">{{ t("toolbar.saveInterval") }}</span>
                <el-select
                  v-model="localSaveSettings.intervalMinutes"
                  class="save-settings-panel__select"
                  :disabled="!localSaveSettings.autoSave"
                >
                  <el-option
                    v-for="item in saveIntervalOptions"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value"
                  />
                </el-select>
              </div>
              <el-button class="save-settings-panel__submit" @click="handleSaveSettingsSubmit">
                {{ t("toolbar.setAndSave") }}
              </el-button>
            </div>
          </template>
        </el-dropdown>
        <el-dropdown trigger="click" placement="bottom-end" @command="handleMoreCommand">
          <el-button class="icon-btn">
            <IconLucideEllipsis />
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="openAi">
                <IconLucideBot class="menu-icon" />
                {{ t("toolbar.aiAssistant") }}
              </el-dropdown-item>
              <el-dropdown-item command="export" divided>
                <IconLucideDownload class="menu-icon" />
                {{ t("toolbar.exportPage") }}
              </el-dropdown-item>
              <el-dropdown-item command="clearCanvas">
                <IconLucideTrash2 class="menu-icon" />
                {{ t("toolbar.clearCanvas") }}
              </el-dropdown-item>
              <el-dropdown-item command="collaboration">{{ t("toolbar.collaboration") }}</el-dropdown-item>
              <el-dropdown-item command="refresh">{{ t("toolbar.refreshCanvas") }}</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </div>
  </header>
</template>

<style scoped>
.toolbar-v2 {
  position: relative;
  display: grid;
  grid-template-columns: max-content 1fr max-content;
  align-items: center;
  gap: var(--designer-gap-sm);
  padding: 0 var(--designer-shell-padding);
}

.toolbar-section {
  display: flex;
  align-items: center;
  min-width: 0;
}

.toolbar-left {
  gap: var(--designer-gap-xs);
  justify-self: start;
}

.toolbar-center {
  position: absolute;
  left: calc(var(--designer-left-panel-width, var(--designer-panel-width)) + var(--designer-rail-width));
  right: calc(var(--designer-right-panel-width, var(--designer-panel-width)) + var(--designer-rail-width));
  top: 0;
  bottom: 0;
  display: flex;
  justify-content: center;
  align-items: center;
  min-width: 0;
  pointer-events: none;
}

.toolbar-right {
  justify-content: flex-end;
  justify-self: end;
}

.toolbar-center-shell {
  display: grid;
  grid-template-columns: max-content max-content minmax(56px, 1fr) max-content;
  align-items: center;
  width: 100%;
  min-width: 0;
  padding: 0 6px;
  pointer-events: auto;
}

.toolbar-group {
  display: inline-flex;
  align-items: center;
  min-width: 0;
}

.toolbar-group--canvas,
.toolbar-group--edit,
.toolbar-group--primary,
.toolbar-group--clipboard {
  gap: var(--designer-gap-xs);
}

.toolbar-group--clipboard {
  padding-right: 6px;
  border-right: 1px solid var(--designer-border-color);
  margin-right: 2px;
}

.toolbar-center-left {
  justify-self: start;
}

.toolbar-center-right {
  justify-self: end;
}

.page-chip {
  display: inline-flex;
  align-items: center;
  gap: var(--designer-gap-2xs);
  min-width: 0;
  height: var(--designer-control-height-lg);
  padding: 0 8px;
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-lg);
  background: var(--designer-chip-surface);
}

.page-chip__label,
.page-chip__name,
.zoom-pill,
.view-selector__size,
.view-btn__text,
.save-status,
.split-action__text {
  font-size: var(--designer-nav-font);
  color: var(--designer-text-regular);
}

.page-chip__label,
.view-selector__size {
  white-space: nowrap;
}

.page-chip__name {
  max-width: 180px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 400;
}

.page-chip__dirty {
  color: var(--designer-warning-text);
  font-size: var(--designer-font-meta);
}

.zoom-group {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.zoom-pill {
  min-width: 54px;
  height: 24px;
  padding: 0 6px;
  border-radius: var(--designer-radius-md);
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.save-status {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 52px;
  height: 20px;
  padding: 0 6px;
  border-radius: 999px;
  white-space: nowrap;
  border: 1px solid transparent;
  font-size: 11px;
}

.save-status.is-saved {
  background: var(--designer-success-surface);
  color: var(--designer-success-text);
}

.save-status.is-dirty {
  background: var(--designer-warning-surface);
  color: var(--designer-warning-text);
}

.save-status.is-saving {
  background: var(--designer-info-surface);
  color: var(--designer-info-text);
}

.split-action__text {
  display: inline-flex;
  align-items: center;
  gap: var(--designer-gap-2xs);
  font-weight: 400;
  letter-spacing: 0;
}

.menu-icon,
.caret-icon {
  width: var(--designer-nav-icon);
  height: var(--designer-nav-icon);
  flex-shrink: 0;
}

.size-panel {
  display: flex;
  flex-direction: column;
  gap: var(--designer-gap-md);
  width: 272px;
  padding: 10px;
  border-radius: var(--designer-radius-lg);
  background: var(--designer-surface-elevated);
  box-shadow: var(--designer-shadow-panel);
  box-sizing: border-box;
}

.size-panel__section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.size-panel__section--custom {
  padding-top: 8px;
  border-top: 1px solid var(--designer-border-soft);
}

.size-panel__title {
  font-size: var(--designer-font-title);
  font-weight: 600;
  line-height: 1.4;
  color: var(--designer-text-primary);
}

.size-preset-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--designer-gap-xs);
}

.size-preset-item {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  min-height: 44px;
  padding: 7px 8px;
  border: 1px solid var(--designer-border-strong);
  border-radius: var(--designer-radius-md);
  background: var(--designer-group-surface);
  color: var(--designer-text-regular);
  cursor: pointer;
  transition:
    border-color 0.16s ease,
    background-color 0.16s ease,
    color 0.16s ease,
    box-shadow 0.16s ease;
}

.size-preset-item:hover {
  border-color: var(--designer-primary-border);
  background: var(--designer-shell-surface);
  box-shadow: inset 0 0 0 1px var(--designer-primary-border);
}

.size-preset-item.is-active {
  border-color: var(--designer-primary-border);
  background: var(--designer-primary-soft);
  color: var(--designer-primary-text);
  box-shadow: inset 0 0 0 1px var(--designer-primary-border);
}

.size-preset-item__label {
  font-size: var(--designer-font-label);
  font-weight: 500;
}

.size-preset-item__meta {
  font-size: var(--designer-font-meta);
  color: var(--designer-text-secondary);
  opacity: 1;
}

.custom-size-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.custom-size-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.custom-size-field__label {
  font-size: var(--designer-font-meta);
  color: var(--designer-text-secondary);
}

.size-panel__submit {
  width: 100%;
  height: 32px;
  margin-top: 2px;
  border-radius: var(--designer-radius-md);
  border: none;
  font-size: var(--designer-font-label);
  font-weight: 500;
  color: var(--designer-surface-elevated);
  background: var(--designer-primary);
}

:deep(.custom-size-field .el-input-number) {
  width: 100%;
}

:deep(.view-btn) {
  height: var(--designer-control-height);
  padding: 0 8px;
  border-radius: var(--designer-radius-md);
  border: none;
  background: transparent;
  color: var(--designer-text-regular);
  font-size: var(--designer-nav-font);
  gap: var(--designer-gap-2xs);
  box-shadow: none;
}

:deep(.view-btn--selector) {
  min-width: auto;
  padding: 0 2px;
  justify-content: center;
}

:deep(.icon-btn) {
  width: var(--designer-control-height);
  height: var(--designer-control-height);
  padding: 0;
  border: none;
  border-radius: var(--designer-radius-sm);
  color: var(--designer-text-regular);
  background: transparent;
  box-shadow: none;
}

:deep(.icon-btn:hover),
:deep(.view-btn:hover) {
  background: var(--designer-hover-surface);
}

:deep(.icon-btn .el-icon),
:deep(.icon-btn svg),
:deep(.view-btn .el-icon),
:deep(.view-btn svg) {
  width: var(--designer-nav-icon);
  height: var(--designer-nav-icon);
}

:deep(.preview-action .el-button-group),
:deep(.save-action .el-button-group) {
  display: inline-flex;
  flex-wrap: nowrap;
  align-items: center;
  vertical-align: middle;
}

:deep(.preview-action .el-button-group > .el-button),
:deep(.preview-action .el-button-group > .el-dropdown__caret-button) {
  height: var(--designer-control-height-lg);
  border: 1px solid var(--designer-border-color);
  background: var(--designer-group-surface);
  color: var(--designer-text-regular);
  box-shadow: none;
  font-size: var(--designer-nav-font);
}

:deep(.preview-action .el-button-group > .el-button:first-child) {
  padding: 0 8px;
  border-top-left-radius: var(--designer-radius-lg);
  border-bottom-left-radius: var(--designer-radius-lg);
}

:deep(.preview-action .el-button-group > .el-dropdown__caret-button:last-child) {
  width: 28px;
  padding: 0;
  border-top-right-radius: var(--designer-radius-lg);
  border-bottom-right-radius: var(--designer-radius-lg);
  border-top-left-radius: 0;
  border-bottom-left-radius: 0;
}

:deep(.save-action .el-button-group > .el-button),
:deep(.save-action .el-button-group > .el-dropdown__caret-button) {
  height: var(--designer-control-height-lg);
  border: 1px solid var(--designer-border-color);
  background: var(--designer-group-surface);
  color: var(--designer-text-regular);
  box-shadow: none;
  font-size: var(--designer-nav-font);
}

:deep(.save-action .el-button-group > .el-button:first-child) {
  min-width: 68px;
  padding: 0 8px;
  border-top-left-radius: var(--designer-radius-lg);
  border-bottom-left-radius: var(--designer-radius-lg);
}

:deep(.save-action .el-button-group > .el-dropdown__caret-button:last-child) {
  width: 28px;
  padding: 0;
  border-top-right-radius: var(--designer-radius-lg);
  border-bottom-right-radius: var(--designer-radius-lg);
  border-top-left-radius: 0;
  border-bottom-left-radius: 0;
}

:deep(.preview-action .el-button-group > .el-dropdown__caret-button .el-icon),
:deep(.save-action .el-button-group > .el-dropdown__caret-button .el-icon) {
  margin-left: 0;
}

:global(.preview-menu-popper .el-dropdown-menu) {
  padding: 0;
}

.preview-identity-panel {
  display: flex;
  flex-direction: column;
  gap: 6px;
  width: 220px;
  padding: 6px 8px 8px;
  border-bottom: 1px solid var(--designer-border-soft);
}

.preview-identity-panel__label {
  color: var(--designer-text-secondary);
  font-size: var(--designer-font-meta);
}

.preview-identity-panel :deep(.el-select) {
  width: 100%;
}

:global(.designer-size-popper.el-popper) {
  padding: 0;
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-lg);
  box-shadow: var(--designer-shadow-panel);
  overflow: hidden;
}

:global(.designer-size-popper .el-popover) {
  padding: 0;
}

:global(.designer-size-popper .el-input-number .el-input__wrapper) {
  min-height: 32px;
  border-radius: var(--designer-radius-md);
  box-shadow: inset 0 0 0 1px var(--designer-border-strong);
  background: var(--designer-group-surface);
}

:global(.designer-size-popper .el-input-number .el-input__inner) {
  font-size: var(--designer-font-label);
  color: var(--designer-text-regular);
}

:global(.save-settings-popper .save-settings-panel) {
  width: 272px;
  padding: 10px;
  border-radius: var(--designer-radius-lg);
  background: var(--designer-surface-elevated);
  box-shadow: var(--designer-shadow-panel);
  box-sizing: border-box;
  font-size: var(--designer-font-label);
}

:global(.save-settings-popper .el-dropdown-menu) {
  padding: 0;
  border: none;
  background: transparent;
}

:global(.save-settings-popper.el-popper) {
  padding: 0;
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-lg);
  box-shadow: var(--designer-shadow-panel);
  overflow: hidden;
}

:global(.save-settings-popper .save-settings-panel__title) {
  font-size: var(--designer-font-title);
  font-weight: 600;
  line-height: 1.4;
  color: var(--designer-text-primary);
  margin-bottom: var(--designer-gap-sm);
}

:global(.save-settings-popper .save-settings-panel__row) {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  margin-bottom: var(--designer-gap-sm);
}

:global(.save-settings-popper .save-settings-panel__row--check) {
  gap: var(--designer-gap-sm);
}

:global(.save-settings-popper .save-settings-panel__row--check .el-checkbox) {
  margin-right: 0;
}

:global(.save-settings-popper .save-settings-panel__label) {
  font-size: var(--designer-font-label);
  line-height: 1.4;
  color: var(--designer-text-regular);
}

:global(.save-settings-popper .save-settings-panel__select) {
  width: 126px;
  margin-left: auto;
}

:global(.save-settings-popper .save-settings-panel__select .el-input__wrapper) {
  min-height: 32px;
  border-radius: var(--designer-radius-md);
  box-shadow: inset 0 0 0 1px var(--designer-border-strong);
  background: var(--designer-group-surface);
}

:global(.save-settings-popper .save-settings-panel__select .el-input__inner) {
  font-size: var(--designer-font-label);
  color: var(--designer-text-regular);
}

:global(.save-settings-popper .save-settings-panel__row--check .el-checkbox__inner) {
  width: 16px;
  height: 16px;
  border-radius: 4px;
}

:global(.save-settings-popper .save-settings-panel__submit) {
  width: 100%;
  height: 32px;
  margin: 4px auto 0;
  display: block;
  border-radius: var(--designer-radius-md);
  border: none;
  font-size: var(--designer-font-label);
  font-weight: 500;
  color: var(--designer-surface-elevated);
  background: var(--designer-primary);
}

@media (max-width: 1360px) {
  .toolbar-center {
    left: calc(var(--designer-left-panel-width, var(--designer-panel-width)) + var(--designer-rail-width));
    right: calc(var(--designer-right-panel-width, var(--designer-panel-width)) + var(--designer-rail-width));
  }

  .toolbar-center-shell {
    grid-template-columns: max-content minmax(32px, 1fr) max-content;
  }
}

@media (max-width: 1180px) {
  .toolbar-v2 {
    gap: var(--designer-gap-sm);
  }

  .toolbar-center {
    left: calc(var(--designer-left-panel-width, var(--designer-panel-width)) + var(--designer-rail-width));
    right: calc(var(--designer-right-panel-width, var(--designer-panel-width)) + var(--designer-rail-width));
  }

  .toolbar-center-shell {
    grid-template-columns: max-content minmax(16px, 1fr) max-content;
  }

  .page-chip__label,
  .view-selector__size {
    display: none;
  }

  :deep(.view-btn--selector) {
    min-width: 126px;
  }
}

@media (max-width: 980px) {
  .toolbar-center {
    position: static;
    left: auto;
    right: auto;
    top: auto;
    bottom: auto;
    pointer-events: auto;
  }

  .toolbar-center-shell {
    width: auto;
    grid-template-columns: max-content;
  }

  .toolbar-center-right {
    display: none;
  }

  .zoom-group {
    display: none;
  }
}
</style>
