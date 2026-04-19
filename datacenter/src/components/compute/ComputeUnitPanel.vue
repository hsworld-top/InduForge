<template>
  <div class="compute-panel">
    <el-alert
      type="info"
      :closable="false"
      show-icon
      title="最小联调面板：先支持创建、运行、调试 compute 单元"
      class="mb-4"
    />

    <el-form label-width="100px" class="mb-4">
      <el-row :gutter="16">
        <el-col :xs="24" :md="12">
          <el-form-item label="名称">
            <el-input v-model="form.name" placeholder="例如：calc.sum" />
          </el-form-item>
        </el-col>
        <el-col :xs="24" :md="12">
          <el-form-item label="语言">
            <el-select v-model="form.language" class="w-full">
              <el-option label="JavaScript" value="js" />
              <el-option label="Python" value="python" />
            </el-select>
          </el-form-item>
        </el-col>
      </el-row>

      <el-row :gutter="16">
        <el-col :xs="24" :md="12">
          <el-form-item label="超时(ms)">
            <el-input-number
              v-model="form.timeoutMs"
              :min="100"
              :max="120000"
              :step="100"
            />
          </el-form-item>
        </el-col>
        <el-col :xs="24" :md="12">
          <el-form-item label="单元ID">
            <el-input
              v-model="computeUnitId"
              placeholder="创建后自动回填，也可手动输入已有ID"
            />
          </el-form-item>
        </el-col>
      </el-row>

      <el-form-item label="脚本">
        <el-input
          v-model="form.scriptCode"
          type="textarea"
          :autosize="{ minRows: 8, maxRows: 14 }"
          placeholder="JavaScript 示例：result = (input.a || 0) + (input.b || 0);"
        />
      </el-form-item>

      <el-form-item label="输入JSON">
        <el-input
          v-model="inputJSON"
          type="textarea"
          :autosize="{ minRows: 4, maxRows: 8 }"
          placeholder='例如：{"a":1,"b":2}'
        />
      </el-form-item>

      <el-form-item>
        <el-button type="primary" :loading="creating" @click="handleCreate">
          创建单元
        </el-button>
        <el-button :loading="running" @click="handleRun">运行</el-button>
        <el-button :loading="debugging" @click="handleDebug">调试</el-button>
      </el-form-item>
    </el-form>

    <el-card shadow="never">
      <template #header>运行结果</template>
      <pre class="result-box">{{ outputText }}</pre>
    </el-card>
  </div>
</template>

<script setup>
import { computed, ref } from "vue";
import { ElMessage } from "element-plus";
import dataAPI from "@/api/data.api";

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
const outputText = ref("尚未执行");

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
    throw new Error("输入 JSON 格式无效");
  }
};

const ensureProject = () => {
  if (!currentProjectId.value) {
    throw new Error("projectId 缺失，无法调用 compute 接口");
  }
};

const ensureUnitID = () => {
  if (!computeUnitId.value.trim()) {
    throw new Error("请先创建单元或手动输入单元ID");
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
    ElMessage.success("计算单元创建成功");
  } catch (error) {
    ElMessage.error(error.response?.data?.message || error.message || "创建失败");
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
    ElMessage.success("运行成功");
  } catch (error) {
    ElMessage.error(error.response?.data?.message || error.message || "运行失败");
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
    ElMessage.success("调试成功");
  } catch (error) {
    ElMessage.error(error.response?.data?.message || error.message || "调试失败");
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
