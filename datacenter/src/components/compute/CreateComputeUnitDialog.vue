<template>
  <DcDialog
    v-model="visible"
    title="新建计算单元"
    width="520px"
    body-max-height="420px"
    :dirty="isDirty"
    :close-disabled="loading"
    @close="resetForm"
  >
    <el-form label-position="top" class="compute-create-dialog">
      <el-form-item label="名称" required>
        <el-input
          v-model="form.name"
          maxlength="60"
          show-word-limit
          placeholder="输入计算单元名称"
        />
      </el-form-item>

      <el-form-item label="语言">
        <el-select v-model="form.lang" class="compute-create-dialog__select">
          <el-option label="JavaScript" value="javascript" />
          <el-option label="Python" value="python" />
        </el-select>
      </el-form-item>

      <el-form-item label="文件夹">
        <el-select
          v-model="form.folderId"
          class="compute-create-dialog__select"
          clearable
          placeholder="根目录"
        >
          <el-option label="根目录" :value="null" />
          <el-option
            v-for="folder in folderOptions"
            :key="folder.id"
            :label="folder.label"
            :value="folder.id"
          />
        </el-select>
      </el-form-item>

      <el-form-item label="描述">
        <el-input
          v-model="form.description"
          type="textarea"
          :rows="3"
          maxlength="200"
          show-word-limit
          placeholder="可选"
        />
      </el-form-item>

      <div v-if="error" class="compute-create-dialog__error">{{ error }}</div>
    </el-form>

    <template #footer>
      <div class="compute-create-dialog__footer">
        <el-button @click="visible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="loading"
          :disabled="!canSubmit"
          @click="submit"
        >
          创建
        </el-button>
      </div>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import type {
  ComputeFolder,
  ComputeUnitSave,
} from "@/api/schemas/compute.schema";
import DcDialog from "@/components/shared/DcDialog.vue";

const props = withDefaults(
  defineProps<{
    modelValue: boolean;
    folders: ComputeFolder[];
    loading?: boolean;
    error?: string;
  }>(),
  {
    loading: false,
    error: "",
  },
);

const emit = defineEmits<{
  (event: "update:modelValue", value: boolean): void;
  (event: "submit", data: ComputeUnitSave): void;
}>();

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit("update:modelValue", value),
});

const form = reactive<ComputeUnitSave>({
  name: "",
  lang: "javascript",
  folderId: null,
  description: "",
  code: "",
});

const canSubmit = computed(() => form.name.trim().length > 0 && !props.loading);
const isDirty = computed(
  () =>
    visible.value &&
    (form.name.trim().length > 0 ||
      form.description.trim().length > 0 ||
      form.lang !== "javascript" ||
      Boolean(form.folderId) ||
      Boolean(form.code)),
);

const flattenFolders = (
  folders: ComputeFolder[],
  depth = 0,
): Array<{ id: string; label: string }> =>
  folders.flatMap((folder) => [
    { id: String(folder.id), label: `${"　".repeat(depth)}${folder.name}` },
    ...flattenFolders(folder.children || [], depth + 1),
  ]);

const folderOptions = computed(() => flattenFolders(props.folders));

function resetForm() {
  form.name = "";
  form.lang = "javascript";
  form.folderId = null;
  form.description = "";
  form.code = "";
}

function submit() {
  if (!canSubmit.value) return;
  emit("submit", {
    name: form.name.trim(),
    lang: form.lang,
    folderId: form.folderId || null,
    description: form.description || "",
    code: form.code || "",
  });
}

watch(
  () => props.modelValue,
  (open) => {
    if (open) resetForm();
  },
);
</script>

<style scoped>
.compute-create-dialog {
  display: grid;
  gap: 2px;
}

.compute-create-dialog__select {
  width: 100%;
}

.compute-create-dialog__error {
  padding: 9px 10px;
  border: 1px solid rgba(220, 38, 38, 0.2);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
  font-size: 13px;
  line-height: 1.5;
}

.compute-create-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
