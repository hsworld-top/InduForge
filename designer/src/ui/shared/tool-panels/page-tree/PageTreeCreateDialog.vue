<!--
  页面树：新建页面 / 分组弹窗
-->
<script setup lang="ts">
import { useI18n } from "vue-i18n";
import IconEpFolder from "~icons/ep/folder";
import IconEpFolderOpened from "~icons/ep/folder-opened";
import IconEpPlus from "~icons/ep/plus";

interface CreateTypeOptionLike {
  value: string;
  icon?: unknown;
  label: string;
  desc?: string;
}

interface PageFolderOptionLike {
  id: string;
  name: string;
}

interface CreatePageFormLike {
  type: string;
  name: string;
  parentId: string | null;
}

const props = defineProps<{
  title: string;
  createTypeOptions: CreateTypeOptionLike[];
  form: CreatePageFormLike;
  rules: Record<string, unknown>;
  folderOptions: PageFolderOptionLike[];
  creating?: boolean;
  isFixedBasicType: (type: string) => boolean;
}>();

const emit = defineEmits(["update:form", "confirm"]);

const open = defineModel<boolean>({ default: false });
const { t } = useI18n();

function patchForm(partial: Partial<CreatePageFormLike>) {
  emit("update:form", { ...props.form, ...partial });
}

function selectType(value: string) {
  patchForm({ type: value });
}

function handleNameUpdate(value: string) {
  patchForm({ name: value });
}

function handleParentIdUpdate(value: string | null) {
  patchForm({ parentId: value });
}
</script>

<template>
  <el-dialog
    v-model="open"
    :title="title"
    width="420px"
    :close-on-click-modal="false"
    class="create-page-dialog"
  >
    <div class="create-type-selector">
      <div
        v-for="option in createTypeOptions"
        :key="option.value"
        class="type-option"
        :class="{ 'is-active': form.type === option.value }"
        @click="selectType(option.value)"
      >
        <component :is="option.icon" class="type-icon" />
        <div class="type-info">
          <div class="type-name">{{ option.label }}</div>
          <div class="type-desc">{{ option.desc }}</div>
        </div>
      </div>
    </div>

    <el-form :model="form" :rules="rules" label-position="top" class="create-form">
      <el-form-item :label="t('pageTree.name')" prop="name">
        <el-input
          :model-value="form.name"
          :disabled="isFixedBasicType(form.type)"
          :placeholder="t('pageTree.enterName')"
          maxlength="50"
          show-word-limit
          @update:model-value="handleNameUpdate"
        />
      </el-form-item>

      <el-form-item v-if="form.type === 'page'" :label="t('pageTree.group')">
        <el-select
          :model-value="form.parentId"
          clearable
          :placeholder="t('pageTree.selectGroupOptional')"
          @update:model-value="handleParentIdUpdate"
        >
          <el-option :label="t('pageTree.root')" :value="null">
            <div class="flex items-center gap-2">
              <IconEpFolderOpened class="text-gray-400" />
              <span>{{ t("pageTree.root") }}</span>
            </div>
          </el-option>
          <el-option
            v-for="folder in folderOptions"
            :key="folder.id"
            :label="folder.name"
            :value="folder.id"
          >
            <div class="flex items-center gap-2">
              <IconEpFolder class="text-yellow-500" />
              <span>{{ folder.name }}</span>
            </div>
          </el-option>
        </el-select>
      </el-form-item>
    </el-form>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="open = false">{{ t("pageTree.cancel") }}</el-button>
        <el-button type="primary" :loading="creating" @click="emit('confirm')">
          <IconEpPlus class="mr-1" />
          {{ t("pageTree.create") }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<style scoped>
.create-type-selector {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 10px;
  margin-bottom: 16px;
}

.type-option {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 11px;
  border: 1px solid var(--designer-border-color, #e6e6e6);
  border-radius: 8px;
  background: #fff;
  cursor: pointer;
  transition:
    border-color 0.2s ease,
    background-color 0.2s ease,
    box-shadow 0.2s ease;
}

.type-option:hover {
  border-color: rgba(37, 99, 235, 0.16);
  background-color: var(--designer-hover-surface, #f5f5f5);
  box-shadow: inset 0 0 0 1px rgba(37, 99, 235, 0.04);
}

.type-option.is-active {
  border-color: rgba(37, 99, 235, 0.24);
  background: rgba(37, 99, 235, 0.06);
  box-shadow: inset 0 0 0 1px rgba(37, 99, 235, 0.08);
}

.dark .type-option {
  border-color: #3a3a3a;
}

.dark .type-option:hover {
  border-color: #4a4a4a;
  background-color: #2a2a2a;
}

.dark .type-option.is-active {
  border-color: #409eff;
  background-color: rgba(64, 158, 255, 0.1);
}

.type-icon {
  width: 24px;
  height: 24px;
  color: var(--designer-text-muted, #8c8c8c);
  flex-shrink: 0;
}

.type-option.is-active .type-icon {
  color: var(--designer-primary-text, #2563eb);
}

.type-info {
  flex: 1;
  min-width: 0;
}

.type-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--designer-text-primary, #191919);
  margin-bottom: 2px;
}

.dark .type-name {
  color: #e5e7eb;
}

.type-desc {
  font-size: 11px;
  color: var(--designer-text-muted, #8c8c8c);
  line-height: 1.4;
}

.create-form {
  margin-top: 12px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
