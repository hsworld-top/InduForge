<template>
  <div class="compute-panel">
    <el-alert
      type="info"
      :closable="false"
      show-icon
      :title="t('compute.title')"
      class="mb-4"
    />

    <el-form label-width="100px" class="mb-4">
      <el-row :gutter="16">
        <el-col :xs="24" :md="12">
          <el-form-item :label="t('compute.name')">
            <el-input v-model="form.name" placeholder="calc.sum" />
          </el-form-item>
        </el-col>
        <el-col :xs="24" :md="12">
          <el-form-item :label="t('compute.language')">
            <el-select v-model="form.language" class="w-full">
              <el-option label="JavaScript" value="js" />
              <el-option label="Python" value="python" />
            </el-select>
          </el-form-item>
        </el-col>
      </el-row>

      <el-row :gutter="16">
        <el-col :xs="24" :md="12">
          <el-form-item :label="t('compute.timeout')">
            <el-input-number
              v-model="form.timeoutMs"
              :min="100"
              :max="120000"
              :step="100"
            />
          </el-form-item>
        </el-col>
        <el-col :xs="24" :md="12">
          <el-form-item :label="t('compute.unitId')">
            <el-input
              v-model="computeUnitId"
              :placeholder="t('compute.unitIdPlaceholder')"
            />
          </el-form-item>
        </el-col>
      </el-row>

      <el-form-item :label="t('compute.script')">
        <el-input
          v-model="form.scriptCode"
          type="textarea"
          :autosize="{ minRows: 8, maxRows: 14 }"
          :placeholder="t('compute.scriptPlaceholder')"
        />
      </el-form-item>

      <el-form-item :label="t('compute.inputJson')">
        <el-input
          v-model="inputJSON"
          type="textarea"
          :autosize="{ minRows: 4, maxRows: 8 }"
          :placeholder="t('compute.inputJsonPlaceholder')"
        />
      </el-form-item>

      <el-form-item>
        <el-button type="primary" :loading="creating" @click="handleCreate">
          {{ t("actions.createUnit") }}
        </el-button>
        <el-button :loading="running" @click="handleRun">{{
          t("actions.run")
        }}</el-button>
        <el-button :loading="debugging" @click="handleDebug">{{
          t("actions.debug")
        }}</el-button>
      </el-form-item>
    </el-form>

    <el-card shadow="never">
      <template #header>{{ t("compute.resultTitle") }}</template>
      <pre class="result-box">{{ outputText }}</pre>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { ElMessage } from "element-plus";
import dataAPI from "@/api/data.api";
import { t } from "@/i18n/runtime";
import { getApiErrorMessage } from "@/utils/request";

const props = defineProps({
  projectId: {
    type: String,
    default: "",
  },
});

const form = ref({
  name: "calc.sum",
  language: "js",
  timeoutMs: 3000,
  scriptCode: "result = (input.a || 0) + (input.b || 0);",
});

const computeUnitId = ref("");
const inputJSON = ref('{"a":1,"b":2}');
const outputText = ref(t("compute.notExecuted"));

const creating = ref(false);
const running = ref(false);
const debugging = ref(false);

const currentProjectId = computed(() => (props.projectId || "").trim());

const parseInput = () => {
  if (!inputJSON.value.trim()) {
    return {};
  }
  try {
    return JSON.parse(inputJSON.value);
  } catch {
    throw new Error(t("compute.invalidJson"));
  }
};

const ensureProject = () => {
  if (!currentProjectId.value) {
    throw new Error(t("compute.missingProjectId"));
  }
};

const ensureUnitID = () => {
  if (!computeUnitId.value.trim()) {
    throw new Error(t("compute.missingUnitId"));
  }
};

const handleCreate = async () => {
  try {
    ensureProject();
    creating.value = true;
    const response = await dataAPI.createComputeUnit(currentProjectId.value, {
      name: form.value.name,
      language: form.value.language,
      timeoutMs: form.value.timeoutMs,
      scriptCode: form.value.scriptCode,
      triggerType: "manual",
    });
    computeUnitId.value = response.data?.id || "";
    outputText.value = JSON.stringify(response.data, null, 2);
    ElMessage.success(t("compute.createSuccess"));
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t("compute.createFailed")));
  } finally {
    creating.value = false;
  }
};

const handleRun = async () => {
  try {
    ensureProject();
    ensureUnitID();
    const input = parseInput();
    running.value = true;
    const response = await dataAPI.runComputeUnit(
      currentProjectId.value,
      computeUnitId.value.trim(),
      input,
    );
    outputText.value = JSON.stringify(response.data, null, 2);
    ElMessage.success(t("compute.runSuccess"));
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t("compute.runFailed")));
  } finally {
    running.value = false;
  }
};

const handleDebug = async () => {
  try {
    ensureProject();
    ensureUnitID();
    const input = parseInput();
    debugging.value = true;
    const response = await dataAPI.debugComputeUnit(
      currentProjectId.value,
      computeUnitId.value.trim(),
      input,
    );
    outputText.value = JSON.stringify(response.data, null, 2);
    ElMessage.success(t("compute.debugSuccess"));
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t("compute.debugFailed")));
  } finally {
    debugging.value = false;
  }
};
</script>

<style scoped>
.compute-panel {
  height: 100%;
  padding: 16px;
  overflow: auto;
}

.result-box {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 12px;
  line-height: 1.5;
}
</style>
