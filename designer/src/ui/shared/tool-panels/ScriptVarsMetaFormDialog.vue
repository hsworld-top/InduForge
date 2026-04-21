<!--
  脚本面板：定时器 / 变量监听 / 自定义脚本 元数据新建或编辑
-->
<script setup lang="ts">
import { useI18n } from "vue-i18n";

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
const { t } = useI18n();

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
        <el-form-item :label="t('scriptPanel.metaDialog.timerName')">
          <el-input v-model="meta.name" />
        </el-form-item>
        <el-form-item :label="t('scriptPanel.metaDialog.timeMs')">
          <el-input-number v-model="meta.interval" :min="100" :step="100" style="width: 100%" />
        </el-form-item>
        <el-form-item :label="t('scriptPanel.metaDialog.description')">
          <el-input v-model="meta.description" />
        </el-form-item>
      </template>
      <template v-else-if="module === 'variableChanges'">
        <el-form-item :label="t('scriptPanel.metaDialog.variable')">
          <el-select v-model="meta.variable" :placeholder="t('scriptPanel.metaDialog.selectVariable')">
            <el-option
              v-for="name in projectVariableNames"
              :key="name"
              :label="name"
              :value="name"
            />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('scriptPanel.metaDialog.description')">
          <el-input v-model="meta.description" />
        </el-form-item>
      </template>
      <template v-else-if="module === 'custom'">
        <el-form-item :label="t('scriptPanel.metaDialog.functionName')">
          <el-input v-model="meta.name" />
        </el-form-item>
        <el-form-item :label="t('scriptPanel.metaDialog.params')">
          <el-input v-model="meta.params" :placeholder="t('scriptPanel.metaDialog.paramsPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('scriptPanel.metaDialog.description')">
          <el-input v-model="meta.description" />
        </el-form-item>
      </template>
    </el-form>
    <template #footer>
      <el-button @click="handleCancel">{{ t("scriptPanel.metaDialog.cancel") }}</el-button>
      <el-button type="primary" @click="handleConfirm">{{ t("scriptPanel.metaDialog.confirm") }}</el-button>
    </template>
  </el-dialog>
</template>
