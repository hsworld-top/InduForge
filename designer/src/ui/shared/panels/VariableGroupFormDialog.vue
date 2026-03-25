<!--
  新建/编辑变量分组（脚本侧与数据点侧共用表单壳）
-->
<script setup>
import { ElButton, ElDialog, ElForm, ElFormItem, ElInput, ElOption, ElSelect } from "element-plus";

defineProps({
  editMode: { type: Boolean, default: false },
  /** @type {{ id: string, name: string }[]} */
  parentOptions: { type: Array, default: () => [] },
});
const emit = defineEmits(["confirm"]);
const visible = defineModel({ type: Boolean, default: false });
const name = defineModel("name", { type: String, default: "" });
const parentId = defineModel("parentId", { default: null });

function handleCancel() {
  visible.value = false;
}

function handleConfirm() {
  emit("confirm");
}
</script>

<template>
  <ElDialog
    v-model="visible"
    :title="editMode ? '编辑分组' : '新建分组'"
    width="420px"
    :close-on-click-modal="false"
    :lock-scroll="false"
  >
    <ElForm label-width="70px">
      <ElFormItem label="名称">
        <ElInput v-model="name" />
      </ElFormItem>
      <ElFormItem label="父级">
        <ElSelect v-model="parentId" placeholder="根目录">
          <ElOption label="根目录" :value="null" />
          <ElOption
            v-for="group in parentOptions"
            :key="group.id"
            :label="group.name"
            :value="group.id"
          />
        </ElSelect>
      </ElFormItem>
    </ElForm>
    <template #footer>
      <ElButton @click="handleCancel">取消</ElButton>
      <ElButton type="primary" @click="handleConfirm">确定</ElButton>
    </template>
  </ElDialog>
</template>
