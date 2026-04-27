<script setup lang="ts">
import type { PageInspectorFormState } from "./page-inspector-types";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import FriendlyColorPicker from "@/ui/shared/widgets/base/FriendlyColorPicker.vue";
import { useEditorStore } from "@/stores/editor-store";
import AssetManagerDialog from "@/ui/shared/widgets/resource/AssetManagerDialog.vue";
import IconEpEditPen from "~icons/ep/edit-pen";
import StyleConfigEditorDialog from "../style-config/StyleConfigEditorDialog.vue";
import type { AssetItem } from "@/types/api";

const props = defineProps<{
  form: PageInspectorFormState;
}>();

const emit = defineEmits<{
  updateConfig: [];
}>();

const { t } = useI18n();
const editorStore = useEditorStore();
const styleDialogVisible = ref(false);
const assetDialogVisible = ref(false);
const backgroundEditorVisible = ref(false);
const gradientDirection = ref("135deg");
const gradientStartColor = ref("#1677ff");
const gradientEndColor = ref("#67c23a");

interface PageStylePreset {
  id: string;
  label: string;
  content: string;
}

const backgroundKindOptions = computed(() => [
  { label: t("pageInspector.backgroundKinds.color"), value: "color" },
  { label: t("pageInspector.backgroundKinds.image"), value: "image" },
  { label: t("pageInspector.backgroundKinds.gradient"), value: "gradient" },
]);

const backgroundSizeOptions = computed(() => [
  { label: t("pageInspector.backgroundSizes.cover"), value: "cover" },
  { label: t("pageInspector.backgroundSizes.contain"), value: "contain" },
  { label: t("pageInspector.backgroundSizes.stretch"), value: "stretch" },
  { label: t("pageInspector.backgroundSizes.auto"), value: "auto" },
]);

const backgroundRepeatOptions = computed(() => [
  { label: t("pageInspector.backgroundRepeats.noRepeat"), value: "no-repeat" },
  { label: t("pageInspector.backgroundRepeats.repeat"), value: "repeat" },
  { label: t("pageInspector.backgroundRepeats.repeatX"), value: "repeat-x" },
  { label: t("pageInspector.backgroundRepeats.repeatY"), value: "repeat-y" },
]);

const gradientDirectionOptions = computed(() => [
  { label: t("pageInspector.backgroundGradient.directions.right"), value: "to right" },
  { label: t("pageInspector.backgroundGradient.directions.bottom"), value: "to bottom" },
  { label: t("pageInspector.backgroundGradient.directions.diagonalDown"), value: "135deg" },
  { label: t("pageInspector.backgroundGradient.directions.diagonalUp"), value: "45deg" },
]);

const gradientPresets = computed(() => [
  {
    id: "blue-green",
    label: t("pageInspector.backgroundGradient.presets.blueGreen"),
    value: "linear-gradient(135deg, #1677ff 0%, #67c23a 100%)",
  },
  {
    id: "indigo-cyan",
    label: t("pageInspector.backgroundGradient.presets.indigoCyan"),
    value: "linear-gradient(135deg, #4f46e5 0%, #06b6d4 100%)",
  },
  {
    id: "orange-red",
    label: t("pageInspector.backgroundGradient.presets.orangeRed"),
    value: "linear-gradient(135deg, #f59e0b 0%, #ef4444 100%)",
  },
  {
    id: "slate",
    label: t("pageInspector.backgroundGradient.presets.slate"),
    value: "linear-gradient(135deg, #f8fafc 0%, #cbd5e1 100%)",
  },
]);

const transitionOptions = computed(() => [
  { label: t("pageInspector.transitionTypes.none"), value: "none" },
  { label: t("pageInspector.transitionTypes.fade"), value: "fade" },
  { label: t("pageInspector.transitionTypes.slide"), value: "slide" },
  { label: t("pageInspector.transitionTypes.zoom"), value: "zoom" },
]);

const showImageBackgroundOptions = computed(() => props.form.backgroundType === "image");
const hasBackgroundImage = computed(
  () => props.form.backgroundType === "image" && isImageUrlValue(props.form.backgroundValue),
);
const hasGradientValue = computed(
  () =>
    props.form.backgroundType === "gradient" &&
    /^(linear-gradient|radial-gradient|conic-gradient)\(/i.test(props.form.backgroundValue.trim()),
);
const backgroundEditorTitle = computed(() => t("pageInspector.backgroundGradient.dialogTitle"));
const pageStylePresets = computed<PageStylePreset[]>(() => [
  {
    id: "page-root",
    label: t("pageInspector.styleConfig.presets.pageRoot"),
    content: "#pageId {\n  background: #f5f7fa;\n  color: #303133;\n}\n",
  },
  {
    id: "page-theme",
    label: t("pageInspector.styleConfig.presets.pageTheme"),
    content:
      "#pageId {\n  --page-primary: #1677ff;\n  --page-success: #52c41a;\n  --page-warning: #faad14;\n  --page-danger: #ff4d4f;\n  --page-radius: 8px;\n}\n",
  },
  {
    id: "all-nodes",
    label: t("pageInspector.styleConfig.presets.allNodes"),
    content: "#pageId .designer-node {\n  border-radius: 8px;\n  box-sizing: border-box;\n}\n",
  },
  {
    id: "text",
    label: t("pageInspector.styleConfig.presets.text"),
    content:
      "#pageId {\n  color: #1f2937;\n  font-size: 14px;\n  line-height: 1.6;\n}\n\n#pageId h1,\n#pageId h2,\n#pageId h3 {\n  color: #111827;\n  font-weight: 600;\n}\n",
  },
  {
    id: "button",
    label: t("pageInspector.styleConfig.presets.button"),
    content:
      "#pageId .el-button {\n  border-radius: var(--page-radius, 8px);\n  font-weight: 600;\n}\n\n#pageId .el-button--primary {\n  background: var(--page-primary, #1677ff);\n  border-color: var(--page-primary, #1677ff);\n}\n",
  },
  {
    id: "input",
    label: t("pageInspector.styleConfig.presets.input"),
    content:
      "#pageId .el-input__wrapper,\n#pageId .el-select__wrapper,\n#pageId .el-textarea__inner {\n  border-radius: var(--page-radius, 8px);\n  box-shadow: 0 0 0 1px #dcdfe6 inset;\n}\n\n#pageId .el-input__wrapper.is-focus,\n#pageId .el-select__wrapper.is-focused {\n  box-shadow: 0 0 0 1px var(--page-primary, #1677ff) inset;\n}\n",
  },
  {
    id: "card",
    label: t("pageInspector.styleConfig.presets.card"),
    content:
      "#pageId .el-card,\n#pageId .designer-node-card {\n  border-radius: 12px;\n  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.08);\n}\n",
  },
  {
    id: "table",
    label: t("pageInspector.styleConfig.presets.table"),
    content:
      "#pageId .el-table {\n  border-radius: 8px;\n  overflow: hidden;\n}\n\n#pageId .el-table th.el-table__cell {\n  background: #f5f7fa;\n  color: #374151;\n  font-weight: 600;\n}\n",
  },
  {
    id: "form",
    label: t("pageInspector.styleConfig.presets.form"),
    content:
      "#pageId .el-form-item {\n  margin-bottom: 16px;\n}\n\n#pageId .el-form-item__label {\n  color: #374151;\n  font-weight: 500;\n}\n",
  },
  {
    id: "image-background",
    label: t("pageInspector.styleConfig.presets.imageBackground"),
    content:
      '#pageId {\n  background-image: url("/static/images/background.png");\n  background-size: cover;\n  background-position: center;\n  background-repeat: no-repeat;\n}\n',
  },
]);
function openStyleEditor(): void {
  styleDialogVisible.value = true;
}

function openAssetDialog(): void {
  assetDialogVisible.value = true;
}

function openBackgroundEditor(): void {
  backgroundEditorVisible.value = true;
}

function isColorValue(value: string): boolean {
  const trimmed = String(value || "").trim();
  return (
    /^#(?:[0-9a-f]{3,4}|[0-9a-f]{6}|[0-9a-f]{8})$/i.test(trimmed) ||
    /^(rgb|rgba|hsl|hsla|var)\(/i.test(trimmed)
  );
}

function isImageUrlValue(value: string): boolean {
  const trimmed = String(value || "").trim();
  if (!trimmed) return false;
  if (trimmed.startsWith("#")) return false;
  if (/^(rgb|hsl|linear-gradient|radial-gradient|conic-gradient)\(/i.test(trimmed)) return false;
  return true;
}

function handleBackgroundTypeChange(): void {
  if (props.form.backgroundType === "color" && !isColorValue(props.form.backgroundValue)) {
    props.form.backgroundValue = "#ffffff";
  }
  if (props.form.backgroundType === "image" && !isImageUrlValue(props.form.backgroundValue)) {
    props.form.backgroundValue = "";
  }
  if (props.form.backgroundType === "gradient" && !hasGradientValue.value) {
    applyGradientPreset(gradientPresets.value[0]?.value || buildLinearGradient());
    return;
  }
  emit("updateConfig");
}

function buildLinearGradient(): string {
  return `linear-gradient(${gradientDirection.value}, ${gradientStartColor.value} 0%, ${gradientEndColor.value} 100%)`;
}

function syncGradientControls(value: string): void {
  const match = String(value || "").match(
    /^linear-gradient\(\s*([^,]+),\s*(#[0-9a-f]{3,8})\s+\d+%?,\s*(#[0-9a-f]{3,8})\s+\d+%?\s*\)$/i,
  );
  const [, direction, startColor, endColor] = match || [];
  if (!direction || !startColor || !endColor) return;
  gradientDirection.value = direction.trim();
  gradientStartColor.value = startColor;
  gradientEndColor.value = endColor;
}

function applyCustomGradient(): void {
  props.form.backgroundValue = buildLinearGradient();
  emit("updateConfig");
}

function applyGradientPreset(value: string): void {
  props.form.backgroundValue = value;
  syncGradientControls(value);
  emit("updateConfig");
}

function closeBackgroundEditor(): void {
  backgroundEditorVisible.value = false;
}

function resolveAssetUrl(asset: Partial<AssetItem> | null | undefined): string {
  return String(asset?.url || asset?.src || asset?.path || "");
}

function selectBackgroundAsset(asset: AssetItem): void {
  const url = resolveAssetUrl(asset);
  if (!url) return;
  props.form.backgroundValue = url;
  assetDialogVisible.value = false;
  emit("updateConfig");
}

function saveStyleConfig(content: string): void {
  props.form.styleConfig = content;
  styleDialogVisible.value = false;
  emit("updateConfig");
}

watch(
  () => props.form.backgroundValue,
  (value) => {
    if (props.form.backgroundType === "gradient") {
      syncGradientControls(value);
    }
  },
  { immediate: true },
);
</script>

<template>
  <div class="page-section-fields">
    <div class="page-prop-item">
      <div class="page-prop-label">{{ t("pageInspector.labels.backgroundType") }}</div>
      <div class="page-prop-editor">
        <el-select v-model="form.backgroundType" size="small" @change="handleBackgroundTypeChange">
          <el-option
            v-for="item in backgroundKindOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </el-select>
      </div>
    </div>

    <div class="page-prop-item page-prop-item--stacked">
      <div class="page-prop-label">{{ t("pageInspector.labels.backgroundValue") }}</div>
      <div class="page-prop-editor">
        <FriendlyColorPicker
          v-if="form.backgroundType === 'color'"
          v-model="form.backgroundValue"
          layout="block"
          :show-recent="false"
          @change="$emit('updateConfig')"
        />
        <div v-else-if="form.backgroundType === 'image'" class="background-image-picker">
          <button type="button" class="background-image-picker__button" @click="openAssetDialog">
            <span v-if="hasBackgroundImage" class="background-image-picker__preview">
              <img :src="form.backgroundValue" alt="" />
            </span>
            <span class="background-image-picker__content">
              <span class="background-image-picker__title">
                {{
                  hasBackgroundImage
                    ? t("pageInspector.backgroundImage.change")
                    : t("pageInspector.backgroundImage.select")
                }}
              </span>
              <span
                v-if="hasBackgroundImage"
                class="background-image-picker__url"
                :title="form.backgroundValue"
              >
                {{ form.backgroundValue }}
              </span>
              <span v-else class="background-image-picker__url">
                {{ t("pageInspector.backgroundImage.empty") }}
              </span>
            </span>
          </button>
        </div>
        <button
          v-else-if="form.backgroundType === 'gradient'"
          type="button"
          class="background-gradient-preview"
          :title="t('pageInspector.backgroundGradient.configure')"
          :aria-label="t('pageInspector.backgroundGradient.configure')"
          @click="openBackgroundEditor"
        >
          <span
            class="background-gradient-preview__swatch"
            :style="{ background: form.backgroundValue || buildLinearGradient() }"
          />
        </button>
        <el-input
          v-else
          v-model="form.backgroundValue"
          size="small"
          :placeholder="t('pageInspector.placeholders.backgroundValue')"
          @blur="$emit('updateConfig')"
        />
      </div>
    </div>

    <template v-if="showImageBackgroundOptions">
      <div class="page-prop-item">
        <div class="page-prop-label">{{ t("pageInspector.labels.backgroundSize") }}</div>
        <div class="page-prop-editor">
          <el-select v-model="form.backgroundSize" size="small" @change="$emit('updateConfig')">
            <el-option
              v-for="item in backgroundSizeOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </div>
      </div>

      <div class="page-prop-item">
        <div class="page-prop-label">{{ t("pageInspector.labels.backgroundPosition") }}</div>
        <div class="page-prop-editor">
          <el-input
            v-model="form.backgroundPosition"
            size="small"
            :placeholder="t('pageInspector.placeholders.backgroundPosition')"
            @blur="$emit('updateConfig')"
          />
        </div>
      </div>

      <div class="page-prop-item">
        <div class="page-prop-label">{{ t("pageInspector.labels.backgroundRepeat") }}</div>
        <div class="page-prop-editor">
          <el-select v-model="form.backgroundRepeat" size="small" @change="$emit('updateConfig')">
            <el-option
              v-for="item in backgroundRepeatOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </div>
      </div>
    </template>

    <div class="page-prop-item">
      <div class="page-prop-label">{{ t("pageInspector.labels.styleConfig") }}</div>
      <div class="page-prop-editor style-config-entry">
        <button
          type="button"
          class="style-config-button"
          :class="{ 'has-config': form.styleConfig }"
          :title="t('pageInspector.styleConfig.openEditor')"
          :aria-label="t('pageInspector.styleConfig.openEditor')"
          @click="openStyleEditor"
        >
          <IconEpEditPen class="style-config-button__icon" />
        </button>
      </div>
    </div>

    <div class="page-prop-item">
      <div class="page-prop-label">{{ t("pageInspector.labels.transitionType") }}</div>
      <div class="page-prop-editor">
        <el-select v-model="form.transitionType" size="small" @change="$emit('updateConfig')">
          <el-option
            v-for="item in transitionOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </el-select>
      </div>
    </div>

    <StyleConfigEditorDialog
      v-model="styleDialogVisible"
      :content="form.styleConfig || ''"
      :title="t('pageInspector.styleConfig.dialogTitle')"
      :presets="pageStylePresets"
      :project-id="editorStore.projectId || ''"
      :selector-label="t('pageInspector.styleConfig.selectorLabel')"
      :selector-tokens="['#pageId']"
      :selector-help="t('pageInspector.styleConfig.hint')"
      :template-label="t('pageInspector.styleConfig.templateLabel')"
      :template-placeholder="t('pageInspector.styleConfig.templatePlaceholder')"
      :filter-label="t('pageInspector.styleConfig.filterLabel')"
      :search-template="t('pageInspector.styleConfig.searchTemplate')"
      :asset-library="t('pageInspector.styleConfig.assetLibrary')"
      :search-assets="t('pageInspector.styleConfig.searchAssets')"
      :clear-text="t('pageInspector.styleConfig.clear')"
      :cancel-text="t('pageInspector.styleConfig.cancel')"
      :save-text="t('pageInspector.styleConfig.save')"
      @save="saveStyleConfig"
    />

    <el-dialog
      v-model="backgroundEditorVisible"
      :title="backgroundEditorTitle"
      width="520px"
      append-to-body
      class="background-editor-dialog"
    >
      <div v-if="form.backgroundType === 'gradient'" class="background-editor">
        <div
          class="background-editor__hero"
          :style="{ background: form.backgroundValue || buildLinearGradient() }"
        >
          <span>{{ t("pageInspector.backgroundGradient.preview") }}</span>
        </div>
        <div class="background-editor__preset-grid">
          <button
            v-for="item in gradientPresets"
            :key="item.id"
            type="button"
            class="background-editor__preset"
            :title="item.label"
            :aria-label="item.label"
            :style="{ background: item.value }"
            @click="applyGradientPreset(item.value)"
          />
        </div>
        <div class="background-editor__field">
          <span>{{ t("pageInspector.backgroundGradient.direction") }}</span>
          <el-select v-model="gradientDirection" size="small" @change="applyCustomGradient">
            <el-option
              v-for="item in gradientDirectionOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </div>
        <div class="background-editor__field">
          <span>{{ t("pageInspector.backgroundGradient.startColor") }}</span>
          <FriendlyColorPicker v-model="gradientStartColor" @change="applyCustomGradient" />
        </div>
        <div class="background-editor__field">
          <span>{{ t("pageInspector.backgroundGradient.endColor") }}</span>
          <FriendlyColorPicker v-model="gradientEndColor" @change="applyCustomGradient" />
        </div>
        <el-input
          v-model="form.backgroundValue"
          size="small"
          :placeholder="t('pageInspector.backgroundGradient.placeholder')"
          @blur="$emit('updateConfig')"
        />
      </div>

      <template #footer>
        <el-button type="primary" @click="closeBackgroundEditor">
          {{ t("pageInspector.backgroundEditor.done") }}
        </el-button>
      </template>
    </el-dialog>

    <AssetManagerDialog
      v-model="assetDialogVisible"
      :project-id="editorStore.projectId || ''"
      :title="t('pageInspector.backgroundImage.dialogTitle')"
      :selected-url="hasBackgroundImage ? form.backgroundValue : ''"
      accept="image/*"
      image-only
      @select="selectBackgroundAsset"
    />
  </div>
</template>

<style scoped>
.page-section-fields {
  display: flex;
  flex-direction: column;
  gap: var(--designer-gap-xs);
}

.page-prop-item {
  display: flex;
  align-items: center;
  gap: var(--designer-gap-sm);
  min-height: 32px;
  padding: 4px 6px;
  border-radius: var(--designer-radius-sm);
}

.page-prop-item--stacked {
  align-items: flex-start;
}

.page-prop-label {
  width: 92px;
  min-width: 92px;
  font-size: var(--designer-font-label);
  color: var(--designer-text-regular);
}

.page-prop-editor {
  flex: 1;
  min-width: 0;
}

.style-config-entry {
  display: flex;
  justify-content: flex-start;
  min-width: 0;
}

.style-config-button {
  display: flex;
  width: 28px;
  height: 28px;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-md);
  background: var(--designer-group-surface);
  color: var(--designer-text-secondary);
  cursor: pointer;
  transition:
    border-color 0.15s ease,
    background-color 0.15s ease,
    color 0.15s ease;
}

.style-config-button:hover,
.style-config-button.has-config {
  border-color: var(--designer-primary-border);
  background: var(--designer-primary-soft);
  color: var(--designer-primary-text);
}

.style-config-button__icon {
  width: var(--designer-panel-icon);
  height: var(--designer-panel-icon);
  flex: 0 0 auto;
}

.background-image-picker {
  min-width: 0;
}

.background-image-picker__button {
  display: flex;
  width: 100%;
  min-width: 0;
  align-items: center;
  gap: 8px;
  padding: 6px;
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-md);
  background: var(--designer-group-surface);
  color: var(--designer-text-primary);
  cursor: pointer;
  text-align: left;
  transition:
    border-color 0.15s ease,
    background-color 0.15s ease;
}

.background-image-picker__button:hover {
  border-color: var(--designer-primary-border);
  background: var(--designer-primary-soft);
}

.background-gradient-preview {
  display: block;
  width: 100%;
  height: 28px;
  min-width: 0;
  overflow: hidden;
  padding: 0;
  border: 0;
  border-radius: var(--designer-radius-sm);
  background: transparent;
  cursor: pointer;
}

.background-gradient-preview__swatch {
  display: block;
  width: 100%;
  height: 100%;
  border-radius: inherit;
  transition: opacity 0.15s ease;
}

.background-gradient-preview:hover .background-gradient-preview__swatch {
  opacity: 0.88;
}

.background-image-picker__preview {
  display: flex;
  width: 44px;
  height: 32px;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border-radius: var(--designer-radius-sm);
  background: var(--designer-shell-surface);
}

.background-image-picker__preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.background-image-picker__content {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 2px;
}

.background-image-picker__title {
  color: var(--designer-text-primary);
  font-size: var(--designer-font-label);
}

.background-image-picker__url {
  min-width: 0;
  overflow: hidden;
  color: var(--designer-text-muted);
  font-size: var(--designer-font-caption);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.background-editor {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 12px;
}

.background-editor__hero {
  display: flex;
  height: 96px;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border: 1px solid var(--designer-border-soft);
  border-radius: var(--designer-radius-lg);
  color: rgba(255, 255, 255, 0.92);
  font-size: var(--designer-font-label);
  font-weight: 600;
  text-shadow: 0 1px 2px rgba(15, 23, 42, 0.35);
}

.background-editor__preset-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
}

.background-editor__preset {
  height: 38px;
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-md);
  cursor: pointer;
}

.background-editor__preset:hover {
  border-color: var(--designer-primary-border);
}

.background-editor__field {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 6px;
}

.background-editor__field > span {
  color: var(--designer-text-secondary);
  font-size: var(--designer-font-caption);
}

.page-prop-editor :deep(.el-select),
.page-prop-editor :deep(.el-input) {
  width: 100%;
}
</style>
