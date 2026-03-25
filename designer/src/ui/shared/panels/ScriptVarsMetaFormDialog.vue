<!--
  脚本面板：定时器 / 变量监听 / 自定义脚本 元数据新建或编辑
-->
<script setup>
import {
  ElButton,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElInputNumber,
  ElOption,
  ElSelect,
} from "element-plus";

defineProps({
  title: { type: String, required: true },
  /** timers | variableChanges | custom */
  module: { type: String, required: true },
  /** @type {string[]} */
  projectVariableNames: { type: Array, default: () => [] },
});
const emit = defineEmits(["confirm"]);
const visible = defineModel({ type: Boolean, default: false });
const meta = defineModel("meta", { type: Object, required: true });

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
    :title="title"
    width="420px"
    :close-on-click-modal="false"
    :lock-scroll="false"
  >
    <ElForm label-width="90px">
      <template v-if="module === 'timers'">
        <ElFormItem label="定时器名称">
          <ElInput v-model="meta.name" />
        </ElFormItem>
        <ElFormItem label="时间(ms)">
          <ElInputNumber v-model="meta.interval" :min="100" :step="100" style="width: 100%" />
        </ElFormItem>
        <ElFormItem label="描述">
          <ElInput v-model="meta.description" />
        </ElFormItem>
      </template>
      <template v-else-if="module === 'variableChanges'">
        <ElFormItem label="变量">
          <ElSelect v-model="meta.variable" placeholder="请选择变量">
            <ElOption
              v-for="name in projectVariableNames"
              :key="name"
              :label="name"
              :value="name"
            />
          </ElSelect>
        </ElFormItem>
        <ElFormItem label="描述">
          <ElInput v-model="meta.description" />
        </ElFormItem>
      </template>
      <template v-else-if="module === 'custom'">
        <ElFormItem label="函数名称">
          <ElInput v-model="meta.name" />
        </ElFormItem>
        <ElFormItem label="入参">
          <ElInput v-model="meta.params" placeholder="例如: id, value" />
        </ElFormItem>
        <ElFormItem label="描述">
          <ElInput v-model="meta.description" />
        </ElFormItem>
      </template>
    </ElForm>
    <template #footer>
      <ElButton @click="handleCancel">取消</ElButton>
      <ElButton type="primary" @click="handleConfirm">确定</ElButton>
    </template>
  </ElDialog>
</template>
