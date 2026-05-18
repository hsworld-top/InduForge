<script setup lang="ts">
import type { ProjectI18nLocale, ProjectI18nSettings } from "@/editor-core/document/types";
import IconLucideCircleHelp from "~icons/lucide/circle-help";
import IconLucidePlus from "~icons/lucide/plus";
import { ElMessage } from "element-plus";
import { storeToRefs } from "pinia";
import { computed, ref } from "vue";
import {
  cloneProjectI18nSettings,
  normalizeProjectI18nSettings,
} from "@/editor-core/i18n/project-i18n";
import { useEditorStore } from "@/stores/editor-store";

const editorStore = useEditorStore();
const { projectI18n } = storeToRefs(editorStore);

const newLocaleCode = ref("");
const newLocaleName = ref("");
const localeDialogVisible = ref(false);

const settings = computed({
  get: () => projectI18n.value,
  set: (value) => {
    editorStore.setProjectI18n(value);
  },
});

const enabledLocales = computed(() => settings.value.locales.filter((item) => item.enabled));

function patchSettings(patch: Partial<ProjectI18nSettings>) {
  settings.value = normalizeProjectI18nSettings({
    ...settings.value,
    ...patch,
  });
}

function handleEnabledChange(value: boolean) {
  patchSettings({ enabled: value });
}

function handleDefaultLocaleChange(value: string) {
  patchSettings({ defaultLocale: value });
}

function addLocale() {
  const code = newLocaleCode.value.trim();
  const name = newLocaleName.value.trim() || code;
  if (!code) {
    ElMessage.warning({ message: "请输入语言编码" } as never);
    return;
  }
  if (settings.value.locales.some((item) => item.code === code)) {
    ElMessage.warning({ message: "语言编码已存在" } as never);
    return;
  }
  const next = cloneProjectI18nSettings(settings.value);
  next.locales.push({ code, name, enabled: true });
  settings.value = next;
  newLocaleCode.value = "";
  newLocaleName.value = "";
  localeDialogVisible.value = false;
}

function updateLocale(locale: ProjectI18nLocale, patch: Partial<ProjectI18nLocale>) {
  const next = cloneProjectI18nSettings(settings.value);
  next.locales = next.locales.map((item) =>
    item.code === locale.code ? { ...item, ...patch } : item,
  );
  if (!next.locales.some((item) => item.code === next.defaultLocale && item.enabled)) {
    next.defaultLocale = next.locales.find((item) => item.enabled)?.code || locale.code;
  }
  settings.value = normalizeProjectI18nSettings(next);
}

function removeLocale(locale: ProjectI18nLocale) {
  if (locale.code === settings.value.defaultLocale) {
    ElMessage.warning({ message: "默认语言不能删除" } as never);
    return;
  }
  const next = cloneProjectI18nSettings(settings.value);
  next.locales = next.locales.filter((item) => item.code !== locale.code);
  Object.values(next.resources).forEach((resource) => {
    delete resource.values[locale.code];
  });
  settings.value = normalizeProjectI18nSettings(next);
}

function openResourceDialog(scan = false) {
  window.dispatchEvent(
    new CustomEvent("designer:i18n-open-resource", {
      detail: { scan },
    }),
  );
}
</script>

<template>
  <div class="i18n-panel">
    <section class="panel-section">
      <div class="section-body">
        <div class="switch-row">
          <div>
            <div class="row-title">
              启用页面国际化
              <el-tooltip content="启用后，访问用户看到的页面文案会按当前语言显示。" placement="top">
                <IconLucideCircleHelp class="help-icon" />
              </el-tooltip>
            </div>
          </div>
          <el-switch :model-value="settings.enabled" @update:model-value="handleEnabledChange" />
        </div>
        <label class="field-block">
          <span>
            默认语言
            <el-tooltip content="缺少翻译时会回退到默认语言。" placement="top">
              <IconLucideCircleHelp class="help-icon" />
            </el-tooltip>
          </span>
          <el-select
            :model-value="settings.defaultLocale"
            size="small"
            placeholder="默认语言"
            @update:model-value="handleDefaultLocaleChange"
          >
            <el-option
              v-for="localeItem in enabledLocales"
              :key="localeItem.code"
              :label="`${localeItem.name} ${localeItem.code}`"
              :value="localeItem.code"
            />
          </el-select>
        </label>
        <div class="button-grid">
          <el-tooltip content="扫描工程内所有页面，列出可维护的静态文案。" placement="top">
            <el-button type="primary" size="small" @click="openResourceDialog(true)">
              扫描并维护文案
            </el-button>
          </el-tooltip>
        </div>
      </div>
    </section>

    <section class="panel-section">
      <div class="section-head">
        <span>语言</span>
        <el-button text size="small" @click="localeDialogVisible = true">
          <IconLucidePlus />
          新增
        </el-button>
      </div>
      <div class="section-body">
        <div class="locale-list">
          <div v-for="localeItem in settings.locales" :key="localeItem.code" class="locale-row">
            <div>
              <div class="row-title">{{ localeItem.name }}</div>
              <div class="row-desc">{{ localeItem.code }}</div>
            </div>
            <div class="locale-actions">
              <el-tag v-if="localeItem.code === settings.defaultLocale" size="small">默认</el-tag>
              <el-switch
                :model-value="localeItem.enabled"
                size="small"
                :disabled="localeItem.code === settings.defaultLocale"
                @update:model-value="(value: boolean) => updateLocale(localeItem, { enabled: value })"
              />
              <el-button
                size="small"
                text
                :disabled="localeItem.code === settings.defaultLocale"
                @click="removeLocale(localeItem)"
              >
                删除
              </el-button>
            </div>
          </div>
        </div>
      </div>
    </section>

    <el-dialog
      v-model="localeDialogVisible"
      title="新增语言"
      width="360px"
      append-to-body
      destroy-on-close
    >
      <div class="locale-dialog-body">
        <label class="field-block">
          <span>语言名称</span>
          <el-input v-model="newLocaleName" placeholder="如 English" />
        </label>
        <label class="field-block">
          <span>语言编码</span>
          <el-input v-model="newLocaleCode" placeholder="如 en-US" />
        </label>
      </div>
      <template #footer>
        <el-button @click="localeDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="addLocale">新增</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.i18n-panel {
  display: flex;
  flex-direction: column;
  gap: var(--designer-gap-md);
  min-width: 0;
}

.panel-section {
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-md);
  background: var(--designer-shell-surface);
  overflow: hidden;
}

.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 32px;
  padding: 0 10px;
  border-bottom: 1px solid var(--designer-border-soft);
  background: var(--designer-group-surface);
  font-size: var(--designer-font-label);
  font-weight: 600;
}

.section-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px;
}

.switch-row,
.locale-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.row-title {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-weight: 600;
  color: var(--designer-text-primary);
}

.row-desc {
  margin-top: 2px;
  color: var(--designer-text-secondary);
  font-size: var(--designer-font-label);
}

.button-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 6px;
}

.button-grid :deep(.el-button) {
  width: 100%;
}

.field-block {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.field-block > span {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--designer-text-secondary);
  font-size: var(--designer-font-label);
}

.help-icon {
  width: 13px;
  height: 13px;
  color: var(--designer-text-tertiary);
  cursor: help;
}

.locale-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.locale-dialog-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.locale-row {
  padding: 8px;
  border: 1px solid var(--designer-border-soft);
  border-radius: var(--designer-radius-md);
}

.locale-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}

</style>
