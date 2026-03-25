<!--
  脚本面板：定时器 / 变量监听 / 自定义脚本 元数据新建或编辑
-->
<script setup lang="ts">
interface ScriptVarsMetaLike {
  name?: string;
  interval?: number;
  description?: string;
  variable?: string;
  params?: string;
}

defineProps<{
  title: string;
  /** timers | variableChanges | custom */
  module: "timers" | "variableChanges" | "custom";
  projectVariableNames?: string[];
}>();

const emit = defineEmits<{
  (event: "confirm"): void;
}>();

const visible = defineModel<boolean>({ default: false });
const meta = defineModel<ScriptVarsMetaLike>("meta", { required: true });

function handleCancel() {
  visible.value = false;
}

function handleConfirm() {
  emit("confirm");
}
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="title"
    width="420px"
    :close-on-click-modal="false"
    :lock-scroll="false"
  >
    <el-form label-width="90px">
      <template v-if="module === 'timers'">
        <el-form-item label="定时器名称">
          <el-input v-model="meta.name" />
        </el-form-item>
        <el-form-item label="时间(ms)">
          <el-input-number v-model="meta.interval" :min="100" :step="100" style="width: 100%" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="meta.description" />
        </el-form-item>
      </template>
      <template v-else-if="module === 'variableChanges'">
        <el-form-item label="变量">
          <el-select v-model="meta.variable" placeholder="请选择变量">
            <el-option
              v-for="name in projectVariableNames"
              :key="name"
              :label="name"
              :value="name"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="meta.description" />
        </el-form-item>
      </template>
      <template v-else-if="module === 'custom'">
        <el-form-item label="函数名称">
          <el-input v-model="meta.name" />
        </el-form-item>
        <el-form-item label="入参">
          <el-input v-model="meta.params" placeholder="例如: id, value" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="meta.description" />
        </el-form-item>
      </template>
    </el-form>
    <template #footer>
      <el-button @click="handleCancel">取消</el-button>
      <el-button type="primary" @click="handleConfirm">确定</el-button>
    </template>
  </el-dialog>
</template>
