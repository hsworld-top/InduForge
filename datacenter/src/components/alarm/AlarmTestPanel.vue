<template>
  <section class="alarm-test-panel">
    <div class="alarm-test-panel__input">
      <div class="alarm-test-panel__bar">
        <div>
          <strong>样本输入</strong>
          <span>JSON Payload</span>
        </div>
        <button type="button" :disabled="running" @click="runTrial">
          {{ running ? "试算中" : "运行试算" }}
        </button>
      </div>

      <textarea
        v-model="sampleText"
        spellcheck="false"
        :disabled="running"
        aria-label="报警试算 JSON 样本"
      />

      <p v-if="parseError" class="alarm-test-panel__error">{{ parseError }}</p>
      <p v-else-if="error" class="alarm-test-panel__error">{{ error }}</p>
    </div>

    <div class="alarm-test-panel__result">
      <div class="alarm-test-panel__metrics">
        <div>
          <span>state</span>
          <strong>{{ result?.state || "-" }}</strong>
        </div>
        <div>
          <span>triggered</span>
          <strong>{{ result ? String(result.triggered) : "-" }}</strong>
        </div>
      </div>

      <div class="alarm-test-panel__block">
        <span>message</span>
        <strong>{{ result?.message || "无输出" }}</strong>
      </div>

      <div class="alarm-test-panel__block">
        <span>diagnostics</span>
        <pre>{{ diagnosticsText }}</pre>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import type {
  AlarmTrialPayload,
  AlarmTrialResult,
} from "@/api/schemas/alarm.schema";

const props = defineProps<{
  result: AlarmTrialResult | null;
  running: boolean;
  error: string;
}>();

const emit = defineEmits<{
  run: [payload: AlarmTrialPayload];
}>();

const sampleText = ref(
  JSON.stringify(
    {
      value: 86.5,
      timestamp: "2026-05-20 10:00:00",
      context: {
        source: "manual-trial",
        quality: "good",
      },
    },
    null,
    2,
  ),
);
const parseError = ref("");

const diagnosticsText = computed(() => {
  const diagnostics = props.result?.diagnostics ?? {};
  return JSON.stringify(diagnostics, null, 2);
});

const runTrial = () => {
  parseError.value = "";
  try {
    const payload = JSON.parse(sampleText.value) as unknown;
    if (!payload || typeof payload !== "object" || Array.isArray(payload)) {
      parseError.value = "JSON 样本必须是对象";
      return;
    }
    emit("run", payload as AlarmTrialPayload);
  } catch (error) {
    parseError.value =
      error instanceof Error ? `JSON 解析失败：${error.message}` : "JSON 解析失败";
  }
};
</script>

<style scoped>
.alarm-test-panel {
  height: 100%;
  min-height: 0;
  display: grid;
  grid-template-columns: minmax(320px, 0.95fr) minmax(320px, 1.05fr);
  gap: 12px;
}

.alarm-test-panel__input,
.alarm-test-panel__result {
  min-height: 0;
  display: grid;
  gap: 10px;
  align-content: start;
}

.alarm-test-panel__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.alarm-test-panel__bar strong,
.alarm-test-panel__bar span,
.alarm-test-panel__block span,
.alarm-test-panel__metrics span {
  display: block;
}

.alarm-test-panel__bar strong {
  color: var(--dc-text);
  font-size: 13px;
}

.alarm-test-panel__bar span,
.alarm-test-panel__block span,
.alarm-test-panel__metrics span {
  color: var(--dc-text-muted);
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
}

.alarm-test-panel__bar button {
  height: 30px;
  padding: 0 12px;
  border: 1px solid var(--dc-primary);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary);
  color: #fff;
  cursor: pointer;
  font-family: inherit;
  font-size: 12px;
  font-weight: 700;
}

.alarm-test-panel__bar button:disabled {
  cursor: not-allowed;
  opacity: 0.58;
}

.alarm-test-panel textarea,
.alarm-test-panel pre {
  min-height: 0;
  margin: 0;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
  font-family: var(--dc-font-mono);
  font-size: 12px;
  line-height: 1.55;
}

.alarm-test-panel textarea {
  width: 100%;
  height: 156px;
  resize: none;
  padding: 10px;
  outline: none;
}

.alarm-test-panel pre {
  max-height: 112px;
  overflow: auto;
  padding: 10px;
  white-space: pre-wrap;
  word-break: break-word;
}

.alarm-test-panel__metrics {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.alarm-test-panel__metrics > div,
.alarm-test-panel__block {
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
}

.alarm-test-panel__metrics strong,
.alarm-test-panel__block strong {
  display: block;
  margin-top: 5px;
  overflow: hidden;
  color: var(--dc-text);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.alarm-test-panel__error {
  margin: 0;
  color: var(--dc-danger, #b91c1c);
  font-size: 12px;
}

@media (max-width: 920px) {
  .alarm-test-panel {
    grid-template-columns: 1fr;
  }
}
</style>
