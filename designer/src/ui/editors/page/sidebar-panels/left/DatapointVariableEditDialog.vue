<!--
  数据点面板：新增/编辑变量表单（含 Monaco 初始值）
-->
<script setup lang="ts">
import { useI18n } from "vue-i18n";
import MonacoEditor from "@/ui/shared/widgets/base/monaco-editor-async";

interface DatapointGroupOptionLike {
  id: string;
  name: string;
}

defineProps<{
  editMode?: boolean;
  types: string[];
  groupOptions?: DatapointGroupOptionLike[];
  rootGroupId: string;
  isEditorType?: boolean;
  isTextType?: boolean;
  editorLanguage?: string;
}>();
const emit = defineEmits<{
  (event: "confirm"): void;
  (event: "typeChange"): void;
  (event: "editValueMarkers", payload: unknown): void;
}>();
const visible = defineModel<boolean>({ default: false });
const editName = defineModel<string>("editName", { default: "" });
const editGroupId = defineModel<string>("editGroupId", { default: "" });
const editType = defineModel<string>("editType", { default: "string" });
const editValue = defineModel<any>("editValue", { default: "" });
const editDescription = defineModel<string>("editDescription", { default: "" });
const mapped = defineModel<boolean>("mapped", { default: false });
const mappedField = defineModel<string>("mappedField", { default: "" });
const mappedSourceLabel = defineModel<string>("mappedSourceLabel", { default: "" });
const { t } = useI18n();

function handleMarkers(payload: unknown) {
  emit("editValueMarkers", payload);
}
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="editMode ? t('datapointPanel.variableDialog.editTitle') : t('datapointPanel.variableDialog.createTitle')"
    width="520px"
    :close-on-click-modal="false"
    :lock-scroll="false"
  >
    <el-form label-width="80px">
      <el-form-item :label="t('datapointPanel.variableDialog.name')">
        <el-input v-model="editName" />
      </el-form-item>
      <el-form-item :label="t('datapointPanel.variableDialog.group')">
        <el-select v-model="editGroupId" :placeholder="t('datapointPanel.variableDialog.selectGroup')">
          <el-option :label="t('datapointPanel.variableDialog.root')" :value="rootGroupId" />
          <el-option
            v-for="group in groupOptions"
            :key="group.id"
            :label="group.name"
            :value="group.id"
          />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('datapointPanel.variableDialog.type')">
        <el-select v-model="editType" :disabled="mapped" @change="emit('typeChange')">
          <el-option v-for="t in types" :key="t" :label="t" :value="t" />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('datapointPanel.variableDialog.initialValue')">
        <div v-if="isEditorType" class="edit-value-block">
          <MonacoEditor
            v-model="editValue"
            :language="editorLanguage"
            height="136px"
            @markers="handleMarkers"
          />
        </div>
        <el-input v-else-if="isTextType" v-model="editValue" type="textarea" :rows="6" />
        <el-input-number
          v-else-if="editType === 'number'"
          v-model="editValue"
          style="width: 100%"
        />
        <el-switch v-else-if="editType === 'boolean'" v-model="editValue" />
        <el-date-picker
          v-else-if="editType === 'date'"
          v-model="editValue"
          type="datetime"
          style="width: 100%"
        />
      </el-form-item>
      <el-form-item :label="t('datapointPanel.variableDialog.description')">
        <el-input v-model="editDescription" type="textarea" :rows="2" />
      </el-form-item>
      <el-form-item :label="t('datapointPanel.variableDialog.mapping')">
        <el-switch v-model="mapped" />
      </el-form-item>
      <template v-if="mapped">
        <el-form-item :label="t('datapointPanel.variableDialog.path')">
          <el-input v-model="mappedField" disabled />
        </el-form-item>
        <el-form-item :label="t('datapointPanel.variableDialog.source')">
          <el-input v-model="mappedSourceLabel" disabled />
        </el-form-item>
      </template>
    </el-form>
    <template #footer>
      <el-button @click="visible = false">{{ t("datapointPanel.variableDialog.cancel") }}</el-button>
      <el-button type="primary" @click="emit('confirm')">{{ t("datapointPanel.variableDialog.confirm") }}</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.edit-value-block {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
</style>
