<!--
  数据点面板：新增/编辑变量表单（含 Monaco 初始值）
-->
<script setup>
import {
  ElButton,
  ElDatePicker,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElInputNumber,
  ElOption,
  ElSelect,
  ElSwitch,
} from "element-plus";
import MonacoEditor from "@/components/common/monaco-editor-async";

defineProps({
  editMode: { type: Boolean, default: false },
  /** @type {string[]} */
  types: { type: Array, required: true },
  /** @type {{ id: string, name: string }[]} */
  groupOptions: { type: Array, default: () => [] },
  rootGroupId: { type: String, required: true },
  isEditorType: { type: Boolean, default: false },
  isTextType: { type: Boolean, default: false },
  editorLanguage: { type: String, default: "json" },
});
const emit = defineEmits(["confirm", "typeChange", "editValueMarkers"]);
const visible = defineModel({ type: Boolean, default: false });
const editName = defineModel("editName", { type: String, default: "" });
const editGroupId = defineModel("editGroupId", { type: String, default: "" });
const editType = defineModel("editType", { type: String, default: "string" });
const editValue = defineModel("editValue", { default: "" });
const editDescription = defineModel("editDescription", { type: String, default: "" });
const mapped = defineModel("mapped", { type: Boolean, default: false });
const mappedField = defineModel("mappedField", { type: String, default: "" });
const mappedSourceLabel = defineModel("mappedSourceLabel", { type: String, default: "" });

function handleMarkers(payload) {
  emit("editValueMarkers", payload);
}
</script>

<template>
  <ElDialog
    v-model="visible"
    :title="editMode ? '编辑变量' : '新增变量'"
    width="520px"
    :close-on-click-modal="false"
    :lock-scroll="false"
  >
    <ElForm label-width="80px">
      <ElFormItem label="变量名">
        <ElInput v-model="editName" />
      </ElFormItem>
      <ElFormItem label="分组">
        <ElSelect v-model="editGroupId" placeholder="请选择分组">
          <ElOption label="根目录" :value="rootGroupId" />
          <ElOption
            v-for="group in groupOptions"
            :key="group.id"
            :label="group.name"
            :value="group.id"
          />
        </ElSelect>
      </ElFormItem>
      <ElFormItem label="类型">
        <ElSelect v-model="editType" :disabled="mapped" @change="emit('typeChange')">
          <ElOption v-for="t in types" :key="t" :label="t" :value="t" />
        </ElSelect>
      </ElFormItem>
      <ElFormItem label="初始值">
        <div v-if="isEditorType" class="edit-value-block">
          <MonacoEditor
            v-model="editValue"
            :language="editorLanguage"
            height="136px"
            @markers="handleMarkers"
          />
        </div>
        <ElInput v-else-if="isTextType" v-model="editValue" type="textarea" :rows="6" />
        <ElInputNumber v-else-if="editType === 'number'" v-model="editValue" style="width: 100%" />
        <ElSwitch v-else-if="editType === 'boolean'" v-model="editValue" />
        <ElDatePicker
          v-else-if="editType === 'date'"
          v-model="editValue"
          type="datetime"
          style="width: 100%"
        />
      </ElFormItem>
      <ElFormItem label="描述">
        <ElInput v-model="editDescription" type="textarea" :rows="2" />
      </ElFormItem>
      <ElFormItem label="映射">
        <ElSwitch v-model="mapped" />
      </ElFormItem>
      <template v-if="mapped">
        <ElFormItem label="路径">
          <ElInput v-model="mappedField" disabled />
        </ElFormItem>
        <ElFormItem label="来源">
          <ElInput v-model="mappedSourceLabel" disabled />
        </ElFormItem>
      </template>
    </ElForm>
    <template #footer>
      <ElButton @click="visible = false">取消</ElButton>
      <ElButton type="primary" @click="emit('confirm')">确定</ElButton>
    </template>
  </ElDialog>
</template>

<style scoped>
.edit-value-block {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
</style>
